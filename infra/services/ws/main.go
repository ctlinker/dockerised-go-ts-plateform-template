package main

import (
	natsgo "app-nats-go"
	"app-utils-go/env"
	"app-utils-go/jwt"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	
	"github.com/nats-io/nats.go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, refine this
	},
}

// ClientManager handles active WebSocket connections
type ClientManager struct {
	clients map[int32]*websocket.Conn
	mu      sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[int32]*websocket.Conn),
	}
}

func (cm *ClientManager) Add(userID int32, conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[userID] = conn
}

func (cm *ClientManager) Remove(userID int32) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, userID)
}

func (cm *ClientManager) Send(userID int32, message []byte) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	conn, ok := cm.clients[userID]
	if !ok {
		return fmt.Errorf("user %d not connected", userID)
	}
	return conn.WriteMessage(websocket.TextMessage, message)
}

func main() {
	// 1. Load Config
	authConf := env.LoadAuthConfig()
	serverConf := env.LoadServerConfig()

	// 2. Connect to NATS
	natsClient, err := natsgo.NewClientFromEnv("ws-service")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	manager := NewClientManager()

	// 3. WS Handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// --- Auth Check ---
		// We use a query parameter for the token because custom headers
		// can be difficult to set with standard browser WebSocket APIs.
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		claims, err := jwt.ValidateToken(token, authConf.JWT_ACCESS_SECRET)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// --- Upgrade to WebSocket ---
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade: %v", err)
			return
		}

		userID := claims.UserID
		manager.Add(userID, conn)
		log.Printf("User %d connected via WebSocket", userID)

		// 4. Subscribe to NATS for this user
		// Any message sent to this subject will be pushed to the user's socket
		userSubject := fmt.Sprintf("ws.user.%d", userID)
		sub, err := natsClient.Conn().Subscribe(userSubject, func(msg *nats.Msg) {
			if err := manager.Send(userID, msg.Data); err != nil {
				log.Printf("Failed to send message to user %d: %v", userID, err)
			}
		})
		if err != nil {
			log.Printf("Failed to subscribe to NATS for user %d: %v", userID, err)
			conn.Close()
			return
		}

		// Handle disconnection
		go func() {
			defer func() {
				sub.Unsubscribe()
				manager.Remove(userID)
				conn.Close()
				log.Printf("User %d disconnected", userID)
			}()

			for {
				// We must read to detect connection closure
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()
	})

	// Use port 8081 by default for WS if SERVER_PORT is occupied by Gateway
	port := serverConf.SERVER_PORT
	if port == "" || port == "8080" {
		port = "8081"
	}

	server := &http.Server{Addr: ":" + port}

	// 5. Run Server
	go func() {
		log.Printf("WebSocket service starting on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start WS server: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down WebSocket service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
