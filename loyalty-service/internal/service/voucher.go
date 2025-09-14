package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jackc/pgx/v5/pgtype"
	utils "github.com/linhhuynhcoding/jss-microservices/jss-shared/utils/format"
	token "github.com/linhhuynhcoding/jss-microservices/jss-shared/utils/token"
	db "github.com/linhhuynhcoding/jss-microservices/loyalty/internal/repository"
	api "github.com/linhhuynhcoding/jss-microservices/rpc/gen/loyalty"
)

// ===== VOUCHER HANDLERS =====

func (s *Service) CreateVoucher(ctx context.Context, req *api.CreateVoucherRequest) (*api.GetVoucherResponse, error) {
	s.logger.Info("Creating voucher",
		zap.String("code", req.Code),
		zap.String("discount_type", req.DiscountType.String()))

	// 1) Auth
	accessToken, err := token.BearerFromMD(ctx)
	if err != nil {
		return nil, err
	}
	valid, _, role, err := s.authClient.Validate(ctx, accessToken)
	if err != nil || !valid {
		return nil, fmt.Errorf("unauthorised")
	}
	if role != "MANAGER" && role != "ADMIN" {
		return nil, fmt.Errorf("unauthorised role: %s, just Manager and Admin could update voucher", role)
	}

	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}
	if req.StartDate == "" || req.EndDate == "" {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	// Validate date format
	// if !utils.IsValidDateFormat(req.StartDate) || !utils.IsValidDateFormat(req.EndDate) {
	// 	return nil, status.Error(codes.InvalidArgument, "dates must be in YYYY-MM-DD format")
	// }

	isGlobal := req.Target == "ALL"
	createParams := db.CreateVoucherParams{
		Code:          req.Code,
		Description:   utils.StringPointerToPgText(req.Description),
		DiscountType:  ConvertAPIDiscountTypeToDBString(req.DiscountType),
		DiscountValue: utils.ToNumeric(req.DiscountValue),
		StartDate:     utils.StringDateToPgDate(req.StartDate),
		EndDate:       utils.StringDateToPgDate(req.EndDate),
		UsageLimit:    utils.Int(req.UsageLimit),
		IsGlobal:      pgtype.Bool{Bool: isGlobal, Valid: true},
	}

	voucher, err := s.queries.CreateVoucher(ctx, createParams)
	if err != nil {
		s.logger.Error("Failed to create voucher", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create voucher")
	}

	response := &api.GetVoucherResponse{
		Voucher: ConvertDBVoucherToAPI(voucher, nil),
	}

	s.logger.Info("Successfully created voucher", zap.Int32("id", voucher.ID))
	return response, nil
}

func (s *Service) GetVoucher(ctx context.Context, req *api.GetVoucherRequest) (*api.GetVoucherResponse, error) {
	s.logger.Info("Getting voucher", zap.Int32("id", req.Id))

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}

	voucher, err := s.queries.GetVoucher(ctx, req.Id)
	if err != nil {
		s.logger.Error("Failed to get voucher", zap.Error(err), zap.Int32("id", req.Id))
		return nil, status.Error(codes.NotFound, "voucher not found")
	}

	response := &api.GetVoucherResponse{
		Voucher: ConvertDBVoucherToAPI(voucher, nil),
	}

	return response, nil
}

func (s *Service) GetVoucherByCode(ctx context.Context, req *api.GetVoucherByCodeRequest) (*api.GetVoucherResponse, error) {
	s.logger.Info("Getting voucher by code", zap.String("code", req.Code))

	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	voucher, err := s.queries.GetVoucherByCode(ctx, req.Code)
	if err != nil {
		s.logger.Error("Failed to get voucher by code", zap.Error(err), zap.String("code", req.Code))
		return nil, status.Error(codes.NotFound, "voucher not found")
	}

	response := &api.GetVoucherResponse{
		Voucher: ConvertDBVoucherToAPI(voucher, nil),
	}

	return response, nil
}

