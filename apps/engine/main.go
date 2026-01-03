package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/startswithzed/pulse/libs/shared/config"
	"github.com/startswithzed/pulse/libs/telemetry"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", "error", err)
		os.Exit(1)
	}

	shutdown, err := telemetry.InitSDK(ctx, "engine", "1.0.0", cfg.Telemetry.ExporterEndpoint, cfg.Service.Environment, cfg.Service.LogJSON)
	if err != nil {
		slog.Error("telemetry_init_failed", "error", err)
		os.Exit(1)
	}

	slog.Info("service_started", "service", "engine")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("service_shutting_down", "service", "engine")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := shutdown(shutdownCtx); err != nil {
		slog.Error("telemetry_shutdown_failed", "error", err)
	}
}
