package collector

import (
	"context"
	"errors"
)

// Collector is the interface every metric source implements. One collector
// per metric category; add a new sensor by writing a struct and registering it.
type Collector interface {
	Name() string
	Collect(ctx context.Context) ([]Metric, error)
}

// Metric is a single collected value. Name follows the dotted namespace
// convention, e.g. "battery.percent".
type Metric struct {
	Name  string
	Value float64
	Unit  string // optional, display-only
}

// ErrNoData means the collector found nothing to collect on this system
// (e.g. battery on a server). It is not a failure: callers log it at debug
// level and simply omit those metrics.
var ErrNoData = errors.New("collector returned no data")

func IsNoDataError(err error) bool {
	return errors.Is(err, ErrNoData)
}
