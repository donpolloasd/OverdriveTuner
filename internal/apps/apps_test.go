package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogIntegrity(t *testing.T) {
	cats := map[string]bool{}
	for _, c := range Categories() {
		cats[c.ID] = true
	}
	ids := map[string]bool{}
	for _, it := range Catalog() {
		if ids[it.ID] || it.ID == "" {
			t.Errorf("id repetido o vacío: %q", it.ID)
		}
		ids[it.ID] = true
		if !cats[it.Cat] || it.Name == "" || it.Desc == "" || (len(it.IDs) == 0 && it.URL == "") {
			t.Errorf("%s incompleto", it.ID)
		}
		if !strings.HasPrefix(it.Site, "https://") {
			t.Errorf("%s: falta la página oficial de descarga", it.ID)
		}
		for _, w := range it.IDs {
			if strings.ContainsAny(w, " \t\"&|<>^%") || !strings.Contains(w, ".") {
				t.Errorf("%s: identificador de winget sospechoso %q", it.ID, w)
			}
		}
	}
	for _, v := range []string{"nvidia", "amd", "intel"} {
		if !strings.HasPrefix(DriverPage[v], "https://") {
			t.Errorf("falta la página de drivers de %s", v)
		}
	}
}

func TestParseExportAndInstalled(t *testing.T) {
	have, err := parseExport([]byte("\ufeff" + `{"Sources":[{"Packages":[{"PackageIdentifier":"Valve.Steam"},{"PackageIdentifier":"Guru3D.RTSS"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	steam, _ := ByID("steam")
	if !IsInstalled(steam, have) {
		t.Fatal("Steam debería figurar instalado")
	}
	ab, _ := ByID("afterburner") // necesita Afterburner y RTSS
	if IsInstalled(ab, have) {
		t.Fatal("con solo RTSS, Afterburner no está instalado")
	}
}

func TestExitCodeNormalization(t *testing.T) {
	// En Windows de 64 bits Go devuelve el código sin signo: 0x8A150061 llega como 2316632161.
	if norm(2316632161) != exitAlreadyInstalled || norm(2316632107) != exitNoUpgrade || norm(0) != 0 {
		t.Fatal("los códigos de winget no se normalizan bien")
	}
}

func TestCoreTempConfig(t *testing.T) {
	pf := t.TempDir()
	t.Setenv("ProgramFiles", pf)
	t.Setenv("ProgramFiles(x86)", "")
	if coreTempDir() != "" {
		t.Fatal("sin Core Temp no debe haber carpeta")
	}
	d := filepath.Join(pf, "Core Temp")
	_ = os.MkdirAll(d, 0o755)
	_ = os.WriteFile(filepath.Join(d, "Core Temp.exe"), []byte("x"), 0o644)
	if err := writeCoreTempConfig(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(d, "CoreTemp.ini"))
	if !strings.Contains(string(b), "Minimized=1;") {
		t.Error("falta la configuración")
	}
	_ = os.WriteFile(filepath.Join(d, "CoreTemp.ini"), []byte("propio"), 0o644)
	_ = writeCoreTempConfig()
	if b, _ := os.ReadFile(filepath.Join(d, "CoreTemp.ini")); string(b) != "propio" {
		t.Error("no debe pisar una configuración existente")
	}
}
