package collector

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	topProcessCount       = 5
	topProcessSampleEvery = 200 * time.Millisecond
)

func init() {
	Register("top_process", NewTopProcessCollector)
}

type topProcessCollector struct {
	logger *slog.Logger
}

func NewTopProcessCollector(logger *slog.Logger) Collector {
	return &topProcessCollector{logger: logger}
}

func (c *topProcessCollector) Name() string { return "top_process" }

func (c *topProcessCollector) Collect(ctx context.Context) ([]Metric, error) {
	// Sample each process's CPU time twice across a short window and take the
	// delta, so we get "CPU usage right now" rather than lifetime average.
	first, err := sampleCPUTimes(ctx)
	if err != nil {
		return nil, err
	}

	select {
	case <-time.After(topProcessSampleEvery):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	second, err := sampleCPUTimes(ctx)
	if err != nil {
		return nil, err
	}

	wall := topProcessSampleEvery.Seconds()
	type entry struct {
		name string
		pid  int32
		cpu  float64
	}
	var entries []entry
	for pid, t1 := range first {
		t2, ok := second[pid]
		if !ok {
			continue // process exited during the sampling window
		}
		cpuPct := (t2.Total() - t1.Total()) / wall * 100
		if cpuPct < 0 {
			cpuPct = 0
		}
		entries = append(entries, entry{pid: pid, cpu: cpuPct})
	}

	for i := range entries {
		if p, err := process.NewProcess(entries[i].pid); err == nil {
			if name, err := p.NameWithContext(ctx); err == nil {
				entries[i].name = name
			}
		}
		if entries[i].name == "" {
			entries[i].name = "unknown"
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].cpu > entries[j].cpu })
	if len(entries) > topProcessCount {
		entries = entries[:topProcessCount]
	}

	var metrics []Metric
	for _, e := range entries {
		tag := sanitize(e.name)
		if tag == "" {
			tag = "unknown"
		}
		metrics = append(metrics,
			Metric{Name: "process.top_cpu." + tag + ".cpu_percent", Value: e.cpu, Unit: "%"},
			Metric{Name: "process.top_cpu." + tag + ".pid", Value: float64(e.pid), Unit: "pid"},
		)
	}
	return metrics, nil
}

func sampleCPUTimes(ctx context.Context) (map[int32]*cpu.TimesStat, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	samples := make(map[int32]*cpu.TimesStat, len(procs))
	for _, p := range procs {
		t, err := p.TimesWithContext(ctx)
		if err != nil {
			continue // process vanished between enumeration and sampling
		}
		samples[p.Pid] = t
	}
	return samples, nil
}
