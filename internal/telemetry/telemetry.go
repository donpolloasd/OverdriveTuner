// Package telemetry mide el uso de CPU, GPU y RAM (y temperaturas cuando
// Windows las expone) sin drivers de kernel ni inyección: solo contadores
// de rendimiento de Windows, compatibles con los anti-cheat.
package telemetry

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// Unknown marca un valor que no se pudo medir (temperatura, por ejemplo).
const Unknown = -1.0

// Sample es una medición.
type Sample struct {
	At        time.Time `json:"at"`
	CPU       float64   `json:"cpu"`       // % total
	CPUThread float64   `json:"cpuThread"` // % del hilo más cargado
	CPUMHz    float64   `json:"cpuMhz"`
	CPUTemp   float64   `json:"cpuTemp"` // Unknown si no hay
	RAMUsedMB float64   `json:"ramUsedMb"`
	RAMTotMB  float64   `json:"ramTotMb"`
	GPU       float64   `json:"gpu"`
	GPUVRAMMB float64   `json:"gpuVramMb"`
	GPUTemp   float64   `json:"gpuTemp"` // Unknown si no hay
	GPULuid   string    `json:"gpuLuid,omitempty"`
}

// RAMPct devuelve el uso de memoria en %.
func (s Sample) RAMPct() float64 {
	if s.RAMTotMB <= 0 {
		return 0
	}
	return 100 * s.RAMUsedMB / s.RAMTotMB
}

type rawGPU struct {
	Luid string  `json:"luid"`
	U    float64 `json:"u"`
	V    float64 `json:"v"` // bytes dedicados en uso
}

type rawLine struct {
	CPU    float64  `json:"cpu"`
	Max    float64  `json:"max"`
	MHz    float64  `json:"mhz"`
	Free   float64  `json:"freeKb"`
	Total  float64  `json:"totalKb"`
	GPU    []rawGPU `json:"gpu"`
	NVTemp *float64 `json:"nv"`
	LHMCPU *float64 `json:"lhmCpu"`
	LHMGPU *float64 `json:"lhmGpu"`
}

// NormLuid unifica el formato del identificador de una gráfica.
func NormLuid(s string) string {
	return strings.Replace(strings.ToUpper(strings.TrimSpace(s)), "0X", "0x", -1)
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// ParseLine interpreta una línea JSON del muestreador. gpuTemps (opcional)
// da la temperatura por luid de la gráfica (vía Windows).
func ParseLine(line string, gpuTemps map[string]float64) (Sample, error) {
	var r rawLine
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &r); err != nil {
		return Sample{}, fmt.Errorf("línea no válida: %w", err)
	}
	s := Sample{At: time.Now(), CPU: clamp(r.CPU, 0, 100), CPUThread: clamp(r.Max, 0, 100), CPUMHz: r.MHz,
		CPUTemp: Unknown, GPUTemp: Unknown}
	if r.Total > 0 {
		s.RAMTotMB = r.Total / 1024
		s.RAMUsedMB = math.Max(0, (r.Total-r.Free)/1024)
	}
	// La gráfica que se está usando: la de mayor uso 3D (si empatan, la de más VRAM).
	best := -1
	for i, g := range r.GPU {
		if best < 0 || g.U > r.GPU[best].U || (g.U == r.GPU[best].U && g.V > r.GPU[best].V) {
			best = i
		}
	}
	if best >= 0 {
		g := r.GPU[best]
		g.Luid = NormLuid(g.Luid)
		s.GPU, s.GPUVRAMMB, s.GPULuid = clamp(g.U, 0, 100), g.V/1048576, g.Luid
		if t, ok := gpuTemps[g.Luid]; ok && t > 0 && t < 150 {
			s.GPUTemp = t
		}
	}
	if r.NVTemp != nil && *r.NVTemp > 0 && *r.NVTemp < 150 {
		s.GPUTemp = *r.NVTemp
	}
	if r.LHMGPU != nil && s.GPUTemp == Unknown && *r.LHMGPU > 0 && *r.LHMGPU < 150 {
		s.GPUTemp = *r.LHMGPU
	}
	if r.LHMCPU != nil && *r.LHMCPU > 0 && *r.LHMCPU < 150 {
		s.CPUTemp = *r.LHMCPU
	}
	return s, nil
}

// Verdict explica qué está limitando los FPS.
type Verdict struct {
	Kind string `json:"kind"` // gpu, cpu_thread, cpu, other, none
	Text string `json:"text"`
}

func avg(xs []Sample, f func(Sample) float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	t := 0.0
	for _, x := range xs {
		t += f(x)
	}
	return t / float64(len(xs))
}

// Bottleneck analiza muestras tomadas jugando (al menos 5).
func Bottleneck(xs []Sample) Verdict {
	if len(xs) < 5 {
		return Verdict{"none", "Faltan datos: jugá unos segundos con el overlay activo."}
	}
	gpu := avg(xs, func(s Sample) float64 { return s.GPU })
	thr := avg(xs, func(s Sample) float64 { return s.CPUThread })
	cpu := avg(xs, func(s Sample) float64 { return s.CPU })
	switch {
	case gpu >= 92:
		return Verdict{"gpu", fmt.Sprintf("Limita la gráfica (%.0f%% de uso). Bajar resolución, sombras o efectos sube los FPS.", gpu)}
	case thr >= 80:
		return Verdict{"cpu_thread", fmt.Sprintf("Limita un solo hilo del procesador (%.0f%%), aunque el uso total sea de apenas %.0f%%. Es normal en CS2: ayudan la RAM a su velocidad (DOCP/XMP), cerrar programas de fondo y bajar efectos que cargan al procesador.", thr, cpu)}
	case cpu >= 90:
		return Verdict{"cpu", fmt.Sprintf("El procesador está saturado (%.0f%%). Cerrá programas de fondo.", cpu)}
	default:
		return Verdict{"other", fmt.Sprintf("Ni el procesador (hilo máx. %.0f%%) ni la gráfica (%.0f%%) llegan al límite. Puede haber un tope de FPS (fps_max, V-Sync), o limita la latencia de la memoria RAM.", thr, gpu)}
	}
}
