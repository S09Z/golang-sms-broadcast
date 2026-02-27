package dlrwebhook
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang-sms-broadcast/internal/adapters/db/postgres"

























































}	_ = fiberApp.Shutdown()	log.Info("shutting down dlr-webhook")	<-ctx.Done()	}()		}			log.Error("fiber listen", "err", err)		if err := fiberApp.Listen(conf.HTTPAddr); err != nil {		log.Info("dlr-webhook listening", "addr", conf.HTTPAddr)	go func() {	defer stop()	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)	handler.Register(fiberApp)	handler := transport.NewHandler(svc, log)	})		AppName: "dlr-webhook",	fiberApp := fiber.New(fiber.Config{	// ── HTTP server (DLR webhook only) ───────────────────────────────────────	svc := app.NewBroadcastService(repo, publisher, provider, log)	// ── Application service ──────────────────────────────────────────────────	provider := httpmock.New(conf.ProviderURL)	defer publisher.Close()	}		os.Exit(1)		log.Error("connect rabbitmq publisher", "err", err)	if err != nil {	publisher, err := rabbitmq.NewPublisher(conf.AMQPURL)	defer repo.Close()	}		os.Exit(1)		log.Error("connect postgres", "err", err)	if err != nil {	repo, err := postgres.New(conf.DatabaseURL)	// ── Adapters ─────────────────────────────────────────────────────────────	conf := cfg.FromEnv()	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))func main() {)	cfg "golang-sms-broadcast/config"	"github.com/gofiber/fiber/v2"	"golang-sms-broadcast/internal/transport"	"golang-sms-broadcast/internal/app"	"golang-sms-broadcast/internal/adapters/queue/rabbitmq"	"golang-sms-broadcast/internal/adapters/provider/httpmock"