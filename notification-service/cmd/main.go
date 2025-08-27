package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	mq "github.com/linhhuynhcoding/jss-microservices/mq"
	mqconfig "github.com/linhhuynhcoding/jss-microservices/mq/config"
	"github.com/linhhuynhcoding/jss-microservices/mq/events"
	"github.com/linhhuynhcoding/jss-microservices/notification-service/internal/grpc/server"
	"github.com/linhhuynhcoding/jss-microservices/notification-service/internal/repository"
	"github.com/linhhuynhcoding/jss-microservices/notification-service/internal/service"
	"github.com/linhhuynhcoding/jss-microservices/notification-service/pkg/config"
	"github.com/linhhuynhcoding/jss-microservices/notification-service/pkg/database"
	"github.com/linhhuynhcoding/jss-microservices/notification-service/pkg/logger"
	notificationpb "github.com/linhhuynhcoding/jss-microservices/rpc/gen/notification"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func main() {
	// 1) Config + logger
	cfg := config.Load()
	logg := logger.New(cfg.LogLevel)
	defer logg.Sync()

	// 2) DB + repo + service
	db := database.Connect(cfg, logg)
	repo := repository.NewNotificationRepository(db, logg)
	svc := service.NewNotificationService(repo, logg)

	// 3) MQ subscriber (dùng /mq) — handler ký tên func([]byte) error
	var sub *mq.Subscriber
	if cfg.RabbitMQURL != "" {
		subscriberName := cfg.SubscriberName
		if subscriberName == "" {
			subscriberName = "notification-service"
		}
		keys := cfg.BindingKeys
		if len(keys) == 0 {
			keys = []string{"notification.create"}
		}

		subCfg := mqconfig.RabbitMQConfig{
			ConnStr:        cfg.RabbitMQURL,   // amqp://guest:guest@rabbitmq:5672/
			SubscriberName: subscriberName,    // tên consumer/queue
			ExchangeType:   "topic",
			ExchangeName:   cfg.ExchangeName,  // ví dụ: notification_exchange
			SubscribeKeys:  keys,              // ví dụ: ["notification.create"]
		}

		var err error
		sub, err = mq.NewSubscriber(subCfg, logg)
		if err != nil {
			logg.Fatal("mq subscriber init failed", zap.Error(err))
		}

		// Consume nhận []byte (envelope), tự giải mã rồi xử lý
		go func() {
			err := sub.Consume(func(b []byte) error {
				var env events.EventEnvelope
				if err := proto.Unmarshal(b, &env); err != nil {
					// lỗi không phục hồi → drop (không requeue)
					logg.Warn("drop: unmarshal envelope failed", zap.Error(err))
					return nil
				}
				var msg notificationpb.CreateNotificationRequest
				if err := proto.Unmarshal(env.Payload, &msg); err != nil {
					// lỗi không phục hồi → drop
					logg.Warn("drop: unmarshal payload failed", zap.Error(err))
					return nil
				}

				// svc.Create(ctx, userIdStr, role, title, message)
				if _, err := svc.Create(context.Background(), msg.UserId, "", msg.Title, msg.Message); err != nil {
					low := strings.ToLower(err.Error())
					if strings.Contains(low, "invalid user id") || strings.Contains(low, "invalid id") {
						// không phục hồi → drop
						logg.Warn("drop: invalid user id", zap.String("userId", msg.UserId))
						return nil
					}
					// lỗi tạm (DB/network) → trả err để lib quyết định Nack(requeue)
					logg.Error("create notification failed (will requeue)", zap.Error(err))
					return err
				}

				// ok → ack
				return nil
			})
			if err != nil {
				logg.Fatal("mq subscriber exited", zap.Error(err))
			}
		}()

		logg.Info("rabbitmq subscriber started",
			zap.String("exchange", subCfg.ExchangeName),
			zap.String("subscriber", subCfg.SubscriberName),
			zap.Strings("keys", subCfg.SubscribeKeys),
		)
	}

	// 4) gRPC server
	grpcServer := grpc.NewServer()
	notificationpb.RegisterNotificationServiceServer(
		grpcServer,
		server.NewServer(svc, logg),
	)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logg.Fatal("failed to listen for gRPC", zap.Error(err))
	}
	go func() {
		logg.Info("gRPC listening", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logg.Fatal("gRPC serve failed", zap.Error(err))
		}
	}()

	// 5) gRPC-Gateway (REST)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{grpc.WithInsecure()}
		if err := notificationpb.RegisterNotificationServiceHandlerFromEndpoint(
			ctx, mux, "localhost:"+cfg.GRPCPort, opts,
		); err != nil {
			log.Fatal("failed to register notification gateway", err)
		}
		log.Printf("HTTP REST gateway listening on :%s", cfg.HTTPPort)
		if err := http.ListenAndServe(":"+cfg.HTTPPort, mux); err != nil {
			log.Fatal("HTTP REST serve failed:", err)
		}
	}()

	// 6) Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logg.Info("shutting down gRPC")
	grpcServer.GracefulStop()
	if sub != nil {
		sub.Close() // không có return value
	}
	logg.Info("exited")
}
