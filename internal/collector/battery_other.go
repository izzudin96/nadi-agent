//go:build !darwin && !linux

package collector

import (
	"context"
	"errors"
)

// batteryMetrics on unsupported platforms: no data, never an error.
func batteryMetrics(context.Context) ([]Metric, error) {
	return nil, errors.New("battery not supported on this OS")
}
