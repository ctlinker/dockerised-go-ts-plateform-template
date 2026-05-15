package main

import (
	natsgo "app-nats-go"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type HealthRequest struct{}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	// Initialize NATS client
	natsClient, err := natsgo.NewClientFromEnv("gateway")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the context of a request, to aid in closeing it net/http
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Request health status from health-service via NATS
		resp, err := natsgo.Request[HealthRequest, HealthResponse](ctx, natsClient, "service.health", HealthRequest{})
		if err != nil {
			log.Printf("Health check request failed: %v", err)
			http.Error(w, "Health service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("Gateway starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Gateway failed to start: %v", err)
	}
}
