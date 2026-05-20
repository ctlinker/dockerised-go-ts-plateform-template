package main

import (
	natsgo "app-nats-go"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Placeholder for future implementation
type AuthRequest struct {
	Action string `json:"action"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func main() {
	// 1. Connect to NATS
	client, err := natsgo.NewClientFromEnv("auth-service")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	subject := "service.auth"
	queue := "auth-service-group"

	// 2. Set up basic responder (Skeleton)
	_, err = natsgo.Respond(client, subject, queue, func(ctx context.Context, req AuthRequest) (AuthResponse, error) {
		log.Printf("Received auth request for action: %s", req.Action)
		return AuthResponse{
			Success: true,
			Message: "Auth service is alive",
		}, nil
	})

	if err != nil {
		log.Fatalf("Failed to set up NATS responder: %v", err)
	}

	log.Printf("Auth service listening on NATS subject: %s", subject)

	// 3. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down auth service")
}
