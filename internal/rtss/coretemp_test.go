package rtss

import (
	"encoding/binary"
	"math"
	"testing"
)

func fakeCT(cores, cpus int, temps []float32, tj uint32, fahr, delta bool) []byte {
	b := make([]byte, CoreTempBytes)
	binary.LittleEndian.PutUint32(b[ctCoreCntOff:], uint32(cores))
	binary.LittleEndian.PutUint32(b[ctCPUCntOff:], uint32(cpus))
	for i := 0; i < cpus; i++ {
		binary.LittleEndian.PutUint32(b[ctTjMaxOff+4*i:], tj)
	}
	for i, t := range temps {
		binary.LittleEndian.PutUint32(b[ctTempOff+4*i:], math.Float32bits(t))
	}
	if fahr {
		b[ctFahrenheit] = 1
	}
	if delta {
		b[ctDelta] = 1
	}
	return b
}

func TestParseCoreTemp(t *testing.T) {
	ss := ParseCoreTemp(fakeCT(6, 1, []float32{55, 61, 58, 60, 49, 57}, 95, false, false))
	if v, ok := CPUTemp(ss); !ok || v != 61 {
		t.Fatalf("debe tomar el núcleo más caliente: %v %v", v, ok)
	}
	if v, _ := CPUTemp(ParseCoreTemp(fakeCT(2, 1, []float32{30, 25}, 100, false, true))); v != 75 {
		t.Fatalf("con distancia al TjMax: %v", v)
	}
	if v, _ := CPUTemp(ParseCoreTemp(fakeCT(1, 1, []float32{140}, 100, true, false))); math.Abs(v-60) > 0.01 {
		t.Fatalf("en Fahrenheit: %v", v)
	}
	if ParseCoreTemp(fakeCT(0, 1, nil, 100, false, false)) != nil || ParseCoreTemp(fakeCT(300, 1, nil, 100, false, false)) != nil {
		t.Fatal("cantidad de núcleos imposible")
	}
	if ParseCoreTemp(fakeCT(2, 1, []float32{0, 0}, 100, false, false)) != nil || ParseCoreTemp(fakeCT(1, 1, []float32{999}, 100, false, false)) != nil {
		t.Fatal("sin datos válidos no hay sensor")
	}
	if ParseCoreTemp(nil) != nil || ParseCoreTemp(make([]byte, 100)) != nil {
		t.Fatal("memoria corta")
	}
}
