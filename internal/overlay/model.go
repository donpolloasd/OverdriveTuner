package overlay

import (
	"fmt"
	"sort"

	"fpsforge/internal/telemetry"
)

// Colores (COLORREF 0x00BBGGRR).
const (
	ColBG     = 0x0A0A0A // #0A0A0A
	ColLabel  = 0x8A8A8A // #8A8A8A
	ColText   = 0xF5F5F5 // #F5F5F5
	ColGood   = 0xC4C4C4 // gris claro: lo normal no llama la atención
	ColWarn   = 0x57C8FF // #FFC857
	ColBad    = 0x785FFF // #FF5F78
	ColAccent = 0xFFFFFF // #FFFFFF
)

// Part es un tramo de texto con color.
type Part struct {
	Text  string
	Color uint32
}

// Row es una fila del overlay: etiqueta a la izquierda y valores.
type Row struct {
	Label string
	Parts []Part
}

// Model es lo que se dibuja: ya decidido qué mostrar y de qué color.
type Model struct {
	FPS      string
	FPSColor uint32
	Low      string // "1% 126"
	Spark    []float32
	Rows     []Row
	Line     string // para el diseño de una sola línea
	LineCols []Part
}

func tempPart(t float64, limit int) Part {
	if t == telemetry.Unknown || t <= 0 {
		return Part{"—", ColLabel}
	}
	c := uint32(ColText)
	switch {
	case t >= float64(limit):
		c = ColBad
	case t >= 70:
		c = ColWarn
	}
	return Part{fmt.Sprintf("%.0f°", t), c}
}

// limit devuelve el límite de aviso (el rojo de la temperatura sigue al límite configurado).
func (sn Snapshot) limit(v int) int {
	if v <= 0 || !sn.Cfg.AlertOn {
		return 85
	}
	return v
}

// BuildModel decide el contenido según la configuración y las mediciones.
func BuildModel(sn Snapshot) Model {
	c, s := sn.Cfg, sn.Sample
	m := Model{FPS: "—", FPSColor: ColLabel}
	if sn.FPS > 0 {
		m.FPS, m.FPSColor = fmt.Sprintf("%.0f", sn.FPS), ColText
	} else if !sn.GameRunning {
		m.FPS = "—"
	}
	if c.ShowLow && sn.Low1 > 0 {
		m.Low = fmt.Sprintf("1%% %.0f", sn.Low1)
	}
	if c.ShowGrph {
		m.Spark = sn.Spark
	}
	if c.ShowCPU {
		r := Row{Label: "CPU", Parts: []Part{{fmt.Sprintf("%.0f%%", s.CPU), ColText}}}
		if c.ShowTemp {
			r.Parts = append(r.Parts, tempPart(s.CPUTemp, sn.limit(c.AlertCPU)))
		}
		m.Rows = append(m.Rows, r)
	}
	if c.ShowGPU {
		r := Row{Label: "GPU", Parts: []Part{{fmt.Sprintf("%.0f%%", s.GPU), ColText}}}
		if c.ShowTemp {
			r.Parts = append(r.Parts, tempPart(s.GPUTemp, sn.limit(c.AlertGPU)))
		}
		m.Rows = append(m.Rows, r)
	}
	if c.ShowVRAM {
		m.Rows = append(m.Rows, Row{Label: "VRAM", Parts: []Part{{fmt.Sprintf("%.1f GB", s.GPUVRAMMB/1024), ColText}}})
	}
	if c.ShowRAM {
		m.Rows = append(m.Rows, Row{Label: "RAM", Parts: []Part{{fmt.Sprintf("%.1f GB", s.RAMUsedMB/1024), ColText}}})
	}
	if c.AlertOn {
		if sn.AlertCPU {
			m.Rows = append(m.Rows, Row{Label: "Aviso", Parts: []Part{{"CPU caliente", ColBad}}})
		}
		if sn.AlertGPU {
			m.Rows = append(m.Rows, Row{Label: "Aviso", Parts: []Part{{"GPU caliente", ColBad}}})
		}
	}
	// Diseño de una línea: «238 FPS · CPU 41% 62° · GPU 70% 54° · RAM 11.2 GB»
	if c.ShowFPS {
		m.LineCols = append(m.LineCols, Part{m.FPS + " FPS", m.FPSColor})
	}
	for _, r := range m.Rows {
		m.LineCols = append(m.LineCols, Part{"  " + r.Label + " ", ColLabel})
		for i, p := range r.Parts {
			if i > 0 {
				p.Text = " " + p.Text
			}
			m.LineCols = append(m.LineCols, p)
		}
	}
	for _, p := range m.LineCols {
		m.Line += p.Text
	}
	return m
}

// SparkScale devuelve el tope vertical del gráfico de tiempos de cuadro: 3
// veces la mediana (mínimo 12 ms) para que los picos sobresalgan.
func SparkScale(sp []float32) float32 {
	if len(sp) == 0 {
		return 12
	}
	c := append([]float32{}, sp...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	m := c[len(c)/2] * 3
	if m < 12 {
		m = 12
	}
	return m
}

// SparkColor colorea una barra según cuánto se aleja de lo normal.
func SparkColor(ft, median float32) uint32 {
	switch {
	case ft > 2.5*median && ft-median > 4:
		return ColBad
	case ft > 1.6*median && ft-median > 2:
		return ColWarn
	}
	return ColGood
}

// Median devuelve la mediana de los tiempos de cuadro.
func Median(sp []float32) float32 {
	if len(sp) == 0 {
		return 0
	}
	c := append([]float32{}, sp...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)/2]
}
