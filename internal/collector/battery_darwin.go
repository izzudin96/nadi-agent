//go:build darwin

package collector

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// batteryMetrics reads battery state from pmset and IORegistry. pmset output
// looks like:
//
//	Now drawing from 'AC Power'
//	 -InternalBattery-0 (id=5005157)	75%; discharging; 4:34 remaining present: true
func batteryMetrics(ctx context.Context) ([]Metric, error) {
	out, err := exec.CommandContext(ctx, "pmset", "-g", "batt").Output()
	if err != nil {
		return nil, err
	}
	text := string(out)

	percent, err := batteryPercent(text)
	if err != nil {
		return nil, err
	}
	charging, pluggedIn := batteryState(text)

	metrics := []Metric{
		{Name: "battery.percent", Value: percent, Unit: "%"},
		{Name: "battery.charging", Value: charging, Unit: "bool"},
		{Name: "battery.plugged_in", Value: pluggedIn, Unit: "bool"},
	}

	// Cycle count + health come from IORegistry; omit silently if unavailable.
	if cycle, health, ok := batteryHealth(ctx); ok {
		metrics = append(metrics,
			Metric{Name: "battery.cycle_count", Value: cycle, Unit: "count"},
			Metric{Name: "battery.health_percent", Value: health, Unit: "%"},
		)
	}
	return metrics, nil
}

func batteryPercent(text string) (float64, error) {
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "%") {
			continue
		}
		idx := strings.Index(line, "%")
		start := idx
		for start > 0 && line[start-1] >= '0' && line[start-1] <= '9' {
			start--
		}
		return strconv.ParseFloat(line[start:idx], 64)
	}
	return 0, fmt.Errorf("no battery percent in pmset output")
}

func batteryState(text string) (charging, pluggedIn float64) {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "ac power") {
		pluggedIn = 1
	}
	switch {
	case strings.Contains(lower, "discharging"):
		return 0, pluggedIn
	case strings.Contains(lower, "charging"):
		return 1, 1
	case strings.Contains(lower, "charged"), strings.Contains(lower, "full"):
		pluggedIn = 1
	}
	return 0, pluggedIn
}

func batteryHealth(ctx context.Context) (cycle, health float64, ok bool) {
	out, err := exec.CommandContext(ctx, "ioreg", "-rn", "AppleSmartBattery").Output()
	if err != nil {
		return 0, 0, false
	}
	text := string(out)

	cycle, err = ioregNumber(text, "CycleCount")
	if err != nil {
		return 0, 0, false
	}
	maxCapacity, err := ioregNumber(text, "MaximumCapacity")
	if err != nil {
		return 0, 0, false
	}
	designCapacity, err := ioregNumber(text, "DesignCapacity")
	if err != nil {
		return 0, 0, false
	}
	if designCapacity <= 0 {
		return 0, 0, false
	}
	return cycle, maxCapacity / designCapacity * 100, true
}

func ioregNumber(text, key string) (float64, error) {
	needle := `"` + key + `"` + " = "
	idx := strings.Index(text, needle)
	if idx < 0 {
		return 0, fmt.Errorf("key %s not found in ioreg output", key)
	}
	rest := text[idx+len(needle):]
	if end := strings.IndexAny(rest, "\n"); end >= 0 {
		rest = rest[:end]
	}
	return strconv.ParseFloat(strings.TrimSpace(rest), 64)
}
