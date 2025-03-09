package main

import (
	"context"
	"github.com/fentezi/export-word/config"
	"github.com/fentezi/export-word/internal/gmail"
	"github.com/fentezi/export-word/internal/kafka"
	"github.com/fentezi/export-word/internal/repository"
	"github.com/fentezi/export-word/internal/service"
	"github.com/fentezi/export-word/pkg/logger"
	"log/slog"
	"os"
)

func run() error {
	cfg := config.MustConfig()
	log := logger.NewLogger(cfg.Env)

	log.Info("init config")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := repository.New(ctx, log, cfg.Mongo)
	if err != nil {
		log.Error("failed to create repository", slog.String("error", err.Error()))
		return err
	}
	defer func() {
		if err := repo.Close(ctx); err != nil {
			log.Error("failed to close repository", slog.String("error", err.Error()))
		}
	}()
	log.Info("repository created")

	email := gmail.New(cfg.Gmail)
	log.Info("gmail client created")

	broker, err := kafka.New(log, cfg.Kafka)
	if err != nil {
		log.Error("failed to create kafka broker", slog.String("error", err.Error()))
		return err
	}
	defer func() {
		if err := broker.Close(); err != nil {
			log.Error("failed to close kafka broker", slog.String("error", err.Error()))
		}
	}()
	log.Info("broker created")

	sv := service.New(log, cfg, broker, email, repo)

	if err := sv.Run(ctx); err != nil {
		log.Error("failed to run service", slog.String("error", err.Error()))
		return err
	}

	log.Info("service run")
	return nil
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}
