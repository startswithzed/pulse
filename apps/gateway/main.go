package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/startswithzed/pulse/libs/shared/config"
	"github.com/startswithzed/pulse/libs/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", "error", err)
		os.Exit(1)
	}

	shutdown, err := telemetry.InitSDK(ctx, "gateway", "1.0.0", cfg.Telemetry.ExporterEndpoint, cfg.Service.Environment, cfg.Service.LogJSON)
	if err != nil {
		slog.Error("telemetry_init_failed", "error", err)
		os.Exit(1)
	}

	router := gin.Default()
	router.Use(otelgin.Middleware("gateway"))

	router.GET("/", func(c *gin.Context) {
		slog.InfoContext(c.Request.Context(), "request_processed")
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello World",
			"status":  "ok",
		})
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Service.Port,
		Handler: router.Handler(),
	}

	slog.Info("service_started", "service", "gateway", "port", cfg.Service.Port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server_start_failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("service_shutting_down", "service", "gateway")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server_shutdown_failed", "error", err)
	}

	if err := shutdown(shutdownCtx); err != nil {
		slog.Error("telemetry_shutdown_failed", "error", err)
	}
}
