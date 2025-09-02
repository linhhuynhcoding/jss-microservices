package service

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/linhhuynhcoding/jss-microservices/order-service/config"
    "github.com/linhhuynhcoding/jss-microservices/order-service/internal/adapter"
    "github.com/linhhuynhcoding/jss-microservices/order-service/internal/domain"
    "github.com/linhhuynhcoding/jss-microservices/order-service/internal/repository"
    orderpb "github.com/linhhuynhcoding/jss-microservices/rpc/gen/order"
    // Import the auth package anonymously so that generated types are referenced
    _ "github.com/linhhuynhcoding/jss-microservices/rpc/gen/auth"
    productpb "github.com/linhhuynhcoding/jss-microservices/rpc/gen/product"
    loyaltypb "github.com/linhhuynhcoding/jss-microservices/rpc/gen/loyalty"
    notificationpb "github.com/linhhuynhcoding/jss-microservices/rpc/gen/notification"
    mqconfig "github.com/linhhuynhcoding/jss-microservices/mq/config"
    "github.com/linhhuynhcoding/jss-microservices/mq"
    "go.mongodb.org/mongo-driver/mongo"
    "go.uber.org/zap"
    "google.golang.org/grpc/metadata"
    "google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the OrderService gRPC server defined in order.proto.
// It orchestrates calls to the auth, product and loyalty services while
// persisting order data to MongoDB and publishing notification events
// via RabbitMQ.
type Service struct {
    orderpb.UnimplementedOrderServiceServer
    repo          *repository.OrderRepository
    authClient    *adapter.AuthClient
    productClient *adapter.ProductClient
    loyaltyClient *adapter.LoyaltyClient
    publisher     *mq.Publisher
    logger        *zap.Logger
}

// New constructs a new Service.  The provided MongoDB database must be
// connected; it will be used to create the OrderRepository.  Clients
// connecting to external services are created here so that a single
// connection is reused for all requests.  If any client fails to
// initialise an error is returned.
func New(cfg config.Config, db *mongo.Database, log *zap.Logger) (*Service, error) {
    repo := repository.New(db)

    // Initialise gRPC clients
    authClient, err := adapter.NewAuthClient(cfg.AuthServiceAddr, log)
    if err != nil {
        return nil, fmt.Errorf("failed to create auth client: %w", err)
    }
    productClient, err := adapter.NewProductClient(cfg.ProductServiceAddr, log)
    if err != nil {
        return nil, fmt.Errorf("failed to create product client: %w", err)
    }
    loyaltyClient, err := adapter.NewLoyaltyClient(cfg.LoyaltyServiceAddr, log)
    if err != nil {
        return nil, fmt.Errorf("failed to create loyalty client: %w", err)
    }

    // Initialise RabbitMQ publisher
    pubCfg := mqconfig.RabbitMQConfig{
        ConnStr:       cfg.RabbitMQURL,
        ExchangeName:  cfg.ExchangeName,
        ExchangeType:  "topic",
        PublisherName: cfg.PublisherName,
    }
    publisher, err := mq.NewPublisher(pubCfg, log)
    if err != nil {
        return nil, fmt.Errorf("failed to create mq publisher: %w", err)
    }

    return &Service{
        repo:          repo,
        authClient:    authClient,
        productClient: productClient,
        loyaltyClient: loyaltyClient,
        publisher:     publisher,
        logger:        log,
    }, nil
}

// Close cleans up all resources held by the service.  It should be
// invoked when shutting down the service to close gRPC connections and
// the RabbitMQ publisher.
func (s *Service) Close() {
    if s.authClient != nil {
        s.authClient.Close()
    }
    if s.productClient != nil {
        s.productClient.Close()
    }
    if s.loyaltyClient != nil {
        s.loyaltyClient.Close()
    }
    if s.publisher != nil {
        s.publisher.Close()
    }
}

