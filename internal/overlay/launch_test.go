package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fpsforge/internal/system"
)

func TestValidLaunch(t *testing.T) {
	for _, ok := range []string{"", "steam://rungameid/730", `C:\Juegos\X\x.exe`, "com.epicgames.launcher://apps/Fortnite?action=launch&silent=true", "battlenet://Pro"} {
		if !ValidLaunch(ok) {
			t.Errorf("debía aceptar %q", ok)
		}
	}
	for _, bad := range []string{"cmd.exe", `C:\x\a.bat`, `C:\x\a.exe" /c calc`, `\\srv\x.exe`, "steam://run/730//-x", "http://x.com/a.exe", "file:///C:/a.exe"} {
		if ValidLaunch(bad) {
			t.Errorf("debía rechazar %q", bad)
		}
	}
}

func TestLaunchAndSteamLookup(t *testing.T) {
	f := system.NewFake()
	if Launch(f, "") == nil || Launch(f, `C:\no\existe.exe`) == nil || len(f.ExecLog) != 0 {
		t.Fatal("sin ruta válida no se abre nada")
	}
	if err := Launch(f, "steam://rungameid/730"); err != nil || !strings.Contains(f.ExecLog[0], "explorer.exe steam://rungameid/730") {
		t.Fatal(err, f.ExecLog)
	}
	root := t.TempDir()
	lib := filepath.Join(root, "steamapps")
	_ = os.MkdirAll(filepath.Join(lib, "common", "Counter-Strike Global Offensive", "game", "bin", "win64"), 0o755)
	_ = os.WriteFile(filepath.Join(lib, "common", "Counter-Strike Global Offensive", "game", "bin", "win64", "cs2.exe"), nil, 0o644)
	_ = os.WriteFile(filepath.Join(lib, "appmanifest_730.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\t\"730\"\n\t\"installdir\"\t\t\"Counter-Strike Global Offensive\"\n}"), 0o644)
	if id := steamAppID(root, "Counter-Strike Global Offensive"); id != "730" {
		t.Fatal(id)
	}
	if p := findIn(filepath.Join(lib, "common"), "CS2.EXE", 6); !strings.HasSuffix(p, "cs2.exe") {
		t.Fatal(p)
	}
	if p := findIn(filepath.Join(lib, "common"), "cs2.exe", 2); p != "" {
		t.Fatal("respeta la profundidad", p)
	}
	_ = os.WriteFile(filepath.Join(lib, "libraryfolders.vdf"), []byte(`"libraryfolders" { "0" { "path" "`+strings.ReplaceAll(root, `\`, `\\`)+`" } "1" { "path" "D:\\SteamLibrary" } }`), 0o644)
	if l := steamLibraries(root); len(l) != 2 || l[1] != `D:\SteamLibrary` {
		t.Fatal(l)
	}
	t.Log("cs2 en esta PC:", FindLaunch("cs2.exe", nil))
}
