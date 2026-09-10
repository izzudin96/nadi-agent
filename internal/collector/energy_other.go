//go:build !darwin && !linux

package collector

import (
	"context"
	"errors"
)

func energyMetrics(*energyCollector, context.Context) ([]Metric, error) {
	return nil, errors.New("energy not supported on this OS")
}
