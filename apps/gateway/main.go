package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/startswithzed/pulse/libs/shared/config"
	"github.com/startswithzed/pulse/libs/shared/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", "error", err)
		os.Exit(1)
	}

	logger.Init("gateway", cfg.Service.LogJSON)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	slog.Info("service_started", "service", "gateway", "port", cfg.Service.Port)
	
	err = http.ListenAndServe(":"+cfg.Service.Port, nil)
	if err != nil {
		slog.Error("server_start_failed", "error", err)
		os.Exit(1)
	}
}
