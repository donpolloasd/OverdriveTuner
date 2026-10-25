package extras

import (
	"testing"

	"fpsforge/internal/system"
)

func TestStartupListAndToggle(t *testing.T) {
	f := system.NewFake()
	run := startSources[0].run
	f.Seed(run, "Discord", "REG_SZ", `C:\Discord\Update.exe`)
	f.Seed(run, "Steam", "REG_SZ", `C:\Steam\steam.exe`)
	f.Seed(startSources[0].approved, "Steam", "REG_BINARY", "030000000000000000000000")
	items, err := ListStartup(f)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if !items[0].Enabled || items[0].Name != "Discord" {
		t.Errorf("Discord debe estar habilitado: %+v", items[0])
	}
	if items[1].Enabled {
		t.Errorf("Steam debe figurar deshabilitado: %+v", items[1])
	}
	if err := SetStartup(f, run, "Discord", false); err != nil {
		t.Fatal(err)
	}
	items, _ = ListStartup(f)
	if items[0].Enabled {
		t.Error("Discord debería quedar deshabilitado")
	}
	if err := SetStartup(f, "HKCU\\Otro", "x", true); err == nil {
		t.Error("debe rechazar orígenes desconocidos")
	}
}
