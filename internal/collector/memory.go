package collector

import (
	"context"
	"log/slog"

	"github.com/shirou/gopsutil/v4/mem"
)

func init() {
	Register("memory", NewMemoryCollector)
}

type memoryCollector struct{}

func NewMemoryCollector(logger *slog.Logger) Collector { return &memoryCollector{} }

func (c *memoryCollector) Name() string { return "memory" }

func (c *memoryCollector) Collect(ctx context.Context) ([]Metric, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}

	metrics := []Metric{
		{Name: "memory.used_percent", Value: vm.UsedPercent, Unit: "%"},
		{Name: "memory.used_bytes", Value: float64(vm.Used), Unit: "bytes"},
		{Name: "memory.total_bytes", Value: float64(vm.Total), Unit: "bytes"},
	}

	// Swap may be absent (containers, some systems); omit rather than fail.
	if sm, err := mem.SwapMemoryWithContext(ctx); err == nil {
		metrics = append(metrics, Metric{Name: "memory.swap_used_percent", Value: sm.UsedPercent, Unit: "%"})
	}

	return metrics, nil
}
