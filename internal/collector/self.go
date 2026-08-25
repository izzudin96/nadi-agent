package collector

import (
	"context"
	"log/slog"
	"time"
)

func init() {
	Register("self", NewSelfCollector)
}

type selfCollector struct {
	logger *slog.Logger
	start  time.Time
}

func NewSelfCollector(logger *slog.Logger) Collector {
	return &selfCollector{logger: logger, start: time.Now()}
}

func (c *selfCollector) Name() string { return "self" }

func (c *selfCollector) Collect(_ context.Context) ([]Metric, error) {
	uptime := time.Since(c.start).Seconds()
	return []Metric{
		{Name: "agent.uptime", Value: uptime, Unit: "s"},
	}, nil
}
