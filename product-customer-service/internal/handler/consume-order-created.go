package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/linhhuynhcoding/jss-microservices/mq"
	mqConfig "github.com/linhhuynhcoding/jss-microservices/mq/config"
	"github.com/linhhuynhcoding/jss-microservices/mq/consts"
	"github.com/linhhuynhcoding/jss-microservices/product/config"
	"github.com/linhhuynhcoding/jss-microservices/rpc/gen/order"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type IOrderCreatedConsumer interface {
	ConsumeOrderCreated(ctx context.Context)
}

type OrderCreatedConsumer struct {
	logger *zap.Logger
	cfg    config.Config
}

func NewOrderCreatedConsumer(
	logger *zap.Logger,
	cfg config.Config,
) IOrderCreatedConsumer {
	return &OrderCreatedConsumer{
		logger: logger,
		cfg:    cfg,
	}
}

func (c *OrderCreatedConsumer) ConsumeOrderCreated(ctx context.Context) {
	logger := c.logger.With(zap.Any("func", "ConsumeOrderCreated"))

	config := mqConfig.RabbitMQConfig{
		ConnStr:        c.cfg.MqConnStr,
		ExchangeName:   consts.EXCHANGE_ORDER_SERVICE,
		ExchangeType:   "topic",
		SubscribeKeys:  []string{consts.TOPIC_CREATE_ORDER},
		PublisherName:  consts.EXCHANGE_ORDER_SERVICE,
		SubscriberName: "",
	}

	subscriber, err := mq.NewSubscriber(config, logger)
	if err != nil {
		logger.Error("Error", zap.Error(err))
		return
	}
	defer subscriber.Close()
	logger.Info("Init Subscriber successfully")

	errCh := make(chan error)

	// Start consuming
	go func() {
		if errCh <- subscriber.Consume(c.handler); errCh != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	select {
	case <-errCh:
		{
			logger.Info("Done")
			return
		}
	}
}

func (c *OrderCreatedConsumer) handler(body []byte) error {
	var data order.Order // TODO: Chờ publisher define event
	_ = proto.Unmarshal(body, &data)
	b, _ := protojson.Marshal(&data)

	fmt.Printf("Received: %v\n", string(b))
	return nil
}
