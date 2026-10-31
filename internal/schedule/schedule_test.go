package schedule

import (
	"strings"
	"testing"

	"fpsforge/internal/extras"
	"fpsforge/internal/system"
)

func TestEnableValidatesAndBuildsScript(t *testing.T) {
	f := system.NewFake()
	if err := Enable(f, "cleanup", `C:\P F\OverdriveTuner.exe`, 0, "20:30"); err != nil {
		t.Fatal(err)
	}
	s := f.PSLog[0]
	for _, want := range []string{"OverdriveTuner Limpieza semanal", "-DaysOfWeek Sunday", "'20:30'", "--task cleanup", "-RunLevel Highest", "-StartWhenAvailable"} {
		if !strings.Contains(s, want) {
			t.Errorf("falta %q en el script:\n%s", want, s)
		}
	}
	for _, bad := range []struct {
		id  string
		day int
		tm  string
	}{{"x", 0, "20:00"}, {"cleanup", 7, "20:00"}, {"cleanup", -1, "20:00"}, {"cleanup", 1, "25:00"}, {"cleanup", 1, "20:00'; calc"}, {"restore", 1, ""}} {
		if Enable(f, bad.id, "a.exe", bad.day, bad.tm) == nil {
			t.Errorf("debía rechazar %+v", bad)
		}
	}
	if Enable(f, "restore", " ", 1, "10:00") == nil {
		t.Error("sin ruta no se programa")
	}
	if err := Disable(f, "restore"); err != nil || !strings.Contains(f.PSLog[len(f.PSLog)-1], "Unregister-ScheduledTask") {
		t.Errorf("Disable: %v", err)
	}
}

func TestParseInfo(t *testing.T) {
	i := parseInfo("cleanup", "ruido\nT|2026-10-25 20:00|2026-11-01 20:00|0|8|20:00\n")
	if !i.Registered || i.Day != 3 || i.Time != "20:00" || i.NextRun != "2026-11-01 20:00" || i.LastResult != 0 {
		t.Errorf("%+v", i)
	}
	if i := parseInfo("cleanup", "T|||267011|1|07:15"); i.Day != 0 || i.Time != "07:15" || i.LastRun != "" {
		t.Errorf("%+v", i)
	}
	if parseInfo("cleanup", "").Registered {
		t.Error("sin salida no está registrada")
	}
}

func TestRunWritesLog(t *testing.T) {
	dir := t.TempDir()
	f := system.NewFake()
	msg, err := Run(f, dir, "restore")
	if err != nil || !strings.Contains(msg, "restauración") {
		t.Fatalf("%q %v", msg, err)
	}
	if !strings.Contains(LastMessage(dir, "restore"), "punto de restauración creado") {
		t.Error("debía quedar anotado")
	}
	if _, err := Run(f, dir, "otra"); err == nil {
		t.Error("tarea desconocida")
	}
	if LastMessage(dir, "cleanup") != "" {
		t.Error("sin ejecuciones no hay mensaje")
	}
	// limpieza: con objetivos simulados (la real borraría los temporales del equipo que corre los tests)
	var ran []string
	cleanTargets = func() []extras.Target {
		return []extras.Target{{ID: "usertemp", Safe: true}, {ID: "shaders", Confirm: "x"}, {ID: "edgecache", Deep: true}}
	}
	cleanRun = func(ids []string) []extras.CleanResult { ran = ids; return nil }
	defer func() { cleanTargets, cleanRun = extras.Targets, extras.Run }()
	if _, err := Run(f, dir, "cleanup"); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "usertemp" {
		t.Fatalf("la limpieza semanal solo borra lo seguro (nunca shaders ni cachés de navegadores): %v", ran)
	}
}
