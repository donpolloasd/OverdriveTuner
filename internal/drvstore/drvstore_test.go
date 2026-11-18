package drvstore

import (
	"strings"
	"testing"

	"fpsforge/internal/system"
)

const drvXML = `<?xml version="1.0" encoding="utf-8"?><PnpUtil>
<Driver DriverName="oem1.inf"><OriginalName>rt640x64.inf</OriginalName><ProviderName>Realtek</ProviderName><ClassName>Net</ClassName><DriverVersion>10/28/2019 10.37.1028.2019</DriverVersion></Driver>
<Driver DriverName="oem2.inf"><OriginalName>rt640x64.inf</OriginalName><ProviderName>Realtek</ProviderName><ClassName>Net</ClassName><DriverVersion>11/11/2021 10.54.1111.2021</DriverVersion></Driver>
<Driver DriverName="oem3.inf"><OriginalName>amd.inf</OriginalName><ProviderName>AMD</ProviderName><ClassName>Display</ClassName><DriverVersion>01/01/2024 1.0.0.1</DriverVersion></Driver>
<Driver DriverName="oem4.inf"><OriginalName>amd.inf</OriginalName><ProviderName>AMD</ProviderName><ClassName>Display</ClassName><DriverVersion>01/01/2025 1.0.0.2</DriverVersion></Driver>
<Driver DriverName="oem5.inf"><OriginalName>solo.inf</OriginalName><ProviderName>X</ProviderName><ClassName>System</ClassName><DriverVersion>01/01/2020 1.0</DriverVersion></Driver>
</PnpUtil>`

// oem3 es viejo, pero lo usa un dispositivo desconectado. El acento viene en la página de códigos de la
// consola (byte 0xE1), como lo escribe pnputil de verdad.
const devXML = "<?xml version=\"1.0\" encoding=\"utf-8\"?><PnpUtil><Device><ManufacturerName>(Dispositivos de sistema est\xe1ndar)</ManufacturerName><Status>Disconnected</Status><DriverName>oem3.inf</DriverName></Device><Device><DriverName>oem2.inf</DriverName></Device></PnpUtil>"

func TestOldOnlyUnusedOlderVersions(t *testing.T) {
	l, err := oldFrom(drvXML, devXML)
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 1 || l[0].Name != "oem1.inf" || l[0].Date != "2019-10-28" || l[0].Newer != "10.54.1111.2021" {
		t.Fatalf("solo el Realtek viejo (el AMD viejo está en uso): %+v", l)
	}
}

func TestDeleteNeverForces(t *testing.T) {
	f := system.NewFake()
	if Delete(f, "oem1.inf; rm") == nil || Delete(f, `..\x.inf`) == nil {
		t.Fatal("solo nombres oemN.inf")
	}
	_ = Delete(f, "OEM12.inf")
	log := strings.Join(f.ExecLog, "\n")
	if !strings.Contains(log, "/delete-driver oem12.inf") || strings.Contains(log, "/force") {
		t.Fatalf("pnputil sin /force: %s", log)
	}
}

func TestUnsupportedWithoutXML(t *testing.T) {
	f := system.NewFake()
	f.ExecHook = func(string, []string) (string, int, error) { return "PNPUTIL [/add-driver ...]", 0, nil }
	if _, err := Old(f); err != ErrUnsupported {
		t.Fatalf("Windows sin /format xml: %v", err)
	}
}
