package overlay

import (
	"strings"
	"testing"

	"fpsforge/internal/telemetry"
)

func snap() Snapshot {
	return Snapshot{Cfg: Default(), GameRunning: true, FPS: 238.4, Low1: 126.2,
		Sample: telemetry.Sample{CPU: 41.2, GPU: 70.4, CPUTemp: telemetry.Unknown, GPUTemp: 54.6, RAMUsedMB: 11800, RAMTotMB: 32768}}
}

func TestBuildModel(t *testing.T) {
	m := BuildModel(snap())
	if m.FPS != "238" || m.Low != "1% 126" || len(m.Rows) != 3 {
		t.Fatalf("%+v", m)
	}
	if got := m.Rows[0].Parts[1]; got.Text != "—" {
		t.Fatalf("temperatura de CPU desconocida debe mostrar —, fue %q", got.Text)
	}
	if got := m.Rows[1].Parts[1]; got.Text != "55°" || got.Color != ColText {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(m.Line, "238 FPS") || !strings.Contains(m.Line, "GPU 70% 55°") || !strings.Contains(m.Line, "RAM 11.5 GB") {
		t.Fatalf("línea: %q", m.Line)
	}
}

func TestBuildModelOptions(t *testing.T) {
	s := snap()
	s.Cfg.ShowCPU, s.Cfg.ShowTemp, s.Cfg.ShowVRAM, s.Cfg.ShowLow = false, false, true, false
	s.FPS = 0
	m := BuildModel(s)
	if m.FPS != "—" || m.Low != "" || len(m.Rows) != 3 {
		t.Fatalf("%+v", m)
	}
	for _, r := range m.Rows {
		if len(r.Parts) != 1 {
			t.Fatalf("sin temperaturas cada fila tiene un valor: %+v", r)
		}
	}
	if tempPart(90, 85).Color != ColBad || tempPart(75, 85).Color != ColWarn {
		t.Fatal("colores de temperatura")
	}
}

func TestSpark(t *testing.T) {
	sp := []float32{4, 4, 4, 4, 4, 4, 4, 4, 4, 40}
	if SparkScale(sp) != 12 || Median(sp) != 4 {
		t.Fatalf("%v %v", SparkScale(sp), Median(sp))
	}
	if SparkColor(40, 4) != ColBad || SparkColor(4.2, 4) != ColGood || SparkColor(8, 4) != ColWarn {
		t.Fatal("colores del gráfico")
	}
}
