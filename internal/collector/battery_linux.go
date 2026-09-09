//go:build linux

package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const powerSupplyDir = "/sys/class/power_supply"

// batteryMetrics reads battery state from the kernel's power_supply interface.
// No battery on the system (typical for a server) is ErrNoData, not an error.
func batteryMetrics(ctx context.Context) ([]Metric, error) {
	batts, err := filepath.Glob(filepath.Join(powerSupplyDir, "BAT*"))
	if err != nil || len(batts) == 0 {
		return nil, fmt.Errorf("no battery in %s", powerSupplyDir)
	}
	dir := batts[0]

	capacity, err := readIntFile(filepath.Join(dir, "capacity"))
	if err != nil {
		return nil, err
	}

	charging, pluggedIn := batteryState(readStringFile(filepath.Join(dir, "status")))

	metrics := []Metric{
		{Name: "battery.percent", Value: capacity, Unit: "%"},
		{Name: "battery.charging", Value: charging, Unit: "bool"},
		{Name: "battery.plugged_in", Value: pluggedIn, Unit: "bool"},
	}

	// Cycle count + health are vendor-dependent files; omit if absent.
	if cycle, err := readIntFile(filepath.Join(dir, "cycle_count")); err == nil {
		metrics = append(metrics, Metric{Name: "battery.cycle_count", Value: cycle, Unit: "count"})
	}
	if energyFull, err := readIntFile(filepath.Join(dir, "energy_full")); err == nil {
		if design, err := readIntFile(filepath.Join(dir, "energy_full_design")); err == nil && design > 0 {
			metrics = append(metrics, Metric{Name: "battery.health_percent", Value: energyFull / design * 100, Unit: "%"})
		}
	}
	return metrics, nil
}

func batteryState(status string) (charging, pluggedIn float64) {
	switch {
	case strings.Contains(status, "Charg"):
		return 1, 1
	case status == "Full":
		return 0, 1
	}
	return 0, 0
}

func readIntFile(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
}

func readStringFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
