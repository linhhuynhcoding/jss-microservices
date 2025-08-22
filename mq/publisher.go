package main

import (
	"context"
	"fmt"
	"time"

	"github.com/linhhuynhcoding/jss-microservices/mq/config"
	"github.com/linhhuynhcoding/jss-microservices/mq/consts"
	"github.com/linhhuynhcoding/jss-microservices/mq/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Publisher struct {
	pubConfig config.RabbitMQConfig
	conn      *amqp.Connection
	ch        *amqp.Channel

	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger
}

func NewPublisher(
	cfg config.RabbitMQConfig,
) (*Publisher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := zap.NewNop()

	conn, err := amqp.Dial(cfg.ConnStr)
	if err != nil {
		logger.Fatal("Failed to connect mq", zap.Error(err))
		defer cancel()
		return nil, err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Fatal("Failed to init mq channel", zap.Error(err))
		defer cancel()
		return nil, err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		cfg.ExchangeName,           // name
		consts.EXCHANGE_TYPE_TOPIC, // type
		true,                       // durable
		false,                      // auto-deleted
		false,                      // internal
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		logger.Fatal("Failed to init mq exchange", zap.Error(err))
		defer cancel()
		return nil, err
	}

	return &Publisher{
		pubConfig: cfg,
		conn:      conn,
		ch:        ch,
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
	}, nil
}

func (p *Publisher) SendMessage(event proto.Message, topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Serialize the event payload
	payloadBytes, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	// Create event envelope
	envelope := &events.EventEnvelope{
		EventType: topic,
		EventId:   generateEventID(),
		Timestamp: timestamppb.Now(),
		Version:   1,
		Payload:   payloadBytes,
		Metadata: map[string]string{
			"publisher": p.pubConfig.PublisherName,
			"go_type":   fmt.Sprintf("%T", event),
		},
	}

	// Serialize the envelope
	body, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event envelope: %v", err)
	}

	err = p.ch.PublishWithContext(ctx,
		p.pubConfig.ExchangeName, // exchange
		topic,                    // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	p.logger.Info(" [x] Sent: ", zap.Any("content", body))
	return nil
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}
