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
	"golang-sms-broadcast/internal/adapters/provider/httpmock"
	"golang-sms-broadcast/internal/adapters/queue/rabbitmq"
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

	repo, err := postgres.New(conf.DatabaseURL)
	if err != nil {
		return errors.New("failed to connect to postgres: " + err.Error())
	}
	defer repo.Close()

	publisher, err := rabbitmq.NewPublisher(conf.AMQPURL)
	if err != nil {
		return errors.New("failed to connect to rabbitmq: " + err.Error())
	}
	defer publisher.Close()

	provider := httpmock.New(conf.ProviderURL)
	svc := app.NewBroadcastService(repo, publisher, provider, log)

	fiberApp := fiber.New(fiber.Config{
		AppName:               "broadcast-api",
		DisableStartupMessage: true,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
		// OWASP: Disable server header to reduce information disclosure
		ServerHeader: "",
		// OWASP: Limit body size to prevent memory exhaustion attacks
		BodyLimit: 1 * 1024 * 1024, // 1MB
	})

	// ═══════════════════════════════════════════════════════════
	// Global Middleware - OWASP 2026 Security Best Practices
	// ═══════════════════════════════════════════════════════════

	// 1. Panic Recovery - prevents application crashes
	fiberApp.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// 2. Request Logging - audit trail for security monitoring
	fiberApp.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// 3. Request ID - tracing and correlation
	fiberApp.Use(middleware.RequestIDMiddleware())

	// 4. Security Headers - OWASP recommended headers
	fiberApp.Use(middleware.SecurityHeaders())

	// 5. CORS - Cross-Origin Resource Sharing
	fiberApp.Use(middleware.CORSConfig())

	// 6. Rate Limiting - prevent brute force and DDoS
	// 100 requests per minute per IP
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)
	fiberApp.Use(rateLimiter.Middleware())

	// ═══════════════════════════════════════════════════════════
	// Routes
	// ═══════════════════════════════════════════════════════════

	fiberApp.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	handler := transport.NewHandler(svc, log)
	api := fiberApp.Group("/api")
	handler.Register(api)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	go func() {
		log.Info("broadcast-api started", "addr", conf.HTTPAddr)
		if err := fiberApp.Listen(conf.HTTPAddr); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errChan:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := fiberApp.ShutdownWithContext(shutdownCtx); err != nil {
		return errors.New("failed to shutdown gracefully: " + err.Error())
	}

	log.Info("broadcast-api stopped gracefully")
	return nil
}
