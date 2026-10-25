package bench

import (
	"math"
)

// Group resume las corridas que comparten estado (ej. «antes», «optimizado»).
type Group struct {
	Label string  `json:"label"`
	N     int     `json:"n"`
	Mean  float64 `json:"mean"`
	SD    float64 `json:"sd"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Low1  float64 `json:"low1"`
	Low01 float64 `json:"low01"`
}

// Verdict compara dos grupos.
type Verdict struct {
	A       string  `json:"a"`
	B       string  `json:"b"`
	DiffPct float64 `json:"diffPct"`
	Real    bool    `json:"real"`
	Text    string  `json:"text"`
}

func meanSD(xs []float64) (m, sd float64) {
	if len(xs) == 0 {
		return
	}
	for _, x := range xs {
		m += x
	}
	m /= float64(len(xs))
	if len(xs) > 1 {
		for _, x := range xs {
			sd += (x - m) * (x - m)
		}
		sd = math.Sqrt(sd / float64(len(xs)-1))
	}
	return
}

// Groups agrupa por etiqueta (Run.Label vacío => «sin etiqueta»), en orden de aparición.
func Groups(es []Entry) []Group {
	idx := map[string]int{}
	var gs []Group
	var avgs, l1, l01 [][]float64
	for _, e := range es {
		l := e.Run.Label
		if l == "" {
			l = "sin etiqueta"
		}
		i, ok := idx[l]
		if !ok {
			i = len(gs)
			idx[l] = i
			gs = append(gs, Group{Label: l, Min: math.Inf(1)})
			avgs, l1, l01 = append(avgs, nil), append(l1, nil), append(l01, nil)
		}
		avgs[i] = append(avgs[i], e.Run.Avg)
		l1[i] = append(l1[i], e.Run.Low1)
		l01[i] = append(l01[i], e.Run.Low01)
		gs[i].Min = math.Min(gs[i].Min, e.Run.Avg)
		gs[i].Max = math.Max(gs[i].Max, e.Run.Avg)
	}
	for i := range gs {
		gs[i].N = len(avgs[i])
		gs[i].Mean, gs[i].SD = meanSD(avgs[i])
		gs[i].Low1, _ = meanSD(l1[i])
		gs[i].Low01, _ = meanSD(l01[i])
	}
	return gs
}

// Compare decide si la diferencia entre medias supera el ruido (Welch, ~95%).
func Compare(a, b Group) Verdict {
	v := Verdict{A: a.Label, B: b.Label, DiffPct: Pct(a.Mean, b.Mean)}
	if a.N < 3 || b.N < 3 {
		v.Text = "Faltan corridas: con menos de 3 por estado no se puede separar el ruido de la diferencia real. Hacé al menos 3 en cada uno, con las mismas apps abiertas."
		return v
	}
	se := math.Sqrt(a.SD*a.SD/float64(a.N) + b.SD*b.SD/float64(b.N))
	if se == 0 {
		v.Real = a.Mean != b.Mean
	} else {
		t := math.Abs(b.Mean-a.Mean) / se
		// t crítico aproximado (95 %, dos colas) con pocos datos.
		df := float64(a.N + b.N - 2)
		crit := 1.96 + 2.4/df + 4.0/(df*df)
		v.Real = t > crit
	}
	switch {
	case !v.Real:
		v.Text = "La diferencia está dentro del ruido normal entre corridas: no hay evidencia de que haya cambiado el rendimiento."
	case v.DiffPct > 0:
		v.Text = "Mejora real: la diferencia supera el ruido."
	default:
		v.Text = "Empeora de verdad: la diferencia supera el ruido. Probá revertir lo último que aplicaste."
	}
	return v
}

// Compares compara cada grupo con el primero.
func Compares(gs []Group) []Verdict {
	var out []Verdict
	for i := 1; i < len(gs); i++ {
		out = append(out, Compare(gs[0], gs[i]))
	}
	return out
}
