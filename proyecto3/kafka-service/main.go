package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/Shopify/sarama"
	"google.golang.org/grpc"

	pb "kafka-service/proto"
)

type server struct {
	pb.UnimplementedWeatherTweetServiceServer
	producer sarama.SyncProducer
}

func (s *server) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	// Convertir el mensaje a JSON
	msg, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Publicar en Kafka
	_, _, err = s.producer.SendMessage(&sarama.ProducerMessage{
		Topic: "weather-tweets",
		Value: sarama.StringEncoder(msg),
	})
	if err != nil {
		return nil, err
	}

	return &pb.WeatherTweetResponse{Status: "Published to Kafka"}, nil
}

func main() {
	// Usar el nombre correcto del servicio Kafka
	kafkaBootstrap := "my-kafka-kafka-bootstrap:9092"

	// Esperar a que Kafka esté disponible
	var producer sarama.SyncProducer
	var err error

	log.Printf("Connecting to Kafka at: %s", kafkaBootstrap)

	for i := 0; i < 30; i++ {
		config := sarama.NewConfig()
		config.Producer.Return.Successes = true
		producer, err = sarama.NewSyncProducer([]string{kafkaBootstrap}, config)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to Kafka (attempt %d/30): %v", i+1, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to Kafka after 30 attempts: %v", err)
	}
	defer producer.Close()

	log.Println("Successfully connected to Kafka")

	// Configurar servidor gRPC
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWeatherTweetServiceServer(s, &server{producer: producer})
	log.Println("gRPC server starting on :9000")
	log.Fatal(s.Serve(lis))
}
