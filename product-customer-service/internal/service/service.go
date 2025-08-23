package service

import (
	"context"

	"github.com/linhhuynhcoding/jss-microservices/product/config"
	"github.com/linhhuynhcoding/jss-microservices/product/internal/adapter/cloudinary"
	"github.com/linhhuynhcoding/jss-microservices/product/internal/repository"
	"github.com/linhhuynhcoding/jss-microservices/rpc/gen/product"
	"go.uber.org/zap"
)

type Adapter struct {
	cloudinaryAdapter cloudinary.ICoundinaryClient
}

type Service struct {
	product.UnimplementedProductCustomerServer

	logger  *zap.Logger
	cfg     config.Config
	queries repository.Store

	adapter *Adapter
}

func NewService(ctx context.Context, logger *zap.Logger, cfg config.Config, store repository.Store) *Service {

	return &Service{
		logger:  logger,
		cfg:     cfg,
		queries: store,

		adapter: &Adapter{
			cloudinaryAdapter: cloudinary.NewCloudinaryClient(logger, cfg),
		},
	}
}
