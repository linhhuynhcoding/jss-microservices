package main

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linhhuynhcoding/jss-microservices/loyalty/config"
	"github.com/linhhuynhcoding/jss-microservices/loyalty/internal/handler"
	"github.com/linhhuynhcoding/jss-microservices/loyalty/internal/repository"
	"github.com/linhhuynhcoding/jss-microservices/loyalty/internal/service"
	"github.com/linhhuynhcoding/jss-microservices/rpc/gen/loyalty"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO: IMPLEMENT FX
func main() {
	// ------------------------------------------------------------
	// 		INIT VARIABLES
	// ------------------------------------------------------------
	ctx := context.Background()
	cfg := config.NewConfig()
	log, _ := zap.NewProduction()

	log.Info("config", zap.Any("cfg", cfg))

	{
		orderCreatedConsumer := handler.NewOrderCreatedConsumer(log, cfg)
		go orderCreatedConsumer.ConsumeOrderCreated(ctx)
	}
	{
		go NewServer(ctx, cfg, log)
	}
	{
		NewGatewayServer(ctx, cfg, log)
	}
}

func NewServer(
	ctx context.Context,
	cfg config.Config,
	log *zap.Logger,
) {
	// ------------------------------------------------------------
	// 		INIT DB
	// ------------------------------------------------------------
	connPool, err := pgxpool.New(ctx, cfg.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db")
	}
	store := repository.NewStore(connPool)

	// ------------------------------------------------------------
	// 		START SERVER
	// ------------------------------------------------------------
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal("failed to listen: %v", zap.Error(err))
	}

	s := grpc.NewServer()
	ser, err := service.NewService(log, config.NewConfig(), store)
	loyalty.RegisterLoyaltyServer(s, ser)

	log.Info("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve: %v", zap.Error(err))
	}
}

func NewGatewayServer(
	ctx context.Context,
	cfg config.Config,
	log *zap.Logger,
) {
	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := loyalty.RegisterLoyaltyHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
	if err != nil {
		log.Fatal("failed to start gateway", zap.Error(err))
	}

	log.Info("gRPC-Gateway listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("failed to serve: %v", zap.Error(err))
	}
}
