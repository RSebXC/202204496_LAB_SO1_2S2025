package main

import (
        "context"
        "encoding/json"
        "log"
        "os"

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
        log.Println("Starting Kafka consumer...")
        
        // Leer variables de entorno
        kafkaBrokers := getEnv("KAFKA_BROKERS", "my-kafka-kafka-bootstrap:9092")
        kafkaTopic := getEnv("KAFKA_TOPIC", "weather-tweets")
        kafkaGroupID := getEnv("KAFKA_GROUP_ID", "kafka-consumer")
        redisHost := getEnv("REDIS_HOST", "valkey-redis-master")
        redisPort := getEnv("REDIS_PORT", "6379")
        redisPassword := getEnv("REDIS_PASSWORD", "")
        
        log.Printf("Connecting to Kafka: %s", kafkaBrokers)
        log.Printf("Connecting to Redis: %s:%s", redisHost, redisPort)
        
        // Conectar a Kafka
        config := sarama.NewConfig()
        config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
        config.Consumer.Offsets.Initial = sarama.OffsetOldest
        consumer, err := sarama.NewConsumerGroup([]string{kafkaBrokers}, kafkaGroupID, config)
        if err != nil {
                log.Fatalf("Failed to create consumer: %v", err)
        }
        defer consumer.Close()

        // Conectar a Valkey (Redis)
        rdb := redis.NewClient(&redis.Options{
                Addr:     redisHost + ":" + redisPort,
                Password: redisPassword,
                DB:       0,
        })
        defer rdb.Close()
        
        // Test Redis connection
        ctx := context.Background()
        if err := rdb.Ping(ctx).Err(); err != nil {
                log.Fatalf("Failed to connect to Redis: %v", err)
        }
        log.Println("Connected to Redis successfully")

        // Consumir mensajes
        handler := &consumerHandler{rdb: rdb}
        log.Printf("Starting to consume from topic: %s", kafkaTopic)
        for {
                err := consumer.Consume(ctx, []string{kafkaTopic}, handler)
                if err != nil {
                        log.Printf("Error consuming: %v", err)
                }
        }
}

func getEnv(key, defaultValue string) string {
        if value := os.Getenv(key); value != "" {
                return value
        }
        return defaultValue
}

type consumerHandler struct {
        rdb *redis.Client
}

func (h *consumerHandler) Setup(_ sarama.ConsumerGroupSession) error {
        log.Println("Consumer group session setup")
        return nil
}

func (h *consumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
        log.Println("Consumer group session cleanup")
        return nil
}

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
        ctx := context.Background()
        for msg := range claim.Messages() {
                log.Printf("Received message: %s", string(msg.Value))
                
                var tweet WeatherTweet
                if err := json.Unmarshal(msg.Value, &tweet); err != nil {
                        log.Printf("Failed to unmarshal message: %v", err)
                        continue
                }

                log.Printf("Processing tweet for %s: temp=%d, humidity=%d, weather=%s", 
                        tweet.Municipality, tweet.Temperature, tweet.Humidity, tweet.Weather)

                // Guardar en Valkey
                key := "tweet:" + tweet.Municipality
                err := h.rdb.HSet(ctx, key, map[string]interface{}{
                        "temperature": tweet.Temperature,
                        "humidity":    tweet.Humidity,
                        "weather":     tweet.Weather,
                }).Err()
                if err != nil {
                        log.Printf("Failed to save to Valkey: %v", err)
                } else {
                        log.Printf("Saved to Valkey with key: %s", key)
                }

                session.MarkMessage(msg, "")
        }
        return nil
}
