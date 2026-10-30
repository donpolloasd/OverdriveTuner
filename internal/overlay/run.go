package overlay

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/fpslog"
	"fpsforge/internal/rtss"
	"fpsforge/internal/system"
	"fpsforge/internal/telemetry"
)

// tasklistRunning dice si hay un proceso con ese nombre (exe ya validado).
func tasklistRunning(sys system.System, exe string) bool {
	if !ValidExe(exe) {
		return false
	}
	if m, ok := runningExes(); ok {
		_, found := m[strings.ToLower(exe)]
		return found
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
	if !demo {
		// Si RTSS está instalado se deja abierto en segundo plano, para que enganche cada juego desde el inicio.
		go func() {
			if rtss.EnsureRunning() {
				fpslog.Diag("RTSS estaba cerrado: se abrió en segundo plano")
			}
		}()
	}
	if !demo {
		// Core Temp (si está instalado) se deja abierto para poder leer la temperatura del procesador.
		go func() {
			if rtss.EnsureCoreTemp() {
				fpslog.Diag("Core Temp estaba cerrado: se abrió en segundo plano")
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
	fpslog.DiagPath = filepath.Join(dataDir, "presentmon_diagnostico.txt")
	d := Deps{Dir: dataDir, Sys: sys, PID: os.Getpid(), Quit: quit}
	if demo {
		d.GameRunning = func(string) bool { return true }
		d.FindGame = func() string { return "cs2.exe" }
		d.StartPM = func(_ string, on func(float64), _ func(error)) (func(), error) { return demoFrames(on), nil }
	} else {
		d.GameRunning = func(exe string) bool { return tasklistRunning(sys, exe) }
		d.Exclusive = exclusiveFullscreen
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
		d.StartPM = func(process string, on func(float64), _ func(error)) (func(), error) {
			fpslog.DiagPath = filepath.Join(dataDir, "presentmon_diagnostico.txt")
			fpslog.Diag("overlay ejecutándose como administrador: %v; los FPS salen de RTSS", sys.IsAdmin())
			rtssMu.Lock()
			defer rtssMu.Unlock()
			if curRTSS != nil { // un reinicio de la medición no debe dejar otro lector de RTSS activo
				curRTSS()
			}
			curRTSS = rtssFeed(process, on)
			return curRTSS, nil
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
				fpslog.Diag("el motor del overlay se detuvo por un error interno: %v\n%s", r, debug.Stack())
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