func (s *Service) GetActiveVouchers(ctx context.Context, req *api.GetActiveVouchersRequest) (*api.GetVouchersResponse, error) {
	s.logger.Info("Getting active vouchers")

	var (
		limit int32 = 10
		page  int32 = 1
	)

	if req.Pagination != nil {
		limit = req.Pagination.Limit
		page = req.Pagination.Page
	}

	params := db.GetActiveVouchersParams{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	vouchers, err := s.queries.GetActiveVouchers(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get active vouchers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get active vouchers")
	}

	apiVouchers := make([]*api.Voucher, len(vouchers))
	for i, v := range vouchers {
		apiVouchers[i] = ConvertDBVoucherToAPI(v, nil)
	}

	response := &api.GetVouchersResponse{
		Vouchers: apiVouchers,
		Pagination: &api.PaginationResponse{
			Total:   int32(len(vouchers)),
			Limit:   limit,
			Page:    page,
			HasNext: len(vouchers) == int(limit),
		},
	}

	return response, nil
}

func (s *Service) GetAllVouchers(ctx context.Context, req *api.GetAllVouchersRequest) (*api.GetVouchersResponse, error) {
	s.logger.Info("Getting all vouchers")

	var (
		limit int32 = 10
		page  int32 = 1
	)
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		page = req.Pagination.Page
	}

	params := db.GetAllVouchersParams{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	vouchers, err := s.queries.GetAllVouchers(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get all vouchers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get vouchers")
	}

	apiVouchers := make([]*api.Voucher, len(vouchers))
	for i, v := range vouchers {
		apiVouchers[i] = ConvertDBVoucherToAPI(v, nil)
	}

	response := &api.GetVouchersResponse{
		Vouchers: apiVouchers,
		Pagination: &api.PaginationResponse{
			Total:   int32(len(vouchers)),
			Limit:   limit,
			Page:    page,
			HasNext: len(vouchers) == int(limit),
		},
	}

	return response, nil
}

func (s *Service) UpdateVoucher(ctx context.Context, req *api.UpdateVoucherRequest) (*api.GetVoucherResponse, error) {
	s.logger.Info("Updating voucher", zap.Int32("id", req.Id))

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}
	// if !utils.IsValidDateFormat(req.StartDate) || !utils.IsValidDateFormat(req.EndDate) {
	// 	return nil, status.Error(codes.InvalidArgument, "dates must be in YYYY-MM-DD format")
	// }

	updateParams := db.UpdateVoucherParams{
		ID:            req.Id,
		Code:          req.Code,
		Description:   utils.StringPointerToPgText(req.Description),
		DiscountType:  ConvertAPIDiscountTypeToDBString(req.DiscountType),
		DiscountValue: utils.ToNumeric(req.DiscountValue),
		StartDate:     utils.StringDateToPgDate(req.StartDate),
		EndDate:       utils.StringDateToPgDate(req.EndDate),
		UsageLimit:    utils.Int(req.UsageLimit),
	}

	voucher, err := s.queries.UpdateVoucher(ctx, updateParams)
	if err != nil {
		s.logger.Error("Failed to update voucher", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update voucher")
	}

	response := &api.GetVoucherResponse{
		Voucher: ConvertDBVoucherToAPI(voucher, nil),
	}

	s.logger.Info("Successfully updated voucher", zap.Int32("id", req.Id))
	return response, nil
}

func (s *Service) DeleteVoucher(ctx context.Context, req *api.DeleteVoucherRequest) (*emptypb.Empty, error) {
	s.logger.Info("Deleting voucher", zap.Int32("id", req.Id))

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}

	err := s.queries.DeleteVoucher(ctx, req.Id)
	if err != nil {
		s.logger.Error("Failed to delete voucher", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete voucher")
	}

	s.logger.Info("Successfully deleted voucher", zap.Int32("id", req.Id))
	return &emptypb.Empty{}, nil
}

// ===== CUSTOMER VOUCHER HANDLERS =====

func (s *Service) CreateCustomerVoucher(ctx context.Context, req *api.CreateCustomerVoucherRequest) (*api.GetCustomerVoucherResponse, error) {
	s.logger.Info("Creating customer voucher",
		zap.String("customer_id", req.CustomerId),
		zap.Int32("voucher_id", req.VoucherId))

	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id must be positive")
	}
	if req.VoucherId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "voucher_id must be positive")
	}

	createParams := db.CreateCustomerVoucherParams{
		CustomerID: req.CustomerId,
		VoucherID:  req.VoucherId,
		Status:     utils.StringToPgText(ConvertAPICustomerVoucherStatusToDBString(req.Status)),
	}

	customerVoucher, err := s.queries.CreateCustomerVoucher(ctx, createParams)
	if err != nil {
		s.logger.Error("Failed to create customer voucher", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create customer voucher")
	}

	response := &api.GetCustomerVoucherResponse{
		CustomerVoucher: ConvertDBCustomerVoucherToAPI(customerVoucher),
	}

	s.logger.Info("Successfully created customer voucher", zap.Int32("id", customerVoucher.ID))
	return response, nil
}

