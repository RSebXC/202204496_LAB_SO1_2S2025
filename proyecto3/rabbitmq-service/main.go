package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"

	pb "rabbitmq-service/proto"
)

type server struct {
	pb.UnimplementedWeatherTweetServiceServer
	channel *amqp091.Channel
}

func (s *server) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	// Convertir el mensaje a JSON
	msg, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Publicar en RabbitMQ
	err = s.channel.PublishWithContext(ctx, "", "weather-tweets", false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        msg,
	})
	if err != nil {
		return nil, err
	}

	return &pb.WeatherTweetResponse{Status: "Published to RabbitMQ"}, nil
}

func main() {
	// Usar las credenciales correctas
	rabbitMQURL := "amqp://admin:admin123@rabbitmq:5672/"

	// Esperar a que RabbitMQ esté disponible
	var conn *amqp091.Connection
	var err error

	log.Printf("Connecting to RabbitMQ...")

	for i := 0; i < 30; i++ {
		conn, err = amqp091.Dial(rabbitMQURL)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d/30): %v", i+1, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after 30 attempts: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer channel.Close()

	// Declarar la cola
	_, err = channel.QueueDeclare("weather-tweets", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	log.Println("Successfully connected to RabbitMQ and declared queue")

	// Configurar servidor gRPC
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWeatherTweetServiceServer(s, &server{channel: channel})
	log.Println("gRPC server starting on :9000")
	log.Fatal(s.Serve(lis))
}
