//go:build linux

package collector

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnergyRAPL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "energy_uj")
	if err := os.WriteFile(path, []byte("1000000\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	old := raplEnergyPath
	raplEnergyPath = path
	defer func() { raplEnergyPath = old }()

	c := NewEnergyCollector(discardLogger()).(*energyCollector)

	// First sample: no baseline yet -> error (ErrNoData at the collector layer).
	if _, err := energyMetrics(c, context.Background()); err == nil {
		t.Fatal("expected error on first RAPL sample")
	}

	// Advance the counter and force elapsed time.
	if err := os.WriteFile(path, []byte("3000000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c.prevTime = c.prevTime.Add(-time.Second) // pretend 1s has passed

	metrics, err := energyMetrics(c, context.Background())
	if err != nil {
		t.Fatalf("energyMetrics() error = %v", err)
	}
	if len(metrics) != 1 || metrics[0].Name != "energy.package_power_watts" {
		t.Fatalf("unexpected metrics: %v", metrics)
	}
	// 2,000,000 µJ over ~1s = ~2 W. The elapsed time is wall-clock, so allow
	// a small tolerance for the real time between the two reads.
	if math.Abs(metrics[0].Value-2) > 0.01 {
		t.Fatalf("watts = %v, want ~2", metrics[0].Value)
	}
}
