package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fpsforge/internal/bench"
	"fpsforge/internal/system"
	"fpsforge/internal/telemetry"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newEng(t *testing.T) (*Engine, *clock, *bool, *system.Fake, string) {
	dir := t.TempDir()
	clk := &clock{time.Date(2026, 10, 13, 22, 0, 0, 0, time.UTC)}
	running := new(bool)
	f := system.NewFake()
	started := 0
	_ = started
	e := NewEngine(Deps{Dir: dir, Sys: f, Now: clk.now, PID: 4242,
		GameRunning: func(string) bool { return *running },
		StartPM: func(p string, on func(float64), _ func(error)) (func(), error) {
			return func() {}, nil
		}})
	return e, clk, running, f, dir
}

func TestLiveFPSAndSnapshot(t *testing.T) {
	e, clk, running, _, _ := newEng(t)
	*running = true
	e.Tick()
	e.Tick() // detecta el juego
	for i := 0; i < 500; i++ {
		e.OnFrame(4) // 250 FPS
	}
	sn := e.Snapshot()
	if !sn.GameRunning || sn.FPS < 249 || sn.FPS > 251 {
		t.Fatalf("%+v", sn)
	}
	if len(sn.Spark) == 0 {
		t.Fatal("falta el gráfico")
	}
	clk.t = clk.t.Add(3 * time.Second) // sin cuadros nuevos: FPS en 0
	if sn := e.Snapshot(); sn.FPS != 0 {
		t.Fatalf("sin cuadros debe mostrar 0, fue %v", sn.FPS)
	}
}

