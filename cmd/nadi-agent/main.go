package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/izzudin96/nadi-agent/internal/agent"
	"github.com/izzudin96/nadi-agent/internal/collector"
	"github.com/izzudin96/nadi-agent/internal/config"
	"github.com/izzudin96/nadi-agent/internal/sender"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	jitter := time.Duration(cfg.JitterSeconds) * time.Second

	reg, err := collector.NewRegistry(cfg.Collectors, logger)
	if err != nil {
		return err
	}

	snd := sender.New(cfg.DeviceID, cfg.APIKey, cfg.ServerURL, version, logger, nil)

	logger.Info("agent starting",
		"version", version,
		"device_id", cfg.DeviceID,
		"server_url", cfg.ServerURL,
		"interval_seconds", cfg.IntervalSeconds,
		"jitter_seconds", cfg.JitterSeconds,
		"collectors", reg.Names(),
	)

	if err := agent.Run(ctx, logger, interval, jitter, reg, snd); err != nil {
		return err
	}

	logger.Info("agent stopped")
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
