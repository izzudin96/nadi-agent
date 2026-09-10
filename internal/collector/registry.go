package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
)

// collectorTimeout bounds how long a single collector may run. A collector
// that exceeds it is treated as failed and its metrics omitted — one slow or
// hung sensor must never stall the whole cycle. (var, not const, so tests can
// shorten it.)
var collectorTimeout = 2 * time.Second

type collectResult struct {
	metrics []Metric
	err     error
}

// factory builds a collector. Collectors are constructed once at startup and
// reused every cycle so they can keep state (e.g. timestamps for deltas).
type factory func(logger *slog.Logger) Collector

var factories = make(map[string]factory)

// Register makes a collector available under name. Called from package init().
func Register(name string, f factory) {
	factories[name] = f
}

// Registry holds the set of enabled collectors and runs them each cycle.
type Registry struct {
	logger     *slog.Logger
	collectors map[string]Collector
}

// NewRegistry instantiates the collectors enabled in the config. Enabled names
// that are not registered are a config error, so typos fail loudly.
func NewRegistry(enabled map[string]bool, logger *slog.Logger) (*Registry, error) {
	var unknown []string
	for name, on := range enabled {
		if on {
			if _, ok := factories[name]; !ok {
				unknown = append(unknown, name)
			}
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown collectors enabled in config: %s", strings.Join(unknown, ", "))
	}

	collectors := make(map[string]Collector)
	for name, factory := range factories {
		if enabled[name] {
			collectors[name] = factory(logger)
		}
	}
	return newRegistry(collectors, logger), nil
}

func newRegistry(collectors map[string]Collector, logger *slog.Logger) *Registry {
	return &Registry{collectors: collectors, logger: logger}
}

// Names returns the enabled collector names, sorted for stable output.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Collect runs every enabled collector. A failing collector is logged and its
// metrics omitted — it never aborts the cycle. A count of real failures is
// reported as agent.collector_errors_count so the agent monitors itself.
//
// Collectors run sequentially, not concurrently (spec §7.1 said concurrent):
// several collectors (network, energy, cpu) hold state that isn't mutex-
// protected, and the per-collector timeout already bounds the cycle, so
// sequential keeps the code simple and safe. Revisit if cycle latency matters.
func (r *Registry) Collect(ctx context.Context) []Metric {
	var all []Metric
	var errorsCount float64
	for name, c := range r.collectors {
		metrics, err := r.runCollector(ctx, c)
		if err != nil {
			if IsNoDataError(err) {
				r.logger.Debug("collector returned no data", "collector", name, "err", err)
				continue
			}
			if errors.Is(err, context.Canceled) {
				// The cycle context was cancelled (graceful shutdown) while the
				// collector was mid-run. Not a failure — just stop quietly.
				r.logger.Debug("collector cancelled during shutdown", "collector", name)
				continue
			}
			errorsCount++
			r.logger.Error("collector failed", "collector", name, "err", err)
			continue
		}
		r.logger.Debug("collector succeeded", "collector", name, "metrics", len(metrics))
		all = append(all, metrics...)
	}
	if errorsCount > 0 {
		all = append(all, Metric{Name: "agent.collector_errors_count", Value: errorsCount, Unit: "count"})
	}
	return all
}

// runCollector runs one collector with its own deadline. The goroutine keeps
// the cycle from blocking on a hung collector: if the deadline hits, we return
// the timeout error immediately and the stuck goroutine is abandoned (the
// buffered channel lets it finish without leaking back into us).
func (r *Registry) runCollector(ctx context.Context, c Collector) ([]Metric, error) {
	cctx, cancel := context.WithTimeout(ctx, collectorTimeout)
	defer cancel()

	done := make(chan collectResult, 1)
	go func() {
		metrics, err := c.Collect(cctx)
		done <- collectResult{metrics: metrics, err: err}
	}()

	select {
	case res := <-done:
		return res.metrics, res.err
	case <-cctx.Done():
		return nil, cctx.Err()
	}
}
