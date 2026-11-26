package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fpsforge/internal/fpslog"
)

// freePos ubica el panel en la posición libre del editor: 0 y 1 son los bordes del monitor, así el panel
// nunca queda fuera de la pantalla aunque cambie de tamaño o de monitor.
func freePos(m Monitor, w, h int, c Config) (int, int) {
	return m.L + int(c.PosX*float64(max(0, m.Width-w))+0.5), m.T + int(c.PosY*float64(max(0, m.Height-h))+0.5)
}

// ---- aviso de temperatura alta en las notificaciones de Windows ----

const heatEvery = 10 * time.Minute // como mucho un aviso por componente en este tiempo

type heatNotice struct {
	kind  string
	since int64
	temp  float64
	last  map[string]time.Time
}

func (h *heatNotice) note(kind string, t float64, now time.Time) {
	if at, ok := h.last[kind]; ok && now.Sub(at) < heatEvery {
		return
	}
	if h.last == nil {
		h.last = map[string]time.Time{}
	}
	h.last[kind] = now
	h.kind, h.temp, h.since = kind, t, now.UnixMilli()
}

// ClaimHeatNotice dice si hay que mostrar la notificación de temperatura (una vez por cada aviso).
func ClaimHeatNotice(dir string, st State) bool {
	if st.Heat == "" || st.HeatSince == 0 {
		return false
	}
	p := filepath.Join(dir, "overlay_heat_aviso.txt")
	mark := strconv.FormatInt(st.HeatSince, 10)
	if b, err := os.ReadFile(p); err == nil && string(b) == mark {
		return false
	}
	return os.WriteFile(p, []byte(mark), 0o644) == nil
}

// HeatNotice es el título y el texto de la notificación de temperatura.
func (st State) HeatNotice(en bool) (title, text string) {
	es := map[string]string{"cpu": "El procesador", "gpu": "La placa de video", "ssd": "El SSD"}
	eng := map[string]string{"cpu": "The processor", "gpu": "The graphics card", "ssd": "The SSD"}
	if en {
		return "High temperature", fmt.Sprintf("%s reached %.0f °C, above the limit you set. Check the airflow and the fans.", eng[st.Heat], st.HeatTemp)
	}
	return "Temperatura alta", fmt.Sprintf("%s llegó a %.0f °C, más que el límite que elegiste. Revisá la ventilación y los ventiladores.", es[st.Heat], st.HeatTemp)
}

// ---- «solo si hay un problema» ----

const problemHold = 6 * time.Second // el panel queda visible un rato para que se llegue a leer

// checkProblem decide (una vez por segundo) si hay un problema para mostrar.
func (e *Engine) checkProblem() {
	e.mu.Lock()
	defer e.mu.Unlock()
	sn := e.snapshotLocked()
	c := e.cfg
	if sn.FPS > 0 && sn.FPS < float64(c.Style.FpsLow) {
		e.lowFPS++
	} else {
		e.lowFPS = 0
	}
	bad := e.lowFPS >= 2 || (c.AlertOn && (e.alertCPU || e.alertGPU || e.alertSSD)) ||
		(c.RAMBad > 0 && e.sample.RAMPct() >= float64(c.RAMBad)) ||
		e.thr.cpuN >= thrCPUSamples || e.thr.gpuN >= thrGPUSamples
	if bad {
		e.problemTill = e.d.Now().Add(problemHold)
	}
}

// ---- acciones de los atajos ----

// checkMic mira si el micrófono está silenciado (solo si se usa esa función).
func (e *Engine) checkMic(c Config) {
	if e.d.MicMuted == nil || (!c.ShowMic || c.HkMic == "") {
		e.mu.Lock()
		e.micMuted = false
		e.mu.Unlock()
		return
	}
	m, err := e.d.MicMuted()
	e.mu.Lock()
	e.micMuted = m && err == nil
	e.mu.Unlock()
}

// ToggleMic silencia o activa el micrófono de Windows.
func (e *Engine) ToggleMic() {
	if e.d.MicMuted == nil || e.d.SetMic == nil {
		return
	}
	m, err := e.d.MicMuted()
	if err == nil {
		err = e.d.SetMic(!m)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err != nil {
		fpslog.Diag("micrófono: %v", err)
		e.msg = err.Error()
		return
	}
	e.micMuted = !m
}

// FreeRAM libera memoria (atajo).
func (e *Engine) FreeRAM() {
	if e.d.FreeRAM != nil {
		e.d.FreeRAM()
	}
}

// OpenApp abre la ventana de OverdriveTuner (atajo).
func (e *Engine) OpenApp() {
	if e.d.OpenApp != nil {
		e.d.OpenApp()
	}
}
