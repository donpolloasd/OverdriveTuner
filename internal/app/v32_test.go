package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotsSaveRestorePreviewAndPrune(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, b := call(t, s, base, "/api/apply", `{"id":"fps.gamemode"}`, nil); c != 200 {
		t.Fatal(c, b)
	}
	c, b := call(t, s, base, "/api/snapshots", `{"name":"Antes de probar Nagle"}`, nil)
	if c != 200 || !strings.Contains(b, "antes-de-probar-nagle.json") {
		t.Fatalf("guardar: %d %s", c, b)
	}
	var saved struct{ ID string }
	_ = json.Unmarshal([]byte(b), &saved)
	// Después se aplica otro ajuste: restaurar la copia ofrece desactivarlo.
	call(t, s, base, "/api/apply", `{"id":"fps.gamedvr"}`, nil)
	_, raw := call(t, s, base, "/api/snapshots/get?id="+saved.ID, "", nil)
	var bundle configBundle
	if err := json.Unmarshal([]byte(raw), &bundle); err != nil || bundle.Name != "Antes de probar Nagle" || len(bundle.Tweaks) != 1 {
		t.Fatalf("copia: %v %+v", err, bundle)
	}
	body, _ := json.Marshal(map[string]any{"bundle": bundle, "apply": false})
	_, pl := call(t, s, base, "/api/config/import", string(body), nil)
	var plan importPlan
	_ = json.Unmarshal([]byte(pl), &plan)
	if len(plan.ToRevert) != 1 || plan.ToRevert[0].ID != "fps.gamedvr" || len(plan.Already) != 1 {
		t.Fatalf("vista previa: %+v", plan)
	}
	// Ids raros no se leen ni se borran.
	if c, _ := call(t, s, base, "/api/snapshots/get?id=..%5Csettings.json", "", nil); c != 404 {
		t.Fatal("solo archivos de la carpeta de copias")
	}
	if c, _ := call(t, s, base, "/api/snapshots/delete", `{"id":"../settings.json"}`, nil); c != 400 {
		t.Fatal("no se borra fuera de la carpeta")
	}
	// Las automáticas se limitan; la del usuario se conserva.
	for i := 0; i < maxAutoSnapshots+3; i++ {
		if _, err := s.saveSnapshot("Auto "+string(rune('a'+i)), true); err != nil {
			t.Fatal(err)
		}
		time.Sleep(1010 * time.Millisecond / 50) // nombres distintos aunque caigan en el mismo segundo
	}
	autos, mine := 0, 0
	for _, it := range s.listSnapshots() {
		if it.Auto {
			autos++
		} else {
			mine++
		}
	}
	if autos > maxAutoSnapshots || mine != 1 {
		t.Fatalf("automáticas %d, del usuario %d", autos, mine)
	}
	if c, _ := call(t, s, base, "/api/snapshots/delete", `{"id":"`+saved.ID+`"}`, nil); c != 200 {
		t.Fatal("borrar la copia del usuario")
	}
}

func TestUIScaleAndRestartReminder(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, _ := call(t, s, base, "/api/settings", `{"uiScale":300}`, nil); c != 400 {
		t.Fatal("tamaño fuera de rango")
	}
	if c, b := call(t, s, base, "/api/settings", `{"uiScale":125}`, nil); c != 200 || !strings.Contains(b, `"uiScale":125`) {
		t.Fatal(c, b)
	}
	_ = updateSettings(s.Dir, func(st *settings) { st.Lang = "es" }) // si no, sigue el idioma de Windows (en GitHub Actions, inglés)
	now := time.Now()
	if r := s.dueReminders(now, nil, 2*24*time.Hour); len(r) != 0 {
		t.Fatal("dos días encendida no es para avisar")
	}
	r := s.dueReminders(now, nil, 9*24*time.Hour)
	if len(r) != 1 || !strings.Contains(r[0].Title, "9 días") {
		t.Fatalf("reinicio: %+v", r)
	}
	if r := s.dueReminders(now.Add(24*time.Hour), nil, 10*24*time.Hour); len(r) != 0 {
		t.Fatal("no se repite al día siguiente")
	}
}

