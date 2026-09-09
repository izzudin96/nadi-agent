package agent

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/izzudin96/nadi-agent/internal/collector"
	"github.com/izzudin96/nadi-agent/internal/sender"
)

// Run fires one cycle every interval + a random jitter. Each cycle runs the
// registry's collectors, then sends the resulting metrics as a heartbeat. It
// blocks until ctx is cancelled, then returns nil so the process can exit
// cleanly.
func Run(ctx context.Context, logger *slog.Logger, interval, jitter time.Duration, reg *collector.Registry, snd *sender.Sender) error {
	for {
		select {
		case <-ctx.Done():
			logger.Info("agent stopping")
			return nil
		case <-time.After(nextInterval(interval, jitter)):
			metrics := reg.Collect(ctx)
			for _, m := range metrics {
				logger.Debug("metric", "name", m.Name, "value", m.Value, "unit", m.Unit)
			}
			if err := snd.Send(ctx, metrics); err != nil {
				// Phase 4 replaces this log with buffering + backoff.
				logger.Error("heartbeat send failed", "err", err, "metrics", len(metrics))
			} else {
				logger.Debug("heartbeat sent", "metrics", len(metrics))
			}
		}
	}
}

// nextInterval returns interval plus a random jitter in [0, jitter).
// Each cycle re-rolls the jitter so multiple devices don't sync up.
func nextInterval(interval, jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return interval
	}
	return interval + time.Duration(rand.Int64N(int64(jitter)))
}
