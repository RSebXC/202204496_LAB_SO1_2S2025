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
	// Conectar al servidor gRPC (Deployment 2 o 3, configuraremos después)
	conn, err := grpc.Dial("kafka-service.dev.svc.cluster.local:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()
	client := pb.NewWeatherTweetServiceClient(conn)

	// Configurar API REST
	r := mux.NewRouter()
	r.HandleFunc("/tweet", func(w http.ResponseWriter, r *http.Request) {
		var tweet WeatherTweet
		if err := json.NewDecoder(r.Body).Decode(&tweet); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Mapear enums
		municipality := pb.Municipalities_value[tweet.Municipality]
		weather := pb.Weathers_value[tweet.Weather]

		// Enviar a gRPC server
		resp, err := client.SendTweet(context.Background(), &pb.WeatherTweetRequest{
			Municipality: pb.Municipalities(municipality),
			Temperature:  tweet.Temperature,
			Humidity:     tweet.Humidity,
			Weather:      pb.Weathers(weather),
		})
		if err != nil {
			http.Error(w, "Failed to send to gRPC server", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": resp.Status})
	}).Methods("POST")

	log.Fatal(http.ListenAndServe(":8080", r))
}