func (s *Service) GetCustomerVoucher(ctx context.Context, req *api.GetCustomerVoucherRequest) (*api.GetCustomerVoucherResponse, error) {
	s.logger.Info("Getting customer voucher", zap.Int32("id", req.Id))

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}

	customerVoucher, err := s.queries.GetCustomerVoucher(ctx, req.Id)
	if err != nil {
		s.logger.Error("Failed to get customer voucher", zap.Error(err), zap.Int32("id", req.Id))
		return nil, status.Error(codes.NotFound, "customer voucher not found")
	}

	response := &api.GetCustomerVoucherResponse{
		CustomerVoucher: ConvertDBCustomerVoucherToAPI(customerVoucher),
	}

	return response, nil
}

func (s *Service) GetCustomerVouchers(ctx context.Context, req *api.GetCustomerVouchersRequest) (*api.GetCustomerVouchersResponse, error) {
	s.logger.Info("Getting customer vouchers", zap.String("customer_id", req.CustomerId))

	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id must be positive")
	}
	var (
		limit int32 = 10
		page  int32 = 1
	)
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		page = req.Pagination.Page
	}

	params := db.GetCustomerVouchersParams{
		CustomerID: req.CustomerId,
		Limit:      limit,
		Offset:     (page - 1) * limit,
	}

	customerVouchers, err := s.queries.GetCustomerVouchers(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get customer vouchers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get customer vouchers")
	}

	apiCustomerVouchers := make([]*api.CustomerVoucher, len(customerVouchers))
	for i, cv := range customerVouchers {
		apiCustomerVouchers[i] = ConvertDBCustomerVoucherRowToAPI(cv)
	}

	response := &api.GetCustomerVouchersResponse{
		CustomerVouchers: apiCustomerVouchers,
		Pagination: &api.PaginationResponse{
			Total:   int32(len(customerVouchers)),
			Limit:   limit,
			Page:    page,
			HasNext: len(customerVouchers) == int(limit),
		},
	}

	return response, nil
}

func (s *Service) GetCustomerVouchersByStatus(ctx context.Context, req *api.GetCustomerVouchersByStatusRequest) (*api.GetCustomerVouchersResponse, error) {
	s.logger.Info("Getting customer vouchers by status",
		zap.String("customer_id", req.CustomerId),
		zap.String("status", req.Status.String()))

	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id must be positive")
	}

	var (
		limit int32 = 10
		page  int32 = 1
	)
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		page = req.Pagination.Page
	}

	params := db.GetCustomerVouchersByStatusParams{
		CustomerID: req.CustomerId,
		Status:     utils.StringToPgText(ConvertAPICustomerVoucherStatusToDBString(&req.Status)),
		Limit:      limit,
		Offset:     (page - 1) * limit,
	}

	customerVouchers, err := s.queries.GetCustomerVouchersByStatus(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get customer vouchers by status", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get customer vouchers")
	}

	apiCustomerVouchers := make([]*api.CustomerVoucher, len(customerVouchers))
	for i, cv := range customerVouchers {
		apiCustomerVouchers[i] = ConvertDBCustomerVoucherStatusRowToAPI(cv)
	}

	response := &api.GetCustomerVouchersResponse{
		CustomerVouchers: apiCustomerVouchers,
		Pagination: &api.PaginationResponse{
			Total:   int32(len(customerVouchers)),
			Limit:   limit,
			Page:    page,
			HasNext: len(customerVouchers) == int(limit),
		},
	}

	return response, nil
}

func (s *Service) GetAllCustomerVouchers(ctx context.Context, req *api.GetAllCustomerVouchersRequest) (*api.GetCustomerVouchersResponse, error) {
	s.logger.Info("Getting all customer vouchers")

	var (
		limit int32 = 10
		page  int32 = 1
	)
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		page = req.Pagination.Page
	}

	params := db.GetAllCustomerVouchersParams{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	customerVouchers, err := s.queries.GetAllCustomerVouchers(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get all customer vouchers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get customer vouchers")
	}

	apiCustomerVouchers := make([]*api.CustomerVoucher, len(customerVouchers))
	for i, cv := range customerVouchers {
		apiCustomerVouchers[i] = ConvertDBAllCustomerVouchersRowToAPI(cv)
	}

	response := &api.GetCustomerVouchersResponse{
		CustomerVouchers: apiCustomerVouchers,
		Pagination: &api.PaginationResponse{
			Total:   int32(len(customerVouchers)),
			Limit:   limit,
			Page:    page,
			HasNext: len(customerVouchers) == int(limit),
		},
	}

	return response, nil
}

