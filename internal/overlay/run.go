package overlay

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fpsforge/internal/presentmon"
	"fpsforge/internal/rtss"
	"fpsforge/internal/system"
	"fpsforge/internal/telemetry"
)

// tasklistRunning dice si hay un proceso con ese nombre (exe ya validado).
func tasklistRunning(sys system.System, exe string) bool {
	if !ValidExe(exe) {
		return false
	}
	out, _, err := sys.Exec(8*time.Second, "tasklist.exe", "/FI", "IMAGENAME eq "+exe, "/NH", "/FO", "CSV")
	return err == nil && strings.Contains(strings.ToLower(out), `"`+strings.ToLower(exe)+`"`)
}

// demoFrames simula cuadros de ~240 FPS con algún pico (modo demostración).
func demoFrames(onFrame func(float64)) func() {
	stop := make(chan struct{})
	go func() {
		tk := time.NewTicker(4 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				ft := 4.2 + rand.Float64()*0.8
				if rand.Intn(900) == 0 {
					ft = 22 + rand.Float64()*20
				}
				onFrame(ft)
			}
		}
	}()
	return func() { close(stop) }
}

// Main es el programa del overlay (OverdriveTuner.exe --overlay).
func Main(dataDir string, demo bool) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	var sys system.System
	if demo {
		sys = system.NewFake()
	} else {
		sys = system.New()
	}
	if !demo && LoadConfig(dataDir).FpsSource != SrcPresentMon {
		// Si RTSS está instalado se deja abierto en segundo plano, para que enganche cada juego desde el inicio.
		go func() {
			if rtss.EnsureRunning() {
				presentmon.Diag("RTSS estaba cerrado: se abrió en segundo plano")
			}
		}()
	}
	if !demo {
		// Core Temp (si está instalado) se deja abierto para poder leer la temperatura del procesador.
		go func() {
			if rtss.EnsureCoreTemp() {
				presentmon.Diag("Core Temp estaba cerrado: se abrió en segundo plano")
			}
		}()
	}
	var quitMu sync.Mutex
	var quitFn func()
	quit := func() {
		quitMu.Lock()
		f := quitFn
		quitMu.Unlock()
		if f != nil {
			f()
		}
	}
	presentmon.DiagPath = filepath.Join(dataDir, "presentmon_diagnostico.txt")
	d := Deps{Dir: dataDir, Sys: sys, PID: os.Getpid(), Quit: quit}
	if demo {
		d.GameRunning = func(string) bool { return true }
		d.FindGame = func() string { return "cs2.exe" }
		d.StartPM = func(_ string, on func(float64), _ func(error)) (func(), error) { return demoFrames(on), nil }
	} else {
		d.GameRunning = func(exe string) bool { return tasklistRunning(sys, exe) }
		d.FindGame = func() string {
			return FindGame(sys, func() ([]string, []string) {
				var l []string
				c := LoadConfig(dataDir)
				for _, g := range c.Games {
					l = append(l, g.Exe)
				}
				return l, c.GameDirs
			})
		}
		var rtssMu sync.Mutex
		var curRTSS func()
		d.StartPM = func(process string, on func(float64), onExit func(error)) (func(), error) {
			presentmon.DiagPath = filepath.Join(dataDir, "presentmon_diagnostico.txt")
			src := LoadConfig(dataDir).FpsSource
			presentmon.Diag("overlay ejecutándose como administrador: %v; fuente de FPS: %s", sys.IsAdmin(), src)
			var lastPM atomic.Int64
			lastPM.Store(time.Now().UnixNano())
			wrapped := func(ms float64) { lastPM.Store(time.Now().UnixNano()); on(ms) }
			stopRTSS := func() {}
			rtssMu.Lock()
			if curRTSS != nil { // un reinicio de PresentMon no debe dejar otro lector de RTSS activo
				curRTSS()
				curRTSS = nil
			}
			if src != SrcPresentMon { // «solo PresentMon» nunca lee RTSS
				if src == SrcRTSS {
					lastPM.Store(0) // sin PresentMon: RTSS desde el primer segundo
				}
				stopRTSS = rtssFeed(process, &lastPM, on)
				curRTSS = stopRTSS
			}
			rtssMu.Unlock()
			if src == SrcRTSS {
				return stopRTSS, nil
			}
			exe, err := presentmon.Extract(filepath.Join(dataDir, "bin"))
			if err != nil {
				return stopRTSS, err
			}
			r, err := presentmon.Start(exe, process, wrapped, onExit)
			if err != nil {
				return stopRTSS, fmt.Errorf("PresentMon no arrancó (hace falta ejecutar como administrador): %w", err)
			}
			return func() { r.Stop(); stopRTSS() }, nil
		}
	}
	e := NewEngine(d)
	stopSamples := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				presentmon.Diag("el motor del overlay se detuvo por un error interno: %v\n%s", r, debug.Stack())
			}
		}()
		e.Run(ctx, telemetry.Stream(stopSamples))
	}()
	err := RunWindow(e, func(f func()) { quitMu.Lock(); quitFn = f; quitMu.Unlock() })
	cancel()
	close(stopSamples)
	<-done
	ClearState(dataDir)
	return err
}
