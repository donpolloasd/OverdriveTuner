package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fpsforge/internal/update"
)

// stubUpdater cuenta las llamadas y devuelve lo que se le indique.
type stubUpdater struct {
	mu        sync.Mutex
	installed bool
	rel       update.Release
	newer     bool
	err       error
	instErr   error
	fresh     bool
	checks    int
	installs  int
}

func (u *stubUpdater) Installed() bool { return u.installed }
func (u *stubUpdater) Check(context.Context) (update.Release, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.checks++
	return u.rel, u.newer, u.err
}
func (u *stubUpdater) Install(context.Context, update.Release) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.installs++
	return u.fresh, u.instErr
}
func (u *stubUpdater) counts() (int, int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.checks, u.installs
}

func newUpdates(t *testing.T, auto bool, stub *stubUpdater) *Updates {
	t.Helper()
	dir := t.TempDir()
	if auto {
		b, _ := json.Marshal(settings{TargetHz: 144, IntroVol: 70, AutoUpdate: true})
		_ = os.WriteFile(filepath.Join(dir, "settings.json"), b, 0o644)
	}
	return &Updates{Upd: stub, Dir: dir}
}

func TestAutoUpdateOffByDefault(t *testing.T) {
	if defaultSettings().AutoUpdate {
		t.Fatal("sin elegirlo el usuario, las actualizaciones automáticas deben estar desactivadas")
	}
	// settings.json de una versión anterior (sin el campo) = desactivado
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"targetHz":144,"lang":"es"}`), 0o644)
	if readSettings(dir).AutoUpdate {
		t.Error("un settings.json viejo no debe activar las actualizaciones")
	}
}

func TestWatchDoesNothingWhenDisabled(t *testing.T) {
	stub := &stubUpdater{installed: true, newer: true, rel: update.Release{Version: "9.0.0"}, fresh: true}
	u := newUpdates(t, false, stub)
	stop := make(chan struct{})
	u.Watch(stop, 10*time.Millisecond, 20*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	close(stop)
	if c, i := stub.counts(); c != 0 || i != 0 {
		t.Fatalf("con las automáticas desactivadas no se consulta ni se instala nada: checks=%d installs=%d", c, i)
	}
}

func TestWatchInstallsWhenEnabled(t *testing.T) {
	stub := &stubUpdater{installed: true, newer: true, rel: update.Release{Version: "9.0.0"}, fresh: true}
	u := newUpdates(t, true, stub)
	got := make(chan string, 4)
	u.OnApplied = func(v string) { got <- v }
	stop := make(chan struct{})
	defer close(stop)
	u.Watch(stop, 10*time.Millisecond, 30*time.Millisecond)
	select {
	case v := <-got:
		if v != "9.0.0" {
			t.Errorf("versión avisada: %s", v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("con las automáticas activadas debía instalarse")
	}
	time.Sleep(150 * time.Millisecond) // varias vueltas más: el aviso no se repite
	if len(got) != 0 {
		t.Error("el aviso de actualización no debe repetirse")
	}
	if st := u.Status(); st.Applied != "9.0.0" || !st.Auto || st.Current != Version {
		t.Errorf("estado: %+v", st)
	}
}

func TestCycleManualDoesNotInstallWithoutAuto(t *testing.T) {
	stub := &stubUpdater{installed: true, newer: true, rel: update.Release{Version: "9.0.0", Notes: "n", Page: "p"}, fresh: true}
	u := newUpdates(t, false, stub)
	u.Cycle(false)
	st := u.Status()
	if !st.Available || st.Latest != "9.0.0" || st.Applied != "" || st.CheckedAt == 0 {
		t.Fatalf("debía avisar que hay una versión nueva sin instalarla: %+v", st)
	}
	if _, i := stub.counts(); i != 0 {
		t.Error("sin automáticas, buscar no instala")
	}
	u.InstallNow() // el usuario lo pide a mano
	if _, i := stub.counts(); i != 1 || u.Status().Applied != "9.0.0" {
		t.Errorf("la instalación pedida a mano debía hacerse: %+v", u.Status())
	}
}

func TestCycleNotInstalledCopyOnlyNotifies(t *testing.T) {
	stub := &stubUpdater{installed: false, newer: true, rel: update.Release{Version: "9.0.0"}}
	u := newUpdates(t, true, stub)
	u.Cycle(true)
	if _, i := stub.counts(); i != 0 {
		t.Error("una copia no instalada (portable) no se reemplaza sola")
	}
	if st := u.Status(); !st.Available || st.Installed {
		t.Errorf("debía avisar sin instalar: %+v", st)
	}
}

func TestCycleErrorsAreFriendly(t *testing.T) {
	stub := &stubUpdater{installed: true, err: update.ErrNoRelease}
	u := newUpdates(t, true, stub)
	u.Cycle(true)
	if st := u.Status(); st.Error == "" || st.Available || st.Busy != "" {
		t.Fatalf("sin Release publicado: %+v", st)
	}
	stub.err = &update.StatusError{Code: 403}
	u.Cycle(true)
	if st := u.Status(); st.Error == "" {
		t.Error("un 403 de GitHub debe verse como error amigable")
	}
	stub.err = nil
	stub.rel = update.Release{Version: "9.0.0"}
	stub.newer = true
	stub.instErr = errors.New("sin permiso")
	u.Cycle(true)
	if st := u.Status(); st.Error == "" || st.Applied != "" {
		t.Errorf("si falla la instalación debe mostrarse y no marcar como aplicada: %+v", st)
	}
	stub.instErr = nil
	stub.fresh = true
	u.Cycle(true)
	if st := u.Status(); st.Error != "" || st.Applied != "9.0.0" {
		t.Errorf("tras un éxito el error se borra: %+v", st)
	}
}

func TestUpdateEndpoints(t *testing.T) {
	s, base, _ := newTestServer(t)
	stub := &stubUpdater{installed: true, newer: true, rel: update.Release{Version: "9.0.0", Notes: "mejoras"}, fresh: true}
	s.Up.Upd = stub
	code, body := call(t, s, base, "/api/update", "", nil)
	var st UpdateStatus
	if code != 200 || json.Unmarshal([]byte(body), &st) != nil || st.Auto || !st.Installed || st.Current != Version {
		t.Fatalf("estado inicial: %d %s", code, body)
	}
	// buscar sin automáticas: avisa pero no instala
	code, body = call(t, s, base, "/api/update/check", "{}", nil)
	_ = json.Unmarshal([]byte(body), &st)
	if code != 200 || !st.Available || st.Latest != "9.0.0" || st.Applied != "" {
		t.Fatalf("check: %d %s", code, body)
	}
	if _, i := stub.counts(); i != 0 {
		t.Error("buscar con las automáticas desactivadas no debe instalar")
	}
	// reiniciar sin nada instalado: error
	if code, _ := call(t, s, base, "/api/update/restart", "{}", nil); code != 400 {
		t.Errorf("reiniciar sin versión instalada: %d", code)
	}
	// instalar a mano
	code, _ = call(t, s, base, "/api/update/install", "{}", nil)
	if code != 200 {
		t.Fatalf("install: %d", code)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.Up.Status().Applied == "9.0.0" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.Up.Status().Applied != "9.0.0" {
		t.Fatal("la instalación a mano no terminó")
	}
	// activar las automáticas desde Configuración
	code, body = call(t, s, base, "/api/settings", `{"autoUpdate":true}`, nil)
	var set settings
	if code != 200 || json.Unmarshal([]byte(body), &set) != nil || !set.AutoUpdate {
		t.Fatalf("settings: %d %s", code, body)
	}
	if !s.loadSettings().AutoUpdate {
		t.Error("el ajuste debía guardarse")
	}
	// y desactivarlas
	call(t, s, base, "/api/settings", `{"autoUpdate":false}`, nil)
	if s.loadSettings().AutoUpdate {
		t.Error("el ajuste debía poder desactivarse")
	}
	// sin token no se puede
	if code, _ := call(t, s, base, "/api/update/check", "{}", map[string]string{"X-Token": ""}); code == 200 {
		t.Error("sin token no debe responder")
	}
}

func TestEnablingAutoUpdateChecksRightAway(t *testing.T) {
	s, base, _ := newTestServer(t)
	stub := &stubUpdater{installed: true, newer: true, rel: update.Release{Version: "9.0.0"}, fresh: true}
	s.Up.Upd = stub
	call(t, s, base, "/api/settings", `{"autoUpdate":true}`, nil)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, i := stub.counts(); i == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("al activar las automáticas debía buscar e instalar enseguida")
}

func TestImportDoesNotEnableAutoUpdate(t *testing.T) {
	s, base, _ := newTestServer(t)
	call(t, s, base, "/api/config/import", `{"app":"OverdriveTuner","format":1,"settings":{"autoUpdate":true,"lang":"en"},"tweaks":[]}`, nil)
	if s.loadSettings().AutoUpdate {
		t.Error("un archivo importado no debe activar las actualizaciones automáticas")
	}
}

func TestScheduleAndToolsEndpoints(t *testing.T) {
	s, base, _ := newTestServer(t)
	code, body := call(t, s, base, "/api/schedule", "", nil)
	var l []map[string]any
	if code != 200 || json.Unmarshal([]byte(body), &l) != nil || len(l) != 2 || l[0]["registered"] == true {
		t.Fatalf("estado inicial: %d %s", code, body)
	}
	code, body = call(t, s, base, "/api/schedule", `{"id":"cleanup","enabled":true,"day":3,"time":"19:30"}`, nil)
	json.Unmarshal([]byte(body), &l)
	if code != 200 || l[0]["registered"] != true || l[0]["time"] != "19:30" || l[0]["day"] != float64(3) {
		t.Fatalf("programar: %d %s", code, body)
	}
	call(t, s, base, "/api/schedule", `{"id":"cleanup","enabled":false}`, nil)
	_, body = call(t, s, base, "/api/schedule", "", nil)
	json.Unmarshal([]byte(body), &l)
	if l[0]["registered"] == true {
		t.Error("debía quitarse")
	}
	if code, _ := call(t, s, base, "/api/schedule", `{"id":"nada","enabled":true}`, nil); code != 404 {
		t.Errorf("tarea desconocida: %d", code)
	}
	if code, _ := call(t, s, base, "/api/diag/action", `{"action":"open_driver_nvidia"}`, nil); code != 200 {
		t.Errorf("abrir página de drivers: %d", code)
	}
	if code, _ := call(t, s, base, "/api/diag/action", `{"action":"open_driver_evil"}`, nil); code == 200 {
		t.Error("un fabricante desconocido no debe abrir nada")
	}
	code, body = call(t, s, base, "/api/gameping", "", nil)
	var gp []map[string]any
	if code != 200 || json.Unmarshal([]byte(body), &gp) != nil || len(gp) < 10 || gp[0]["status"] == nil {
		t.Fatalf("gameping: %d %.200s", code, body)
	}
}
