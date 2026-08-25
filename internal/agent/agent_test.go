package agent

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/izzudin96/nadi-agent/internal/collector"
)

func TestNextIntervalNoJitter(t *testing.T) {
	interval := 5 * time.Second
	if got := nextInterval(interval, 0); got != interval {
		t.Errorf("nextInterval(%v, 0) = %v, want %v", interval, got, interval)
	}
}

func TestNextIntervalWithinBounds(t *testing.T) {
	interval := 10 * time.Second
	jitter := 3 * time.Second
	for i := 0; i < 1000; i++ {
		got := nextInterval(interval, jitter)
		if got < interval || got >= interval+jitter {
			t.Fatalf("nextInterval() = %v, want in [%v, %v)", got, interval, interval+jitter)
		}
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg, err := collector.NewRegistry(nil, logger)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, logger, time.Millisecond, 0, reg)
	}()

	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancel")
	}
}
