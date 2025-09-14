package service

import (
	"fmt"

	"github.com/linhhuynhcoding/jss-microservices/loyalty/config"
	"github.com/linhhuynhcoding/jss-microservices/loyalty/internal/adapter"
	"github.com/linhhuynhcoding/jss-microservices/loyalty/internal/repository"
	"github.com/linhhuynhcoding/jss-microservices/rpc/gen/loyalty"
	"go.uber.org/zap"
)

type Service struct {
	loyalty.UnimplementedLoyaltyServer

	logger  *zap.Logger
	cfg     config.Config
	queries repository.Store
	//
	authClient *adapter.AuthClient
}

func NewService(
	logger *zap.Logger,
	cfg config.Config,
	store repository.Store,
) (*Service, error) {
	authClient, err := adapter.NewAuthClient(cfg.AuthServiceAddr, logger)
	if err != nil {
		logger.Error("failed to create auth client", zap.Error(err))
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	return &Service{
		logger:     logger,
		cfg:        cfg,
		queries:    store,
		authClient: authClient,
	}, nil
}
