package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

type WeatherTweet struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
}

func main() {
	// Conectar a Kafka
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumer, err := sarama.NewConsumerGroup([]string{"my-kafka-kafka-bootstrap:9092"}, "weather-consumer-group", config)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Conectar a Valkey (Redis)
	rdb := redis.NewClient(&redis.Options{
		Addr: "valkey:6379",
	})
	defer rdb.Close()

	// Consumir mensajes
	ctx := context.Background()
	handler := &consumerHandler{rdb: rdb}
	for {
		err := consumer.Consume(ctx, []string{"weather-tweets"}, handler)
		if err != nil {
			log.Printf("Error consuming: %v", err)
		}
	}
}

type consumerHandler struct {
	rdb *redis.Client
}

func (h *consumerHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := context.Background()
	for msg := range claim.Messages() {
		var tweet WeatherTweet
		if err := json.Unmarshal(msg.Value, &tweet); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Guardar en Valkey
		key := "tweet:" + tweet.Municipality
		err := h.rdb.HSet(ctx, key, map[string]interface{}{
			"temperature": tweet.Temperature,
			"humidity":    tweet.Humidity,
			"weather":     tweet.Weather,
		}).Err()
		if err != nil {
			log.Printf("Failed to save to Valkey: %v", err)
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
