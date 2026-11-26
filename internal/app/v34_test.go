package app

import (
	"strings"
	"testing"
	"time"
)

func waitFor(t *testing.T, what string, f func() bool) {
	t.Helper()
	for i := 0; i < 300; i++ {
		if f() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("no terminó: " + what)
}

func TestPerfRunCancelAndRank(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, _ := call(t, s, base, "/api/perf/run", `{"kind":"x"}`, nil); c != 400 {
		t.Fatal("prueba desconocida", c)
	}
	if c, _ := call(t, s, base, "/api/perf/run", `{"kind":"diskr","root":"..\\x"}`, nil); c != 400 {
		t.Fatal("solo unidades reales", c)
	}
	if c, b := call(t, s, base, "/api/perf/run", `{"kind":"cpuall"}`, nil); c != 200 {
		t.Fatal(c, b)
	}
	if c, _ := call(t, s, base, "/api/perf/run", `{"kind":"cpu1"}`, nil); c != 409 {
		t.Fatal("una a la vez", c)
	}
	call(t, s, base, "/api/perf/run", `{"cancel":true}`, nil)
	waitFor(t, "cancelar", func() bool {
		_, b := call(t, s, base, "/api/perf", "", nil)
		return strings.Contains(b, `"running":""`)
	})
	if _, b := call(t, s, base, "/api/perf", "", nil); !strings.Contains(b, "Prueba cancelada") || !strings.Contains(b, `"list":[]`) {
		t.Fatal("lo cancelado no entra en el ranking", b)
	}
	call(t, s, base, "/api/perf/run", `{"kind":"ramlat"}`, nil)
	waitFor(t, "latencia", func() bool {
		_, b := call(t, s, base, "/api/perf", "", nil)
		return strings.Contains(b, `"running":""`)
	})
	if _, b := call(t, s, base, "/api/perf", "", nil); !strings.Contains(b, `"rank":1`) {
		t.Fatal(b)
	}
	if c, _ := call(t, s, base, "/api/perf/delete", `{}`, nil); c != 400 {
		t.Fatal("sin decir qué borrar")
	}
	if c, _ := call(t, s, base, "/api/perf/delete", `{"all":true}`, nil); c != 200 {
		t.Fatal("borrar todo")
	}
}

func TestMicWinOptsUpgradesAndCleanPlan(t *testing.T) {
	s, base, f := newTestServer(t)
	if _, b := call(t, s, base, "/api/mic", `{"mute":true}`, nil); !strings.Contains(b, `"muted":true`) {
		t.Fatal(b)
	}
	call(t, s, base, "/api/mic", `{"mute":false}`, nil)
	if c, _ := call(t, s, base, "/api/mic/wav", "", nil); c != 404 {
		t.Fatal("sin grabación", c)
	}
	call(t, s, base, "/api/mic", `{"record":true}`, nil)
	waitFor(t, "grabar", func() bool {
		_, b := call(t, s, base, "/api/mic", "", nil)
		return strings.Contains(b, `"verdict":"ok"`)
	})
	if c, b := call(t, s, base, "/api/mic/wav", "", nil); c != 200 || !strings.HasPrefix(b, "RIFF") {
		t.Fatal(c)
	}
	call(t, s, base, "/api/mic", `{"forget":true}`, nil)
	if c, _ := call(t, s, base, "/api/mic/wav", "", nil); c != 404 {
		t.Fatal("descartada")
	}

	const id = "{00000000-0000-0000-0000-000000000001}"
	if c, b := call(t, s, base, "/api/winopts", `{"id":"`+id+`","fav":true}`, nil); c != 200 || !strings.Contains(b, id) {
		t.Fatal(c, b)
	}
	if c, _ := call(t, s, base, "/api/winopts", `{"id":"`+id+`"}`, nil); c != 200 {
		t.Fatal(c)
	}
	if c, _ := call(t, s, base, "/api/winopts", `{"id":"{11111111-0000-0000-0000-000000000001}"}`, nil); c != 404 {
		t.Fatal("solo opciones de la lista")
	}
	if c, _ := call(t, s, base, "/api/winopts/folder", `{}`, nil); c != 200 {
		t.Fatal(c)
	}

	if _, b := call(t, s, base, "/api/apps/upgrades", "", nil); !strings.Contains(b, "7zip.7zip") {
		t.Fatal(b)
	}
	if c, _ := call(t, s, base, "/api/apps/upgrades", `{"id":"--all"}`, nil); c != 400 {
		t.Fatal("identificador inválido")
	}
	if c, _ := call(t, s, base, "/api/apps/upgrades", `{"id":"7zip.7zip"}`, nil); c != 200 || !strings.Contains(strings.Join(f.ExecLog, "\n"), "upgrade --id 7zip.7zip --exact") {
		t.Fatal(f.ExecLog)
	}

	if _, b := call(t, s, base, "/api/clean/plan", "", nil); strings.Contains(b, `"shaders"`) || !strings.Contains(b, `"usertemp"`) {
		t.Fatal("los shaders nunca se limpian solos", b)
	}
	if c, _ := call(t, s, base, "/api/schedule", `{"id":"cleanup","clean":["shaders"]}`, nil); c != 400 {
		t.Fatal("nada permitido")
	}
	if c, b := call(t, s, base, "/api/schedule", `{"id":"cleanup","enabled":true,"freq":"daily","day":1,"time":"10:00","clean":["usertemp","discordcache"]}`, nil); c != 200 || !strings.Contains(b, `"freq":"daily"`) {
		t.Fatal(c, b)
	}
	if c, _ := call(t, s, base, "/api/dnsbench", `{"custom":"no-es-ip"}`, nil); c != 400 {
		t.Fatal("DNS propio inválido")
	}
	if c, b := call(t, s, base, "/api/hotkeys", "", nil); c != 200 || !strings.Contains(b, "hkMic") {
		t.Fatal(c, b)
	}
}

func TestGameLaunch(t *testing.T) {
	s, base, f := newTestServer(t)
	s.SpawnOverlay = func() error { return nil }
	if c, _ := call(t, s, base, "/api/gamemode/find", `{"exe":"a b.exe"}`, nil); c != 400 {
		t.Fatal("nombre inválido")
	}
	if _, b := call(t, s, base, "/api/gamemode/find", `{"exe":"cs2.exe"}`, nil); !strings.Contains(b, "steam://rungameid/730") {
		t.Fatal(b)
	}
	if c, _ := call(t, s, base, "/api/gamemode/launch", `{"exe":"cs2.exe"}`, nil); c != 400 {
		t.Fatal("sin forma de abrirlo")
	}
	_, cfg := call(t, s, base, "/api/overlay", "", nil)
	_ = cfg
	body := `{"show":true,"gameExe":"cs2.exe","games":[{"exe":"cs2.exe","launch":"steam://rungameid/730"},{"exe":"x.exe","launch":"cmd.exe /c calc"}]}`
	if c, b := call(t, s, base, "/api/overlay/config", body, nil); c != 200 || strings.Contains(b, "calc") {
		t.Fatal(c, b)
	}
	if c, b := call(t, s, base, "/api/gamemode/launch", `{"exe":"cs2.exe"}`, nil); c != 200 || !strings.Contains(b, `"overlay":true`) {
		t.Fatal(c, b)
	}
	if !strings.Contains(strings.Join(f.ExecLog, " | "), "explorer.exe steam://rungameid/730") {
		t.Fatal(f.ExecLog)
	}
}
