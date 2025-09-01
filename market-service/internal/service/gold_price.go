package service

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	db "github.com/linhhuynhcoding/jss-microservices/market/internal/repository"
	utils "github.com/linhhuynhcoding/jss-microservices/market/internal/utils/numeric"
	api "github.com/linhhuynhcoding/jss-microservices/rpc/gen/market"
)

func (s *Service) CreateGoldPrice(ctx context.Context, req *api.CreateGoldPriceRequest) (*api.CreateGoldPriceResponse, error) {
	s.logger.Info("CreateGoldPrice called", zap.Any("req", req))

	arg := db.CreateGoldPriceParams{
		GoldType:  req.GoldType,
		BuyPrice:  utils.ToNumeric(float64(req.BuyPrice)),
		SellPrice: utils.ToNumeric(float64(req.SellPrice)),
		Date:      utils.ToPgTimestamp(req.Date),
	}

	gp, err := s.queries.CreateGoldPrice(ctx, arg)
	if err != nil {
		s.logger.Error("failed to create gold price", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create gold price: %v", err)
	}

	s.logger.Info("Gold price created successfully", zap.Any("gold_price", gp))
	return &api.CreateGoldPriceResponse{
		GoldPrice: &api.GoldPrice{
			Id:        int64(gp.ID),
			GoldType:  gp.GoldType,
			BuyPrice:  float32(utils.NumericToFloat64(gp.BuyPrice)),
			SellPrice: float32(utils.NumericToFloat64(gp.SellPrice)),
			Date:      utils.PgToPbTimestamp(gp.Date),
		},
	}, nil
}

func (s *Service) GetGoldPrice(ctx context.Context, req *api.GetGoldPriceRequest) (*api.GetGoldPriceResponse, error) {
	s.logger.Info("GetGoldPrice called", zap.Any("req", req))

	gp, err := s.queries.GetGoldPrice(ctx, int32(req.Id))
	if err != nil {
		s.logger.Error("gold price not found", zap.Error(err), zap.Int64("id", req.Id))
		return nil, status.Errorf(codes.NotFound, "gold price not found: %v", err)
	}

	s.logger.Info("Gold price retrieved", zap.Any("gold_price", gp))
	return &api.GetGoldPriceResponse{
		GoldPrice: &api.GoldPrice{
			Id:        int64(gp.ID),
			GoldType:  gp.GoldType,
			BuyPrice:  float32(utils.NumericToFloat64(gp.BuyPrice)),
			SellPrice: float32(utils.NumericToFloat64(gp.SellPrice)),
			Date:      utils.PgToPbTimestamp(gp.Date),
		},
	}, nil
}

func (s *Service) ListGoldPrices(ctx context.Context, req *api.ListGoldPricesRequest) (*api.ListGoldPricesResponse, error) {
	s.logger.Info("ListGoldPrices called", zap.Any("req", req))

	arg := db.GetGoldPricesParams{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	gps, err := s.queries.GetGoldPrices(ctx, arg)
	s.logger.Info("Gold price retrieved", zap.Any("gold_price", gps))
	if err != nil {
		s.logger.Error("failed to list gold prices", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list gold prices: %v", err)
	}

	var results []*api.GoldPrice
	for _, gp := range gps {
		results = append(results, &api.GoldPrice{
			Id:        int64(gp.ID),
			GoldType:  gp.GoldType,
			BuyPrice:  float32(utils.NumericToFloat64(gp.BuyPrice)),
			SellPrice: float32(utils.NumericToFloat64(gp.SellPrice)),
			Date:      utils.PgToPbTimestamp(gp.Date),
		})
	}

	s.logger.Info("Gold prices listed", zap.Int("count", len(results)))
	return &api.ListGoldPricesResponse{GoldPrices: results}, nil
}

func (s *Service) UpdateGoldPrice(ctx context.Context, req *api.UpdateGoldPriceRequest) (*api.UpdateGoldPriceResponse, error) {
	s.logger.Info("UpdateGoldPrice called", zap.Any("req", req))

	arg := db.UpdateGoldPriceParams{
		ID:        int32(req.Id),
		BuyPrice:  utils.ToNumeric(float64(req.BuyPrice)),
		SellPrice: utils.ToNumeric(float64(req.SellPrice)),
	}

	gp, err := s.queries.UpdateGoldPrice(ctx, arg)
	if err != nil {
		s.logger.Error("failed to update gold price", zap.Error(err), zap.Int64("id", req.Id))
		return nil, status.Errorf(codes.Internal, "failed to update gold price: %v", err)
	}

	s.logger.Info("Gold price updated", zap.Any("gold_price", gp))
	return &api.UpdateGoldPriceResponse{
		GoldPrice: &api.GoldPrice{
			Id:        int64(gp.ID),
			GoldType:  gp.GoldType,
			BuyPrice:  float32(utils.NumericToFloat64(gp.BuyPrice)),
			SellPrice: float32(utils.NumericToFloat64(gp.SellPrice)),
			Date:      utils.PgToPbTimestamp(gp.Date),
		},
	}, nil
}

func (s *Service) DeleteGoldPrice(ctx context.Context, req *api.DeleteGoldPriceRequest) (*emptypb.Empty, error) {
	s.logger.Info("DeleteGoldPrice called", zap.Any("req", req))

	err := s.queries.DeleteGoldPrice(ctx, int32(req.Id))
	if err != nil {
		s.logger.Error("failed to delete gold price", zap.Error(err), zap.Int64("id", req.Id))
		return nil, status.Errorf(codes.Internal, "failed to delete gold price: %v", err)
	}

	s.logger.Info("Gold price deleted", zap.Int64("id", req.Id))
	return &emptypb.Empty{}, nil
}
