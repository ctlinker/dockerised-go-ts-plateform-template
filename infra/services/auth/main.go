package main

import (
	database "app-database"
	"app-database/schema"
	natsgo "app-nats-go"
	"app-utils-go/env"
	"app-utils-go/jwt"
	"app-utils-go/password"
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
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

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	AccessToken string `json:"access_token"`
}

type AuthResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func main() {
	// 1. Load Env & Connect to DB
	dbConf := env.LoadDBConfig()
	authConf := env.LoadAuthConfig()
	db := database.Connect(dbConf)
	defer db.Close()

	// 2. Connect to NATS
	client, err := natsgo.NewClientFromEnv("auth-service")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// 3. Register Handlers
	setupHandlers(client, db, authConf)

	log.Println("Auth service is running...")

	// 4. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down auth service")
}

func setupHandlers(nc *natsgo.Client, db *database.DB, authConf env.AuthConfig) {
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

		// 1. Generate Access Token (Short-lived: 15 mins)
		accessToken, err := jwt.GenerateToken(user.ID, user.Email, authConf.JWT_ACCESS_SECRET, 15*time.Minute)
		if err != nil {
			log.Printf("Failed to generate access token: %v", err)
			return AuthResponse{Success: false, Message: "Internal error"}, err
		}

		// 2. Generate Refresh Token (Long-lived: 7 days)
		refreshToken, err := jwt.GenerateToken(user.ID, user.Email, authConf.JWT_REFRESH_SECRET, 7*24*time.Hour)
		if err != nil {
			log.Printf("Failed to generate refresh token: %v", err)
			return AuthResponse{Success: false, Message: "Internal error"}, err
		}

		// 3. Store Session in DB
		_, err = db.CreateSession(ctx, schema.CreateSessionParams{
			UserID:           user.ID,
			TokenHash:        accessToken,
			RefreshTokenHash: sql.NullString{String: refreshToken, Valid: true},
			ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		})
		if err != nil {
			log.Printf("Failed to store session: %v", err)
		}

		return AuthResponse{
			Success:      true,
			Message:      "Logged in successfully",
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil
	})

	// Refresh Token Handler
	natsgo.Respond(nc, "auth.refresh", "auth-service-group", func(ctx context.Context, req RefreshRequest) (AuthResponse, error) {
		// 1. Validate the refresh token
		claims, err := jwt.ValidateToken(req.RefreshToken, authConf.JWT_REFRESH_SECRET)
		if err != nil {
			return AuthResponse{Success: false, Message: "Invalid refresh token"}, nil
		}

		// 2. Check if the session exists in DB and isn't revoked
		session, err := db.GetSessionByRefreshTokenHash(ctx, sql.NullString{String: req.RefreshToken, Valid: true})
		if err != nil {
			return AuthResponse{Success: false, Message: "Session expired or revoked"}, nil
		}

		// 3. Generate new Access Token
		newAccessToken, err := jwt.GenerateToken(claims.UserID, claims.Email, authConf.JWT_ACCESS_SECRET, 15*time.Minute)
		if err != nil {
			return AuthResponse{Success: false, Message: "Internal error"}, err
		}

		// 4. Update session with new access token hash
		err = db.UpdateSessionTokenHash(ctx, schema.UpdateSessionTokenHashParams{
			TokenHash:   session.TokenHash, // Old hash to find
			TokenHash_2: newAccessToken,    // New hash to set
		})

		return AuthResponse{
			Success:      true,
			Message:      "Token refreshed successfully",
			AccessToken:  newAccessToken,
			RefreshToken: req.RefreshToken, // Keep using the same refresh token
		}, nil
	})

	// Logout Handler
	natsgo.Respond(nc, "auth.logout", "auth-service-group", func(ctx context.Context, req LogoutRequest) (AuthResponse, error) {
		// Mark session as deleted based on access token
		err := db.SoftDeleteSessionByTokenHash(ctx, req.AccessToken)
		if err != nil {
			return AuthResponse{Success: false, Message: "Failed to logout"}, err
		}

		return AuthResponse{Success: true, Message: "Logged out successfully"}, nil
	})
}
