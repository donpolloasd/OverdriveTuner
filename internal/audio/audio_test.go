package audio

import (
	"math"
	"testing"
)

func tone(amp float64, sec float64, quietHalf bool) []int16 {
	n := int(sec * Rate)
	out := make([]int16, n)
	for i := range out {
		a := amp
		if quietHalf && i < n/2 {
			a = 0.001
		}
		out[i] = int16(math.Max(-32768, math.Min(32767, a*32767*math.Sin(float64(i)/20))))
	}
	return out
}

func TestAnalyze(t *testing.T) {
	for want, pcm := range map[string][]int16{
		"silent": make([]int16, Rate), "ok": tone(0.3, 2, true), "low": tone(0.005, 2, true), "loud": tone(1.5, 2, true), "noisy": tone(0.2, 2, false),
	} {
		if r := Analyze(pcm); r.Verdict != want {
			t.Errorf("%s: %+v", want, r)
		}
	}
	if r := Analyze(nil); r.Verdict != "silent" {
		t.Error("vacío", r)
	}
	if b := WAV(make([]int16, 10)); len(b) != 64 || string(b[:4]) != "RIFF" {
		t.Error("wav", len(b))
	}
	m, err := MicMuted() // solo lectura: no cambia nada en el equipo
	t.Log("silenciado:", m, err)
}
