package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fpsforge/internal/system"
)

func TestValidateDir(t *testing.T) {
	bad := []string{"", "relativa", `C:\`, `C:\Windows`, `c:\users`, `C:\Program Files`}
	for _, d := range bad {
		if ValidateDir(d) == nil {
			t.Errorf("%q debería rechazarse", d)
		}
	}
}

func TestInstallUninstall(t *testing.T) {
	f := system.NewFake()
	dir := filepath.Join(t.TempDir(), "nueva", "OverdriveTuner")
	payload := make([]byte, 4096)
	var steps []int
	err := Install(f, Options{Dir: dir, Desktop: true, StartMenu: true, Version: "1.0.0", Exe: payload},
		func(p int, _ string) { steps = append(steps, p) })
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(dir, ExeName)); err != nil || st.Size() != 4096 {
		t.Fatalf("no se copió el ejecutable: %v", err)
	}
	if steps[len(steps)-1] != 100 {
		t.Errorf("el progreso debe terminar en 100: %v", steps)
	}
	if got := InstalledDir(f); got != dir {
		t.Errorf("InstalledDir=%q", got)
	}
	v, _ := system.RegGet(f, uninstKey, "UninstallString")
	if !strings.HasSuffix(v.Data, "--uninstall") {
		t.Errorf("UninstallString mal formado: %q", v.Data)
	}
	if len(f.PSLog) != 2 || !strings.Contains(f.PSLog[0], "CommonPrograms") || !strings.Contains(f.PSLog[1], "CommonDesktopDirectory") {
		t.Errorf("accesos directos: %v", f.PSLog)
	}
	if err := Uninstall(f, dir); err != nil {
		t.Fatal(err)
	}
	if InstalledDir(f) != "" {
		t.Error("debe quitar la entrada del registro")
	}
}

func TestInstallRejectsEmptyPayload(t *testing.T) {
	err := Install(system.NewFake(), Options{Dir: filepath.Join(t.TempDir(), "x", "y"), Exe: []byte("hola")}, nil)
	if err == nil {
		t.Fatal("debe rechazar un instalador sin programa")
	}
}

func TestScheduleDeleteAndSamePath(t *testing.T) {
	f := system.NewFake()
	if err := ScheduleDelete(f, `C:\Program Files\OverdriveTuner`); err != nil {
		t.Fatal(err)
	}
	if len(f.PSLog) != 1 || !strings.Contains(f.PSLog[0], "Remove-Item") || !strings.Contains(f.PSLog[0], "OverdriveTuner") {
		t.Fatalf("script inesperado: %v", f.PSLog)
	}
	if !SamePath("/a/b/", "/a/b") {
		t.Error("SamePath debe ignorar la barra final")
	}
}

func TestPatchRequiresInstall(t *testing.T) {
	f := system.NewFake()
	if _, err := Patch(f, make([]byte, 2048), nil, "1.1.0", nil); err != ErrNotInstalled {
		t.Fatalf("sin instalar debe fallar con ErrNotInstalled, fue %v", err)
	}
	dir := t.TempDir() + "/OverdriveTuner"
	if err := Install(f, Options{Dir: dir, Version: "1.0.0", Exe: make([]byte, 2048)}, nil); err != nil {
		t.Fatal(err)
	}
	got, err := Patch(f, append(make([]byte, 2047), 7), nil, "1.1.0", nil)
	if err != nil || got != dir {
		t.Fatalf("%v %q", err, got)
	}
	b, _ := os.ReadFile(dir + "/" + ExeName)
	if len(b) != 2048 || b[2047] != 7 {
		t.Fatal("el ejecutable no se actualizó")
	}
	if v, _ := system.RegGet(f, uninstKey, "DisplayVersion"); v.Data != "1.1.0" {
		t.Fatalf("versión registrada: %q", v.Data)
	}
}

func TestPatchMigratesLegacyFPSForge(t *testing.T) {
	f := system.NewFake()
	t.Setenv("ProgramFiles", t.TempDir())
	old := t.TempDir() + "/FPS Forge"
	f.RegSet(legacyKey, "InstallLocation", "REG_SZ", old)
	f.RegSet(legacyKey, "DisplayName", "REG_SZ", "FPS Forge")
	if _, err := Patch(f, make([]byte, 2048), nil, "1.3.0", nil); err != nil {
		t.Fatal(err)
	}
	if f.RegKeyExists(legacyKey) {
		t.Fatal("la instalación vieja debe quitarse del registro")
	}
	if !f.RegKeyExists(uninstKey) {
		t.Fatal("falta la instalación nueva")
	}
	found := false
	for _, s := range f.PSLog {
		if strings.Contains(s, "FPS Forge.lnk") {
			found = true
		}
	}
	if !found {
		t.Fatal("deben borrarse los accesos directos viejos")
	}
}
