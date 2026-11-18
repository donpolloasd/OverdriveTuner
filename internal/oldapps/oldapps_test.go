package oldapps

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestSteamGamesAndSort(t *testing.T) {
	dir := t.TempDir()
	lib2 := filepath.Join(dir, "lib2")
	_ = os.MkdirAll(filepath.Join(dir, "steamapps"), 0o755)
	_ = os.MkdirAll(filepath.Join(lib2, "steamapps"), 0o755)
	vdf := `"libraryfolders" { "0" { "path" "` + filepath.ToSlash(dir) + `" } "1" { "path" "` + filepath.ToSlash(lib2) + `" } }`
	_ = os.WriteFile(filepath.Join(dir, "steamapps", "libraryfolders.vdf"), []byte(vdf), 0o644)
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	ts := func(d time.Duration) string { return strconv.FormatInt(now.Add(-d).Unix(), 10) }
	_ = os.WriteFile(filepath.Join(dir, "steamapps", "appmanifest_10.acf"), []byte(`"AppState" { "appid" "10" "name" "Counter-Strike" "LastPlayed" "`+ts(200*24*time.Hour)+`" "SizeOnDisk" "318799907" }`), 0o644)
	_ = os.WriteFile(filepath.Join(lib2, "steamapps", "appmanifest_730.acf"), []byte(`"AppState" { "appid" "730" "name" "CS2" "LastPlayed" "`+ts(time.Hour)+`" "SizeOnDisk" "40000000000" }`), 0o644)
	_ = os.WriteFile(filepath.Join(lib2, "steamapps", "appmanifest_228980.acf"), []byte(`"AppState" { "appid" "228980" "name" "Steamworks Common Redistributables" }`), 0o644)
	g := SteamGames(dir, now)
	if len(g) != 2 {
		t.Fatalf("dos juegos (sin los redistribuibles): %+v", g)
	}
	items := append(g, Item{Name: "sin datos", Days: -1, SizeMB: 9999})
	Sort(items)
	if items[0].Name != "Counter-Strike" || items[0].Days != 200 || items[0].SizeMB != 304 || items[2].Days != -1 {
		t.Fatalf("orden: %+v", items)
	}
}

func TestParseProgramsAndUninstallValidation(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.Local)
	out := `[{"key":"HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\X","name":"X","pub":"Y","mb":120,"last":"2026-11-07"},{"key":"HKLM\\SOFTWARE\\Otra\\Cosa","name":"Mala","mb":1,"last":""}]`
	items, err := parsePrograms(out, now)
	if err != nil || len(items) != 1 || items[0].Days != 10 || items[0].ID != `reg:HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\X` {
		t.Fatalf("%v %+v", err, items)
	}
	for _, bad := range []string{`reg:HKLM\SOFTWARE\Otra`, "steam:abc", "otro:1", `reg:HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\a\b`} {
		if Uninstall(nil, bad) == nil {
			t.Errorf("%s debe rechazarse", bad)
		}
	}
}
