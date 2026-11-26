//go:build !windows

package perf

import (
	"context"
	"errors"
)

// Disk solo existe en Windows.
func Disk(ctx context.Context, dir string, write bool, prog func(float64)) (mbs, iops float64, err error) {
	return 0, 0, errors.New("la prueba de disco solo funciona en Windows")
}
