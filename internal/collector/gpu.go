package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

func init() {
	Register("gpu", NewGPUCollector)
}

type gpuCollector struct {
	logger *slog.Logger
}

func NewGPUCollector(logger *slog.Logger) Collector {
	return &gpuCollector{logger: logger}
}

func (c *gpuCollector) Name() string { return "gpu" }

// Collect reads NVIDIA GPU stats via nvidia-smi. v1 scope is NVIDIA only (see
// spec §5.5): if nvidia-smi is not on PATH — as on most dev Macs — the
// collector returns ErrNoData and the metrics are omitted.
func (c *gpuCollector) Collect(ctx context.Context) ([]Metric, error) {
	metrics, err := gpuMetrics(ctx)
	if err != nil {
		c.logger.Debug("gpu unavailable", "err", err)
		return nil, ErrNoData
	}
	return metrics, nil
}

func gpuMetrics(ctx context.Context) ([]Metric, error) {
	if !commandExists("nvidia-smi") {
		return nil, fmt.Errorf("nvidia-smi not found")
	}
	out, err := runCommand(ctx, "nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits")
	if err != nil {
		return nil, err
	}
	return parseNvidiaSMI(string(out))
}

// parseNvidiaSMI parses a single CSV row like "42, 1024, 8192, 65, 120.5".
// memory.used/total are reported in MiB and converted to bytes.
func parseNvidiaSMI(text string) ([]Metric, error) {
	line := strings.TrimSpace(strings.Split(text, "\n")[0])
	fields := strings.Split(line, ",")
	if len(fields) < 5 {
		return nil, fmt.Errorf("unexpected nvidia-smi output: %q", line)
	}

	val := func(i int) (float64, error) {
		return strconv.ParseFloat(strings.TrimSpace(fields[i]), 64)
	}
	util, err := val(0)
	if err != nil {
		return nil, err
	}
	memUsed, err := val(1)
	if err != nil {
		return nil, err
	}
	memTotal, err := val(2)
	if err != nil {
		return nil, err
	}
	temp, err := val(3)
	if err != nil {
		return nil, err
	}
	power, err := val(4)
	if err != nil {
		return nil, err
	}

	const mib = 1024 * 1024
	return []Metric{
		{Name: "gpu.usage_percent", Value: util, Unit: "%"},
		{Name: "gpu.memory_used_bytes", Value: memUsed * mib, Unit: "bytes"},
		{Name: "gpu.memory_total_bytes", Value: memTotal * mib, Unit: "bytes"},
		{Name: "gpu.temperature_celsius", Value: temp, Unit: "C"},
		{Name: "gpu.power_watts", Value: power, Unit: "W"},
	}, nil
}
