package bench

import "testing"

func es(label string, vals ...float64) []Entry {
	var o []Entry
	for _, v := range vals {
		o = append(o, Entry{Run: Run{Label: label, Avg: v, Low1: v / 2}})
	}
	return o
}

func TestNoiseIsNotReal(t *testing.T) {
	// Datos reales del usuario: antes vs. optimizado (ruido ±20 FPS).
	gs := Groups(append(es("antes", 268, 242.3, 259.2), es("opt", 254.8, 228.8, 267.9)...))
	v := Compare(gs[0], gs[1])
	if v.Real {
		t.Fatalf("debería ser ruido: %+v", v)
	}
}

func TestRealDifference(t *testing.T) {
	gs := Groups(append(es("a", 200, 201, 199, 200), es("b", 240, 241, 239, 240)...))
	if v := Compare(gs[0], gs[1]); !v.Real || v.DiffPct < 19 {
		t.Fatalf("%+v", v)
	}
	gs = Groups(append(es("a", 240, 241, 239), es("b", 200, 201, 199)...))
	if v := Compare(gs[0], gs[1]); !v.Real || v.DiffPct > 0 {
		t.Fatalf("%+v", v)
	}
}

func TestTooFewRuns(t *testing.T) {
	gs := Groups(append(es("a", 200, 210), es("b", 250, 255)...))
	if v := Compare(gs[0], gs[1]); v.Real {
		t.Fatalf("con 2 corridas no se afirma nada: %+v", v)
	}
}
