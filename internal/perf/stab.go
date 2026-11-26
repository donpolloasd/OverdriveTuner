package perf

import (
	"context"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"fpsforge/internal/telemetry"
)

// MaxTemp es la temperatura (°C) del procesador que detiene la prueba de estabilidad.
const MaxTemp = 98

// StabPoint es un momento de la prueba de estabilidad.
type StabPoint struct {
	T     float64 `json:"t"`     // segundos desde el inicio
	Rate  float64 `json:"rate"`  // bloques por segundo, en total
	Temp  float64 `json:"temp"`  // °C del procesador (Unknown si no se sabe)
	Limit float64 `json:"limit"` // % de la velocidad máxima permitida por el firmware (0 = no se sabe)
	MHz   float64 `json:"mhz"`
}

// StabResult es el resultado de la prueba de estabilidad.
type StabResult struct {
	Seconds  float64     `json:"seconds"`
	Errors   int64       `json:"errors"`   // cuentas que dieron mal: inestabilidad real
	Score    float64     `json:"score"`    // puntos sostenidos (como la prueba de todos los núcleos)
	DropPct  float64     `json:"dropPct"`  // cuánto bajó la velocidad entre el principio y el final
	MaxTemp  float64     `json:"maxTemp"`  // Unknown si no se pudo leer
	MinLimit float64     `json:"minLimit"` // 0 si no se sabe
	Throttle string      `json:"throttle"` // «», limit (lo frena el firmware) o drop (perdió velocidad)
	Stopped  string      `json:"stopped"`  // «», heat o user
	Points   []StabPoint `json:"points"`
}

// Stable dice si no hubo errores de cálculo.
func (r StabResult) Stable() bool { return r.Errors == 0 }

// Stress carga todos los hilos durante d comprobando cada cuenta. sample da la última medición de los
// sensores (nil: no hay). Cada segundo entrega un punto.
func Stress(ctx context.Context, d time.Duration, sample func() (telemetry.Sample, bool), onPoint func(StabPoint)) StabResult {
	return stress(ctx, d, time.Second, sample, onPoint, block)
}

func stress(ctx context.Context, d, step time.Duration, sample func() (telemetry.Sample, bool), onPoint func(StabPoint), calc func(uint64) uint64) StabResult {
	const seeds = 8
	var want [seeds]uint64
	for i := range want {
		want[i] = block(uint64(i))
	}
	run, cancel := context.WithCancel(ctx)
	defer cancel()
	var blocks, errs atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := w; run.Err() == nil; i++ {
				s := i % seeds
				if calc(uint64(s)) != want[s] {
					errs.Add(1)
				}
				blocks.Add(1)
			}
		}(w)
	}
	res := StabResult{MaxTemp: telemetry.Unknown}
	t0, last, lastN := time.Now(), time.Now(), int64(0)
	hot := 0
	tk := time.NewTicker(step)
	defer tk.Stop()
loop:
	for {
		select {
		case <-ctx.Done():
			res.Stopped = "user"
			break loop
		case <-tk.C:
		}
		now, n := time.Now(), blocks.Load()
		p := StabPoint{T: now.Sub(t0).Seconds(), Rate: float64(n-lastN) / now.Sub(last).Seconds(), Temp: telemetry.Unknown}
		last, lastN = now, n
		if sample != nil {
			if s, ok := sample(); ok {
				p.Temp, p.Limit, p.MHz = s.CPUTemp, s.CPULimit, s.CPUMHz
			}
		}
		res.Points = append(res.Points, p)
		if onPoint != nil {
			onPoint(p)
		}
		if p.Temp >= MaxTemp {
			hot++
		} else {
			hot = 0
		}
		if hot >= 3 { // tres segundos seguidos: se corta para cuidar el equipo
			res.Stopped = "heat"
			break
		}
		if now.Sub(t0) >= d {
			break
		}
	}
	cancel()
	wg.Wait()
	res.Seconds, res.Errors = time.Since(t0).Seconds(), errs.Load()
	res.summarize()
	return res
}

// summarize saca las conclusiones de los puntos.
func (r *StabResult) summarize() {
	n := len(r.Points)
	if n == 0 {
		return
	}
	limited := 0
	sum := 0.0
	for _, p := range r.Points {
		sum += p.Rate
		r.MaxTemp = math.Max(r.MaxTemp, p.Temp)
		if p.Limit > 0 {
			if r.MinLimit == 0 || p.Limit < r.MinLimit {
				r.MinLimit = p.Limit
			}
			if p.Limit < 95 {
				limited++
			}
		}
	}
	r.Score = sum / float64(n) / refRate * 1000
	if n >= 12 {
		// Los primeros segundos (sin el arranque) contra los últimos.
		k := n / 5
		r.DropPct = 100 * (1 - mean(r.Points[n-k:])/mean(r.Points[1:1+k]))
		if r.DropPct < 0 {
			r.DropPct = 0
		}
	}
	switch {
	case limited >= 5:
		r.Throttle = "limit"
	case r.DropPct >= 10:
		r.Throttle = "drop"
	}
}

func mean(ps []StabPoint) float64 {
	t := 0.0
	for _, p := range ps {
		t += p.Rate
	}
	if len(ps) == 0 || t == 0 {
		return 1
	}
	return t / float64(len(ps))
}
