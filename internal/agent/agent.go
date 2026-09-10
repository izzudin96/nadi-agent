package agent

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/izzudin96/nadi-agent/internal/backoff"
	"github.com/izzudin96/nadi-agent/internal/buffer"
	"github.com/izzudin96/nadi-agent/internal/collector"
	"github.com/izzudin96/nadi-agent/internal/sender"
)

// Run is the agent's main loop. Each cycle it:
//  1. runs the collectors,
//  2. flushes any buffered heartbeats (oldest first),
//  3. sends the current heartbeat.
//
// A failed send is buffered to disk and retried on an exponential-backoff
// timer, so a downed server costs no data. It blocks until ctx is cancelled.
func Run(ctx context.Context, logger *slog.Logger, interval, jitter time.Duration, reg *collector.Registry, snd *sender.Sender, buf *buffer.Buffer) error {
	r := &retrier{backoff: backoff.New(time.Second, interval), logger: logger}

	// A persistent tick timer, reset only after a collection tick. Using
	// time.After here instead would re-arm the timer on every retry, delaying
	// collection during an outage (retries would keep resetting the tick).
	ticker := time.NewTimer(nextInterval(interval, jitter))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.disarm()
			logger.Info("agent stopping")
			return nil

		case <-ticker.C:
			metrics := reg.Collect(ctx)
			for _, m := range metrics {
				logger.Debug("metric", "name", m.Name, "value", m.Value, "unit", m.Unit)
			}

			data, err := snd.Build(metrics)
			if err != nil {
				logger.Error("building heartbeat", "err", err)
				ticker.Reset(nextInterval(interval, jitter))
				continue
			}

			flushBuffer(ctx, snd, buf, logger)

			if err := snd.SendRaw(ctx, data); err != nil {
				logger.Error("heartbeat send failed", "err", err, "metrics", len(metrics))
				if aerr := buf.Append(data); aerr != nil {
					logger.Error("buffering heartbeat", "err", aerr)
				}
				r.arm()
			} else {
				logger.Debug("heartbeat sent", "metrics", len(metrics))
				r.reset()
			}

			ticker.Reset(nextInterval(interval, jitter))

		case <-r.C():
			r.fired()
			flushBuffer(ctx, snd, buf, logger)
			if buf.Len() == 0 {
				logger.Info("buffer drained")
				r.reset()
			} else {
				r.arm()
			}
		}
	}
}

// flushBuffer sends buffered payloads oldest-first until one fails or the
// buffer empties, then compacts the file to drop the sent entries.
func flushBuffer(ctx context.Context, snd *sender.Sender, buf *buffer.Buffer, logger *slog.Logger) {
	flushed := 0
	for {
		oldest, ok := buf.Peek()
		if !ok {
			break
		}
		if err := snd.SendRaw(ctx, oldest); err != nil {
			break // server still down; keep the rest
		}
		buf.Pop()
		flushed++
	}
	if flushed > 0 {
		if err := buf.Persist(); err != nil {
			logger.Error("compacting buffer", "err", err)
		}
		logger.Debug("flushed buffered heartbeats", "count", flushed)
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

// retrier manages the one-shot backoff timer. A nil timer means "disarmed";
// its C() channel (nil) then blocks forever in select, so the retry case is
// simply skipped until armed again.
type retrier struct {
	backoff *backoff.Backoff
	logger  *slog.Logger
	timer   *time.Timer
}

func (r *retrier) arm() {
	if r.timer != nil {
		return // already armed
	}
	d := r.backoff.Next()
	r.timer = time.NewTimer(d)
	r.logger.Debug("retry scheduled", "delay", d.String())
}

func (r *retrier) fired() {
	r.timer = nil
}

func (r *retrier) disarm() {
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
}

func (r *retrier) reset() {
	r.backoff.Reset()
	r.disarm()
}

func (r *retrier) C() <-chan time.Time {
	if r.timer == nil {
		return nil
	}
	return r.timer.C
}
