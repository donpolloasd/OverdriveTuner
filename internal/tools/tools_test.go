package tools

import (
	"strings"
	"testing"

	"fpsforge/internal/system"
)

const pingEN = `Pinging 8.8.8.8 with 32 bytes of data:
Reply from 8.8.8.8: bytes=32 time=12ms TTL=118
Reply from 8.8.8.8: bytes=32 time=14ms TTL=118
Reply from 8.8.8.8: bytes=32 time<1ms TTL=118
Ping statistics for 8.8.8.8:
    Packets: Sent = 3, Received = 3, Lost = 0 (0% loss),
Approximate round trip times in milli-seconds:
    Minimum = 12ms, Maximum = 14ms, Average = 13ms`

const pingES = `Haciendo ping a 1.1.1.1 con 32 bytes de datos:
Respuesta desde 1.1.1.1: bytes=32 tiempo=20ms TTL=58
Respuesta desde 1.1.1.1: bytes=32 tiempo=60ms TTL=58
Estadísticas de ping para 1.1.1.1:
    Paquetes: enviados = 15, recibidos = 12, perdidos = 3 (20% perdidos),
Tiempos aproximados de ida y vuelta en milisegundos:
    Mínimo = 20ms, Máximo = 60ms, Media = 40ms`

func TestParsePing(t *testing.T) {
	loss, mn, mx, avg, jit, ok := ParsePing(pingEN)
	if !ok || loss != 0 || mn != 12 || mx != 14 || avg != 13 || jit < 5 {
		t.Fatalf("%v %v %v %v %v %v", loss, mn, mx, avg, jit, ok)
	}
	loss, mn, mx, avg, jit, ok = ParsePing(pingES)
	if !ok || loss != 20 || mn != 20 || mx != 60 || avg != 40 || jit != 40 {
		t.Fatalf("es: %v %v %v %v %v %v", loss, mn, mx, avg, jit, ok)
	}
	if _, _, _, _, _, ok := ParsePing("Request timed out."); ok {
		t.Fatal("sin estadísticas no es válido")
	}
}

func TestNetTestAndJudge(t *testing.T) {
	f := system.NewFake()
	f.ExecHook = func(name string, args []string) (string, int, error) {
		if args[len(args)-1] == "1.1.1.1" {
			return pingES, 0, nil
		}
		return pingEN, 0, nil
	}
	rs := NetTest(f)
	if len(rs) != 3 || rs[0].Status != "ok" || rs[1].Status != "bad" {
		t.Fatalf("%+v", rs)
	}
	f.ExecHook = func(string, []string) (string, int, error) { return "", 1, nil }
	if rs := NetTest(f); rs[0].Status != "error" {
		t.Fatalf("%+v", rs)
	}
}

func TestDisks(t *testing.T) {
	ds, err := parseDisks(`[{"name":"Samsung SSD","media":"SSD","health":"Healthy","op":"OK","size":447},{"name":"HDD","media":"HDD","health":"Warning","op":"OK","size":894}]`)
	if err != nil || len(ds) != 2 || ds[0].Status != "ok" || ds[1].Status != "warn" {
		t.Fatalf("%v %+v", err, ds)
	}
	if ds, _ := parseDisks(`{"name":"x","media":"SSD","health":"Unhealthy","size":10}`); ds[0].Status != "bad" {
		t.Fatalf("%+v", ds)
	}
}

func TestVideoReview(t *testing.T) {
	txt := `"video.cfg"
{
	"Version"		"15"
	"setting.mat_vsync"		"1"
	"setting.msaa_samples"		"8"
	"setting.fullscreen"		"0"
	"setting.refreshrate_numerator"		"144000"
	"setting.refreshrate_denominator"		"1000"
}`
	f := ReviewVideo(ParseVideo(txt), 240)
	if len(f) != 4 || f[0].Status != "bad" || !strings.Contains(f[0].Advice, "240") {
		t.Fatalf("%+v", f)
	}
	for _, x := range f {
		if x.Status == "ok" {
			t.Fatalf("nada de esto está bien: %+v", x)
		}
	}
	ok := ReviewVideo(ParseVideo(`"setting.mat_vsync" "0" "setting.msaa_samples" "2" "setting.refreshrate_numerator" "240000" "setting.refreshrate_denominator" "1000"`), 240)
	for _, x := range ok {
		if x.Status != "ok" {
			t.Fatalf("%+v", x)
		}
	}
	// Sin Steam: mensaje claro, no un error.
	if r := Video(system.NewFake(), 240); r.Found || r.Message == "" || r.Findings == nil {
		t.Fatalf("%+v", r)
	}
}

func TestParseDisksSmartAndTips(t *testing.T) {
	ds, err := parseDisks(`[{"name":"NVMe","media":"SSD","bus":"NVMe","health":"Healthy","size":447,"temp":72,"wear":10,"hours":900,"errors":0,"trim":"off","volumes":[{"letter":"C","freeGB":30,"sizeGB":447}]},
	{"name":"HDD","media":"HDD","bus":"SATA","health":"Healthy","size":894,"temp":40,"wear":-1,"hours":-1,"errors":2,"trim":"off","volumes":[{"letter":"D","freeGB":20,"sizeGB":894}]},
	{"name":"Gastado","media":"SSD","bus":"SATA","health":"Healthy","size":240,"temp":30,"wear":96,"hours":1,"errors":0,"trim":"on"}]`)
	if err != nil || len(ds) != 3 {
		t.Fatal(err, ds)
	}
	if ds[0].Status != "warn" || !strings.Contains(ds[0].Detail, "caliente") {
		t.Fatalf("temperatura SSD: %+v", ds[0])
	}
	if len(ds[0].Tips) < 2 {
		t.Fatalf("faltan consejos de TRIM e instalación: %+v", ds[0].Tips)
	}
	if ds[1].Status != "warn" {
		t.Fatalf("errores o poco espacio deben avisar: %+v", ds[1])
	}
	if ds[2].Status != "bad" || ds[2].Volumes == nil {
		t.Fatalf("desgaste crítico: %+v", ds[2])
	}
}
