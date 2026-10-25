package telemetry

import (
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	line := `{"cpu":41.5,"max":88,"mhz":4350,"freeKb":20971520,"totalKb":33554432,"gpu":[{"luid":"0x00000000_0x0000A1","u":5,"v":104857600},{"luid":"0x00000000_0x0000B2","u":71.2,"v":2147483648}],"nv":null}`
	s, err := ParseLine(line, map[string]float64{"0x00000000_0x0000B2": 54.5})
	if err != nil {
		t.Fatal(err)
	}
	if s.GPU != 71.2 || s.GPULuid != "0x00000000_0x0000B2" || s.GPUTemp != 54.5 {
		t.Fatalf("%+v", s)
	}
	if s.RAMTotMB != 32768 || s.RAMUsedMB != 12288 || int(s.RAMPct()) != 37 {
		t.Fatalf("RAM: %+v", s)
	}
	if s.CPUTemp != Unknown {
		t.Fatal("sin sensor la temperatura de CPU es desconocida")
	}
	if _, err := ParseLine("basura", nil); err == nil {
		t.Fatal("debe fallar")
	}
	if s, _ := ParseLine(`{"cpu":150,"max":-4}`, nil); s.CPU != 100 || s.CPUThread != 0 || s.GPUTemp != Unknown {
		t.Fatalf("límites: %+v", s)
	}
}

func TestTempSources(t *testing.T) {
	s, _ := ParseLine(`{"gpu":[{"luid":"a","u":10,"v":1}],"nv":61,"lhmCpu":58.5}`, map[string]float64{"a": 40})
	if s.GPUTemp != 61 || s.CPUTemp != 58.5 {
		t.Fatalf("%+v", s)
	}
	s, _ = ParseLine(`{"gpu":[{"luid":"a","u":10,"v":1}],"lhmGpu":47}`, nil)
	if s.GPUTemp != 47 {
		t.Fatalf("%+v", s)
	}
	s, _ = ParseLine(`{"gpu":[{"luid":"a","u":10,"v":1}]}`, map[string]float64{"a": 300})
	if s.GPUTemp != Unknown {
		t.Fatal("temperatura absurda debe descartarse")
	}
}

func mk(cpu, thr, gpu float64) []Sample {
	var o []Sample
	for i := 0; i < 10; i++ {
		o = append(o, Sample{CPU: cpu, CPUThread: thr, GPU: gpu})
	}
	return o
}

func TestBottleneck(t *testing.T) {
	cases := []struct {
		xs   []Sample
		kind string
	}{
		{mk(40, 60, 97), "gpu"},
		{mk(50, 92, 70), "cpu_thread"}, // el caso del usuario: CPU 50 %, GPU 70 %
		{mk(95, 70, 60), "cpu"},
		{mk(30, 50, 50), "other"},
		{mk(30, 50, 50)[:3], "none"},
	}
	for _, c := range cases {
		if v := Bottleneck(c.xs); v.Kind != c.kind {
			t.Errorf("esperaba %s, fue %+v", c.kind, v)
		}
	}
	if v := Bottleneck(mk(50, 92, 70)); !strings.Contains(v.Text, "un solo hilo") {
		t.Fatal(v.Text)
	}
}