// CreateOrder creates a new order.  It validates the JWT from the
// Authorization header, obtains the next order ID, calls the product
// service to update stock and record the order, applies vouchers via
// the loyalty service, persists the order to MongoDB and publishes a
// notification event.  If any step fails an error is returned and
// nothing is persisted.
func (s *Service) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
    s.logger.Info("CreateOrder called", zap.Any("req", req))
    // Extract the bearer token from the incoming metadata
    md, ok := metadata.FromIncomingContext(ctx)
    var accessToken string
    if ok {
        if vals, exists := md["authorization"]; exists && len(vals) > 0 {
            authHeader := vals[0]
            if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
                accessToken = strings.TrimSpace(authHeader[7:])
            }
        }
    }
    if accessToken == "" {
        return nil, fmt.Errorf("missing authorization token")
    }

    // Validate the token via auth service
    valid, userID, role, err := s.authClient.Validate(ctx, accessToken)
    if err != nil {
        s.logger.Error("ValidateToken RPC failed", zap.Error(err))
        return nil, fmt.Errorf("unable to validate token: %w", err)
    }
    if !valid {
        return nil, fmt.Errorf("invalid token")
    }
    // Only staff, manager and admin can create orders
    if role != "STAFF" && role != "MANAGER" && role != "ADMIN" {
        return nil, fmt.Errorf("unauthorised role: %s", role)
    }

    // Ensure there are items to purchase
    if len(req.Items) == 0 {
        return nil, fmt.Errorf("order must contain at least one product")
    }

    // Generate the next sequential order ID
    orderID, err := s.repo.NextOrderID(ctx)
    if err != nil {
        s.logger.Error("failed to get next order id", zap.Error(err))
        return nil, fmt.Errorf("failed to generate order id: %w", err)
    }

    // Build product purchase request
    pReq := &productpb.PurchaseProductRequest{
        CustomerId: req.CustomerId,
        OrderId:    orderID,
        Products:   []*productpb.PurchaseProductRequest_Product{},
    }
    for _, item := range req.Items {
        pReq.Products = append(pReq.Products, &productpb.PurchaseProductRequest_Product{
            ProductId: item.ProductId,
            Quantity:  item.Quantity,
        })
    }
    pResp, err := s.productClient.PurchaseProduct(ctx, pReq)
    if err != nil {
        s.logger.Error("PurchaseProduct RPC failed", zap.Error(err))
        return nil, fmt.Errorf("failed to purchase products: %w", err)
    }

    // Calculate total product amount based on returned selling prices
    totalProductAmount := 0.0
    items := make([]domain.OrderItem, 0, len(req.Items))
    // Build a map of productId -> selling_price for quick lookup
    priceMap := make(map[int32]float64)
    for _, p := range pResp.GetProducts() {
        priceMap[p.GetId()] = p.GetSellingPrice()
    }
    for _, item := range req.Items {
        price, ok := priceMap[item.ProductId]
        if !ok {
            return nil, fmt.Errorf("product price not returned for product_id %d", item.ProductId)
        }
        totalProductAmount += price * float64(item.Quantity)
        items = append(items, domain.OrderItem{
            ProductID: item.ProductId,
            Quantity:  item.Quantity,
            UnitPrice: price,
        })
    }

    // Determine shipping cost (defaults to zero if not supplied)
    shippingCost := req.GetShippingCost()

    // Apply vouchers if provided
    discountAmount := 0.0
    if len(req.VoucherCodes) > 0 {
        vReq := &loyaltypb.UsingVoucherRequest{
            Vouchers:           req.VoucherCodes,
            TotalProductAmount: totalProductAmount,
            TotalShippingAmount: shippingCost,
            CustomerId:         req.CustomerId,
            OrderId:            orderID,
        }
        vResp, err := s.loyaltyClient.UsingVoucher(ctx, vReq)
        if err != nil {
            s.logger.Error("UsingVoucher RPC failed", zap.Error(err))
            return nil, fmt.Errorf("failed to apply vouchers: %w", err)
        }
        discountAmount = vResp.GetTotalDiscountAmount()
    }

    finalPrice := totalProductAmount + shippingCost - discountAmount

    // Persist the order in MongoDB
    order := &domain.Order{
        OrderID:       orderID,
        CustomerID:    req.CustomerId,
        StaffID:       userID,
        Items:         items,
        VoucherCodes:  req.VoucherCodes,
        TotalPrice:    totalProductAmount,
        DiscountAmount: discountAmount,
        FinalPrice:    finalPrice,
        ShippingCost:  shippingCost,
        CreatedAt:     time.Now(),
    }
    if err := s.repo.Create(ctx, order); err != nil {
        s.logger.Error("failed to persist order", zap.Error(err))
        return nil, fmt.Errorf("failed to save order: %w", err)
    }

    // Publish a notification event via RabbitMQ
    evt := &notificationpb.CreateNotificationRequest{
        UserId:  fmt.Sprintf("%d", req.CustomerId),
        Role:    role,
        Title:   "Order created", // Title of the notification
        Message: fmt.Sprintf("Your order #%d has been created successfully", orderID),
    }
    if err := s.publisher.SendMessage(evt, "notification.create"); err != nil {
        s.logger.Error("failed to publish notification", zap.Error(err))
        // We don't fail the order creation if notification fails
    }

    // Map domain order to protobuf order for response
    respOrder := &orderpb.Order{
        OrderId:      order.OrderID,
        CustomerId:   order.CustomerID,
        StaffId:      order.StaffID,
        VoucherCodes: order.VoucherCodes,
        TotalPrice:   order.TotalPrice,
        DiscountAmount: order.DiscountAmount,
        FinalPrice:   order.FinalPrice,
        ShippingCost: order.ShippingCost,
        CreatedAt:    timestamppb.New(order.CreatedAt),
    }
    for _, it := range order.Items {
        respOrder.Items = append(respOrder.Items, &orderpb.OrderItem{
            ProductId: it.ProductID,
            Quantity:  it.Quantity,
            UnitPrice: it.UnitPrice,
        })
    }
    return &orderpb.CreateOrderResponse{Order: respOrder}, nil
}

