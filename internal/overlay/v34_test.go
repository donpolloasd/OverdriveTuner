package overlay

import (
	"testing"
	"time"

	"fpsforge/internal/system"
	"fpsforge/internal/telemetry"
)

func TestFreePosStaysOnScreen(t *testing.T) {
	m := Monitor{L: 1920, T: 100, Width: 1920, Height: 1080}
	c := Config{PosX: 1, PosY: 1}
	if x, y := freePos(m, 200, 100, c); x != 1920+1720 || y != 100+980 {
		t.Fatal(x, y)
	}
	c = Config{}
	if x, y := freePos(m, 5000, 5000, c); x != 1920 || y != 100 {
		t.Fatal("más grande que la pantalla", x, y)
	}
}

func TestV34Normalize(t *testing.T) {
	c := Default()
	c.PosX, c.PosY, c.CPUWarn, c.CPUBad, c.GPUBad, c.TempWarn, c.AlertSSD = 4, -1, 95, 60, 300, 5, 500
	c.HkMic, c.HkRAM, c.HkOpen = "Ctrl+Alt+O", "x", "ctrl+alt+k"
	c.Normalize()
	if c.PosX != 0.02 || c.PosY != 0.03 || c.CPUWarn != 60 || c.CPUBad != 95 || c.GPUBad != 0 || c.TempWarn != 70 || c.AlertSSD != 70 {
		t.Fatalf("%+v", c)
	}
	if c.HkMic != "" || c.HkRAM != "" || c.HkOpen != "Ctrl+Alt+K" {
		t.Fatal("los atajos repetidos o inválidos quedan sin asignar", c.HkMic, c.HkRAM, c.HkOpen)
	}
	c = Default()
	c.HkMic = "Ctrl+Alt+O"
	if c.HotkeyConflict() == "" {
		t.Fatal("repetido con mostrar/ocultar")
	}
}

func TestLevelColorsAndSSDRow(t *testing.T) {
	c := Default()
	c.Levels, c.ShowSSD = true, true
	sn := Snapshot{Cfg: c, Sample: telemetry.Sample{CPU: 99, RAMUsedMB: 85, RAMTotMB: 100, CPUTemp: -1, GPUTemp: -1, SSDTemp: 75}, AlertSSD: true, MicMuted: true}
	m := BuildModel(sn)
	p := m.Pal
	col := map[string]uint32{}
	for _, r := range m.Rows {
		col[r.Label] = r.Parts[0].Color
	}
	if col["CPU"] != p.Bad || col["RAM"] != p.Warn || col["SSD"] != p.Bad || col["GPU"] != p.Value || col["Mic"] != p.Bad || col["Aviso"] != p.Bad {
		t.Fatalf("%+v", m.Rows)
	}
	sn.Cfg.Levels = false
	for _, r := range BuildModel(sn).Rows {
		if r.Label == "CPU" && r.Parts[0].Color != p.Value {
			t.Fatal("sin umbrales no cambia de color")
		}
	}
}

func TestProblemHeatAndMic(t *testing.T) {
	now := time.Unix(1000, 0)
	muted := false
	e := NewEngine(Deps{Dir: t.TempDir(), Sys: system.NewFake(), Now: func() time.Time { return now },
		MicMuted: func() (bool, error) { return muted, nil }, SetMic: func(m bool) error { muted = m; return nil }})
	e.cfg.AlertTray = true
	e.checkProblem()
	if e.Snapshot().Problem {
		t.Fatal("sin datos no hay problema")
	}
	e.OnSample(telemetry.Sample{CPUTemp: -1, GPUTemp: 99, SSDTemp: 30})
	e.checkProblem()
	if !e.Snapshot().Problem || e.heat.kind != "gpu" {
		t.Fatal("la gráfica caliente es un problema")
	}
	first := e.heat.since
	e.OnSample(telemetry.Sample{CPUTemp: -1, GPUTemp: 40, SSDTemp: 30})
	now = now.Add(time.Minute)
	e.checkProblem()
	if e.Snapshot().Problem {
		t.Fatal("ya pasó")
	}
	e.OnSample(telemetry.Sample{CPUTemp: -1, GPUTemp: 99, SSDTemp: 30})
	if e.heat.since != first {
		t.Fatal("no se repite el aviso enseguida")
	}
	st := State{Heat: "gpu", HeatSince: first, HeatTemp: 99}
	dir := t.TempDir()
	if !ClaimHeatNotice(dir, st) || ClaimHeatNotice(dir, st) {
		t.Fatal("se avisa una vez")
	}
	if a, _ := st.HeatNotice(true); a == "" {
		t.Fatal("texto")
	}
	e.ToggleMic()
	if !muted || !e.Snapshot().MicMuted {
		t.Fatal("silenciar")
	}
	e.ToggleMic()
	if muted {
		t.Fatal("activar")
	}
	for i := 0; i < 300; i++ {
		e.OnFrame(50) // 20 FPS
	}
	now = now.Add(time.Hour)
	e.OnSample(telemetry.Sample{CPUTemp: -1, GPUTemp: 40})
	e.lastFrame = now
	e.checkProblem()
	e.checkProblem()
	if !e.Snapshot().Problem {
		t.Fatal("FPS bajos")
	}
}
