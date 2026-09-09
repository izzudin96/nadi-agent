//go:build darwin

package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// cpuTemperature reads the CPU die temperature from powermetrics. Requires
// sudo and an undocumented output format, so it usually fails on dev Macs —
// that's fine (best-effort, silently omitted).
func cpuTemperature(ctx context.Context) (float64, error) {
	if !commandExists("powermetrics") {
		return 0, fmt.Errorf("powermetrics not found")
	}
	out, err := runCommand(ctx, "powermetrics", "--samplers", "smc", "-n", "1")
	if err != nil {
		return 0, err
	}
	return parsePowermetricsCPUTemp(string(out))
}

// parsePowermetricsCPUTemp extracts the value from a line like:
//
//	CPU die temperature: 55.48 C
func parsePowermetricsCPUTemp(text string) (float64, error) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "CPU die temperature:") {
			continue
		}
		fields := strings.Fields(line) // ["CPU","die","temperature:","55.48","C"]
		if len(fields) >= 4 {
			if v, err := strconv.ParseFloat(fields[3], 64); err == nil {
				return v, nil
			}
		}
	}
	return 0, fmt.Errorf("no CPU die temperature in powermetrics output")
}
