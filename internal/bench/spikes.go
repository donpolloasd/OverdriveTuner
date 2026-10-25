package bench

import (
	"math"
	"sort"
)

// Spike es un cuadro anormalmente lento dentro de una grabación.
type Spike struct {
	At float64 `json:"at"` // segundo aproximado dentro de la sesión
	Ms float64 `json:"ms"`
}

// SpikeReport resume los microcortes de una sesión y su causa más probable.
type SpikeReport struct {
	Count     int      `json:"count"`
	PerMinute float64  `json:"perMinute"`
	MedianMs  float64  `json:"medianMs"`
	WorstMs   float64  `json:"worstMs"`
	Score     int      `json:"score"` // 0-100, más alto = más fluido
	Pattern   string   `json:"pattern"`
	PeriodSec float64  `json:"periodSec"`
	Cause     string   `json:"cause"`
	Advice    []string `json:"advice"`
	Spikes    []Spike  `json:"spikes"`
}

// AnalyzeSpikes busca microcortes en los tiempos de cuadro (ms) de una sesión
// de «seconds» segundos. total es la cantidad real de picos medida por el
// overlay (0 si no se conoce): los frames guardados están reducidos y cada
// valor representa el peor cuadro de un tramo.
func AnalyzeSpikes(frames []float32, seconds float64, total int) SpikeReport {
	rep := SpikeReport{Advice: []string{}, Spikes: []Spike{}, Pattern: "none"}
	n := len(frames)
	if n < 10 || seconds <= 0 {
		rep.Cause = "Muy pocos datos para analizar. Grabá al menos 30 segundos de partida."
		rep.Score = 100
		return rep
	}
	sorted := make([]float64, n)
	for i, v := range frames {
		sorted[i] = float64(v)
	}
	sort.Float64s(sorted)
	med := sorted[n/2]
	rep.MedianMs = med
	rep.WorstMs = sorted[n-1]
	limit := math.Max(2*med, med+4)
	step := seconds / float64(n)

	// Picos contiguos se cuentan como uno solo.
	inSpike := false
	var cur Spike
	flush := func() {
		if inSpike {
			rep.Spikes = append(rep.Spikes, cur)
		}
		inSpike = false
	}
	for i, v := range frames {
		if float64(v) > limit {
			t := (float64(i) + 0.5) * step
			if !inSpike {
				cur = Spike{At: t, Ms: float64(v)}
				inSpike = true
			} else if float64(v) > cur.Ms {
				cur.Ms = float64(v)
			}
		} else {
			flush()
		}
	}
	flush()

	rep.Count = len(rep.Spikes)
	if total > rep.Count {
		rep.Count = total
	}
	rep.PerMinute = float64(rep.Count) / (seconds / 60)
	pen := rep.PerMinute*5 + math.Max(0, rep.WorstMs-3*med)/math.Max(med, 1)*2
	rep.Score = int(math.Max(0, math.Min(100, 100-pen)))
	classify(&rep, seconds)
	if len(rep.Spikes) > 40 {
		sort.Slice(rep.Spikes, func(i, j int) bool { return rep.Spikes[i].Ms > rep.Spikes[j].Ms })
		rep.Spikes = rep.Spikes[:40]
		sort.Slice(rep.Spikes, func(i, j int) bool { return rep.Spikes[i].At < rep.Spikes[j].At })
	}
	return rep
}

func classify(r *SpikeReport, seconds float64) {
	sp := r.Spikes
	if r.PerMinute < 0.5 || len(sp) < 2 {
		r.Pattern = "clean"
		r.Cause = "Prácticamente sin microcortes: la partida salió fluida."
		return
	}
	early := 0
	for _, s := range sp {
		if s.At < math.Max(15, seconds*0.15) {
			early++
		}
	}
	var gaps []float64
	for i := 1; i < len(sp); i++ {
		gaps = append(gaps, sp[i].At-sp[i-1].At)
	}
	sort.Float64s(gaps)
	medGap := gaps[len(gaps)/2]
	var mean, sq float64
	for _, g := range gaps {
		mean += g
	}
	mean /= float64(len(gaps))
	for _, g := range gaps {
		sq += (g - mean) * (g - mean)
	}
	cv := math.Sqrt(sq/float64(len(gaps))) / math.Max(mean, 0.001)
	short := 0
	for _, g := range gaps {
		if g < 3 {
			short++
		}
	}

	switch {
	case float64(early) >= 0.6*float64(len(sp)):
		r.Pattern = "start"
		r.Cause = "Los cortes se concentran al empezar: casi seguro es carga de recursos y compilación de shaders."
		r.Advice = []string{
			"Jugá una partida completa antes de medir: los shaders se compilan una vez y quedan guardados.",
			"No limpies la caché de shaders justo antes de jugar.",
			"Instalá el juego en el SSD si está en el disco duro.",
		}
	case len(gaps) >= 4 && cv < 0.25:
		r.Pattern = "periodic"
		r.PeriodSec = medGap
		r.Cause = "Los cortes se repiten con un ritmo parejo: suele ser una tarea de fondo que despierta cada tanto."
		r.Advice = []string{
			"Cerrá lanzadores, antivirus con escaneo, sincronización de nube y programas de RGB o monitoreo.",
			"Mirá «Procesos que más consumen» en Diagnóstico durante una partida.",
			"Si el intervalo ronda los 60 segundos, sospechá de Windows Update o del antivirus.",
		}
	case float64(short) >= 0.5*float64(len(gaps)):
		r.Pattern = "bursts"
		r.Cause = "Los cortes vienen en ráfagas: suele pasar en zonas o momentos concretos del mapa (mucha carga de CPU, memoria de video llena o limitación térmica)."
		r.Advice = []string{
			"Probá bajar texturas y sombras para ver si la ráfaga desaparece.",
			"Revisá temperaturas y frecuencias con el overlay durante esas ráfagas.",
			"Un límite de FPS apenas por debajo de tu tasa de refresco reduce los picos.",
		}
	default:
		r.Pattern = "scattered"
		r.Cause = "Los cortes aparecen sin patrón claro: lo más común es un núcleo del procesador al límite o procesos de fondo."
		r.Advice = []string{
			"Cerrá navegador y overlays de terceros antes de jugar.",
			"Comprobá que la memoria corra a su velocidad (DOCP/EXPO/XMP) y en dual channel.",
			"Compará con el benchmark guiado antes y después de cada ajuste.",
		}
	}
}
