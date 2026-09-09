package collector

import (
	"context"
	"log/slog"
)

func init() {
	Register("battery", NewBatteryCollector)
}

type batteryCollector struct {
	logger *slog.Logger
}

func NewBatteryCollector(logger *slog.Logger) Collector {
	return &batteryCollector{logger: logger}
}

func (c *batteryCollector) Name() string { return "battery" }

// Collect is the cross-platform shell. The actual reading lives in
// battery_darwin.go / battery_linux.go / battery_other.go (see build tags).
// Battery is best-effort: any inability to read it (no battery, no tooling)
// becomes ErrNoData, never a collector failure.
func (c *batteryCollector) Collect(ctx context.Context) ([]Metric, error) {
	metrics, err := batteryMetrics(ctx)
	if err != nil {
		c.logger.Debug("battery unavailable", "err", err)
		return nil, ErrNoData
	}
	return metrics, nil
}
