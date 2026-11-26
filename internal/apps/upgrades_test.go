package apps

import (
	"testing"

	"fpsforge/internal/system"
)

const wingetTable = "   - \r   \\ \r                \rNombre            Id                    Versión   Disponible Origen\r\n" +
	"--------------------------------------------------------------------\r\n" +
	"7-Zip 26.05 (x64) 7zip.7zip             26.05     26.06      winget\r\n" +
	"原神              miHoYo.GenshinImpact  1.0       2.0        winget\r\n" +
	"Cortado           Muy.Largo…            1.0       2.0        winget\r\n" +
	"Vis C++           Microsoft.VCRedist.2015+.x64 1 2 winget\r\n" +
	"3 actualizaciones disponibles.\r\n"

func TestParseUpgrades(t *testing.T) {
	l := parseUpgrades(wingetTable)
	if len(l) < 2 || l[0] != (Upgrade{"7-Zip 26.05 (x64)", "7zip.7zip", "26.05", "26.06"}) || l[1].ID != "miHoYo.GenshinImpact" || l[1].To != "2.0" {
		t.Fatalf("%+v", l)
	}
	for _, u := range l {
		if !ValidPackage(u.ID) || u.ID == "Muy.Largo…" {
			t.Fatalf("%+v", u)
		}
	}
	if len(parseUpgrades("No se encontró ningún paquete")) != 0 {
		t.Fatal("sin tabla no hay filas")
	}
}

func TestUpgradeOneValidates(t *testing.T) {
	f := system.NewFake()
	if UpgradeOne(f, "--all").OK || UpgradeOne(f, "a b").OK || len(f.ExecLog) != 0 {
		t.Fatal("solo identificadores válidos")
	}
	if !UpgradeOne(f, "Notepad++.Notepad++").OK {
		t.Fatal(f.ExecLog)
	}
	f.ExecHook = func(string, []string) (string, int, error) { return "error", 5, nil }
	if UpgradeOne(f, "a.b").OK {
		t.Fatal("un código de error es un fallo")
	}
}