func TestRecordingSavesSession(t *testing.T) {
	e, _, running, _, dir := newEng(t)
	*running = true
	e.Tick()
	e.Tick()
	e.StartRecording("optimizado")
	for i := 0; i < 2000; i++ {
		ft := 4.0
		if i == 1000 {
			ft = 50
		}
		e.OnFrame(ft)
	}
	id := e.StopRecording()
	if id == "" {
		t.Fatalf("no guardó: %s", e.msg)
	}
	b, err := os.ReadFile(filepath.Join(dir, "sessions", id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var s bench.Session
	if json.Unmarshal(b, &s) != nil || s.Run.Frames != 2000 || s.Spikes != 1 || s.Label != "optimizado" || len(s.Frames) == 0 {
		t.Fatalf("%+v", s)
	}
	if s.Run.Label != "optimizado" || s.Run.Src != "fpsforge" {
		t.Fatalf("%+v", s.Run)
	}
	// Una grabación corta no se guarda.
	e.StartRecording("x")
	e.OnFrame(4)
	if e.StopRecording() != "" {
		t.Fatal("una grabación de 1 cuadro no debe guardarse")
	}
}

func TestCommandsAndState(t *testing.T) {
	e, _, _, _, dir := newEng(t)
	e.OnSample(telemetry.Sample{CPU: 40, GPU: 70, CPUTemp: telemetry.Unknown, GPUTemp: 55})
	e.Tick()
	st, ok := LoadState(dir)
	if !ok || st.PID != 4242 || st.Sample.GPU != 70 || st.Sample.GPUTemp != 55 {
		t.Fatalf("%+v", st)
	}
	if err := SendCmd(dir, "record_start", "antes"); err != nil {
		t.Fatal(err)
	}
	e.Tick()
	if !e.IsRecording() {
		t.Fatal("debía empezar a grabar")
	}
	e.Tick() // la misma orden no se repite
	quit := false
	e.d.Quit = func() { quit = true }
	time.Sleep(2 * time.Millisecond)
	_ = SendCmd(dir, "quit", "")
	e.Tick()
	if !quit || e.IsRecording() {
		t.Fatal("quit debe detener la grabación y salir")
	}
}

func TestOldCommandIgnoredAtStart(t *testing.T) {
	dir := t.TempDir()
	_ = SendCmd(dir, "quit", "")
	e := NewEngine(Deps{Dir: dir})
	quit := false
	e.d.Quit = func() { quit = true }
	e.Tick()
	if quit {
		t.Fatal("una orden vieja no debe cerrar el overlay recién abierto")
	}
}

func TestGameProfileAppliedAndRestored(t *testing.T) {
	e, _, running, f, dir := newEng(t)
	plan := "381b4222-f694-41f0-9685-ff5bb260df2e"
	cfg := Default()
	cfg.Games = []Game{{Exe: "cs2.exe", Priority: true, PowerPlan: plan, Close: []string{"chrome.exe", "lsass.exe"}}}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	f.ExecHook = func(name string, args []string) (string, int, error) {
		if len(args) > 0 && args[0] == "/getactivescheme" {
			return "GUID del esquema de energía: 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c  (Alto rendimiento)", 0, nil
		}
		return "", 0, nil
	}
	*running = true
	e.Tick()
	e.Tick()
	var closed, prio bool
	for _, s := range f.PSLog {
		if strings.Contains(s, "'chrome'") && strings.Contains(s, "Stop-Process") {
			closed = true
		}
		if strings.Contains(s, "lsass") {
			t.Fatal("jamás debe intentar cerrar un proceso protegido")
		}
		if strings.Contains(s, "PriorityClass") {
			prio = true
		}
	}
	if !closed || !prio {
		t.Fatalf("faltan acciones: cerrar=%v prioridad=%v log=%v", closed, prio, f.PSLog)
	}
	*running = false
	e.Tick()
	e.Tick()
	last := f.ExecLog[len(f.ExecLog)-1]
	if !strings.Contains(last, "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c") {
		t.Fatalf("no restauró el plan anterior: %v", f.ExecLog)
	}
}

func TestConfigNormalize(t *testing.T) {
	c := Default()
	c.Corner, c.Scale, c.Opacity, c.GameExe = "zz", 9, 0, `..\evil.exe`
	c.Games = []Game{{Exe: "a b.exe"}, {Exe: "ok.exe", PowerPlan: "no-es-guid", Close: []string{"x.exe", `..\y.exe`}}}
	c.Normalize()
	if c.Corner != "tl" || c.Scale != 1 || c.Opacity != 0.82 || c.GameExe != "cs2.exe" {
		t.Fatalf("%+v", c)
	}
	if len(c.Games) != 1 || c.Games[0].PowerPlan != "" || len(c.Games[0].Close) != 1 {
		t.Fatalf("%+v", c.Games)
	}
}

func TestLiveLow1(t *testing.T) {
	fr := make([]float64, 1000)
	for i := range fr {
		fr[i] = 4 // 250 FPS
	}
	for i := 0; i < 5; i++ {
		fr[i*100] = 100 // 0,5 % de picos: no deben hundir el percentil 99
	}
	if v := liveLow1(fr); v < 249 || v > 251 {
		t.Fatalf("p99 con 0,5%% de picos debe seguir en 250, dio %.1f", v)
	}
	for i := 0; i < 20; i++ {
		fr[i*50+1] = 10 // 2 % de cuadros lentos sí cuentan
	}
	if v := liveLow1(fr); v > 101 {
		t.Fatalf("con 2%% de cuadros de 10 ms el 1%% low debe ser ~100, dio %.1f", v)
	}
}

func TestFPSWindow(t *testing.T) {
	if fpsWindowMs(50) != 250 || fpsWindowMs(100) != 400 || fpsWindowMs(500) != 1000 {
		t.Fatal("la ventana de promedio debe ser de 250 ms a 1 s")
	}
	fr := make([]float64, 0, 200)
	for i := 0; i < 100; i++ {
		fr = append(fr, 10) // 100 FPS
	}
	for i := 0; i < 50; i++ {
		fr = append(fr, 5) // 200 FPS en el último cuarto de segundo
	}
	if v := liveFPS(fr, 250); v < 195 || v > 205 {
		t.Fatalf("con 250 ms debía seguir el cambio (≈200), fue %.0f", v)
	}
}
