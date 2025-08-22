package main

import (
	"fmt"
	"log"
	"time"

	"github.com/linhhuynhcoding/jss-microservices/mq"
	"github.com/linhhuynhcoding/jss-microservices/mq/config"
	"github.com/linhhuynhcoding/jss-microservices/mq/consts"
	"github.com/linhhuynhcoding/jss-microservices/mq/events"
)

func main() {
	config := config.RabbitMQConfig{
		ConnStr:        "amqp://localhost:5672/",
		ExchangeName:   "my-exchange",
		ExchangeType:   "topic",
		SubscribeKeys:  []string{"user.created", "user.updated"},
		PublisherName:  "linh-publisher",
		SubscriberName: "linh-subscriber",
	}
	fmt.Println("Start")

	publisher, err := mq.NewPublisher(config)
	if err != nil {
		fmt.Printf("Error %v", err)
	}
	fmt.Println("Init Publisher successfully")

	go func() {
		<-time.After(time.Second * 3)
		err = publisher.SendMessage(&events.ProductEvent{
			ProductId: "productID",
		}, consts.TOPIC_CREATE_PRODUCT)
	}()

	subscriber, err := mq.NewSubscriber(config)
	if err != nil {
		fmt.Printf("Error %v", err)
	}
	fmt.Println("Init Subscriber successfully")

	// Start consuming
	go func() {
		if err := subscriber.Consume(func(body []byte) error {
			// Your message handler logic
			fmt.Printf("Received: %s\n", body)
			return nil
		}); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	fmt.Println("Done")
}
