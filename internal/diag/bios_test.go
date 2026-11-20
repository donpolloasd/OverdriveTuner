package diag

import (
	"strings"
	"testing"
	"time"

	"fpsforge/internal/scan"
	"fpsforge/internal/system"
)

func TestBIOSAge(t *testing.T) {
	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	if _, ok := BIOSAge(scan.Hardware{}, now); ok {
		t.Fatal("sin fecha no hay chequeo")
	}
	c, ok := BIOSAge(scan.Hardware{Board: "Gigabyte Technology Co., Ltd. A520M K V2", BIOSVer: "F7c", BIOSDate: "2026-08-17"}, now)
	if !ok || c.Status != OK || c.Action != "open_bios" {
		t.Fatalf("BIOS reciente: %+v", c)
	}
	c, _ = BIOSAge(scan.Hardware{Board: "ASRock A520M-K V2", BIOSVer: "P1.60", BIOSDate: "2023-04-12"}, now)
	if c.Status != Warn || !strings.Contains(c.Detail, "hace 43 meses") {
		t.Fatalf("BIOS vieja: %+v", c)
	}
	if c, _ := BIOSAge(scan.Hardware{Board: "Marca Rara X1", BIOSDate: "2020-01-01"}, now); c.Action != "" {
		t.Fatal("sin página conocida no hay botón")
	}
	f := system.NewFake()
	if OpenBIOSPage(f, "Marca Rara") == nil {
		t.Fatal("solo páginas fijas")
	}
	_ = OpenBIOSPage(f, "Micro-Star International Co., Ltd. B550")
	if !strings.Contains(strings.Join(f.ExecLog, " "), "https://www.msi.com/support/download") {
		t.Fatalf("MSI: %v", f.ExecLog)
	}
}

func TestFastIncludesBIOS(t *testing.T) {
	cs := Fast(system.NewFake(), scan.Demo(), 240)
	for _, c := range cs {
		if c.ID == "bios" {
			return
		}
	}
	t.Fatal("el Diagnóstico incluye la BIOS")
}
