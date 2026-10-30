package diag

import (
	"strings"
	"testing"
	"time"

	"fpsforge/internal/scan"
	"fpsforge/internal/system"
)

func find(cs []Check, id string) *Check {
	for i := range cs {
		if cs[i].ID == id {
			return &cs[i]
		}
	}
	return nil
}

func hw() scan.Hardware {
	return scan.Hardware{RAMMTs: 3200, RAMRated: 3200, RAMSticks: 2, Monitors: 1, GPUs: []scan.GPU{
		{Name: "AMD Radeon RX 7600", Vendor: "amd", Refresh: 240, Active: true},
		{Name: "AMD Radeon(TM) Graphics", Vendor: "amd"},
	}}
}

func TestIntegratedDetection(t *testing.T) {
	for name, want := range map[string]bool{
		"AMD Radeon RX 7600": false, "AMD Radeon(TM) Graphics": true, "Intel(R) UHD Graphics 630": true,
		"NVIDIA GeForce RTX 4060": false, "AMD Radeon Vega 8 Graphics": true, "Intel(R) Arc(TM) A770 Graphics": false,
	} {
		if got := (scan.GPU{Name: name}).Integrated(); got != want {
			t.Errorf("%q integrada=%v, esperaba %v", name, got, want)
		}
	}
}

func TestMonitorOnDiscreteIsOK(t *testing.T) {
	cs := Run(system.NewFake(), hw(), 240)
	if c := find(cs, "display_gpu"); c == nil || c.Status != OK {
		t.Fatalf("%+v", c)
	}
	if c := find(cs, "refresh"); c == nil || c.Status != OK {
		t.Fatalf("%+v", c)
	}
}

func TestMonitorOnIntegratedIsBad(t *testing.T) {
	h := hw()
	h.GPUs[0].Active, h.GPUs[1].Active, h.GPUs[1].Refresh = false, true, 144
	cs := Run(system.NewFake(), h, 240)
	c := find(cs, "display_gpu")
	if c == nil || c.Status != Bad || !strings.Contains(c.Detail, "integrada") {
		t.Fatalf("%+v", c)
	}
	if r := find(cs, "refresh"); r == nil || r.Status != Bad || r.Action != "open_display" {
		t.Fatalf("144 Hz con monitor de 240 debe avisar: %+v", r)
	}
}

func TestOnlyIntegratedGPUIsNotAWarning(t *testing.T) {
	h := scan.Hardware{GPUs: []scan.GPU{{Name: "AMD Radeon(TM) Graphics", Active: true, Refresh: 60}}}
	if c := find(Run(system.NewFake(), h, 60), "display_gpu"); c.Status != OK {
		t.Fatalf("sin gráfica dedicada no hay nada que mover: %+v", c)
	}
}

func TestRAMChecks(t *testing.T) {
	h := hw()
	h.RAMMTs = 2133
	h.RAMSticks = 1
	cs := Run(system.NewFake(), h, 240)
	if c := find(cs, "ram_xmp"); c.Status != Bad || !strings.Contains(c.Detail, "DOCP") {
		t.Fatalf("%+v", c)
	}
	if c := find(cs, "ram_channels"); c.Status != Warn {
		t.Fatalf("%+v", c)
	}
}

