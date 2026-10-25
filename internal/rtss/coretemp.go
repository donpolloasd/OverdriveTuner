package rtss

import (
	"encoding/binary"
	"math"
)

// Core Temp publica la temperatura de cada núcleo en la memoria compartida «CoreTempMappingObject»
// (solo lectura, sin límite de horas). Disposición según su SDK:
//
//	uint32 load[256], uint32 tjMax[128], uint32 coreCount, uint32 cpuCount, float32 temp[256],
//	float32 vid, speed, fsb, multiplier, char cpuName[100], uint8 fahrenheit, uint8 deltaToTjMax
const (
	ctTjMaxOff    = 1024
	ctCoreCntOff  = 1536
	ctCPUCntOff   = 1540
	ctTempOff     = 1544
	ctFahrenheit  = 2684
	ctDelta       = 2685
	CoreTempBytes = 2688
)

// ParseCoreTemp devuelve la temperatura del procesador («CPU temperature»), que es la del núcleo más caliente.
func ParseCoreTemp(b []byte) []Sensor {
	if len(b) < CoreTempBytes {
		return nil
	}
	cores := int(binary.LittleEndian.Uint32(b[ctCoreCntOff:]))
	cpus := int(binary.LittleEndian.Uint32(b[ctCPUCntOff:]))
	if cores < 1 || cores > 256 || cpus < 1 || cpus > 8 || cores*cpus > 256 {
		return nil
	}
	fahr, delta := b[ctFahrenheit] != 0, b[ctDelta] != 0
	best, found := 0.0, false
	for i := 0; i < cores*cpus; i++ {
		v := float64(math.Float32frombits(binary.LittleEndian.Uint32(b[ctTempOff+4*i:])))
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if delta { // algunos procesadores informan la distancia al TjMax en vez de la temperatura
			tj := float64(binary.LittleEndian.Uint32(b[ctTjMaxOff+4*(i/cores):]))
			if tj <= 0 {
				continue
			}
			v = tj - v
		}
		if fahr {
			v = (v - 32) * 5 / 9
		}
		if v <= 0 || v > 150 {
			continue
		}
		if v > best {
			best, found = v, true
		}
	}
	if !found {
		return nil
	}
	return []Sensor{{Name: "CPU temperature", Units: "°C", Value: best}}
}
