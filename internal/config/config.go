package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr     string
	RabbitMQConn string
	PostgresConn string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
		RabbitMQConn: os.Getenv("RABBITMQ_CONN"),
		PostgresConn: os.Getenv("POSTGRES_CONN"),
	}

	var missing []string
	if cfg.RabbitMQConn == "" {
		missing = append(missing, "RABBITMQ_CONN")
	}
	if cfg.PostgresConn == "" {
		missing = append(missing, "POSTGRES_CONN")
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
