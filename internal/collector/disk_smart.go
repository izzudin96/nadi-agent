package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register("disk_smart", NewDiskSmartCollector)
}

type diskSmartCollector struct {
	logger *slog.Logger
}

func NewDiskSmartCollector(logger *slog.Logger) Collector {
	return &diskSmartCollector{logger: logger}
}

func (c *diskSmartCollector) Name() string { return "disk_smart" }

// Collect reads SMART health for the first disk. Requires smartctl (and, in
// practice, elevated privileges). Any inability — tool missing, no readable
// disk, parse failure — is ErrNoData, never a failure.
func (c *diskSmartCollector) Collect(ctx context.Context) ([]Metric, error) {
	metrics, err := smartMetrics(ctx)
	if err != nil {
		c.logger.Debug("smart unavailable", "err", err)
		return nil, ErrNoData
	}
	return metrics, nil
}

func smartMetrics(ctx context.Context) ([]Metric, error) {
	if !commandExists("smartctl") {
		return nil, fmt.Errorf("smartctl not found")
	}

	out, err := runCommand(ctx, "smartctl", "--scan")
	if err != nil {
		return nil, err
	}
	device := firstDevice(string(out))
	if device == "" {
		return nil, fmt.Errorf("no devices from smartctl --scan")
	}

	out, err = runCommand(ctx, "smartctl", "-a", "--json", device)
	if err != nil {
		return nil, err
	}
	return parseSmartctlJSON(out)
}

// firstDevice extracts the first /dev/... token from `smartctl --scan` output,
// e.g. "/dev/sda -d sat # /dev/sda, ATA device".
func firstDevice(scan string) string {
	for _, line := range strings.Split(scan, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) > 0 && strings.HasPrefix(fields[0], "/dev/") {
			return fields[0]
		}
	}
	return ""
}

// smartctlOutput is the subset of smartctl --json we care about. SMART
// attribute IDs follow the ATA spec: 5 reallocated, 9 power-on hours,
// 177 wear leveling, 190/194 temperature, 197 pending, 198 uncorrectable.
type smartctlOutput struct {
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	ATAAttributes struct {
		Table []struct {
			ID  int `json:"id"`
			Raw struct {
				Value float64 `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
}

func parseSmartctlJSON(data []byte) ([]Metric, error) {
	var out smartctlOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	metrics := []Metric{
		{Name: "disk.smart.health_passed", Value: boolToFloat(out.SmartStatus.Passed), Unit: "bool"},
	}

	byID := map[int]float64{}
	for _, attr := range out.ATAAttributes.Table {
		byID[attr.ID] = attr.Raw.Value
	}

	add := func(id int, name, unit string) {
		if v, ok := byID[id]; ok {
			metrics = append(metrics, Metric{Name: name, Value: v, Unit: unit})
		}
	}
	add(5, "disk.smart.reallocated_sectors", "count")
	add(197, "disk.smart.pending_sectors", "count")
	add(198, "disk.smart.uncorrectable_sectors", "count")
	add(9, "disk.smart.power_on_hours", "hours")
	add(177, "disk.smart.wear_leveling_percent", "%")

	// Temperature is reported under either id 194 or 190, depending on vendor.
	if v, ok := byID[194]; ok {
		metrics = append(metrics, Metric{Name: "disk.smart.temperature_celsius", Value: v, Unit: "C"})
	} else if v, ok := byID[190]; ok {
		metrics = append(metrics, Metric{Name: "disk.smart.temperature_celsius", Value: v, Unit: "C"})
	}

	return metrics, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