func TestPowerPlanAndRegistryChecks(t *testing.T) {
	f := system.NewFake()
	f.ExecHook = func(name string, args []string) (string, int, error) {
		return "GUID del esquema de energía: a1841308-3541-4fab-bc81-f71556f20b4a  (Economizador)\n", 0, nil
	}
	f.Seed(`HKCU\System\GameConfigStore`, "GameDVR_Enabled", "REG_DWORD", "1")
	f.Seed(`HKCU\Software\Microsoft\GameBar`, "AutoGameModeEnabled", "REG_DWORD", "0")
	f.Seed(`HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, "HwSchMode", "REG_DWORD", "2")
	cs := Run(f, hw(), 240)
	if c := find(cs, "power"); c.Status != Bad || !strings.Contains(c.Detail, "Economizador") {
		t.Fatalf("%+v", c)
	}
	for _, id := range []string{"dvr", "gamemode"} {
		if c := find(cs, id); c == nil || c.Status != Warn {
			t.Errorf("%s debe avisar: %+v", id, c)
		}
	}
	if c := find(cs, "hags"); c == nil || !strings.Contains(c.Detail, "activada") {
		t.Fatalf("%+v", c)
	}
	bad, warn := Summary(cs)
	if bad != 1 || warn != 2 {
		t.Errorf("resumen: %d críticos, %d avisos", bad, warn)
	}
}

func TestAntiCheatCheck(t *testing.T) {
	f := system.NewFake()
	f.PSHook = func(string) (string, error) { return `{"secureboot":true,"tpm":true,"tpmready":true,"hvci":true}`, nil }
	if c := find(Run(f, hw(), 240), "anticheat"); c == nil || c.Status != OK {
		t.Fatalf("%+v", c)
	}
	f.PSHook = func(string) (string, error) {
		return `{"secureboot":false,"tpm":true,"tpmready":true,"hvci":false}`, nil
	}
	c := find(Run(f, hw(), 240), "anticheat")
	if c == nil || c.Status != Warn || !strings.Contains(c.Detail, "Secure Boot") || !strings.Contains(c.Detail, "HVCI") {
		t.Fatalf("%+v", c)
	}
}

func TestAntiCheatIOMMUAndReboot(t *testing.T) {
	f := system.NewFake()
	f.PSHook = func(string) (string, error) {
		return `{"secureboot":true,"tpm":true,"tpmready":true,"hvci":true,"dma":false,"reboot":true}`, nil
	}
	c := find(Run(f, hw(), 240), "anticheat")
	if c == nil || c.Status != Warn || !strings.Contains(c.Detail, "IOMMU") || !strings.Contains(c.Detail, "pendientes") {
		t.Fatalf("%+v", c)
	}
}

func TestParseHealth(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	raw := `{"bar1MiB":256,"gpus":[{"name":"NVIDIA GeForce RTX 3060","version":"31.0.15.3623","date":"2025-01-10"}],
	"disks":[{"name":"Samsung SSD","health":"Healthy","wear":85,"temp":40},{"name":"WD HDD","health":"Warning","wear":null,"temp":null}],
	"sysFreeGB":30.2,"sysFreePct":8}`
	cs := ParseHealth(raw, scan.Hardware{}, now)
	by := map[string][]Check{}
	for _, c := range cs {
		by[c.ID] = append(by[c.ID], c)
	}
	if by["rebar"][0].Status != Warn {
		t.Errorf("BAR1 de 256 MiB = ReBAR desactivado: %+v", by["rebar"])
	}
	if d := by["driver_nvidia"]; len(d) != 1 || d[0].Status != Warn || d[0].Action != "open_driver_nvidia" {
		t.Errorf("driver de hace 21 meses debe avisar: %+v", d)
	}
	if ds := by["disk_health"]; len(ds) != 2 || ds[0].Status != Warn || ds[1].Status != Bad {
		t.Errorf("discos: %+v", ds)
	}
	if by["disk_space"][0].Status != Bad {
		t.Errorf("8%% libre es problema: %+v", by["disk_space"])
	}
	ok := ParseHealth(`{"bar1MiB":16384,"gpus":[{"name":"AMD Radeon RX 7600","version":"32.0","date":"2026-08-01"}],"disks":[],"sysFreePct":50,"sysFreeGB":200}`, scan.Hardware{}, now)
	for _, c := range ok {
		if c.Status != OK {
			t.Errorf("todo debía estar bien: %+v", c)
		}
	}
	if ParseHealth("basura", scan.Hardware{}, now) != nil {
		t.Error("salida inválida no debe dar chequeos")
	}
	if monthsSince("1999-01-01", now) != -1 || monthsSince("2026-09-30", now) != 1 {
		t.Error("monthsSince")
	}
	if c := BIOSGuide(scan.Hardware{CPUVendor: "amd"}); !strings.Contains(c.Detail, "EXPO") || !strings.Contains(c.Detail, "SVM") {
		t.Errorf("guía BIOS AMD: %s", c.Detail)
	}
}
