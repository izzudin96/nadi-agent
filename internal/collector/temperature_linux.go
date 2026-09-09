//go:build linux

package collector

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// cpuTemperature reads the CPU temperature from lm-sensors.
func cpuTemperature(ctx context.Context) (float64, error) {
	if !commandExists("sensors") {
		return 0, fmt.Errorf("sensors not found")
	}
	out, err := runCommand(ctx, "sensors")
	if err != nil {
		return 0, err
	}
	return parseSensorsCPUTemp(string(out))
}

var tempRe = regexp.MustCompile(`\+?([0-9]+(?:\.[0-9]+)?)°C`)

// parseSensorsCPUTemp extracts the CPU temperature from `sensors` output. It
// prefers a package/cpu-level reading (lines naming package, tdie, or tctl),
// and falls back to the first °C value anywhere.
func parseSensorsCPUTemp(text string) (float64, error) {
	var fallback float64
	foundFallback := false
	for _, line := range strings.Split(text, "\n") {
		m := tempRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "package") ||
			strings.Contains(lower, "tdie") ||
			strings.Contains(lower, "tctl") ||
			strings.Contains(lower, "cpu") {
			return v, nil
		}
		if !foundFallback {
			fallback = v
			foundFallback = true
		}
	}
	if foundFallback {
		return fallback, nil
	}
	return 0, fmt.Errorf("no temperature in sensors output")
}
