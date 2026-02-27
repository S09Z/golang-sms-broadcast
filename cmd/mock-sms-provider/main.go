package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// mockSendRequest mirrors what httpmock.Client sends to /send.
type mockSendRequest struct {
	MessageID string `json:"message_id"`
	To        string `json:"to"`
	Body      string `json:"body"`
	DLRHook   string `json:"dlr_webhook_url"`
}

// mockSendResponse is what the mock returns.
type mockSendResponse struct {
	ProviderID string `json:"provider_id"`
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := getenv("HTTP_ADDR", ":9090")
	dlrHook := getenv("DLR_WEBHOOK_URL", "http://localhost:8081/dlr")

	fiberApp := fiber.New(fiber.Config{AppName: "mock-sms-provider"})

	// POST /send — accepts an SMS submission and echoes back a generated provider ID.
	fiberApp.Post("/send", func(c *fiber.Ctx) error {
		var req mockSendRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		providerID := uuid.New().String()
		log.Info("mock provider received message",
			"message_id", req.MessageID,
			"to", req.To,
			"provider_id", providerID,
		)

		// Use the hook URL from the request body; fall back to env var.
		hook := req.DLRHook
		if hook == "" {
			hook = dlrHook
		}
		go simulateDLR(hook, providerID, log)

		return c.Status(fiber.StatusAccepted).JSON(mockSendResponse{ProviderID: providerID})
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("mock-sms-provider listening", "addr", addr)
		if err := fiberApp.Listen(addr); err != nil {
			log.Error("fiber listen", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down mock-sms-provider")
	_ = fiberApp.Shutdown()
}

// simulateDLR posts a delivery receipt to the DLR webhook after a short delay.
func simulateDLR(hookURL, providerID string, log *slog.Logger) {
	time.Sleep(500 * time.Millisecond) // simulate async network delivery

	payload := map[string]string{
		"provider_id": providerID,
		"status":      "delivered",
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hookURL, bytes.NewReader(body))
	if err != nil {
		log.Error("create dlr request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("dlr webhook call failed", "provider_id", providerID, "err", err)
		return
	}
	defer resp.Body.Close()
	log.Info("dlr webhook called", "provider_id", providerID, "status", resp.StatusCode)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
