// Package audio silencia el micrófono de Windows (el mismo control que Configuración → Sonido) y hace la
// prueba de micrófono. No instala nada ni guarda grabaciones.
package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"time"
)

// Rate es la frecuencia de muestreo de la prueba de micrófono.
const Rate = 44100

// ErrNoMic: no hay un micrófono conectado.
var ErrNoMic = errors.New("No se encontró ningún micrófono conectado.")

func peak(s []int16) float64 {
	m := 0.0
	for _, v := range s {
		m = math.Max(m, math.Abs(float64(v)))
	}
	return m / 32768
}

// Report es el resultado de la prueba de micrófono. Los niveles están en dBFS (0 = máximo).
type Report struct {
	Seconds float64 `json:"seconds"`
	PeakDB  float64 `json:"peakDb"`
	NoiseDB float64 `json:"noiseDb"` // ruido de fondo (los momentos más silenciosos)
	VoiceDB float64 `json:"voiceDb"` // los momentos más fuertes
	Clipped bool    `json:"clipped"` // saturó
	Verdict string  `json:"verdict"` // silent, low, loud, noisy, ok
}

func db(v float64) float64 {
	if v < 1e-5 {
		return -100
	}
	return math.Round(200*math.Log10(v)) / 10
}

// Analyze saca las conclusiones de una grabación.
func Analyze(pcm []int16) Report {
	r := Report{Seconds: float64(len(pcm)) / Rate, PeakDB: db(peak(pcm))}
	const win = Rate / 20
	var rms []float64
	clip := 0
	for i := 0; i+win <= len(pcm); i += win {
		t := 0.0
		for _, v := range pcm[i : i+win] {
			f := float64(v) / 32768
			t += f * f
			if v >= 32700 || v <= -32700 {
				clip++
			}
		}
		rms = append(rms, math.Sqrt(t/win))
	}
	r.NoiseDB, r.VoiceDB = -100, -100
	if len(rms) > 0 {
		sort.Float64s(rms)
		r.NoiseDB, r.VoiceDB = db(rms[len(rms)/10]), db(rms[len(rms)*9/10])
	}
	r.Clipped = clip > 20
	switch {
	case r.PeakDB < -60:
		r.Verdict = "silent"
	case r.Clipped:
		r.Verdict = "loud"
	case r.VoiceDB < -40:
		r.Verdict = "low"
	case r.NoiseDB > -45:
		r.Verdict = "noisy"
	default:
		r.Verdict = "ok"
	}
	return r
}

// WAV arma un archivo .wav (mono, 16 bits) en memoria.
func WAV(pcm []int16) []byte {
	var b bytes.Buffer
	n := uint32(len(pcm) * 2)
	w := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(36 + n)
	b.WriteString("WAVEfmt ")
	w(uint32(16))
	w(uint16(1))
	w(uint16(1))
	w(uint32(Rate))
	w(uint32(Rate * 2))
	w(uint16(2))
	w(uint16(16))
	b.WriteString("data")
	w(n)
	w(pcm)
	return b.Bytes()
}

// Demo arma una grabación de ejemplo (modo demostración: no se usa el micrófono real).
func Demo(d time.Duration) []int16 {
	out := make([]int16, int(d.Seconds()*Rate))
	for i := range out {
		a := 0.001
		if i > len(out)/2 {
			a = 0.3
		}
		out[i] = int16(a * 32767 * math.Sin(float64(i)/30))
	}
	return out
}