func (s *Service) GetAvailableVouchersForCustomer(ctx context.Context, req *api.GetAvailableVouchersForCustomerRequest) (*api.GetVouchersResponse, error) {
	s.logger.Info("Getting available vouchers for customer", zap.String("customer_id", req.CustomerId))

	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id must be positive")
	}

	var (
		limit int32 = 10
		page  int32 = 1
	)
	if req.Pagination != nil {
		limit = req.Pagination.Limit
		page = req.Pagination.Page
	}

	params := db.GetAvailableVouchersForCustomerParams{
		CustomerID: req.CustomerId,
		Limit:      limit,
		Offset:     (page - 1) * limit,
	}

	vouchers, err := s.queries.GetAvailableVouchersForCustomer(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get available vouchers for customer", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get available vouchers")
	}

	apiVouchers := make([]*api.Voucher, len(vouchers))
	for i, v := range vouchers {
		apiVouchers[i] = ConvertDBVoucherToAPI(v, nil)
	}

	response := &api.GetVouchersResponse{
		Vouchers: apiVouchers,
		Pagination: &api.PaginationResponse{
			Total:   int32(len(vouchers)),
			Limit:   limit,
			Page:    page,
			HasNext: len(vouchers) == int(limit),
		},
	}

	return response, nil
}

func (s *Service) UseCustomerVoucher(ctx context.Context, req *api.UseCustomerVoucherRequest) (*api.GetCustomerVoucherResponse, error) {
	s.logger.Info("Using customer voucher", zap.Int32("id", req.Id))

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	return nil, nil
}

func (s *Service) CalculateDiscountAmount(ctx context.Context, req *api.CalculateDiscountAmountRequest) (*api.CalculateDiscountAmountResponse, error) {
	logger := s.logger.With(zap.String("method", "CalculateDiscountAmount"))
	logger.Info("Calculating discount amount", zap.Any("customer_id", req.CustomerId))

	if req.CustomerId == "" {
		logger.Error("Invalid customer_id", zap.Any("customer_id", req.CustomerId))
		return nil, fmt.Errorf("invalid customer_id: %d", req.CustomerId)
	}

	if len(req.Vouchers) == 0 {
		logger.Error("No vouchers provided")
		return nil, fmt.Errorf("no vouchers provided")
	}

	vouchers, err := s.validateVoucher(ctx, req.CustomerId, req.Vouchers)
	if err != nil {
		logger.Error("Failed to validate vouchers", zap.Error(err))
		return nil, err
	}
	vouchersResp, totalDiscountAmount := s.calculateDiscountAmount(vouchers, req.TotalProductAmount)

	return &api.CalculateDiscountAmountResponse{
		TotalDiscountAmount: totalDiscountAmount,
		Vouchers:            vouchersResp,
	}, nil
}

func (s *Service) UsingVoucher(ctx context.Context, req *api.UsingVoucherRequest) (*api.UsingVoucherResponse, error) {
	logger := s.logger.With(zap.String("method", "UsingVoucher"))
	logger.Info("Using voucher", zap.Any("customer_id", req.CustomerId))

	vouchers, err := s.validateVoucher(ctx, req.CustomerId, req.Vouchers)
	if err != nil {
		logger.Error("Failed to validate vouchers", zap.Error(err))
		return nil, err
	}
	vouchersResp, totalDiscountAmount := s.calculateDiscountAmount(vouchers, req.TotalProductAmount)

	err = s.queries.ExecTx(ctx, func(q *db.Queries) error {
		for _, voucher := range vouchersResp {
			_, err = s.queries.UpsertUsageRecord(ctx, db.UpsertUsageRecordParams{
				CustomerID: req.CustomerId,
				VoucherID:  voucher.Id,
				OrderID:    req.OrderId,
				CreatedAt: pgtype.Timestamp{
					Time:  time.Now(),
					Valid: true,
				},
				UpdatedAt: pgtype.Timestamp{
					Time: time.Now(),
				},
			})
			if err != nil {
				logger.Error("Failed to upsert usage record", zap.Error(err))
				return err
			}

			_, err = s.queries.UpdateCustomerVoucherStatus(ctx, db.UpdateCustomerVoucherStatusParams{
				ID:     voucher.Id,
				Status: utils.StringToPgText("used"),
			})
			if err != nil {
				logger.Error("Failed to update customer voucher status", zap.Error(err))
				return err
			}

			err = s.queries.DecreaseVoucher(ctx, voucher.Id)
			if err != nil {
				logger.Error("Failed to decrease voucher", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("Failed to upsert usage record", zap.Error(err))
		return nil, err
	}

	return &api.UsingVoucherResponse{
		TotalDiscountAmount: totalDiscountAmount,
		Vouchers:            vouchersResp,
	}, nil
}
