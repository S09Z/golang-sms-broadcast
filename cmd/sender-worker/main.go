package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang-sms-broadcast/internal/adapters/db/postgres"
	"golang-sms-broadcast/internal/adapters/provider/httpmock"
	"golang-sms-broadcast/internal/adapters/queue/rabbitmq"
	"golang-sms-broadcast/internal/app"
	cfg "golang-sms-broadcast/internal/config"
	"golang-sms-broadcast/internal/domain"
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

	// ── Initialize dependencies ──────────────────────────────────────────────
	repo, err := postgres.New(conf.DatabaseURL)
	if err != nil {
		return errors.New("failed to connect to postgres: " + err.Error())
	}
	defer repo.Close()

	consumer, err := rabbitmq.NewConsumer(conf.AMQPURL, log)
	if err != nil {
		return errors.New("failed to connect to rabbitmq consumer: " + err.Error())
	}
	defer consumer.Close()

	provider := httpmock.New(conf.ProviderURL)

	// Sender worker doesn't need publisher
	svc := app.NewBroadcastService(repo, nil, provider, log)

	// ── Setup consumer ───────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("sender-worker started")

	// Consume blocks until context is cancelled or fatal error
	if err := consumer.Consume(ctx, func(ctx context.Context, msg domain.Message) error {
		return svc.SendMessage(ctx, msg)
	}); err != nil {
		// If context was cancelled, it's a graceful shutdown
		if ctx.Err() != nil {
			log.Info("shutdown signal received")
			return nil
		}
		return errors.New("consumer error: " + err.Error())
	}

	log.Info("sender-worker stopped gracefully")
	return nil
}
