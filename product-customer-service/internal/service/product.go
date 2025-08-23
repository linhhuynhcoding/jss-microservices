package service

import (
	"context"
	"database/sql"

	db "github.com/linhhuynhcoding/jss-microservices/product/internal/repository"
	utils "github.com/linhhuynhcoding/jss-microservices/product/internal/utils/numeric"
	api "github.com/linhhuynhcoding/jss-microservices/rpc/gen/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) CreateProduct(ctx context.Context, req *api.CreateProductRequest) (*api.ProductResponse, error) {
	arg := db.CreateProductParams{
		Name:            req.Name,
		Code:            req.Code,
		CategoryID:      req.CategoryId,
		Weight:          utils.ToNumeric(req.Weight),
		GoldPriceAtTime: utils.ToNumeric(req.GoldPriceAtTime),
		LaborCost:       utils.ToNumeric(req.LaborCost),
		StoneCost:       utils.ToNumeric(req.StoneCost),
		MarkupRate:      utils.ToNumeric(req.MarkupRate),
		SellingPrice:    utils.ToNumeric(req.SellingPrice),
		WarrantyPeriod:  utils.Int32(req.WarrantyPeriod),
		Image:           req.Image,
	}	

	product, err := s.queries.CreateProduct(ctx, arg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	return &api.ProductResponse{Product: s.productToProto(product)}, nil
}

func (s *Service) GetProduct(ctx context.Context, req *api.GetProductRequest) (*api.ProductResponse, error) {
	product, err := s.queries.GetProductByID(ctx, req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	return &api.ProductResponse{Product: s.productToProto(product)}, nil
}

func (s *Service) ListProducts(ctx context.Context, req *api.ListProductsRequest) (*api.ListProductsResponse, error) {
	products, err := s.queries.ListProducts(ctx, db.ListProductsParams{
		Limit:  req.Limit,
		Offset: req.Page * req.Limit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
	}

	resp := &api.ListProductsResponse{}
	for _, p := range products {
		resp.Products = append(resp.Products, s.productToProto(p))
	}
	return resp, nil
}

func (s *Service) UpdateProduct(ctx context.Context, req *api.UpdateProductRequest) (*api.ProductResponse, error) {
	arg := db.UpsertProductParams{
		Name:            req.Name,
		Code:            req.Code,
		CategoryID:      req.CategoryId,
		Weight:          utils.ToNumeric(req.Weight),
		GoldPriceAtTime: utils.ToNumeric(req.GoldPriceAtTime),
		LaborCost:       utils.ToNumeric(req.LaborCost),
		StoneCost:       utils.ToNumeric(req.StoneCost),
		MarkupRate:      utils.ToNumeric(req.MarkupRate),
		SellingPrice:    utils.ToNumeric(req.SellingPrice),
		WarrantyPeriod:  utils.Int32(req.WarrantyPeriod),
		Image:           req.Image,
	}

	product, err := s.queries.UpsertProduct(ctx, arg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}

	return &api.ProductResponse{Product: s.productToProto(product)}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *api.DeleteProductRequest) (*api.DeleteProductResponse, error) {
	err := s.queries.DeleteProduct(ctx, req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
	}

	return &api.DeleteProductResponse{Success: true}, nil
}

func (s *Service) ListProductCategories(ctx context.Context, req *api.ListProductCategoriesRequest) (*api.ListProductCategoriesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListProductCategories not implemented")
}
