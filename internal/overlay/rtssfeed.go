package overlay

import (
	"math"
	"sync"
	"time"

	"fpsforge/internal/fpslog"
	"fpsforge/internal/rtss"
)

// rtssFeed lee los FPS que publica RTSS y los entrega como cuadros.
// Devuelve la función que lo detiene.
func rtssFeed(exe string, onFrame func(float64)) func() {
	stop := make(chan struct{})
	go func() {
		defer func() { // un fallo acá nunca debe cerrar el overlay
			if r := recover(); r != nil {
				fpslog.Diag("RTSS: el lector se detuvo por un error interno: %v", r)
			}
		}()
		const every = 50 * time.Millisecond
		tk := time.NewTicker(every)
		defer tk.Stop()
		using := false
		state := ""
		note := func(st, msg string) {
			if st != state {
				state = st
				fpslog.Diag("RTSS: %s", msg)
			}
		}
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
			}
			rtss.Fault = ""
			es := rtss.Read()
			if rtss.Fault != "" {
				note("fallo", rtss.Fault)
			}
			if es == nil {
				if rtss.Fault != "" {
					continue
				}
				note("cerrado", "RTSS/MSI Afterburner no está abierto (no se puede usar como respaldo)")
				continue
			}
			e, ok := rtss.Find(es, exe)
			if !ok {
				note("sin-juego", "RTSS está abierto pero todavía no mide "+exe)
				continue
			}
			note("midiendo", "RTSS mide "+exe)
			fps, ms, ok := e.FPS(rtss.Tick())
			if !ok {
				continue
			}
			if !using {
				using = true
				fpslog.Diag("RTSS: llegan FPS de RTSS (%.0f FPS)", fps)
			}
			n := int(math.Round(fps * every.Seconds()))
			if n < 1 {
				n = 1
			}
			for i := 0; i < n; i++ {
				onFrame(ms)
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(stop) }) } // se puede llamar más de una vez
}
