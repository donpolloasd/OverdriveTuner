package rtss

import (
	"encoding/binary"
	"testing"
)

func fake(name string, t0, t1, frames, ft uint32) []byte {
	b := make([]byte, 40+2*296)
	binary.LittleEndian.PutUint32(b[0:], signature)
	binary.LittleEndian.PutUint32(b[4:], 0x20005)
	binary.LittleEndian.PutUint32(b[8:], 296)
	binary.LittleEndian.PutUint32(b[12:], 40)
	binary.LittleEndian.PutUint32(b[16:], 2)
	p := 40 + 296 // segunda entrada; la primera queda vacía
	binary.LittleEndian.PutUint32(b[p:], 4321)
	copy(b[p+4:], name)
	binary.LittleEndian.PutUint32(b[p+268:], t0)
	binary.LittleEndian.PutUint32(b[p+272:], t1)
	binary.LittleEndian.PutUint32(b[p+276:], frames)
	binary.LittleEndian.PutUint32(b[p+280:], ft)
	return b
}

func TestParseAndFPS(t *testing.T) {
	es := Parse(fake(`C:\Games\EA SPORTS FC 26\FC26.exe`, 10000, 11000, 300, 3200))
	e, ok := Find(es, "fc26.exe")
	if !ok || e.PID != 4321 {
		t.Fatalf("%+v %v", es, ok)
	}
	fps, ms, ok := e.FPS(11500)
	if !ok || fps != 300 || ms != 3.2 {
		t.Fatalf("fps=%v ms=%v ok=%v", fps, ms, ok)
	}
	if _, _, ok := e.FPS(14000); ok {
		t.Fatal("un programa que dejó de dibujar no debe dar FPS")
	}
	if Parse([]byte("nada")) != nil {
		t.Fatal("firma inválida")
	}
}
