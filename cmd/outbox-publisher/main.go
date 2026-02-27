package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang-sms-broadcast/internal/adapters/db/postgres"
	"golang-sms-broadcast/internal/adapters/queue/rabbitmq"
	"golang-sms-broadcast/internal/app"
	cfg "golang-sms-broadcast/internal/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	if err := run(log); err != nil {
		log.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	conf := cfg.FromEnv()

	// Configurable polling interval
	interval := getEnvDuration("OUTBOX_POLL_INTERVAL", 5*time.Second)
	batchSize := getEnvInt("OUTBOX_BATCH_SIZE", 100)

	// ── Initialize dependencies ──────────────────────────────────────────────
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

	// Outbox publisher doesn't need provider
	svc := app.NewBroadcastService(repo, publisher, nil, log)

	// ── Setup polling loop ───────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("outbox-publisher started",
		"interval", interval.String(),
		"batch_size", batchSize,
	)

	// Initial poll immediately
	if err := pollOnce(ctx, svc, batchSize, log); err != nil {
		log.Error("initial poll failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("shutdown signal received")
			return nil

		case <-ticker.C:
			if err := pollOnce(ctx, svc, batchSize, log); err != nil {
				log.Error("poll failed", "error", err)
				// Continue on error - don't crash the service
			}
		}
	}
}

func pollOnce(ctx context.Context, svc *app.BroadcastService, batchSize int, log *slog.Logger) error {
	n, err := svc.PublishPendingMessages(ctx, batchSize)
	if err != nil {
		return err
	}

	if n > 0 {
		log.Info("published messages", "count", n)
	}

	return nil
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}

	return d
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}

	return i
}
