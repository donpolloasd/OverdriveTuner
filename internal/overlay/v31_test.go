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
