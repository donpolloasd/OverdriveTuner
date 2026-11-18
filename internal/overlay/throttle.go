package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"fpsforge/internal/telemetry"
)

// Detector de throttling: avisa cuando, con un juego abierto, el procesador trabaja por debajo de su
// velocidad máxima porque el firmware lo limita (temperatura o energía), o cuando la gráfica pasa mucho
// tiempo a temperatura de recorte. Usa contadores de Windows y los sensores que ya lee el overlay.

const (
	thrCPUSamples = 8  // mediciones seguidas (una por segundo) con el procesador limitado
	thrGPUSamples = 15 // y con la gráfica muy caliente y a pleno
	thrCPULimit   = 95 // % de la velocidad máxima por debajo del cual hay límite
	thrGPUTemp    = 90 // °C
)

type throttleState struct {
	cpuN, gpuN int
	kind       string // «», cpu, gpu
	since      int64  // Unix ms: identifica cada aviso
	limit      float64
	temp       float64
}

// feed suma una medición. Devuelve true si recién ahora se detectó un throttling.
func (t *throttleState) feed(s telemetry.Sample, nowMs int64) bool {
	if s.CPULimit > 0 && s.CPULimit < thrCPULimit && s.CPU >= 25 {
		t.cpuN++
	} else {
		t.cpuN = 0
	}
	if s.GPUTemp != telemetry.Unknown && s.GPUTemp >= thrGPUTemp && s.GPU >= 80 {
		t.gpuN++
	} else {
		t.gpuN = 0
	}
	if t.kind != "" {
		return false // un aviso por partida
	}
	switch {
	case t.cpuN >= thrCPUSamples:
		t.kind, t.limit, t.temp = "cpu", s.CPULimit, s.CPUTemp
	case t.gpuN >= thrGPUSamples:
		t.kind, t.temp = "gpu", s.GPUTemp
	default:
		return false
	}
	t.since = nowMs
	return true
}

func (t *throttleState) reset() { *t = throttleState{} }

// ClaimThrottleNotice dice si hay que avisar del throttling en la bandeja (una vez por cada detección).
func ClaimThrottleNotice(dir string, st State) bool {
	if st.Throttle == "" || st.ThrottleSince == 0 {
		return false
	}
	p := filepath.Join(dir, "overlay_throttle_aviso.txt")
	mark := strconv.FormatInt(st.ThrottleSince, 10)
	if b, err := os.ReadFile(p); err == nil && string(b) == mark {
		return false
	}
	return os.WriteFile(p, []byte(mark), 0o644) == nil
}

// ThrottleTitle es el título del aviso de throttling.
func (st State) ThrottleTitle(en bool) string {
	switch {
	case st.Throttle == "cpu" && en:
		return "Your CPU is slowing down"
	case st.Throttle == "cpu":
		return "Tu procesador está bajando la velocidad"
	case en:
		return "Your graphics card is very hot"
	default:
		return "Tu placa de video está muy caliente"
	}
}

// ThrottleText explica el aviso de throttling y qué hacer.
func (st State) ThrottleText(en bool) string {
	temp := ""
	if st.ThrottleTemp > 0 {
		temp = fmt.Sprintf(" (%.0f °C)", st.ThrottleTemp)
	}
	if st.Throttle == "cpu" {
		if en {
			return fmt.Sprintf("Windows limits it to %.0f%% of its maximum speed%s: it is overheating or the power plan/firmware is holding it back. Check the cooler and the power plan.", st.ThrottleLimit, temp)
		}
		return fmt.Sprintf("Windows lo limita al %.0f%% de su velocidad máxima%s: se está calentando o el plan de energía o el firmware lo frenan. Revisá el disipador y el plan de energía.", st.ThrottleLimit, temp)
	}
	if en {
		return fmt.Sprintf("It has been above 90 °C%s at full load: it may be lowering its clocks. Check the airflow and clean the fans.", temp)
	}
	return fmt.Sprintf("Lleva un rato por encima de 90 °C%s a pleno: puede estar bajando su velocidad. Revisá la ventilación y limpiá los ventiladores.", temp)
}
