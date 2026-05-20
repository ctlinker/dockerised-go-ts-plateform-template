package main

import (
	natsgo "app-nats-go"
	"app-utils-go/env"
	"app-utils-go/jwt"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
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

	// Initialize KV for checking banned tokens
	kv, err := natsClient.GetKV("banned_tokens")
	if err != nil {
		log.Printf("Warning: Failed to connect to banned_tokens KV: %v. Denylist check will be skipped.", err)
	}

	gatewayConf := env.LoadGatewayConfig()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the context of a request, to aid in closeing it net/http
	r.Use(middleware.Timeout(60 * time.Second))

	// WebSocket Proxy
	wsURL, err := url.Parse(gatewayConf.WS_SERVICE_URL)
	if err != nil {
		log.Fatalf("Invalid WS_SERVICE_URL: %v", err)
	}
	wsProxy := httputil.NewSingleHostReverseProxy(wsURL)

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying WebSocket request to %s", gatewayConf.WS_SERVICE_URL)
		wsProxy.ServeHTTP(w, r)
	})

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

	r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := natsgo.Request[any, any](ctx, natsClient, "auth.register", req)
		if err != nil {
			http.Error(w, "Auth service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := natsgo.Request[any, any](ctx, natsClient, "auth.login", req)
		if err != nil {
			http.Error(w, "Auth service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := natsgo.Request[any, any](ctx, natsClient, "auth.refresh", req)
		if err != nil {
			http.Error(w, "Auth service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Protected Routes
	r.Group(func(r chi.Router) {
		authConf := env.LoadAuthConfig()
		r.Use(AuthMiddleware(authConf.JWT_ACCESS_SECRET, kv))

		r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value("user_id")
			email := r.Context().Value("email")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user_id": userID,
				"email":   email,
			})
		})

		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			// Extract token from header to identify the session
			authHeader := r.Header.Get("Authorization")
			tokenString := authHeader[7:]

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			req := struct {
				AccessToken string `json:"access_token"`
			}{AccessToken: tokenString}

			resp, err := natsgo.Request[any, any](ctx, natsClient, "auth.logout", req)
			if err != nil {
				http.Error(w, "Auth service unavailable", http.StatusServiceUnavailable)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})
	})

	log.Println("Gateway starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Gateway failed to start: %v", err)
	}
}

func AuthMiddleware(secret string, kv nats.KeyValue) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
				http.Error(w, "Unauthorized: Missing or invalid token", http.StatusUnauthorized)
				return
			}

			tokenString := authHeader[7:]

			// CHECK THE DENYLIST (NATS KV)
			if kv != nil {
				entry, err := kv.Get(tokenString)
				if err == nil && entry != nil {
					http.Error(w, "Unauthorized: Token has been revoked (logged out)", http.StatusUnauthorized)
					return
				}
			}

			claims, err := jwt.ValidateToken(tokenString, secret)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "email", claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
