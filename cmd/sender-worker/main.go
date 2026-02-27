package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang-sms-broadcast/internal/adapters/db/postgres"
	"golang-sms-broadcast/internal/adapters/provider/httpmock"
	"golang-sms-broadcast/internal/adapters/queue/rabbitmq"
	"golang-sms-broadcast/internal/app"
	"golang-sms-broadcast/internal/domain"

	cfg "golang-sms-broadcast/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	conf := cfg.FromEnv()

	// ── Adapters ─────────────────────────────────────────────────────────────
	repo, err := postgres.New(conf.DatabaseURL)
	if err != nil {
		log.Error("connect postgres", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	publisher, err := rabbitmq.NewPublisher(conf.AMQPURL)
	if err != nil {
		log.Error("connect rabbitmq publisher", "err", err)
		os.Exit(1)
	}
	defer publisher.Close()

	consumer, err := rabbitmq.NewConsumer(conf.AMQPURL, log)
	if err != nil {
		log.Error("connect rabbitmq consumer", "err", err)
		os.Exit(1)
	}
	defer consumer.Close()

	provider := httpmock.New(conf.ProviderURL)

	// ── Application service ──────────────────────────────────────────────────
	svc := app.NewBroadcastService(repo, publisher, provider, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("sender-worker started")

	if err := consumer.Consume(ctx, func(ctx context.Context, msg domain.Message) error {
		return svc.SendMessage(ctx, msg)
	}); err != nil && ctx.Err() == nil {
		log.Error("consumer error", "err", err)
		os.Exit(1)
	}

	log.Info("shutting down sender-worker")
}
