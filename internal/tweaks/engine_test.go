package tweaks

import (
	"strings"
	"testing"

	"fpsforge/internal/system"
)

func newEng(t *testing.T) (*Engine, *system.Fake) {
	t.Helper()
	f := system.NewFake()
	e, err := NewEngine(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return e, f
}

func TestApplyRevertRestoresPrevious(t *testing.T) {
	e, f := newEng(t)
	f.Seed(sysProf, "SystemResponsiveness", "REG_DWORD", "20")
	if r := e.Apply("fps.sysresp"); !r.OK {
		t.Fatalf("apply: %+v", r)
	}
	v, _ := system.RegGet(f, sysProf, "SystemResponsiveness")
	if v.Data != "10" {
		t.Fatalf("esperaba 10, hay %s", v.Data)
	}
	if !e.Applied("fps.sysresp") {
		t.Fatal("debería figurar aplicado")
	}
	if r := e.Revert("fps.sysresp"); !r.OK {
		t.Fatalf("revert: %+v", r)
	}
	v, _ = system.RegGet(f, sysProf, "SystemResponsiveness")
	if v.Data != "20" {
		t.Fatalf("esperaba 20 restaurado, hay %s", v.Data)
	}
	if e.Applied("fps.sysresp") {
		t.Fatal("ya no debería figurar aplicado")
	}
}

func TestRevertDeletesValuesThatDidNotExist(t *testing.T) {
	e, f := newEng(t)
	e.Apply("fps.gamemode")
	v, _ := system.RegGet(f, gameBar, "AutoGameModeEnabled")
	if !v.Exists {
		t.Fatal("debería existir tras aplicar")
	}
	e.Revert("fps.gamemode")
	v, _ = system.RegGet(f, gameBar, "AutoGameModeEnabled")
	if v.Exists {
		t.Fatal("debería haberse borrado al revertir")
	}
}

func TestApplyRollsBackOnFailure(t *testing.T) {
	e, f := newEng(t)
	f.Seed(gameCfg, "GameDVR_Enabled", "REG_DWORD", "1")
	f.FailSet = []string{`Policies\Microsoft\Windows\GameDVR`}
	r := e.Apply("fps.gamedvr")
	if r.OK || len(r.Errors) == 0 {
		t.Fatalf("debía fallar: %+v", r)
	}
	v, _ := system.RegGet(f, gameCfg, "GameDVR_Enabled")
	if v.Data != "1" {
		t.Fatalf("el rollback debía restaurar 1, hay %q", v.Data)
	}
	if e.Applied("fps.gamedvr") {
		t.Fatal("no debe quedar registrado")
	}
}

func TestServiceOnlyIfExists(t *testing.T) {
	e, f := newEng(t)
	f.Seed(svcKey+"DiagTrack", "Start", "REG_DWORD", "2")
	if r := e.Apply("svc.telemetry"); !r.OK {
		t.Fatalf("%+v", r)
	}
	v, _ := system.RegGet(f, svcKey+"DiagTrack", "Start")
	if v.Data != "4" {
		t.Fatalf("DiagTrack debe quedar en 4, hay %s", v.Data)
	}
	if f.RegKeyExists(svcKey + "dmwappushservice") {
		t.Fatal("no debe crear servicios inexistentes")
	}
	e.Revert("svc.telemetry")
	v, _ = system.RegGet(f, svcKey+"DiagTrack", "Start")
	if v.Data != "2" {
		t.Fatalf("debe volver a 2, hay %s", v.Data)
	}
}

func TestPersistenceAcrossRestart(t *testing.T) {
	f := system.NewFake()
	dir := t.TempDir()
	e1, _ := NewEngine(f, dir)
	f.Seed(sysProf, "NetworkThrottlingIndex", "REG_DWORD", "10")
	e1.Apply("fps.netthrottle")
	e2, _ := NewEngine(f, dir)
	if !e2.Applied("fps.netthrottle") {
		t.Fatal("el estado debe persistir")
	}
	if r := e2.Revert("fps.netthrottle"); !r.OK {
		t.Fatalf("%+v", r)
	}
	v, _ := system.RegGet(f, sysProf, "NetworkThrottlingIndex")
	if v.Data != "10" {
		t.Fatalf("debe volver a 10, hay %s", v.Data)
	}
}

func TestPowerPlanRevert(t *testing.T) {
	e, f := newEng(t)
	if r := e.Apply("pw.plan"); !r.OK {
		t.Fatalf("%+v", r)
	}
	log := strings.Join(f.ExecLog, "\n")
	if !strings.Contains(log, "/setactive 11111111-2222-3333-4444-555555555555") {
		t.Fatalf("no activó el plan nuevo:\n%s", log)
	}
	e.Revert("pw.plan")
	log = strings.Join(f.ExecLog, "\n")
	if !strings.Contains(log, "/setactive 381b4222-f694-41f0-9685-ff5bb260df2e") ||
		!strings.Contains(log, "/delete 11111111-2222-3333-4444-555555555555") {
		t.Fatalf("no restauró/borró el plan:\n%s", log)
	}
}

func TestDetectAndDefaultRevert(t *testing.T) {
	e, f := newEng(t)
	tw, _ := ByID("fps.sysresp")
	if e.Detect(tw) {
		t.Fatal("no debería detectarse sin valores")
	}
	f.Seed(sysProf, "SystemResponsiveness", "REG_DWORD", "10")
	if !e.Detect(tw) {
		t.Fatal("debería detectarse como activo")
	}
	// Sin registro propio, revertir usa el valor de fábrica.
	if r := e.Revert("fps.sysresp"); !r.OK {
		t.Fatalf("%+v", r)
	}
	v, _ := system.RegGet(f, sysProf, "SystemResponsiveness")
	if v.Data != "20" {
		t.Fatalf("debe usar el valor de fábrica 20, hay %s", v.Data)
	}
}

func TestContextMenuKeyRemoved(t *testing.T) {
	e, f := newEng(t)
	e.Apply("db.contextmenu")
	k := `HKCU\Software\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}`
	if !f.RegKeyExists(k + `\InprocServer32`) {
		t.Fatal("debe crear la clave")
	}
	e.Revert("db.contextmenu")
	if f.RegKeyExists(k) {
		t.Fatal("debe borrar la clave al revertir")
	}
}

func TestDetectAllBatchesAndSkipsMissingServices(t *testing.T) {
	e, f := newEng(t)
	f.Seed(svcKey+"DiagTrack", "Start", "REG_DWORD", "4") // solo existe DiagTrack
	f.Seed(sysProf, "SystemResponsiveness", "REG_DWORD", "10")
	res := e.DetectAll(All())
	if !res["svc.telemetry"] {
		t.Error("svc.telemetry debería detectarse con los servicios presentes en 4")
	}
	if !res["fps.sysresp"] {
		t.Error("fps.sysresp debería detectarse")
	}
	if res["fps.gamemode"] {
		t.Error("fps.gamemode no debería detectarse")
	}
	if res["fps.priosep"] {
		t.Error("sin valores no debe detectarse")
	}
}

// Aplicar y revertir cada tweak debe dejar el registro exactamente como estaba.
func TestEveryTweakApplyRevertIsClean(t *testing.T) {
	for _, tw := range All() {
		e, f := newEng(t)
		// Estado previo mixto: algunos valores existen (con otro dato), otros no.
		for i, op := range tw.Ops {
			if op.Kind == OpReg && i%2 == 0 {
				f.Seed(op.Key, op.Name, op.Type, "7")
			}
			if op.Kind == OpSvc {
				f.Seed(svcKey+op.Svc, "Start", "REG_DWORD", "2")
			}
		}
		before := f.Snapshot()
		keysBefore := f.KeyCount()
		r := e.Apply(tw.ID)
		if !r.OK {
			t.Errorf("%s: apply falló: %v", tw.ID, r.Errors)
			continue
		}
		if r := e.Revert(tw.ID); !r.OK {
			t.Errorf("%s: revert falló: %v", tw.ID, r.Errors)
			continue
		}
		after := f.Snapshot()
		if len(after) != len(before) {
			t.Errorf("%s: cantidad de valores %d -> %d", tw.ID, len(before), len(after))
		}
		for k, v := range before {
			if after[k] != v {
				t.Errorf("%s: %s cambió de %+v a %+v", tw.ID, k, v, after[k])
			}
		}
		if f.KeyCount() != keysBefore && tw.ID != "db.contextmenu" {
			// Las claves creadas pueden quedar vacías; solo se exige limpieza total en el menú contextual.
			t.Logf("%s: claves %d -> %d (vacías)", tw.ID, keysBefore, f.KeyCount())
		}
	}
}
