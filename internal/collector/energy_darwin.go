//go:build darwin

package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// energyMetrics reads package power from powermetrics. Requires sudo and an
// undocumented, version-sensitive output format — treat as fragile.
func energyMetrics(c *energyCollector, ctx context.Context) ([]Metric, error) {
	if !commandExists("powermetrics") {
		return nil, fmt.Errorf("powermetrics not found")
	}
	out, err := runCommand(ctx, "powermetrics", "--samplers", "cpu_power", "-n", "1")
	if err != nil {
		return nil, err
	}
	watts, err := parsePowermetricsPackagePower(string(out))
	if err != nil {
		return nil, err
	}
	return []Metric{{Name: "energy.package_power_watts", Value: watts, Unit: "W"}}, nil
}

// parsePowermetricsPackagePower extracts package power from powermetrics
// output. It handles the Intel-style "package power (CPUs+GT+SA): 12.34W" line
// and the Apple-Silicon "Combined Power (CPU + GPU + ANE): 12.34 mW" line.
func parsePowermetricsPackagePower(text string) (float64, error) {
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "package power"):
			return wattsFromLine(line, "W")
		case strings.Contains(lower, "combined power"):
			return wattsFromLine(line, "mW")
		}
	}
	return 0, fmt.Errorf("no package power in powermetrics output")
}

// wattsFromLine finds the number before the given unit suffix on a line, e.g.
// "...: 12.34W" or "...: 234.5 mW". If unit is "mW" the value is converted to
// watts.
func wattsFromLine(line, unit string) (float64, error) {
	idx := strings.Index(line, unit)
	if idx < 0 {
		return 0, fmt.Errorf("unit %s not found", unit)
	}
	// The number ends right before the unit, minus any whitespace.
	end := idx
	for end > 0 && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	start := end
	for start > 0 {
		c := line[start-1]
		if (c >= '0' && c <= '9') || c == '.' {
			start--
		} else {
			break
		}
	}
	v, err := strconv.ParseFloat(line[start:end], 64)
	if err != nil {
		return 0, err
	}
	if unit == "mW" {
		v /= 1000
	}
	return v, nil
}
