package bench

import (
	"math"
	"testing"
)

func TestFromFrameTimes(t *testing.T) {
	ms := make([]float64, 1000)
	for i := range ms {
		ms[i] = 4 // 250 FPS
	}
	ms[10], ms[500] = 40, 60 // dos picos
	r, spikes := FromFrameTimes(ms)
	want := 1000 / (float64(4*998+100) / 1000) // cuadros / segundos
	if r.Frames != 1000 || math.Abs(r.Avg-want/1000*1000) > 0.01 {
		t.Fatalf("%+v quería avg %.3f", r, want/1000*1000)
	}
	if spikes != 2 {
		t.Fatalf("picos: %d", spikes)
	}
	if r.Low01 >= r.Low1 || r.Low1 >= r.Avg || r.Min >= r.Max {
		t.Fatalf("orden de métricas: %+v", r)
	}
	if r.Src != "fpsforge" {
		t.Fatal("falta Src")
	}
}

func TestFromFrameTimesEdge(t *testing.T) {
	if r, _ := FromFrameTimes([]float64{0, -1, math.NaN()}); r.Frames != 0 {
		t.Fatal("datos inválidos no deben contar")
	}
	r, s := FromFrameTimes([]float64{5, 5, 5, 5})
	if math.Abs(r.Avg-200) > 0.01 || s != 0 {
		t.Fatalf("%+v %d", r, s)
	}
}

func TestDownsampleKeepsSpikes(t *testing.T) {
	ms := make([]float64, 10000)
	for i := range ms {
		ms[i] = 4
	}
	ms[7777] = 90
	d := Downsample(ms, 500)
	if len(d) != 500 {
		t.Fatalf("%d", len(d))
	}
	var mx float32
	for _, v := range d {
		if v > mx {
			mx = v
		}
	}
	if mx != 90 {
		t.Fatalf("se perdió el pico: %v", mx)
	}
}
