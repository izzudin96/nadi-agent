package collector

import (
	"context"
	"log/slog"
)

func init() {
	Register("temperature", NewTemperatureCollector)
}

type temperatureCollector struct {
	logger *slog.Logger
}

func NewTemperatureCollector(logger *slog.Logger) Collector {
	return &temperatureCollector{logger: logger}
}

func (c *temperatureCollector) Name() string { return "temperature" }

// Collect reads the CPU temperature. The platform-specific reading lives in
// temperature_darwin.go / temperature_linux.go / temperature_other.go. Any
// inability (no sensors, no privileges) is ErrNoData, never a failure.
func (c *temperatureCollector) Collect(ctx context.Context) ([]Metric, error) {
	celsius, err := cpuTemperature(ctx)
	if err != nil {
		c.logger.Debug("temperature unavailable", "err", err)
		return nil, ErrNoData
	}
	return []Metric{{Name: "temperature.cpu_celsius", Value: celsius, Unit: "C"}}, nil
}
