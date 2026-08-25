package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/izzudin96/nadi-agent/internal/config"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("agent exiting", "err", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "agent.yaml", "path to agent config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger := newLogger(cfg.LogLevel)

	logger.Info("agent starting",
		"version", version,
		"device_id", cfg.DeviceID,
		"server_url", cfg.ServerURL,
		"interval_seconds", cfg.IntervalSeconds,
	)

	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
