package agent

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
)

// Run fires one cycle every interval + a random jitter. It blocks until ctx
// is cancelled, then returns nil so the process can exit cleanly.
func Run(ctx context.Context, logger *slog.Logger, interval, jitter time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			logger.Info("agent stopping")
			return nil
		case <-time.After(nextInterval(interval, jitter)):
			logger.Debug("heartbeat tick")
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
