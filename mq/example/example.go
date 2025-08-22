package example

import (
	"fmt"
	"log"

	"github.com/linhhuynhcoding/jss-microservices/mq"
	"github.com/linhhuynhcoding/jss-microservices/mq/config"
	"go.uber.org/zap"
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

	logger, _ := zap.NewProduction()
	subscriber, err := mq.NewSubscriber(config, logger)
	if err != nil {
		log.Fatal(err)
	}
	defer subscriber.Close()

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
}
