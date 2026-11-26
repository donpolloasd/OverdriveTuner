package telemetry

import (
	"encoding/binary"
	"testing"
)

func TestParseDiskTemp(t *testing.T) {
	b := make([]byte, 24+2*16)
	binary.LittleEndian.PutUint16(b[12:], 2)
	binary.LittleEndian.PutUint16(b[24+2:], 38)
	binary.LittleEndian.PutUint16(b[40+2:], 47)
	if v, ok := parseDiskTemp(b); !ok || v != 47 {
		t.Fatalf("%v %v", v, ok)
	}
	if _, ok := parseDiskTemp(b[:20]); ok {
		t.Fatal("una respuesta corta no es válida")
	}
	if _, ok := parseDiskTemp(make([]byte, 56)); ok {
		t.Fatal("sin sensores no hay temperatura")
	}
	t.Logf("SSD de este equipo: %v", SSDTemp())
	for n := 0; n < 3; n++ {
		t.Logf("disco %d: %d bytes, %d, %d", n, len(diskQuery(n, 7)), len(diskQuery(n, 52)), len(diskQuery(n, 51)))
	}
}
