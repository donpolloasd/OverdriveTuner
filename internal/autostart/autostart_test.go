package autostart

import (
	"strings"
	"testing"

	"fpsforge/internal/system"
)

func TestEnableDisable(t *testing.T) {
	f := system.NewFake()
	if err := Enable(f, `C:\Program Files\OverdriveTuner\OverdriveTuner.exe`); err != nil {
		t.Fatal(err)
	}
	s := f.PSLog[len(f.PSLog)-1]
	for _, w := range []string{"-RunLevel Highest", "--overlay --boot", "-WorkingDirectory", "-AtLogOn", `'C:\Program Files\OverdriveTuner\OverdriveTuner.exe'`, "-AllowStartIfOnBatteries"} {
		if !strings.Contains(s, w) {
			t.Fatalf("falta %q en el script:\n%s", w, s)
		}
	}
	if err := Enable(f, ""); err == nil {
		t.Fatal("sin ruta debe fallar")
	}
	if Enabled(f) {
		t.Fatal("el simulado no tiene la tarea")
	}
	f.PSHook = func(string) (string, error) { return "si\n", nil }
	if !Enabled(f) {
		t.Fatal("debe detectar la tarea")
	}
	if err := Disable(f); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.PSLog[len(f.PSLog)-1], "Unregister-ScheduledTask") {
		t.Fatal("Disable debe quitar la tarea")
	}
	if Script(`C:\it's\a.exe`) == "" || !strings.Contains(Script(`C:\it's\a.exe`), `'C:\it''s\a.exe'`) {
		t.Fatal("las comillas simples deben escaparse")
	}
}

func TestParseInfo(t *testing.T) {
	i := parseInfo("ruido\r\nT|Ready|2026-10-23 17:02|0\r\n")
	if !i.Registered || i.State != "Ready" || i.LastRun != "2026-10-23 17:02" || i.LastResult != 0 {
		t.Fatalf("%+v", i)
	}
	if j := parseInfo("T|Ready||267011"); !j.Registered || j.LastRun != "" || j.LastResult != 267011 {
		t.Fatalf("%+v", j)
	}
	if parseInfo("").Registered || parseInfo("nada").Registered {
		t.Fatal("sin tarea no está registrada")
	}
	if RunNow(system.NewFake()) != nil {
		t.Fatal("RunNow en el simulado")
	}
}
