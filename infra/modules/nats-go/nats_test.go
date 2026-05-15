package natsgo

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestNatsGeneric(t *testing.T) {
	// Try to connect to local NATS, skip if not available
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS server not available, skipping integration test")
	}
	defer nc.Close()

	client := &Client{nc: nc}

	type HelloRequest struct {
		Name string `json:"name"`
	}
	type HelloResponse struct {
		Greeting string `json:"greeting"`
	}

	subject := "test.hello"
	queue := "test.queue"

	// Set up responder
	sub, err := Respond(client, subject, queue, func(ctx context.Context, req HelloRequest) (HelloResponse, error) {
		return HelloResponse{Greeting: "Hello, " + req.Name}, nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Perform request
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := Request[HelloRequest, HelloResponse](ctx, client, subject, HelloRequest{Name: "World"})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	expected := "Hello, World"
	if resp.Greeting != expected {
		t.Errorf("Expected %q, got %q", expected, resp.Greeting)
	}
}
