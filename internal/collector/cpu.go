package collector

import (
	"context"
	"errors"
	"log/slog"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

func init() {
	Register("cpu", NewCPUCollector)
}

type cpuCollector struct {
	logger *slog.Logger
}

func NewCPUCollector(logger *slog.Logger) Collector {
	c := &cpuCollector{logger: logger}
	// cpu.Percent reports usage since the previous call, so prime the delta
	// baseline at construction — otherwise the first tick reads 0.
	_, _ = cpu.Percent(0, false)
	return c
}

func (c *cpuCollector) Name() string { return "cpu" }

func (c *cpuCollector) Collect(ctx context.Context) ([]Metric, error) {
	usage, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return nil, err
	}
	if len(usage) == 0 {
		return nil, errors.New("no CPU usage data returned")
	}

	metrics := []Metric{
		{Name: "cpu.usage_percent", Value: usage[0], Unit: "%"},
	}

	// Load average is Linux/macOS only. On unsupported systems gopsutil
	// errors here; omit the metrics rather than fail the whole collector.
	if la, err := load.AvgWithContext(ctx); err == nil {
		metrics = append(metrics,
			Metric{Name: "cpu.load_1m", Value: la.Load1},
			Metric{Name: "cpu.load_5m", Value: la.Load5},
			Metric{Name: "cpu.load_15m", Value: la.Load15},
		)
	} else {
		c.logger.Debug("load average unavailable", "err", err)
	}

	return metrics, nil
}
