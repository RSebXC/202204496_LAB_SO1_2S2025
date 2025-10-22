package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type WeatherTweet struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
}

func main() {
	// Conectar a RabbitMQ
	conn, err := amqp091.Dial("amqp://admin:admin123@weather-rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer channel.Close()

	// Declarar la cola
	q, err := channel.QueueDeclare("weather-tweets", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Conectar a Valkey
	rdb := redis.NewClient(&redis.Options{
		Addr: "valkey:6379",
	})
	defer rdb.Close()

	// Consumir mensajes
	msgs, err := channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	ctx := context.Background()
	for msg := range msgs {
		var tweet WeatherTweet
		if err := json.Unmarshal(msg.Body, &tweet); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Guardar en Valkey
		key := "tweet:" + tweet.Municipality
		err = rdb.HSet(ctx, key, map[string]interface{}{
			"temperature": tweet.Temperature,
			"humidity":    tweet.Humidity,
			"weather":     tweet.Weather,
		}).Err()
		if err != nil {
			log.Printf("Failed to save to Valkey: %v", err)
		}
	}
}
