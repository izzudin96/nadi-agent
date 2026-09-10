package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

func init() {
	Register("energy", NewEnergyCollector)
}

type energyCollector struct {
	logger *slog.Logger

	// State for rate-based sources (RAPL is a cumulative counter). Reused
	// across ticks so the delta between reads is the power draw.
	mu         sync.Mutex
	prevEnergy float64
	prevTime   time.Time
}

func NewEnergyCollector(logger *slog.Logger) Collector {
	return &energyCollector{logger: logger}
}

func (c *energyCollector) Name() string { return "energy" }

// Collect reads package power draw. The platform-specific source lives in
// energy_linux.go (RAPL) / energy_darwin.go (powermetrics) / energy_other.go.
// Any inability is ErrNoData, never a failure.
func (c *energyCollector) Collect(ctx context.Context) ([]Metric, error) {
	metrics, err := energyMetrics(c, ctx)
	if err != nil {
		c.logger.Debug("energy unavailable", "err", err)
		return nil, ErrNoData
	}
	return metrics, nil
}
