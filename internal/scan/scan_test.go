package scan

import "testing"

func TestParseAndConditions(t *testing.T) {
	j := `{"computer":"PC-Demo","cpu":"AMD Ryzen 5 5600G","cpuVendor":"amd","cores":6,"threads":12,"maxMHz":3900,"ramGB":32,"ramMTs":3200,"ramSticks":2,"board":"ASRock A520M-K V2","os":"Microsoft Windows 11 Home","build":26100,"laptop":false,"gpus":[{"name":"AMD Radeon RX 7600","vendor":"amd","vramMB":8192,"driver":"32.0","refresh":144}],"disks":[{"model":"SSD","type":"SSD","sizeGB":447,"system":true},{"model":"HDD","type":"HDD","sizeGB":894,"system":false}]}`
	h, err := Parse("WARNING: x\n" + j)
	if err != nil {
		t.Fatal(err)
	}
	for c, want := range map[string]bool{"ssd": true, "ram16": true, "desktop": true, "laptop": false,
		"amd": true, "nvidia": false, "win11": true, "": true} {
		if h.Has(c) != want {
			t.Errorf("Has(%q) = %v, esperaba %v", c, !want, want)
		}
	}
	h.Disks[0].Type = "HDD"
	if h.Has("ssd") {
		t.Error("no debe asumir SSD si el disco del sistema es HDD")
	}
	h.Disks = nil
	if h.Has("ssd") {
		t.Error("sin datos de disco no se asume SSD")
	}
}
