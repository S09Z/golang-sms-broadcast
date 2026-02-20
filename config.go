package config

import (
	"os"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	AMQPURL       string
	ProviderURL   string
	DLRWebhookURL string
}

func FromEnv() Config {
	return Config{
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/sms?sslmode=disable"),
		AMQPURL:       getenv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
		ProviderURL:   getenv("PROVIDER_URL", "http://localhost:9090"),
		DLRWebhookURL: getenv("DLR_WEBHOOK_URL", "http://localhost:8081/dlr"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
