package collector

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

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
// metrics omitted — it never aborts the cycle.
func (r *Registry) Collect(ctx context.Context) []Metric {
	var all []Metric
	for name, c := range r.collectors {
		metrics, err := c.Collect(ctx)
		if err != nil {
			if IsNoDataError(err) {
				r.logger.Debug("collector returned no data", "collector", name, "err", err)
			} else {
				r.logger.Error("collector failed", "collector", name, "err", err)
			}
			continue
		}
		r.logger.Debug("collector succeeded", "collector", name, "metrics", len(metrics))
		all = append(all, metrics...)
	}
	return all
}
