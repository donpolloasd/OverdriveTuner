package rtss

import (
	"encoding/binary"
	"math"
	"strings"
)

// Sensor es una lectura que MSI Afterburner publica en «MAHMSharedMemory».
type Sensor struct {
	Name  string
	Units string
	Value float64
}

const mahmSignature = 0x4D41484D // 'MAHM'

// ParseSensors interpreta la memoria compartida de Afterburner. Cada entrada tiene cinco textos de
// 260 bytes (nombre, unidad, nombre local, unidad local, formato) y después el valor (float).
func ParseSensors(b []byte) []Sensor {
	if len(b) < 32 || binary.LittleEndian.Uint32(b[0:]) != mahmSignature {
		return nil
	}
	off := int(binary.LittleEndian.Uint32(b[8:]))
	n := int(binary.LittleEndian.Uint32(b[12:]))
	size := int(binary.LittleEndian.Uint32(b[16:]))
	if size < 1324 || n < 0 || n > 512 || off < 0 {
		return nil
	}
	var out []Sensor
	for i := 0; i < n; i++ {
		p := off + i*size
		if p+1304 > len(b) {
			break
		}
		str := func(o int) string {
			t := b[p+o : p+o+260]
			if z := indexZero(t); z >= 0 {
				t = t[:z]
			}
			return string(t)
		}
		v := float64(math.Float32frombits(binary.LittleEndian.Uint32(b[p+1300:])))
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		out = append(out, Sensor{Name: str(0), Units: str(260), Value: v})
	}
	return out
}

// CPUTemp devuelve la temperatura del procesador (°C) entre los sensores; ok=false si no hay.
// Prefiere «CPU temperature» y, si no existe, el máximo de los «CPUn temperature» (núcleos).
func CPUTemp(ss []Sensor) (float64, bool) {
	best, found := 0.0, false
	for _, s := range ss {
		n := strings.ToLower(s.Name)
		if !strings.HasPrefix(n, "cpu") || !strings.Contains(n, "temp") || s.Value <= 0 || s.Value > 150 {
			continue
		}
		if n == "cpu temperature" {
			return s.Value, true
		}
		if s.Value > best {
			best, found = s.Value, true
		}
	}
	return best, found
}

// GPUTemp devuelve la temperatura de la gráfica (°C) entre los sensores.
func GPUTemp(ss []Sensor) (float64, bool) {
	for _, s := range ss {
		n := strings.ToLower(s.Name)
		if strings.HasPrefix(n, "gpu") && strings.Contains(n, "temp") && s.Value > 0 && s.Value < 150 {
			return s.Value, true
		}
	}
	return 0, false
}
