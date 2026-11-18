package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fpsforge/internal/diag"
)

func TestReapplyRestoresUndoneTweak(t *testing.T) {
	s, base, f := newTestServer(t)
	if c, b := call(t, s, base, "/api/apply", `{"id":"fps.gamemode"}`, nil); c != 200 {
		t.Fatal(c, b)
	}
	// Windows deshace el ajuste: el valor vuelve a 0.
	f.Seed(`HKCU\Software\Microsoft\GameBar`, "AutoGameModeEnabled", "REG_DWORD", "0")
	drift := func() bool {
		_, b := call(t, s, base, "/api/state", "", nil)
		var st struct {
			Tweaks []struct {
				ID    string `json:"id"`
				Drift bool   `json:"drift"`
			} `json:"tweaks"`
		}
		_ = json.Unmarshal([]byte(b), &st)
		for _, x := range st.Tweaks {
			if x.ID == "fps.gamemode" {
				return x.Drift
			}
		}
		return false
	}
	if !drift() {
		t.Fatal("el ajuste deshecho por Windows debe marcarse")
	}
	if c, b := call(t, s, base, "/api/reapply", `{"ids":["fps.gamemode"]}`, nil); c != 200 || !strings.Contains(b, `"ok":true`) {
		t.Fatal(c, b)
	}
	if drift() {
		t.Fatal("después de volver a aplicarlo ya no está deshecho")
	}
	if c, b := call(t, s, base, "/api/revert", `{"id":"fps.gamemode"}`, nil); c != 200 {
		t.Fatal(c, b)
	}
}

func TestWinUpdateInfo(t *testing.T) {
	now := time.Now()
	if winUpdateInfo(settings{LastBuild: 26200}, now) != nil {
		t.Fatal("sin compilación anterior no hay actualización")
	}
	if winUpdateInfo(settings{LastBuild: 26200, PrevBuild: 26100, BuildAt: now.Add(-40 * 24 * time.Hour)}, now) != nil {
		t.Fatal("una actualización de hace más de un mes ya no se muestra")
	}
	if w := winUpdateInfo(settings{LastBuild: 26200, PrevBuild: 26100, BuildAt: now}, now); w == nil || w["from"] != 26100 {
		t.Fatalf("actualización reciente: %v", w)
	}
}

func TestWhatsNew(t *testing.T) {
	s, base, _ := newTestServer(t)
	// Instalación nueva: no muestra nada y queda anotada la versión.
	if _, b := call(t, s, base, "/api/whatsnew", "", nil); strings.Contains(b, `"t"`) {
		t.Fatalf("instalación nueva: %s", b)
	}
	_ = updateSettings(s.Dir, func(st *settings) { st.SeenVersion, st.TermsOK = "3.0.0", true })
	_, b := call(t, s, base, "/api/whatsnew?lang=en", "", nil)
	if !strings.Contains(b, `"version":"`+Version+`"`) || !strings.Contains(b, "First-run assistant") {
		t.Fatalf("novedades en inglés: %s", b)
	}
	call(t, s, base, "/api/whatsnew", "{}", nil)
	if _, b := call(t, s, base, "/api/whatsnew", "", nil); strings.Contains(b, `"t"`) {
		t.Fatalf("ya vista: %s", b)
	}
}

func TestIssueURLAndDraft(t *testing.T) {
	s, base, _ := newTestServer(t)
	_, b := call(t, s, base, "/api/issue?lang=es", "", nil)
	if !strings.Contains(b, "OverdriveTuner "+Version) || strings.Contains(b, "PC-Demo") {
		t.Fatalf("el borrador lleva la versión y no el nombre del equipo: %s", b)
	}
	if c, _ := call(t, s, base, "/api/issue", `{"title":"","body":"x"}`, nil); c != 400 {
		t.Fatal("sin título debe fallar")
	}
	u := issueURL("Título & más", strings.Repeat("x", 9000))
	if !strings.HasPrefix(u, issuesURL+"?title=T%C3%ADtulo+%26+m%C3%A1s&body=") || len(u) > 16000 {
		t.Fatalf("url: %.120s (%d)", u, len(u))
	}
}

func TestReminders(t *testing.T) {
	s, _, _ := newTestServer(t)
	now := time.Now()
	if r := s.dueReminders(now, nil); len(r) != 0 {
		t.Fatal("el primer día no hay recordatorios")
	}
	_ = updateSettings(s.Dir, func(st *settings) { st.LastClean = now.Add(-40 * 24 * time.Hour) })
	drv := []diag.Check{{ID: "driver_amd", Status: diag.Warn}}
	if r := s.dueReminders(now, drv); len(r) != 2 {
		t.Fatalf("limpieza y driver: %+v", r)
	}
	if r := s.dueReminders(now.Add(time.Hour), drv); len(r) != 0 {
		t.Fatal("no se repiten enseguida")
	}
	_ = updateSettings(s.Dir, func(st *settings) { st.RemindOff = true })
	if r := s.dueReminders(now.Add(60*24*time.Hour), drv); len(r) != 0 {
		t.Fatal("desactivados no avisan")
	}
}

func TestDemoToolsAndDriverConfirm(t *testing.T) {
	s, base, _ := newTestServer(t)
	for _, p := range []string{"/api/oldapps", "/api/drivers/old", "/api/dpc"} {
		if c, b := call(t, s, base, p, "", nil); c != 200 || len(b) < 10 {
			t.Fatalf("%s: %d %s", p, c, b)
		}
	}
	if c, _ := call(t, s, base, "/api/drivers/delete", `{"names":["oem41.inf"]}`, nil); c != 428 {
		t.Fatalf("borrar drivers sin confirmar debe ser 428, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/opensettings", `{"page":"https://malo.example"}`, nil); c != 400 {
		t.Fatal("solo páginas fijas")
	}
	if c, _ := call(t, s, base, "/api/settings", `{"fresh":{"rm -rf":true}}`, nil); c != 400 {
		t.Fatal("pasos desconocidos de «Después de formatear» no se aceptan")
	}
}

func TestMigrateDataDir(t *testing.T) {
	pd := t.TempDir()
	t.Setenv("ProgramData", pd)
	old := filepath.Join(pd, "FPSForge")
	_ = os.MkdirAll(filepath.Join(old, "appx"), 0o755)
	_ = os.WriteFile(filepath.Join(old, "state.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(old, "appx", "a.json"), []byte("[]"), 0o644)
	if DefaultDataDir() != old || !NeedsMigration() {
		t.Fatal("antes de mover se sigue usando la carpeta vieja")
	}
	dir, err := MigrateDataDir()
	if err != nil || dir != filepath.Join(pd, "OverdriveTuner") || NeedsMigration() || DefaultDataDir() != dir {
		t.Fatalf("migración: %s %v", dir, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "appx", "a.json")); err != nil {
		t.Fatal("se movió todo el contenido")
	}
	// Si las dos existen, se pasa lo que falta sin pisar nada.
	_ = os.MkdirAll(old, 0o755)
	_ = os.WriteFile(filepath.Join(old, "state.json"), []byte("viejo"), 0o644)
	_ = os.WriteFile(filepath.Join(old, "dns-backup.json"), []byte("[]"), 0o644)
	if _, err := MigrateDataDir(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "state.json")); string(b) != "{}" {
		t.Fatal("no se pisa lo que ya estaba en la carpeta nueva")
	}
	if _, err := os.Stat(filepath.Join(dir, "dns-backup.json")); err != nil {
		t.Fatal("lo que faltaba se pasa a la carpeta nueva")
	}
}
