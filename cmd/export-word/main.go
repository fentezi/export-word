package main

import (
	"context"
	"github.com/fentezi/export-word/internal/config"
	"github.com/fentezi/export-word/internal/repository"
	"github.com/fentezi/export-word/internal/service"
	"github.com/fentezi/export-word/pkg/logging"
	"log"
	"log/slog"
	"os"
)

func main() {
	log.Print("config initializing")
	cfg := config.MustConfig()

	log.Print("logger initialized")
	logger := logging.New(cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Print("repository initialized")
	repo, err := repository.New(ctx, cfg.Mongo, logger)
	if err != nil {
		logger.Error("failed to create repository", slog.String("error", err.Error()))
	}
	defer repo.Close(ctx)

	expWord, err := service.New(logger, cfg, repo)
	if err != nil {
		logger.Error("failed to create service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := expWord.Run(ctx); err != nil {
		logger.Error("failed to run service", slog.String("error", err.Error()))
	}

	logger.Info("service run")
}
