package main

import (
	database "app-database"
	"app-database/schema"
	natsgo "app-nats-go"
	"app-utils-go/env"
	"app-utils-go/password"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

func main() {
	// 1. Load Env & Connect to DB
	dbConf := env.LoadDBConfig()
	db := database.Connect(dbConf)
	defer db.Close()

	// 2. Connect to NATS
	client, err := natsgo.NewClientFromEnv("auth-service")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// 3. Register Handlers
	setupHandlers(client, db)

	log.Println("Auth service is running...")

	// 4. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down auth service")
}

func setupHandlers(nc *natsgo.Client, db *database.DB) {
	// Register Handler
	natsgo.Respond(nc, "auth.register", "auth-service-group", func(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
		// Check if user exists
		_, err := db.GetUserByEmail(ctx, req.Email)
		if err == nil {
			return AuthResponse{Success: false, Message: "User already exists"}, nil
		}

		// Hash password
		hashedPassword, err := password.HashPassword(req.Password)
		if err != nil {
			return AuthResponse{Success: false, Message: "Internal error"}, err
		}

		// Create user
		_, err = db.CreateUser(ctx, schema.CreateUserParams{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: hashedPassword,
		})
		if err != nil {
			return AuthResponse{Success: false, Message: "Failed to create user"}, err
		}

		return AuthResponse{Success: true, Message: "User created successfully"}, nil
	})

	// Login Handler
	natsgo.Respond(nc, "auth.login", "auth-service-group", func(ctx context.Context, req LoginRequest) (AuthResponse, error) {
		user, err := db.GetUserByEmail(ctx, req.Email)
		if err != nil {
			return AuthResponse{Success: false, Message: "Invalid credentials"}, nil
		}

		if !password.CheckPasswordHash(req.Password, user.PasswordHash) {
			return AuthResponse{Success: false, Message: "Invalid credentials"}, nil
		}

		// TODO: Generate JWT
		return AuthResponse{
			Success: true,
			Message: "Logged in successfully",
			Token:   "placeholder-jwt-token",
		}, nil
	})
}
