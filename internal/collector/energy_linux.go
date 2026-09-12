//go:build linux

package collector

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// raplEnergyPath is the RAPL package energy counter, a monotonically
// increasing value in microjoules. A var so tests can point it elsewhere.
var raplEnergyPath = "/sys/class/powercap/intel-rapl:0/energy_uj"

// energyMetrics computes package power draw as the rate of change of the RAPL
// energy counter. The first sample has no baseline, so it returns ErrNoData.
func energyMetrics(c *energyCollector, ctx context.Context) ([]Metric, error) {
	data, err := os.ReadFile(raplEnergyPath)
	if err != nil {
		return nil, fmt.Errorf("reading RAPL counter: %w", err)
	}
	energy, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return nil, fmt.Errorf("parsing RAPL counter: %w", err)
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.prevTime.IsZero() {
		c.prevEnergy, c.prevTime = energy, now
		return nil, fmt.Errorf("no RAPL baseline yet")
	}

	dt := now.Sub(c.prevTime).Seconds()
	if dt <= 0 {
		return nil, fmt.Errorf("zero elapsed time between RAPL samples")
	}

	watts := (energy - c.prevEnergy) / 1e6 / dt // microjoules → joules → watts
	c.prevEnergy, c.prevTime = energy, now
	if watts < 0 {
		watts = 0
	}
	return []Metric{{Name: "energy.package_power_watts", Value: watts, Unit: "W"}}, nil
}
