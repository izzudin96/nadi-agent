package agent

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/izzudin96/nadi-agent/internal/buffer"
	"github.com/izzudin96/nadi-agent/internal/collector"
	"github.com/izzudin96/nadi-agent/internal/sender"
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

	// Hermetic send target so the loop can POST without touching the network.
	server := httptest.NewServer(nil)
	defer server.Close()
	snd := sender.New("test-device", "test-key", server.URL, "dev", logger, server.Client())
	buf, err := buffer.New(filepath.Join(t.TempDir(), "b.jsonl"), 1)
	if err != nil {
		t.Fatalf("buffer.New() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, logger, time.Millisecond, 0, reg, snd, buf)
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

func TestRunSendsHeartbeatEachTick(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	snd := sender.New("test-device", "test-key", server.URL, "dev", logger, server.Client())
	buf, err := buffer.New(filepath.Join(t.TempDir(), "b.jsonl"), 1)
	if err != nil {
		t.Fatalf("buffer.New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, logger, 10*time.Millisecond, 0, regWithSelf(logger), snd, buf)
	}()

	// Allow a few ticks then stop.
	time.Sleep(60 * time.Millisecond)
	cancel()
	<-done

	if got := atomic.LoadInt32(&hits); got < 2 {
		t.Errorf("expected at least 2 heartbeats, got %d", got)
	}
}

func regWithSelf(logger *slog.Logger) *collector.Registry {
	reg, err := collector.NewRegistry(map[string]bool{"self": true}, logger)
	if err != nil {
		panic(err)
	}
	return reg
}
