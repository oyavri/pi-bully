package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/oyavri/pi-bully/config"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		zap.L().Fatal("failed to load config", zap.Error(err))
		os.Exit(1)
	}

	logger, err := cfg.Log.BuildLogger()
	if err != nil {
		os.Exit(1)
	}
	defer logger.Sync()

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	<-ctx.Done()
	logger.Info("shutting down")
}
