package overlay

import (
	"math"
	"sync/atomic"
	"time"

	"fpsforge/internal/presentmon"
	"fpsforge/internal/rtss"
)

// rtssFeed lee los FPS que publica RTSS (MSI Afterburner) y los entrega como cuadros, pero solo
// mientras PresentMon no esté entregando datos (lastPM = último cuadro de PresentMon, en UnixNano).
// Devuelve la función que lo detiene.
func rtssFeed(exe string, lastPM *atomic.Int64, onFrame func(float64)) func() {
	stop := make(chan struct{})
	go func() {
		defer func() { // un fallo acá nunca debe cerrar el overlay
			if r := recover(); r != nil {
				presentmon.Diag("RTSS: el lector se detuvo por un error interno: %v", r)
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
				presentmon.Diag("RTSS: %s", msg)
			}
		}
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
			}
			if time.Since(time.Unix(0, lastPM.Load())) < 3*time.Second {
				if using {
					using = false
					presentmon.Diag("RTSS: PresentMon volvió a entregar cuadros, se deja de usar RTSS")
				}
				continue
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
				presentmon.Diag("RTSS: se usan los FPS de RTSS/MSI Afterburner (%.0f FPS) porque PresentMon no entrega cuadros", fps)
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
	return func() { close(stop) }
}