func TestThemeAndHomeSettings(t *testing.T) {
	s, base, _ := newTestServer(t)
	for _, bad := range []string{`{"theme":"rosa"}`, `{"home":["hero","hero"]}`, `{"home":["<script>"]}`} {
		if c, _ := call(t, s, base, "/api/settings", bad, nil); c != 400 {
			t.Fatalf("%s debe rechazarse", bad)
		}
	}
	c, b := call(t, s, base, "/api/settings", `{"theme":"system","home":["status","hero"]}`, nil)
	if c != 200 || !strings.Contains(b, `"theme":"system"`) || !strings.Contains(b, `"home":["status","hero"]`) {
		t.Fatal(c, b)
	}
	if _, b := call(t, s, base, "/api/state", "", nil); !strings.Contains(b, `"uptimeH":`) {
		t.Fatal("el estado trae las horas encendida")
	}
}

func TestSpaceScanViewAndRecycle(t *testing.T) {
	s, base, _ := newTestServer(t)
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "juego"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "juego", "a.pak"), make([]byte, 2<<20), 0o644)
	if c, _ := call(t, s, base, "/api/space/dir", "", nil); c != 409 {
		t.Fatal("sin análisis todavía")
	}
	body, _ := json.Marshal(map[string]string{"root": root})
	if c, b := call(t, s, base, "/api/space/scan", string(body), nil); c != 200 {
		t.Fatal(c, b)
	}
	for i := 0; i < 50; i++ {
		if _, b := call(t, s, base, "/api/space", "", nil); strings.Contains(b, `"done"`) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	_, b := call(t, s, base, "/api/space/dir", "", nil)
	if !strings.Contains(b, `"name":"juego"`) {
		t.Fatalf("vista: %s", b)
	}
	f := filepath.Join(root, "juego", "a.pak")
	req, _ := json.Marshal(map[string]any{"path": f})
	if c, _ := call(t, s, base, "/api/space/recycle", string(req), nil); c != 428 {
		t.Fatal("sin confirmar")
	}
	req, _ = json.Marshal(map[string]any{"path": os.Getenv("WINDIR") + `\System32`, "confirm": 1})
	if os.Getenv("WINDIR") != "" {
		if c, _ := call(t, s, base, "/api/space/recycle", string(req), nil); c != 403 {
			t.Fatal("carpeta del sistema protegida")
		}
	}
	req, _ = json.Marshal(map[string]any{"path": filepath.Join(root, "juego"), "confirm": 1})
	if c, b := call(t, s, base, "/api/space/recycle", string(req), nil); c != 200 {
		t.Fatal(c, b)
	}
	if _, err := os.Stat(f); err != nil {
		t.Fatal("en demostración no se borra nada de verdad")
	}
	if _, b := call(t, s, base, "/api/space/dir", "", nil); strings.Contains(b, `"name":"juego"`) {
		t.Fatalf("el análisis descuenta lo enviado a la Papelera: %s", b)
	}
}

func TestRecycleBinAndRestorePointsConfirm(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, b := call(t, s, base, "/api/recyclebin", "", nil); c != 200 || !strings.Contains(b, `"items"`) {
		t.Fatal(c, b)
	}
	if c, _ := call(t, s, base, "/api/recyclebin", `{}`, nil); c != 428 {
		t.Fatal("vaciar pide confirmación")
	}
	if c, _ := call(t, s, base, "/api/restorepoints", `{"confirm":1}`, nil); c != 428 {
		t.Fatal("borrar puntos pide doble confirmación")
	}
	if c, b := call(t, s, base, "/api/restorepoints", `{"confirm":2}`, nil); c != 200 || !strings.Contains(b, `"deleted"`) {
		t.Fatal(c, b)
	}
}
