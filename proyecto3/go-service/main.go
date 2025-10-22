package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "go-service/proto"
)

type WeatherTweet struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
}

func main() {
	// Conexión a Kafka gRPC Server
	connKafka, err := grpc.Dial("kafka-service.dev.svc.cluster.local:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Kafka gRPC server: %v", err)
	}
	defer connKafka.Close()
	clientKafka := pb.NewWeatherTweetServiceClient(connKafka)

	// Conexión a RabbitMQ gRPC Server
	connRabbit, err := grpc.Dial("rabbitmq-service.dev.svc.cluster.local:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ gRPC server: %v", err)
	}
	defer connRabbit.Close()
	clientRabbit := pb.NewWeatherTweetServiceClient(connRabbit)

	r := mux.NewRouter()
	r.HandleFunc("/tweet", func(w http.ResponseWriter, r *http.Request) {
		var tweet WeatherTweet
		if err := json.NewDecoder(r.Body).Decode(&tweet); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		municipality := pb.Municipalities_value[tweet.Municipality]
		weather := pb.Weathers_value[tweet.Weather]

		// Enviar a Kafka
		_, err = clientKafka.SendTweet(context.Background(), &pb.WeatherTweetRequest{
			Municipality: pb.Municipalities(municipality),
			Temperature:  tweet.Temperature,
			Humidity:     tweet.Humidity,
			Weather:      pb.Weathers(weather),
		})
		if err != nil {
			log.Printf("Kafka send failed: %v", err)
		}

		// Enviar a RabbitMQ
		_, err = clientRabbit.SendTweet(context.Background(), &pb.WeatherTweetRequest{
			Municipality: pb.Municipalities(municipality),
			Temperature:  tweet.Temperature,
			Humidity:     tweet.Humidity,
			Weather:      pb.Weathers(weather),
		})
		if err != nil {
			log.Printf("RabbitMQ send failed: %v", err)
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}).Methods("POST")

	log.Fatal(http.ListenAndServe(":8080", r))
}