// GetOrder retrieves a single order by its ID.  Access control is
// enforced: staff can only retrieve orders they created while managers
// and admins can retrieve any order.
func (s *Service) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.Order, error) {
    // Authenticate the requester
    md, ok := metadata.FromIncomingContext(ctx)
    var accessToken string
    if ok {
        if vals, exists := md["authorization"]; exists && len(vals) > 0 {
            authHeader := vals[0]
            if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
                accessToken = strings.TrimSpace(authHeader[7:])
            }
        }
    }
    if accessToken == "" {
        return nil, fmt.Errorf("missing authorization token")
    }
    valid, userID, role, err := s.authClient.Validate(ctx, accessToken)
    if err != nil || !valid {
        return nil, fmt.Errorf("unauthorised")
    }
    // Retrieve order from repository
    order, err := s.repo.Get(ctx, req.OrderId)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, fmt.Errorf("order not found")
        }
        return nil, err
    }
    // Enforce that staff can only access their own orders
    if role == "STAFF" && order.StaffID != userID {
        return nil, fmt.Errorf("forbidden")
    }
    // Map to protobuf
    pbOrder := &orderpb.Order{
        OrderId:      order.OrderID,
        CustomerId:   order.CustomerID,
        StaffId:      order.StaffID,
        VoucherCodes: order.VoucherCodes,
        TotalPrice:   order.TotalPrice,
        DiscountAmount: order.DiscountAmount,
        FinalPrice:   order.FinalPrice,
        ShippingCost: order.ShippingCost,
        CreatedAt:    timestamppb.New(order.CreatedAt),
    }
    for _, it := range order.Items {
        pbOrder.Items = append(pbOrder.Items, &orderpb.OrderItem{
            ProductId: it.ProductID,
            Quantity:  it.Quantity,
            UnitPrice: it.UnitPrice,
        })
    }
    return pbOrder, nil
}

// ListOrders returns a paginated list of orders.  Staff see only their
// own orders; managers and admins see all orders.  Pagination metadata
// is returned in the response.  Page numbers are zero based.
func (s *Service) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
    // Authenticate the requester
    md, ok := metadata.FromIncomingContext(ctx)
    var accessToken string
    if ok {
        if vals, exists := md["authorization"]; exists && len(vals) > 0 {
            authHeader := vals[0]
            if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
                accessToken = strings.TrimSpace(authHeader[7:])
            }
        }
    }
    if accessToken == "" {
        return nil, fmt.Errorf("missing authorization token")
    }
    valid, userID, role, err := s.authClient.Validate(ctx, accessToken)
    if err != nil || !valid {
        return nil, fmt.Errorf("unauthorised")
    }
    // Set default pagination values if not provided
    page := req.GetPage()
    limit := req.GetLimit()
    if limit == 0 {
        limit = 10
    }
    // Retrieve list and total count
    orders, total, err := s.repo.List(ctx, page, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to list orders: %w", err)
    }
    // Filter orders for staff role
    filtered := make([]domain.Order, 0, len(orders))
    if role == "STAFF" {
        for _, o := range orders {
            if o.StaffID == userID {
                filtered = append(filtered, o)
            }
        }
    } else {
        filtered = orders
    }
    // Map to protobuf and compute pagination metadata
    pbOrders := make([]*orderpb.Order, 0, len(filtered))
    for _, o := range filtered {
        pb := &orderpb.Order{
            OrderId:      o.OrderID,
            CustomerId:   o.CustomerID,
            StaffId:      o.StaffID,
            VoucherCodes: o.VoucherCodes,
            TotalPrice:   o.TotalPrice,
            DiscountAmount: o.DiscountAmount,
            FinalPrice:   o.FinalPrice,
            ShippingCost: o.ShippingCost,
            CreatedAt:    timestamppb.New(o.CreatedAt),
        }
        for _, it := range o.Items {
            pb.Items = append(pb.Items, &orderpb.OrderItem{
                ProductId: it.ProductID,
                Quantity:  it.Quantity,
                UnitPrice: it.UnitPrice,
            })
        }
        pbOrders = append(pbOrders, pb)
    }
    // Calculate hasNext.  Note: we use the original total count, not the
    // filtered length, to reflect that there may be more orders overall.
    hasNext := int32((page+1)*limit) < total
    pagination := &loyaltypb.PaginationResponse{
        Total:   total,
        Limit:   limit,
        Page:    page,
        HasNext: hasNext,
    }
    return &orderpb.ListOrdersResponse{
        Orders:     pbOrders,
        Pagination: pagination,
    }, nil
}