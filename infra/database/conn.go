package database

import (
	"database/sql"
	"fmt"
	"log"
	"app-database/schema"
	"app-utils-go/env"
	"strconv"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
	*schema.Queries
}

func parseSSLMode(mode string) (pq.SSLMode, error) {
	switch mode {
	case "disable":
		return pq.SSLModeDisable, nil
	case "require":
		return pq.SSLModeRequire, nil
	case "verify-ca":
		return pq.SSLModeVerifyCA, nil
	case "verify-full":
		return pq.SSLModeVerifyFull, nil
	default:
		return "", fmt.Errorf("invalid SSL mode: %s", mode)
	}
}

// Connect to the database
func Connect(conf env.DBConfig) *DB {

	port, err := strconv.Atoi(conf.Port)
	if err != nil {
		log.Fatal("Failed to convert port to int", err)
	}

	sslMode, err := parseSSLMode(conf.SSLMode)
	if err != nil {
		log.Fatal("Failed to parse SSL mode", err)
	}

	cfg := pq.Config{
		Host:     conf.Host,
		Port:     uint16(port),
		User:     conf.User,
		Password: conf.Password,
		Database: conf.Database,
		SSLMode:  sslMode,
	}

	c, err := pq.NewConnectorConfig(cfg)
	if err != nil {
		log.Fatal("Failed to create connector config", err)
	}

	// Create connection pool.
	db := sql.OpenDB(c)

	// Make sure it works.
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database", err)
	}

	return &DB{db, schema.New(db)}
}
