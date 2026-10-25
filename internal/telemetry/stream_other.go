//go:build !windows

package telemetry

import (
	"math"
	"time"
)

// GPUTemps no hace nada fuera de Windows.
func GPUTemps() map[string]float64 { return map[string]float64{} }

// Stream genera mediciones simuladas (modo demostración fuera de Windows).
func Stream(stop <-chan struct{}) <-chan Sample {
	out := make(chan Sample, 4)
	go func() {
		defer close(out)
		t0 := time.Now()
		tk := time.NewTicker(500 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				x := time.Since(t0).Seconds()
				out <- Sample{At: time.Now(), CPU: 45 + 8*math.Sin(x/3), CPUThread: 82 + 6*math.Sin(x/2), CPUMHz: 4300,
					CPUTemp: Unknown, RAMUsedMB: 11800, RAMTotMB: 32768, GPU: 70 + 10*math.Sin(x/4), GPUVRAMMB: 3900,
					GPUTemp: 56 + 3*math.Sin(x/6)}
			}
		}
	}()
	return out
}
