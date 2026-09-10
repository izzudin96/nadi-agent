package collector

import (
	"context"
	"log/slog"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func init() {
	Register("turn_process", NewTurnCollector)
}

// coturnProcessNames are the process names coturn runs under (the binary and
// its common symlink alias).
var coturnProcessNames = map[string]bool{
	"coturn":     true,
	"turnserver": true,
}

// turnCPUWindow is how long we sample the coturn process to compute current
// CPU%, since gopsutil's CPUPercent() is a lifetime average.
const turnCPUWindow = 200 * time.Millisecond

type turnCollector struct {
	logger *slog.Logger
}

func NewTurnCollector(logger *slog.Logger) Collector {
	return &turnCollector{logger: logger}
}

func (c *turnCollector) Name() string { return "turn_process" }

// Collect reports the coturn process state. If coturn is not running, it emits
// a single turn.process_up=0 (a valid observation, not "no data"). When coturn
// is running, it adds process CPU/memory metrics; stats come from the CLI in
// commit 2.
func (c *turnCollector) Collect(ctx context.Context) ([]Metric, error) {
	pid, found, err := findTurnProcess(ctx)
	if err != nil {
		return nil, err
	}
	if !found {
		return []Metric{{Name: "turn.process_up", Value: 0, Unit: "bool"}}, nil
	}

	metrics := []Metric{{Name: "turn.process_up", Value: 1, Unit: "bool"}}

	if cpuPct, ok := c.processCPUPercent(ctx, pid); ok {
		metrics = append(metrics, Metric{Name: "turn.process_cpu_percent", Value: cpuPct, Unit: "%"})
	}
	if rss, ok := c.processMemBytes(ctx, pid); ok {
		metrics = append(metrics, Metric{Name: "turn.process_mem_bytes", Value: float64(rss), Unit: "bytes"})
	}

	// coturn stats (sessions + relayed bytes) come from its CLI port. Best-
	// effort: if the CLI is unreachable or the output is unrecognized, omit.
	if sessions, bytes, ok := readCoturnStats(ctx, turnStatsAddr); ok {
		metrics = append(metrics,
			Metric{Name: "turn.active_sessions", Value: sessions, Unit: "count"},
			Metric{Name: "turn.bytes_relayed", Value: bytes, Unit: "bytes"},
		)
	} else {
		c.logger.Debug("coturn stats unavailable", "addr", turnStatsAddr)
	}

	return metrics, nil
}

func findTurnProcess(ctx context.Context) (int32, bool, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return 0, false, err
	}
	for _, p := range procs {
		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue // permission or process exited mid-scan
		}
		if coturnProcessNames[name] {
			return p.Pid, true, nil
		}
	}
	return 0, false, nil
}

func (c *turnCollector) processCPUPercent(ctx context.Context, pid int32) (float64, bool) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return 0, false
	}
	t1, err := p.TimesWithContext(ctx)
	if err != nil {
		return 0, false
	}

	start := time.Now()
	select {
	case <-time.After(turnCPUWindow):
	case <-ctx.Done():
		return 0, false
	}

	t2, err := p.TimesWithContext(ctx)
	if err != nil {
		return 0, false
	}

	wall := time.Since(start).Seconds()
	if wall <= 0 {
		return 0, false
	}
	pct := (t2.Total() - t1.Total()) / wall * 100
	if pct < 0 {
		pct = 0
	}
	return pct, true
}

func (c *turnCollector) processMemBytes(ctx context.Context, pid int32) (uint64, bool) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return 0, false
	}
	mi, err := p.MemoryInfoWithContext(ctx)
	if err != nil {
		return 0, false
	}
	return mi.RSS, true
}
