package overlay

import (
	"os"
	"testing"

	"fpsforge/internal/system"
	"fpsforge/internal/telemetry"
)

func TestThrottleDetection(t *testing.T) {
	var th throttleState
	hot := telemetry.Sample{CPU: 60, CPULimit: 70, CPUTemp: 96, GPUTemp: telemetry.Unknown}
	for i := 0; i < thrCPUSamples-1; i++ {
		if th.feed(hot, 1) {
			t.Fatal("necesita varias mediciones seguidas")
		}
	}
	if !th.feed(hot, 42) || th.kind != "cpu" || th.since != 42 {
		t.Fatalf("%+v", th)
	}
	if th.feed(hot, 43) {
		t.Fatal("un aviso por partida")
	}
	var g throttleState
	idle := telemetry.Sample{CPU: 5, CPULimit: 60, GPUTemp: 50}
	for i := 0; i < 30; i++ {
		if g.feed(idle, 1) {
			t.Fatal("con poca carga no es throttling")
		}
	}
	st := State{Throttle: "cpu", ThrottleSince: 42, ThrottleLimit: 70, ThrottleTemp: 96}
	dir := t.TempDir()
	if !ClaimThrottleNotice(dir, st) || ClaimThrottleNotice(dir, st) {
		t.Fatal("se avisa una sola vez")
	}
	if st.ThrottleText(true) == st.ThrottleText(false) || st.ThrottleTitle(false) == "" {
		t.Fatal("textos en los dos idiomas")
	}
}

func TestQuietRestoresNotifications(t *testing.T) {
	f := system.NewFake()
	dir := t.TempDir()
	quietOn(f, dir)
	if v, _ := system.RegGet(f, pushKey, "ToastEnabled"); !v.Exists || !v.Matches("REG_DWORD", "0") {
		t.Fatal("se apagan durante el juego")
	}
	quietOff(f, dir)
	if v, _ := system.RegGet(f, pushKey, "ToastEnabled"); v.Exists {
		t.Fatal("no existía: se borra al terminar")
	}
	if _, err := os.Stat(quietPath(dir)); err == nil {
		t.Fatal("la copia se borra al restaurar")
	}
	// Si el usuario ya las tenía apagadas, no se toca nada ni se «restaura» después.
	f.Seed(pushKey, "ToastEnabled", "REG_DWORD", "0")
	quietOn(f, dir)
	if _, err := os.Stat(quietPath(dir)); err == nil {
		t.Fatal("ya estaban apagadas: no hay nada que restaurar")
	}
	f.Seed(pushKey, "ToastEnabled", "REG_DWORD", "1")
	quietOn(f, dir)
	quietOff(f, dir)
	if v, _ := system.RegGet(f, pushKey, "ToastEnabled"); !v.Matches("REG_DWORD", "1") {
		t.Fatal("vuelven a como estaban")
	}
}

func TestMiniLayoutAndGameQuiet(t *testing.T) {
	c := Default()
	c.Layout = "mini"
	c.Normalize()
	if c.Layout != "mini" {
		t.Fatal("el diseño mínimo es válido")
	}
	c.Layout = "raro"
	c.Normalize()
	if c.Layout != Default().Layout {
		t.Fatal("diseño desconocido vuelve al de siempre")
	}
	f := system.NewFake()
	f.Seed(pushKey, "ToastEnabled", "REG_DWORD", "1")
	dir := t.TempDir()
	cfg := Default()
	cfg.Games = []Game{{Exe: "cs2.exe", Quiet: true, FreeRAM: true}}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(Deps{Dir: dir, Sys: f})
	e.gameStarted(LoadConfig(dir), "cs2.exe")
	if v, _ := system.RegGet(f, pushKey, "ToastEnabled"); !v.Matches("REG_DWORD", "0") {
		t.Fatal("el perfil del juego silencia las notificaciones")
	}
	e.gameEnded()
	if v, _ := system.RegGet(f, pushKey, "ToastEnabled"); !v.Matches("REG_DWORD", "1") {
		t.Fatal("al cerrar el juego vuelven")
	}
}

func TestWidgetSnapshot(t *testing.T) {
	c := Default()
	c.Widget, c.WidgetPos, c.Layout, c.ShowCPU, c.ShowFPS = true, "bl", "mini", false, true
	w := WidgetSnapshot(Snapshot{Cfg: c})
	if w.Cfg.Layout != "card" || w.Cfg.Corner != "bl" || w.Cfg.ShowFPS || !w.Cfg.ShowCPU || !w.Cfg.ShowTemp || !w.Visible {
		t.Fatalf("%+v", w.Cfg)
	}
	if !WidgetVisible(Snapshot{Cfg: c}) || WidgetVisible(Snapshot{Cfg: c, GameRunning: true}) {
		t.Fatal("se ve en el escritorio y se esconde en el juego")
	}
	c.WidgetPos = "centro"
	c.Normalize()
	if c.WidgetPos != "tr" {
		t.Fatal("esquina desconocida")
	}
}

func TestLow01(t *testing.T) {
	fr := make([]float64, 2000)
	for i := range fr {
		fr[i] = 4 // 250 FPS
	}
	fr[100], fr[900], fr[1500] = 20, 20, 20 // tres tirones de 50 FPS: entran en el 0.1% pero no en el 1%
	if v := liveLow01(fr[:500]); v != 0 {
		t.Fatal("con pocos cuadros no se muestra")
	}
	if v := liveLow01(fr); v < 49 || v > 51 {
		t.Fatalf("0.1%% low: %v", v)
	}
	if v := liveLow1(fr); v < 249 {
		t.Fatalf("1%% low: %v", v)
	}
	c := Default()
	c.ShowLow01 = true
	m := BuildModel(Snapshot{Cfg: c, FPS: 240, Low1: 200, Low01: 120})
	if m.Low != "1% 200  0.1% 120" {
		t.Fatalf("texto: %q", m.Low)
	}
}
