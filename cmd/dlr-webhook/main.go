package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang-sms-broadcast/internal/adapters/db/postgres"
	"golang-sms-broadcast/internal/app"
	cfg "golang-sms-broadcast/internal/config"
	"golang-sms-broadcast/internal/middleware"
	"golang-sms-broadcast/internal/transport"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	if err := run(log); err != nil {
		log.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	conf := cfg.FromEnv()
	addr := getenvOrDefault("DLR_WEBHOOK_ADDR", ":8081")

	repo, err := postgres.New(conf.DatabaseURL)
	if err != nil {
		return errors.New("failed to connect to postgres: " + err.Error())
	}
	defer repo.Close()

	svc := app.NewBroadcastService(repo, nil, nil, log)

	fiberApp := fiber.New(fiber.Config{
		AppName:               "dlr-webhook",
		DisableStartupMessage: true,
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          5 * time.Second,
		IdleTimeout:           60 * time.Second,
		ServerHeader:          "",
		BodyLimit:             512 * 1024, // 512KB - webhooks are small
	})

	// Security Middleware
	fiberApp.Use(recover.New())
	fiberApp.Use(logger.New())
	fiberApp.Use(middleware.RequestIDMiddleware())
	fiberApp.Use(middleware.SecurityHeaders())

	// Rate limiting for webhook endpoint (200 req/min per IP)
	rateLimiter := middleware.NewRateLimiter(200, 1*time.Minute)
	fiberApp.Use(rateLimiter.Middleware())

	fiberApp.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	handler := transport.NewHandler(svc, log)
	fiberApp.Post("/dlr", handler.HandleDLR)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	go func() {
		log.Info("dlr-webhook started", "addr", addr)
		if err := fiberApp.Listen(addr); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errChan:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := fiberApp.ShutdownWithContext(shutdownCtx); err != nil {
		return errors.New("failed to shutdown gracefully: " + err.Error())
	}

	log.Info("dlr-webhook stopped gracefully")
	return nil
}

func getenvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
