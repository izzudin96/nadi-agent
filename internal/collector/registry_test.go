package collector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeCollector struct {
	name    string
	metrics []Metric
	err     error
}

func (f *fakeCollector) Name() string { return f.name }
func (f *fakeCollector) Collect(context.Context) ([]Metric, error) {
	return f.metrics, f.err
}

func TestNewRegistryRejectsUnknownCollector(t *testing.T) {
	_, err := NewRegistry(map[string]bool{"does-not-exist": true}, discardLogger())
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected unknown collector error, got: %v", err)
	}
}

func TestNewRegistryEnablesRegisteredCollectors(t *testing.T) {
	reg, err := NewRegistry(map[string]bool{"self": true}, discardLogger())
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if len(reg.collectors) != 1 {
		t.Fatalf("got %d collectors, want 1", len(reg.collectors))
	}
}

func TestCollectAggregatesMetrics(t *testing.T) {
	reg := newRegistry(map[string]Collector{
		"a": &fakeCollector{name: "a", metrics: []Metric{{Name: "a.x", Value: 1}}},
		"b": &fakeCollector{name: "b", metrics: []Metric{{Name: "b.x", Value: 2}, {Name: "b.y", Value: 3}}},
	}, discardLogger())

	got := reg.Collect(context.Background())
	if len(got) != 3 {
		t.Fatalf("Collected %d metrics, want 3", len(got))
	}
}

func TestCollectSkipsFailingCollector(t *testing.T) {
	reg := newRegistry(map[string]Collector{
		"bad": &fakeCollector{name: "bad", err: errors.New("boom")},
		"ok":  &fakeCollector{name: "ok", metrics: []Metric{{Name: "ok.x", Value: 1}}},
	}, discardLogger())

	got := reg.Collect(context.Background())
	if len(got) != 2 {
		t.Fatalf("expected ok.x + agent.collector_errors_count, got %v", got)
	}
	if got[0].Name != "ok.x" || got[1].Name != "agent.collector_errors_count" {
		t.Fatalf("unexpected metrics: %v", got)
	}
}

func TestCollectOmitsNoDataCollector(t *testing.T) {
	reg := newRegistry(map[string]Collector{
		"nodata": &fakeCollector{name: "nodata", err: ErrNoData},
		"ok":     &fakeCollector{name: "ok", metrics: []Metric{{Name: "ok.x", Value: 1}}},
	}, discardLogger())

	got := reg.Collect(context.Background())
	if len(got) != 1 || got[0].Name != "ok.x" {
		t.Fatalf("expected only ok.x metric (no errors_count for ErrNoData), got %v", got)
	}
}

type slowCollector struct {
	name string
}

func (s *slowCollector) Name() string { return s.name }
func (s *slowCollector) Collect(ctx context.Context) ([]Metric, error) {
	<-ctx.Done() // respect the deadline, then report it
	return nil, ctx.Err()
}

func TestCollectSkipsTimedOutCollector(t *testing.T) {
	old := collectorTimeout
	collectorTimeout = 50 * time.Millisecond
	defer func() { collectorTimeout = old }()

	reg := newRegistry(map[string]Collector{
		"slow": &slowCollector{name: "slow"},
		"ok":   &fakeCollector{name: "ok", metrics: []Metric{{Name: "ok.x", Value: 1}}},
	}, discardLogger())

	start := time.Now()
	got := reg.Collect(context.Background())
	elapsed := time.Since(start)

	if len(got) != 2 {
		t.Fatalf("expected ok.x + errors_count, got %v", got)
	}
	if elapsed > time.Second {
		t.Fatalf("cycle took %v, slow collector stalled it", elapsed)
	}
	if got[1].Name != "agent.collector_errors_count" || got[1].Value != 1 {
		t.Fatalf("expected 1 collector error, got %v", got[1])
	}
}
