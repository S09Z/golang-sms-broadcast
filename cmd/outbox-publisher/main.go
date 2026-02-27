package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang-sms-broadcast/internal/adapters/db/postgres"
	"golang-sms-broadcast/internal/adapters/provider/httpmock"
	"golang-sms-broadcast/internal/adapters/queue/rabbitmq"
	"golang-sms-broadcast/internal/app"

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

	provider := httpmock.New(conf.ProviderURL)

	// ── Application service ──────────────────────────────────────────────────
	svc := app.NewBroadcastService(repo, publisher, provider, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Info("outbox-publisher started", "interval", "5s")

	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down outbox-publisher")
			return

		case <-ticker.C:
			n, err := svc.PublishPendingMessages(ctx, 100)
			if err != nil {
				log.Error("publish pending messages", "err", err)
				continue
			}
			if n > 0 {
				log.Info("published pending messages", "count", n)
			}
		}
	}
}
