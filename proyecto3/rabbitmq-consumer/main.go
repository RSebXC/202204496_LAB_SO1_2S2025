package main

import (
        "context"
        "encoding/json"
        "log"
        "os"

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
        log.Println("Starting RabbitMQ consumer...")
        
        // Leer variables de entorno
        rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://admin:admin123@rabbitmq:5672/")
        queueName := getEnv("RABBITMQ_QUEUE", "weather-tweets")
        redisHost := getEnv("REDIS_HOST", "valkey-redis-master")
        redisPort := getEnv("REDIS_PORT", "6379")
        redisPassword := getEnv("REDIS_PASSWORD", "")
        
        log.Printf("Connecting to RabbitMQ: %s", rabbitmqURL)
        log.Printf("Connecting to Redis: %s:%s", redisHost, redisPort)
        
        // Conectar a RabbitMQ
        conn, err := amqp091.Dial(rabbitmqURL)
        if err != nil {
                log.Fatalf("Failed to connect to RabbitMQ: %v", err)
        }
        defer conn.Close()
        log.Println("Connected to RabbitMQ successfully")

        channel, err := conn.Channel()
        if err != nil {
                log.Fatalf("Failed to open a channel: %v", err)
        }
        defer channel.Close()

        // Declarar la cola
        q, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
        if err != nil {
                log.Fatalf("Failed to declare queue: %v", err)
        }
        log.Printf("Queue declared: %s", q.Name)

        // Conectar a Valkey
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
        msgs, err := channel.Consume(q.Name, "", true, false, false, false, nil)
        if err != nil {
                log.Fatalf("Failed to register consumer: %v", err)
        }
        
        log.Println("Waiting for messages...")
        for msg := range msgs {
                log.Printf("Received message: %s", string(msg.Body))
                
                var tweet WeatherTweet
                if err := json.Unmarshal(msg.Body, &tweet); err != nil {
                        log.Printf("Failed to unmarshal message: %v", err)
                        continue
                }

                log.Printf("Processing tweet for %s: temp=%d, humidity=%d, weather=%s", 
                        tweet.Municipality, tweet.Temperature, tweet.Humidity, tweet.Weather)

                // Guardar en Valkey
                key := "tweet:" + tweet.Municipality
                err = rdb.HSet(ctx, key, map[string]interface{}{
                        "temperature": tweet.Temperature,
                        "humidity":    tweet.Humidity,
                        "weather":     tweet.Weather,
                }).Err()
                if err != nil {
                        log.Printf("Failed to save to Valkey: %v", err)
                } else {
                        log.Printf("Saved to Valkey with key: %s", key)
                }
        }
}

func getEnv(key, defaultValue string) string {
        if value := os.Getenv(key); value != "" {
                return value
        }
        return defaultValue
}
