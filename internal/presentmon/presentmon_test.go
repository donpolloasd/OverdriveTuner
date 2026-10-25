package presentmon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEmbeddedHash(t *testing.T) {
	if err := Verify(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p, err := Extract(dir)
	if err != nil || filepath.Base(p) != "PresentMon.exe" {
		t.Fatalf("%v %s", err, p)
	}
	if err := os.WriteFile(p, []byte("alterado"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(dir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); len(b) < 100000 {
		t.Fatal("un ejecutable alterado debe reemplazarse por el original")
	}
}

func TestParserV1AndV2(t *testing.T) {
	var p Parser
	if _, ok := p.Line("Application,ProcessID,SwapChainAddress,Runtime,SyncInterval,PresentFlags,Dropped,TimeInSeconds,msInPresentAPI,msBetweenPresents,AllowsTearing"); ok {
		t.Fatal("el encabezado no es un cuadro")
	}
	if ms, ok := p.Line("cs2.exe,1234,0x1,DXGI,0,0,0,1.5,0.1,4.16,1"); !ok || ms != 4.16 {
		t.Fatalf("%v %v", ms, ok)
	}
	var q Parser
	q.Line("Application,ProcessID,SwapChainAddress,PresentRuntime,SyncInterval,PresentFlags,AllowsTearing,PresentMode,CPUStartTime,FrameTime,CPUBusy,CPUWait")
	if _, ok := q.Line("cs2.exe,1,0x1,DXGI,0,0,1,Hardware,1.0,5.5,4,1"); ok {
		t.Fatal("v2: la primera fila solo marca el inicio")
	}
	if ms, ok := q.Line("cs2.exe,1,0x1,DXGI,0,0,1,Hardware,6.5,5.4,4,1"); !ok || ms != 5.5 {
		t.Fatalf("v2: %v %v", ms, ok)
	}
	if _, ok := q.Line("cs2.exe,1,0x1"); ok {
		t.Fatal("línea corta")
	}
	if _, ok := q.Line("cs2.exe,1,0x1,DXGI,0,0,1,Hardware,1.0,abc,4,1"); ok {
		t.Fatal("valor no numérico")
	}
}

func TestParserFiltersInflatedFrames(t *testing.T) {
	now := time.Unix(1000, 0)
	p := Parser{Now: func() time.Time { return now }}
	p.Line("Application,ProcessID,SwapChainAddress,PresentRuntime,FrameType,FrameTime")
	// filas duplicadas de ~0 ms no cuentan
	if _, ok := p.Line("cs2.exe,1,0xA,DXGI,Application,0.01"); ok {
		t.Fatal("un duplicado de 0,01 ms no es un cuadro")
	}
	// cuadros generados no cuentan
	if _, ok := p.Line("cs2.exe,1,0xA,DXGI,AMD_AFMF,2.5"); ok {
		t.Fatal("un cuadro generado no es real")
	}
	if ms, ok := p.Line("cs2.exe,1,0xA,DXGI,Application,5.0"); !ok || ms != 5 {
		t.Fatal("cuadro normal")
	}
	// una cadena secundaria con menos cuadros se ignora cuando ya hay una dominante
	for i := 0; i < 20; i++ {
		p.Line("cs2.exe,1,0xA,DXGI,Application,5.0")
	}
	now = now.Add(1100 * time.Millisecond)
	p.Line("cs2.exe,1,0xA,DXGI,Application,5.0")
	if _, ok := p.Line("cs2.exe,1,0xB,DXGI,Application,5.0"); ok {
		t.Fatal("la cadena secundaria debe ignorarse")
	}
	if _, ok := p.Line("cs2.exe,1,0xA,DXGI,Application,5.0"); !ok {
		t.Fatal("la cadena dominante sigue contando")
	}
}

// Muestra real (PresentMon 2.3 en CS2): cada cuadro viene precedido por una fila de espera.
func TestParserRealSample(t *testing.T) {
	b, err := os.ReadFile("testdata/muestra_real.csv")
	if err != nil {
		t.Fatal(err)
	}
	var p Parser
	var frames []float64
	rows := 0
	for _, l := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(l) != "" {
			rows++
		}
		if ms, ok := p.Line(l); ok {
			frames = append(frames, ms)
		}
	}
	rows--                                           // encabezado
	if want := (rows+1)/2 - 1; len(frames) != want { // el primer cuadro solo marca el inicio
		t.Fatalf("de %d filas deben quedar la mitad (%d), quedaron %d", rows, want, len(frames))
	}
	for _, ms := range frames {
		if ms < 5 {
			t.Fatalf("quedó una fila de espera (%.3f ms)", ms)
		}
	}
}

// A ~300 FPS: filas de espera (0,2 ms, mismo inicio) intercaladas con los cuadros reales. Tienen que
// salir 300 cuadros por segundo, no 600, y en cualquier orden de las filas del par.
func TestParserSteadyHighFPS(t *testing.T) {
	for _, waitFirst := range []bool{true, false} {
		var p Parser
		p.Line("Application,ProcessID,SwapChainAddress,PresentRuntime,SyncInterval,PresentFlags,CPUStartTime,FrameTime,CPUBusy,CPUWait")
		start, n, sum := 1000.0, 0, 0.0
		for i := 0; i < 3000; i++ {
			frame := fmt.Sprintf("cs2.exe,1,0xA,DXGI,0,512,%.4f,3.3333,3.1,0.2333", start)
			wait := fmt.Sprintf("cs2.exe,1,0xA,DXGI,0,512,%.4f,0.2333,0.0000,0.2333", start)
			rowsIn := []string{wait, frame}
			if !waitFirst {
				rowsIn = []string{frame, wait}
			}
			for _, l := range rowsIn {
				if ms, ok := p.Line(l); ok {
					n++
					sum += ms
				}
			}
			start += 3.3333
		}
		if n < 2990 || n > 3000 {
			t.Fatalf("waitFirst=%v: de 3000 cuadros salieron %d", waitFirst, n)
		}
		if fps := float64(n) * 1000 / sum; fps < 295 || fps > 305 {
			t.Fatalf("waitFirst=%v: FPS %.1f, esperaba ~300", waitFirst, fps)
		}
	}
}
