package app

import (
	"encoding/json"
	"fpsforge/internal/scan"
	"strings"
	"testing"
)

func TestAppsCatalogAndInstall(t *testing.T) {
	s, base, f := newTestServer(t)
	c, body := call(t, s, base, "/api/apps", "", nil)
	if c != 200 {
		t.Fatal(c, body)
	}
	var cat struct {
		Items []struct {
			ID string `json:"id"`
			AC string `json:"ac"`
		} `json:"items"`
		Winget bool `json:"winget"`
	}
	if err := json.Unmarshal([]byte(body), &cat); err != nil || len(cat.Items) < 25 || !cat.Winget {
		t.Fatalf("catálogo: %v %d winget=%v", err, len(cat.Items), cat.Winget)
	}
	if c, b := call(t, s, base, "/api/apps/install", `{"id":"afterburner"}`, nil); c != 200 || !strings.Contains(b, `"ok":true`) {
		t.Fatalf("%d %s", c, b)
	}
	log := strings.Join(f.ExecLog, "\n")
	for _, id := range []string{"Guru3D.Afterburner", "Guru3D.RTSS"} {
		if !strings.Contains(log, "install --id "+id+" --exact") {
			t.Errorf("no se instaló %s:\n%s", id, log)
		}
	}
	if c, _ := call(t, s, base, "/api/apps/install", `{"id":"no-existe"}`, nil); c != 404 {
		t.Fatalf("programa desconocido debe ser 404, fue %d", c)
	}
	// Los drivers solo abren direcciones fijas.
	if c, _ := call(t, s, base, "/api/apps/drivers", `{"vendor":"https://malo.example"}`, nil); c != 400 {
		t.Fatalf("fabricante inválido debe ser 400, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/apps/drivers", `{"vendor":"amd"}`, nil); c != 200 {
		t.Fatalf("drivers amd: %d", c)
	}
	if !strings.Contains(strings.Join(f.ExecLog, "\n"), "https://www.amd.com/") {
		t.Error("no se abrió la página de AMD")
	}
}

func TestRetiredAndVendorTweaksHidden(t *testing.T) {
	s, base, _ := newTestServer(t)
	_ = s.hardware(true)
	c, body := call(t, s, base, "/api/state", "", nil)
	if c != 200 {
		t.Fatal(c)
	}
	var st struct {
		Tweaks []struct {
			ID string `json:"id"`
		} `json:"tweaks"`
	}
	if err := json.Unmarshal([]byte(body), &st); err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, x := range st.Tweaks {
		have[x.ID] = true
	}
	for _, id := range []string{"svc.spooler", "adv.spectre", "adv.hypervisor", "ui.shake"} {
		if have[id] {
			t.Errorf("%s está retirado y no debería mostrarse", id)
		}
	}
	for _, id := range []string{"upd.pause", "net.noipv6", "ux.dark", "pr.wpbt"} {
		if !have[id] {
			t.Errorf("falta el ajuste nuevo %s", id)
		}
	}
	// El equipo de demostración es AMD: el ajuste de NVIDIA no se muestra, el de AMD sí.
	if have["gpu.nvtelemetry"] || !have["gpu.amdcrash"] {
		t.Errorf("filtro por fabricante: nvidia=%v amd=%v", have["gpu.nvtelemetry"], have["gpu.amdcrash"])
	}
}

func TestRetiredTweakStaysRevertible(t *testing.T) {
	s, base, _ := newTestServer(t)
	// Un ajuste retirado se puede seguir aplicando por API (copias de seguridad antiguas) y, si está aplicado, se muestra.
	if c, b := call(t, s, base, "/api/apply", `{"id":"ui.shake"}`, nil); c != 200 {
		t.Fatalf("%d %s", c, b)
	}
	_, body := call(t, s, base, "/api/state", "", nil)
	if !strings.Contains(body, `"id":"ui.shake"`) {
		t.Error("un ajuste retirado pero aplicado debe verse para poder revertirlo")
	}
	if c, b := call(t, s, base, "/api/revert", `{"id":"ui.shake"}`, nil); c != 200 {
		t.Fatalf("%d %s", c, b)
	}
}

func TestUpdatePauseNeedsDoubleConfirmAndDrifts(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, _ := call(t, s, base, "/api/apply", `{"id":"upd.pause"}`, nil); c == 200 {
		t.Fatal("la pausa de actualizaciones no debe aplicarse sin doble confirmación")
	}
	if c, b := call(t, s, base, "/api/apply", `{"id":"upd.pause","confirm":2}`, nil); c != 200 {
		t.Fatalf("%d %s", c, b)
	}
	// Windows (o una actualización) deshace la política: la interfaz debe avisarlo.
	if err := s.Sys.RegDeleteValue(`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`, "NoAutoUpdate"); err != nil {
		t.Fatal(err)
	}
	_, body := call(t, s, base, "/api/state", "", nil)
	var st struct {
		Tweaks []struct {
			ID    string `json:"id"`
			Drift bool   `json:"drift"`
		} `json:"tweaks"`
	}
	_ = json.Unmarshal([]byte(body), &st)
	found := false
	for _, x := range st.Tweaks {
		if x.ID == "upd.pause" {
			found = true
			if !x.Drift {
				t.Error("debe marcar que Windows deshizo la pausa")
			}
		}
	}
	if !found {
		t.Fatal("no aparece upd.pause")
	}
}

func TestTargetHzAutomatico(t *testing.T) {
	hw := scan.Hardware{MaxHz: 165}
	st := defaultSettings()
	if got := st.targetHz(hw); got != 165 {
		t.Fatalf("sin valor manual debe usar el máximo detectado, dio %d", got)
	}
	if got := st.targetHz(scan.Hardware{}); got != 0 {
		t.Fatalf("sin detección no se compara contra nada, dio %d", got)
	}
	st.TargetHz, st.HzManual = 144, true
	if got := st.targetHz(hw); got != 144 {
		t.Fatalf("el valor manual manda, dio %d", got)
	}
	var p settingsPatch
	auto := true
	p.HzAuto = &auto
	p.apply(&st)
	if st.HzManual || st.targetHz(hw) != 165 {
		t.Fatal("«usar el detectado» debe volver al automático")
	}
}
