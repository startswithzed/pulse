package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/startswithzed/pulse/libs/shared/config"
	"github.com/startswithzed/pulse/libs/shared/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", "error", err)
		os.Exit(1)
	}

	logger.Init("engine", cfg.Service.LogJSON)

	slog.Info("service_started", "service", "engine")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
