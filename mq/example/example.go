package main

import (
	"fmt"
	"log"
	"time"

	"github.com/linhhuynhcoding/jss-microservices/mq"
	"github.com/linhhuynhcoding/jss-microservices/mq/config"
	"github.com/linhhuynhcoding/jss-microservices/mq/consts"
	"github.com/linhhuynhcoding/jss-microservices/mq/events"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	config := config.RabbitMQConfig{
		ConnStr:        "amqp://admin:admin@localhost:5672/",
		ExchangeName:   "my-exchange",
		ExchangeType:   "topic",
		SubscribeKeys:  []string{consts.TOPIC_CREATE_PRODUCT},
		PublisherName:  "linh-publisher",
		SubscriberName: "linh-subscriber",
	}
	fmt.Println("Start")

	publisher, err := mq.NewPublisher(config, logger)
	if err != nil {
		fmt.Printf("Error %v", err)
	}
	defer publisher.Close()
	fmt.Println("Init Publisher successfully")

	go func() {
		<-time.After(time.Second * 3)
		err = publisher.SendMessage(&events.ProductEvent{
			ProductId: "productID",
		}, consts.TOPIC_CREATE_PRODUCT)
	}()

	subscriber, err := mq.NewSubscriber(config, logger)
	if err != nil {
		fmt.Printf("Error %v", err)
	}
	fmt.Println("Init Subscriber successfully")

	errCh := make(chan error)

	// Start consuming
	go func() {
		if errCh <- subscriber.Consume(func(body []byte) error {
			fmt.Printf("Received: %s\n", body)
			return nil
		}); errCh != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	select {
	case <-errCh:
		{
			fmt.Println("Done")
		}
	}
}
