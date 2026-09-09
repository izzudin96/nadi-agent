//go:build !darwin && !linux

package collector

import (
	"context"
	"errors"
)

func cpuTemperature(context.Context) (float64, error) {
	return 0, errors.New("temperature not supported on this OS")
}
