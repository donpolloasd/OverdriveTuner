package bench

import (
	"math"
	"testing"
)

// Texto real pegado por el usuario (CS2, MSI Afterburner).
const sample = `12-10-2026, 02:07:19 cs2.exe benchmark completed, 31529 frames rendered in 113.500 s
                     Average framerate  :  277.7 FPS
                     Minimum framerate  :  198.9 FPS
                     Maximum framerate  :  487.7 FPS
                     1% low framerate   :  130.5 FPS
                     0.1% low framerate :  89.1 FPS

DESPUES OPTIMIZACION CLAUDE con FPS FORGE 

Try 1: 
12-10-2026, 21:07:57 cs2.exe benchmark completed, 30496 frames rendered in 119.641 s
                     Average framerate  :  254.8 FPS
                     Minimum framerate  :  133.4 FPS
                     Maximum framerate  :  451.4 FPS
                     1% low framerate   :  106.9 FPS
                     0.1% low framerate :  77.0 FPS
try 2:
`

func TestParseUserFile(t *testing.T) {
	runs := Parse(sample)
	if len(runs) != 2 {
		t.Fatalf("esperaba 2 corridas, hay %d: %+v", len(runs), runs)
	}
	a, b := runs[0], runs[1]
	if a.App != "cs2.exe" || a.Frames != 31529 || a.Seconds != 113.5 || a.Avg != 277.7 || a.Min != 198.9 || a.Max != 487.7 || a.Low1 != 130.5 || a.Low01 != 89.1 {
		t.Errorf("corrida 1 mal leída: %+v", a)
	}
	if b.Avg != 254.8 || b.Min != 133.4 || b.Low1 != 106.9 || b.Low01 != 77.0 {
		t.Errorf("corrida 2 mal leída: %+v", b)
	}
	if b.Label != "Try 1" {
		t.Errorf("etiqueta = %q", b.Label)
	}
	if a.Date != "12-10-2026" {
		t.Errorf("fecha = %q", a.Date)
	}
	if d := Pct(a.Avg, b.Avg); math.Abs(d-(-8.2494)) > 0.01 {
		t.Errorf("variación del promedio = %.3f", d)
	}
	if Recovered(a, b, 3) {
		t.Error("−8% de promedio no debe considerarse recuperado con 3% de tolerancia")
	}
	if !Recovered(a, Run{Avg: 275, Low1: 128}, 3) {
		t.Error("−1% y −2% sí están dentro de la tolerancia")
	}
}

func TestParseCommaDecimalsAndNoise(t *testing.T) {
	r := Parse("cs2.exe benchmark completed, 1.000 frames rendered in 10,5 s\nAverage framerate : 100,5 FPS\n1% low framerate : 80,2 FPS")
	if len(r) != 1 || r[0].Avg != 100.5 || r[0].Low1 != 80.2 || r[0].Seconds != 10.5 {
		t.Fatalf("%+v", r)
	}
	if len(Parse("hola, esto no es un benchmark")) != 0 {
		t.Fatal("no debe inventar corridas")
	}
}

func TestStoreBaselineAndPersistence(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	if _, err := s.Add("nada", "", false); err == nil {
		t.Fatal("debe rechazar texto sin resultados")
	}
	if _, err := s.Add(sample, "inicial", true); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	base, ok := s2.Baseline()
	if !ok || base.Run.Avg != 277.7 || !base.Baseline {
		t.Fatalf("la referencia debe ser la primera: %+v", base)
	}
	if len(s2.List()) != 2 {
		t.Fatalf("debe haber 2 corridas, hay %d", len(s2.List()))
	}
	s2.Delete(base.ID)
	if len(s2.List()) != 1 {
		t.Fatal("no se borró")
	}
}
