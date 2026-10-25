package uninst

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeDataDir(t *testing.T) {
	t.Setenv("ProgramData", `/pd`)
	if !SafeDataDir(filepath.Join("/pd", "FPSForge")) || !SafeDataDir(filepath.Join("/pd", "fpsforge")) {
		t.Fatal("la carpeta de datos esperada debe aceptarse")
	}
	for _, bad := range []string{"", "/pd", "/", filepath.Join("/pd", "FPSForge", "x"), filepath.Join("/pd", "Otro"), "/home/FPSForge"} {
		if SafeDataDir(bad) {
			t.Fatalf("no debe aceptar %q", bad)
		}
	}
	t.Setenv("ProgramData", "")
	if SafeDataDir("/pd/FPSForge") {
		t.Fatal("sin ProgramData no se borra nada")
	}
}

func TestAppliedCount(t *testing.T) {
	d := t.TempDir()
	if AppliedCount(d) != 0 {
		t.Fatal("sin archivo")
	}
	_ = os.WriteFile(filepath.Join(d, "state.json"), []byte(`[{"id":"a"},{"id":"b"}]`), 0o644)
	if AppliedCount(d) != 2 {
		t.Fatal("dos ajustes")
	}
	_ = os.WriteFile(filepath.Join(d, "state.json"), []byte(`basura`), 0o644)
	if AppliedCount(d) != 0 {
		t.Fatal("estado corrupto")
	}
}
