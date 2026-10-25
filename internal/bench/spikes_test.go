package bench

import "testing"

func flat(n int, v float32) []float32 {
	f := make([]float32, n)
	for i := range f {
		f[i] = v
	}
	return f
}

func TestSpikesClean(t *testing.T) {
	r := AnalyzeSpikes(flat(500, 4), 120, 0)
	if r.Pattern != "clean" || r.Score < 95 {
		t.Fatalf("%+v", r)
	}
}

func TestSpikesTooFew(t *testing.T) {
	r := AnalyzeSpikes(flat(3, 4), 5, 0)
	if r.Cause == "" || r.Score != 100 {
		t.Fatalf("%+v", r)
	}
}

func TestSpikesPeriodic(t *testing.T) {
	f := flat(600, 4)
	for i := 50; i < 600; i += 60 {
		f[i] = 40
	}
	r := AnalyzeSpikes(f, 120, 0)
	if r.Pattern != "periodic" || r.PeriodSec < 10 || r.PeriodSec > 14 {
		t.Fatalf("%+v", r)
	}
}

func TestSpikesStart(t *testing.T) {
	f := flat(600, 4)
	for _, i := range []int{5, 10, 15, 20, 25, 30} {
		f[i] = 50
	}
	f[400] = 50
	r := AnalyzeSpikes(f, 120, 0)
	if r.Pattern != "start" {
		t.Fatalf("%+v", r)
	}
}

func TestSpikesBursts(t *testing.T) {
	f := flat(600, 4)
	for _, i := range []int{300, 302, 304, 306, 308, 500} {
		f[i] = 45
	}
	r := AnalyzeSpikes(f, 120, 0)
	if r.Pattern != "bursts" {
		t.Fatalf("%+v", r)
	}
}

func TestSpikesWorseScoresLower(t *testing.T) {
	a := flat(600, 4)
	b := flat(600, 4)
	for i := 10; i < 600; i += 100 {
		a[i] = 20
	}
	for i := 10; i < 600; i += 10 {
		b[i] = 60
	}
	if AnalyzeSpikes(b, 120, 0).Score >= AnalyzeSpikes(a, 120, 0).Score {
		t.Fatal("score no monótono")
	}
}

func TestSpikesTotalWins(t *testing.T) {
	r := AnalyzeSpikes(flat(500, 4), 120, 37)
	if r.Count != 37 {
		t.Fatalf("%d", r.Count)
	}
}
