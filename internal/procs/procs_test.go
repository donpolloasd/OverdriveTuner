package procs

import (
	"testing"

	"fpsforge/internal/system"
)

func TestParseAndSafety(t *testing.T) {
	js := `[{"pid":10,"name":"chrome","cpu":30.5,"mb":900,"path":"C:\\Program Files\\Google\\chrome.exe"},
{"pid":11,"name":"svchost","cpu":5,"mb":50,"path":"C:\\Windows\\System32\\svchost.exe"},
{"pid":12,"name":"cs2","cpu":99,"mb":3000,"path":"D:\\cs2.exe"},
{"pid":13,"name":"weird","cpu":1,"mb":5,"path":""}]`
	ps, err := parse(js)
	if err != nil || len(ps) != 4 || ps[0].Name != "cs2" {
		t.Fatalf("%v %+v", err, ps)
	}
	safe := map[string]bool{}
	for _, p := range ps {
		safe[p.Name] = p.Safe
	}
	if !safe["chrome"] || safe["svchost"] || safe["cs2"] || safe["weird"] {
		t.Fatalf("%+v", safe)
	}
	if one, err := parse(`{"pid":1,"name":"a","cpu":2,"mb":1,"path":"C:\\x.exe"}`); err != nil || len(one) != 1 {
		t.Fatalf("objeto único: %v %v", one, err)
	}
	if e, err := parse(""); err != nil || e == nil || len(e) != 0 {
		t.Fatalf("vacío debe dar lista vacía")
	}
}

func TestKillRefusals(t *testing.T) {
	f := system.NewFake()
	for _, c := range []struct {
		pid  int
		name string
	}{{4, "x"}, {100, "lsass"}, {100, "CS2.exe"}, {100, ""}} {
		if Kill(f, c.pid, c.name) == nil {
			t.Errorf("debía rechazar %+v", c)
		}
	}
	f.PSHook = func(string) (string, error) { return "OK", nil }
	if err := Kill(f, 100, "chrome"); err != nil {
		t.Fatal(err)
	}
	f.PSHook = func(string) (string, error) { return "DIFF", nil }
	if Kill(f, 100, "chrome") == nil {
		t.Fatal("PID reutilizado debe fallar")
	}
}
