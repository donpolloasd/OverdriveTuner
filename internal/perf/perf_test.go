package perf

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"fpsforge/internal/telemetry"
)

func nop(float64) {}

func TestBenchmarksReturnSaneNumbers(t *testing.T) {
	ctx := context.Background()
	if block(7) != block(7) {
		t.Fatal("el bloque tiene que ser determinista")
	}
	one, err := CPUSingle(ctx, 300*time.Millisecond, nop)
	if err != nil || one <= 0 {
		t.Fatal(one, err)
	}
	all, err := CPUMulti(ctx, 300*time.Millisecond, nop)
	if err != nil || all <= 0 {
		t.Fatal(all, err)
	}
	r, w, c, err := RAM(ctx, 300*time.Millisecond, 64, nop)
	if err != nil || r <= 0 || w <= 0 || c <= 0 {
		t.Fatal(r, w, c, err)
	}
	l, err := RAMLatency(ctx, 200*time.Millisecond, 32, nop)
	if err != nil || l <= 0 || l > 100000 {
		t.Fatal(l, err)
	}
	t.Logf("un núcleo %.0f · todos %.0f · RAM %.1f/%.1f/%.1f GB/s · latencia %.1f ns", one, all, r, w, c, l)
}

func TestCancelStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CPUMulti(ctx, time.Minute, nop); err == nil {
		t.Fatal("cancelado")
	}
}

func TestStressDetectsErrorsHeatAndThrottle(t *testing.T) {
	r := stress(context.Background(), 250*time.Millisecond, 50*time.Millisecond, nil, nil, block)
	if !r.Stable() || r.Score <= 0 || r.Stopped != "" {
		t.Fatalf("%+v", r)
	}
	var n atomic.Int64
	bad := stress(context.Background(), 150*time.Millisecond, 50*time.Millisecond, nil, nil, func(s uint64) uint64 {
		if n.Add(1) == 3 {
			return 1
		}
		return block(s)
	})
	if bad.Stable() {
		t.Fatal("una cuenta mal tiene que detectarse")
	}
	hot := func() (telemetry.Sample, bool) { return telemetry.Sample{CPUTemp: 99}, true }
	if h := stress(context.Background(), time.Minute, 20*time.Millisecond, hot, nil, block); h.Stopped != "heat" {
		t.Fatalf("%+v", h.Stopped)
	}
	var pts []StabPoint
	for i := 0; i < 20; i++ {
		p := StabPoint{Rate: 100, Temp: telemetry.Unknown}
		if i >= 10 {
			p.Rate = 80
		}
		pts = append(pts, p)
	}
	d := StabResult{Points: pts, MaxTemp: telemetry.Unknown}
	if d.summarize(); d.Throttle != "drop" || d.DropPct < 19 {
		t.Fatalf("%+v", d)
	}
	for i := range pts {
		pts[i].Limit = 70
	}
	d = StabResult{Points: pts}
	if d.summarize(); d.Throttle != "limit" || d.MinLimit != 70 {
		t.Fatalf("%+v", d)
	}
}

func TestStoreRanksAndPrunes(t *testing.T) {
	dir := t.TempDir()
	s := OpenStore(dir)
	_, _ = s.Add(Entry{Kind: CPU1, Value: 900})
	if e, _ := s.Add(Entry{Kind: CPU1, Value: 1000}); e.Rank != 1 {
		t.Fatal("más puntos es mejor", e)
	}
	_, _ = s.Add(Entry{Kind: RAMLat, Value: 90})
	if e, _ := s.Add(Entry{Kind: RAMLat, Value: 70}); e.Rank != 1 {
		t.Fatal("menos latencia es mejor", e)
	}
	_, _ = s.Add(Entry{Kind: DiskRead, Target: `C:\`, Value: 500})
	if e, _ := s.Add(Entry{Kind: DiskRead, Target: `D:\`, Value: 100}); e.Rank != 1 {
		t.Fatal("cada unidad tiene su ranking", e)
	}
	for i := 0; i < maxPerKind+5; i++ {
		_, _ = s.Add(Entry{Kind: CPUAll, Value: 100})
	}
	worst, _ := s.Add(Entry{Kind: CPUAll, Value: 1})
	s = OpenStore(dir)
	n, found := 0, false
	for _, e := range s.List() {
		if e.Kind == CPUAll {
			n++
			found = found || e.ID == worst.ID
		}
	}
	if n != maxPerKind || !found {
		t.Fatal(n, found)
	}
	if s.Delete(worst.ID) != nil || s.Delete(0) != nil || len(s.List()) != 0 {
		t.Fatal("borrar")
	}
	_ = os.WriteFile(dir+"/pruebas.json", []byte("{roto"), 0o644)
	if len(OpenStore(dir).List()) != 0 {
		t.Fatal("archivo dañado: empieza vacío")
	}
}
