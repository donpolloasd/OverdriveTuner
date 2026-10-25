package rtss

import (
	"encoding/binary"
	"math"
	"testing"
)

func mahmBuf(names []string, vals []float32) []byte {
	const size = 1324
	b := make([]byte, 32+len(names)*size)
	binary.LittleEndian.PutUint32(b[0:], mahmSignature)
	binary.LittleEndian.PutUint32(b[8:], 32)
	binary.LittleEndian.PutUint32(b[12:], uint32(len(names)))
	binary.LittleEndian.PutUint32(b[16:], size)
	for i, n := range names {
		p := 32 + i*size
		copy(b[p:], n)
		copy(b[p+260:], "C")
		binary.LittleEndian.PutUint32(b[p+1300:], math.Float32bits(vals[i]))
	}
	return b
}

func TestCPUTemp(t *testing.T) {
	ss := ParseSensors(mahmBuf([]string{"GPU temperature", "CPU1 temperature", "CPU2 temperature", "CPU usage"}, []float32{55, 61, 68, 20}))
	if v, ok := CPUTemp(ss); !ok || v != 68 {
		t.Fatalf("debía tomar el máximo de los núcleos (68), fue %v %v", v, ok)
	}
	if v, ok := GPUTemp(ss); !ok || v != 55 {
		t.Fatalf("GPU 55, fue %v %v", v, ok)
	}
	ss = ParseSensors(mahmBuf([]string{"CPU temperature", "CPU1 temperature"}, []float32{47, 70}))
	if v, _ := CPUTemp(ss); v != 47 {
		t.Fatalf("«CPU temperature» tiene prioridad, fue %v", v)
	}
	if ParseSensors([]byte("nada")) != nil || ParseSensors(make([]byte, 64)) != nil {
		t.Fatal("datos inválidos deben dar nil")
	}
}
