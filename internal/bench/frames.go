package bench

import (
	"math"
	"sort"
)

// Session es una grabación de tiempos de cuadro hecha por el overlay.
type Session struct {
	ID       string    `json:"id"`
	At       string    `json:"at"`
	Label    string    `json:"label"`
	Game     string    `json:"game"`
	Run      Run       `json:"run"`
	Spikes   int       `json:"spikes"`
	Imported bool      `json:"imported,omitempty"`
	Frames   []float32 `json:"frames"` // tiempos de cuadro (ms), reducidos para el gráfico
}

// FromFrameTimes calcula las estadísticas de una corrida a partir de los
// tiempos de cuadro en milisegundos. «1% low» es el promedio de FPS del 1 %
// de cuadros más lentos (no es el mismo método que usa Afterburner, por eso
// las corridas se marcan con Src y solo se comparan entre sí).
func FromFrameTimes(ms []float64) (Run, int) {
	r := Run{Src: "fpsforge"}
	clean := make([]float64, 0, len(ms))
	for _, v := range ms {
		if v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
			clean = append(clean, v)
		}
	}
	if len(clean) < 2 {
		return r, 0
	}
	sorted := append([]float64{}, clean...)
	sort.Float64s(sorted)
	var total float64
	for _, v := range clean {
		total += v
	}
	n := len(clean)
	r.Frames = n
	r.Seconds = total / 1000
	r.Avg = float64(n) / r.Seconds
	r.Min = 1000 / sorted[n-1]
	r.Max = 1000 / sorted[0]
	r.Low1 = lowAvg(sorted, 0.01)
	r.Low01 = lowAvg(sorted, 0.001)
	med := sorted[n/2]
	spikes := 0
	for _, v := range clean {
		if v > 2*med && v-med > 4 {
			spikes++
		}
	}
	return r, spikes
}

// lowAvg promedia el peor tramo (fracción frac) de tiempos ya ordenados y lo pasa a FPS.
func lowAvg(sorted []float64, frac float64) float64 {
	n := len(sorted)
	k := int(math.Ceil(float64(n) * frac))
	if k < 1 {
		k = 1
	}
	var s float64
	for _, v := range sorted[n-k:] {
		s += v
	}
	return 1000 / (s / float64(k))
}

// Downsample reduce a como máximo max puntos conservando el peor cuadro de
// cada tramo, para que los picos se vean en el gráfico.
func Downsample(ms []float64, max int) []float32 {
	if max < 2 || len(ms) == 0 {
		return nil
	}
	if len(ms) <= max {
		out := make([]float32, len(ms))
		for i, v := range ms {
			out[i] = float32(v)
		}
		return out
	}
	out := make([]float32, 0, max)
	step := float64(len(ms)) / float64(max)
	for i := 0; i < max; i++ {
		a, b := int(float64(i)*step), int(float64(i+1)*step)
		if b > len(ms) {
			b = len(ms)
		}
		w := 0.0
		for _, v := range ms[a:b] {
			if v > w {
				w = v
			}
		}
		out = append(out, float32(w))
	}
	return out
}
