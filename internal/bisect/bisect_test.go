package bisect

import (
	"fmt"
	"math/bits"
	"sort"
	"testing"
)

func ids(n int) []string {
	var o []string
	for i := 0; i < n; i++ {
		o = append(o, fmt.Sprintf("t%02d", i))
	}
	return o
}

// simula: el problema existe si el culpable NO está revertido.
func run(t *testing.T, n int, culprit string) (State, int) {
	applied := map[string]bool{}
	for _, id := range ids(n) {
		applied[id] = true
	}
	apply := func(a Actions) {
		for _, x := range a.Revert {
			if !applied[x] {
				t.Fatalf("revierte %s que ya estaba revertido", x)
			}
			applied[x] = false
		}
		for _, x := range a.Apply {
			if applied[x] {
				t.Fatalf("aplica %s que ya estaba aplicado", x)
			}
			applied[x] = true
		}
	}
	s, a, err := Start(ids(n))
	if err != nil {
		t.Fatal(err)
	}
	apply(a)
	rounds := 1
	for s.Active {
		good := !applied[culprit] // sin el culpable el rendimiento vuelve
		s, a, err = s.Answer(good)
		if err != nil {
			t.Fatal(err)
		}
		apply(a)
		rounds++
		if rounds > 20 {
			t.Fatal("no termina")
		}
	}
	return s, rounds
}

func TestFindsEveryCulprit(t *testing.T) {
	for _, n := range []int{2, 3, 7, 16, 17, 41} {
		for _, c := range ids(n) {
			s, rounds := run(t, n, c)
			if s.Culprit != c || !s.Done {
				t.Fatalf("n=%d culpable %s: %+v", n, c, s)
			}
			if len(s.Reverted) != 1 || s.Reverted[0] != c {
				t.Fatalf("solo el culpable debe quedar revertido: %v", s.Reverted)
			}
			if max := bits.Len(uint(n)) + 2; rounds > max {
				t.Fatalf("n=%d: %d rondas", n, rounds)
			}
		}
	}
}

func TestNoCulprit(t *testing.T) {
	// El problema es ruido: nunca mejora.
	s, a, _ := Start(ids(8))
	_ = a
	for s.Active {
		s, a, _ = s.Answer(false)
	}
	if s.Culprit != "" || !s.Done || len(s.Reverted) != 0 {
		t.Fatalf("%+v", s)
	}
	sort.Strings(a.Apply)
}

func TestCancelAndGuards(t *testing.T) {
	if _, _, err := Start([]string{"a"}); err == nil {
		t.Fatal("con un solo ajuste no hay nada que buscar")
	}
	s, _, _ := Start(ids(6))
	s2, a := s.Cancel()
	if len(a.Apply) != 3 || s2.Active || len(s2.Reverted) != 0 {
		t.Fatalf("%+v %+v", s2, a)
	}
	if _, _, err := s2.Answer(true); err == nil {
		t.Fatal("responder sin búsqueda activa debe fallar")
	}
}

func TestSaveLoad(t *testing.T) {
	d := t.TempDir()
	s, _, _ := Start(ids(4))
	if err := Save(d, s); err != nil {
		t.Fatal(err)
	}
	if l := Load(d); !l.Active || len(l.Tested) != 2 {
		t.Fatalf("%+v", l)
	}
}
