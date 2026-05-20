package natsgo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps a NATS connection and provides helper methods for common patterns.
type Client struct {
	nc *nats.Conn
}

// NewClient creates a new NATS client.
func NewClient(url string, name string) (*Client, error) {
	nc, err := nats.Connect(url, nats.Name(name),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("Disconnected from NATS: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("Reconnected to NATS: %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Client{nc: nc}, nil
}

// NewClientFromEnv creates a new NATS client using environment variables.
// It looks for NATS_URL (defaulting to nats.DefaultURL) and NATS_CLIENT_NAME.
func NewClientFromEnv(defaultName string) (*Client, error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}
	name := os.Getenv("NATS_CLIENT_NAME")
	if name == "" {
		name = defaultName
	}
	return NewClient(url, name)
}

// Close closes the NATS connection.
func (c *Client) Close() {
	c.nc.Close()
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn {
	return c.nc
}

// JetStream returns a JetStream context.
func (c *Client) JetStream() (nats.JetStreamContext, error) {
	return c.nc.JetStream()
}

// GetKV returns a Key-Value bucket.
func (c *Client) GetKV(bucket string) (nats.KeyValue, error) {
	js, err := c.JetStream()
	if err != nil {
		return nil, err
	}
	kv, err := js.KeyValue(bucket)
	if err != nil {
		// Try to create it if it doesn't exist
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: bucket,
			TTL:    24 * time.Hour, // Default TTL for banned tokens
		})
	}
	return kv, err
}

// Request sends a request and unmarshals the response.
func Request[Req any, Resp any](ctx context.Context, c *Client, subject string, req Req) (Resp, error) {
	var resp Resp
	data, err := json.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := c.nc.RequestWithContext(ctx, subject, data)
	if err != nil {
		return resp, fmt.Errorf("request failed: %w", err)
	}

	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return resp, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return resp, nil
}

// Handler defines a function that processes a request and returns a response.
type Handler[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)

// Respond subscribes to a subject and handles requests using the provided handler.
func Respond[Req any, Resp any](c *Client, subject string, queue string, handler Handler[Req, Resp]) (*nats.Subscription, error) {
	return c.nc.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		var req Req
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Error unmarshaling request: %v", err)
			return
		}

		// Use a background context for now, or consider how to pass context through NATS headers if needed
		resp, err := handler(context.Background(), req)
		if err != nil {
			log.Printf("Error handling request: %v", err)
			// Send error response? NATS doesn't have built-in error signaling in simple Request/Response
			// but we could wrap the response in a standard envelope.
			return
		}

		respData, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshaling response: %v", err)
			return
		}

		if err := msg.Respond(respData); err != nil {
			log.Printf("Error sending response: %v", err)
		}
	})
}

// Publish marshals the data and publishes it to the subject.
func Publish[T any](c *Client, subject string, data T) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	return c.nc.Publish(subject, payload)
}

// Subscribe subscribes to a subject and unmarshals incoming data.
func Subscribe[T any](c *Client, subject string, queue string, handler func(T)) (*nats.Subscription, error) {
	return c.nc.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		var data T
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("Error unmarshaling data: %v", err)
			return
		}
		handler(data)
	})
}
