package main

import (
	natsgo "app-nats-go"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type HealthRequest struct{}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	client, err := natsgo.NewClientFromEnv("health-service")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	subject := "service.health"
	queue := "health-service-group"

	_, err = natsgo.Respond(client, subject, queue, func(ctx context.Context, req HealthRequest) (HealthResponse, error) {
		log.Println("Received health check request")
		return HealthResponse{
			Status:  "UP",
			Service: "health-service",
		}, nil
	})

	if err != nil {
		log.Fatalf("Failed to set up NATS responder: %v", err)
	}

	log.Printf("Health service listening on NATS subject: %s", subject)

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down health service")
}
