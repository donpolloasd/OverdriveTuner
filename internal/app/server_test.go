package app

import (
	"context"
	"encoding/json"
	"fpsforge/internal/scan"
	"fpsforge/internal/tweaks"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fpsforge/internal/system"
)

func newTestServer(t *testing.T) (*Server, string, *system.Fake) {
	t.Helper()
	f := system.NewFake()
	s, err := New(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, strings.TrimSuffix(u, "/"), f
}

func call(t *testing.T, s *Server, base, path, body string, hdr map[string]string) (int, string) {
	t.Helper()
	method := http.MethodGet
	var rd io.Reader
	if body != "" {
		method = http.MethodPost
		rd = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, base+path, rd)
	req.Header.Set("X-Token", s.Token)
	for k, v := range hdr {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestAuthGuard(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, _ := call(t, s, base, "/api/state", "", map[string]string{"X-Token": ""}); c != 403 {
		t.Errorf("sin token debe ser 403, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/state", "", map[string]string{"X-Token": "otro"}); c != 403 {
		t.Errorf("token incorrecto debe ser 403, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/apply", `{"id":"fps.gamemode"}`, map[string]string{"Origin": "http://evil.example"}); c != 403 {
		t.Errorf("origen ajeno debe ser 403, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/state", "", nil); c != 200 {
		t.Errorf("con token debe ser 200, fue %d", c)
	}
}

func TestIndexInjectsToken(t *testing.T) {
	s, base, _ := newTestServer(t)
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(b), s.Token) || strings.Contains(string(b), "__TOKEN__") {
		t.Fatal("el token debe inyectarse en la página")
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("falta la política CSP")
	}
}

func TestApplyAndProfileFlow(t *testing.T) {
	s, base, f := newTestServer(t)
	_, body := call(t, s, base, "/api/profile", `{"name":"competitive"}`, nil)
	var p struct{ IDs []string }
	if err := json.Unmarshal([]byte(body), &p); err != nil || len(p.IDs) < 30 {
		t.Fatalf("perfil competitivo: %v %s", err, body)
	}
	for _, id := range p.IDs {
		c, b := call(t, s, base, "/api/apply", `{"id":"`+id+`"}`, nil)
		if c != 200 || !strings.Contains(b, `"ok":true`) {
			t.Fatalf("apply %s: %d %s", id, c, b)
		}
	}
	// Tras aplicar, el perfil ya no tiene pendientes.
	_, body = call(t, s, base, "/api/profile", `{"name":"competitive"}`, nil)
	json.Unmarshal([]byte(body), &p)
	if len(p.IDs) != 0 {
		t.Fatalf("quedan pendientes: %v", p.IDs)
	}
	// No admin: se rechaza.
	f.Admin = false
	if c, _ := call(t, s, base, "/api/apply", `{"id":"fps.gamebar"}`, nil); c != 403 {
		t.Errorf("sin admin debe ser 403, fue %d", c)
	}
	f.Admin = true
	// Revertir todo vuelve a dejar el perfil completo disponible.
	for _, id := range s.Eng.AppliedIDs() {
		if c, b := call(t, s, base, "/api/revert", `{"id":"`+id+`"}`, nil); c != 200 || !strings.Contains(b, `"ok":true`) {
			t.Fatalf("revert %s: %d %s", id, c, b)
		}
	}
	if n := len(s.Eng.AppliedIDs()); n != 0 {
		t.Fatalf("quedan %d aplicados", n)
	}
}

func TestProfilesExcludeAppsAndAdvanced(t *testing.T) {
	s, base, _ := newTestServer(t)
	for _, name := range []string{"recommended", "competitive"} {
		_, body := call(t, s, base, "/api/profile", `{"name":"`+name+`"}`, nil)
		var p struct{ IDs []string }
		json.Unmarshal([]byte(body), &p)
		for _, id := range p.IDs {
			if strings.HasPrefix(id, "apps.") || strings.HasPrefix(id, "adv.") {
				t.Errorf("%s incluye %s, que requiere decisión manual", name, id)
			}
		}
	}
}

func TestAntiCheatNeedsDoubleConfirm(t *testing.T) {
	s, base, _ := newTestServer(t)
	for _, id := range []string{"adv.vbs", "adv.hypervisor"} {
		if c, b := call(t, s, base, "/api/apply", `{"id":"`+id+`"}`, nil); c != 428 {
			t.Fatalf("%s sin confirmar debe ser 428, fue %d %s", id, c, b)
		}
		if c, _ := call(t, s, base, "/api/apply", `{"id":"`+id+`","confirm":1}`, nil); c != 428 {
			t.Fatalf("%s con una sola confirmación debe ser 428, fue %d", id, c)
		}
	}
	if n := len(s.Eng.AppliedIDs()); n != 0 {
		t.Fatalf("no debía aplicarse nada, hay %d", n)
	}
	if c, b := call(t, s, base, "/api/apply", `{"id":"adv.vbs","confirm":2}`, nil); c != 200 {
		t.Fatalf("con doble confirmación debe aplicarse: %d %s", c, b)
	}
	for _, p := range []string{"recommended", "competitive"} {
		_, body := call(t, s, base, "/api/profile", `{"name":"`+p+`"}`, nil)
		if strings.Contains(body, "adv.vbs") || strings.Contains(body, "adv.hypervisor") {
			t.Fatalf("el perfil %s no puede incluir ajustes anti-cheat", p)
		}
	}
}

func TestV11Endpoints(t *testing.T) {
	s, base, f := newTestServer(t)
	f.PSHook = func(sc string) (string, error) { return "", nil }
	if c, b := call(t, s, base, "/api/diag", ``, nil); c != 200 || !strings.Contains(b, `"checks"`) {
		t.Fatalf("diag: %d %s", c, b)
	}
	txt := "13-10-2026, 17:24:25 cs2.exe benchmark completed, 31157 frames rendered in 116.234 s\\n Average framerate  :  268.0 FPS\\n Minimum framerate  :  200.8 FPS\\n Maximum framerate  :  461.7 FPS\\n 1% low framerate   :  126.1 FPS\\n 0.1% low framerate :   84.4 FPS\\n"
	if c, b := call(t, s, base, "/api/bench/add", `{"text":"`+txt+`","label":"antes","baseline":true}`, nil); c != 200 || !strings.Contains(b, `"antes"`) {
		t.Fatalf("bench add: %d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/bench/add", `{"text":"nada"}`, nil); c != 422 {
		t.Fatalf("texto inválido debe ser 422, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/procs/kill", `{"pid":99999,"name":"lsass"}`, nil); c != 409 {
		t.Fatalf("proceso protegido debe ser 409, fue %d", c)
	}
	call(t, s, base, "/api/apply", `{"id":"fps.gamebar"}`, nil)
	if c, b := call(t, s, base, "/api/revertall", `{}`, nil); c != 200 || !strings.Contains(b, `"ok":true`) {
		t.Fatalf("revertall: %d %s", c, b)
	}
	if n := len(s.Eng.AppliedIDs()); n != 0 {
		t.Fatalf("quedan %d", n)
	}
	if c, _ := call(t, s, base, "/api/settings", `{"targetHz":5}`, nil); c != 400 {
		t.Fatalf("hz inválido debe ser 400, fue %d", c)
	}
}

func TestBenchEmptyListsAreArrays(t *testing.T) {
	s, base, _ := newTestServer(t)
	_, b := call(t, s, base, "/api/bench", ``, nil)
	for _, k := range []string{`"entries":[]`, `"groups":[]`, `"verdicts":[]`} {
		if !strings.Contains(b, k) {
			t.Fatalf("falta %s en %s", k, b)
		}
	}
}

func TestOverlayAndSessions(t *testing.T) {
	s, base, f := newTestServer(t)
	spawned := 0
	s.SpawnOverlay = func() error { spawned++; return nil }
	f.ExecHook = func(string, []string) (string, int, error) {
		return "GUID de plan: 381b4222-f694-41f0-9685-ff5bb260df2e  (Economizador)\nGUID de plan: 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c  (Alto rendimiento) *\n", 0, nil
	}
	if c, b := call(t, s, base, "/api/overlay", ``, nil); c != 200 || !strings.Contains(b, `"running":false`) || !strings.Contains(b, "Alto rendimiento") {
		t.Fatalf("%d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/overlay/start", `{}`, nil); c != 200 || spawned != 1 {
		t.Fatalf("start %d %d", c, spawned)
	}
	if c, _ := call(t, s, base, "/api/overlay/record", `{"action":"start","label":"x"}`, nil); c != 409 {
		t.Fatalf("sin overlay abierto debe ser 409, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/overlay/config", `{"gameExe":"..\\x.exe"}`, nil); c != 400 {
		t.Fatalf("exe inválido debe ser 400, fue %d", c)
	}
	if c, b := call(t, s, base, "/api/overlay/config", `{"gameExe":"cs2.exe","corner":"br","scale":1.2,"opacity":0.7,"layout":"line"}`, nil); c != 200 || !strings.Contains(b, `"corner":"br"`) {
		t.Fatalf("%d %s", c, b)
	}
	// Sesión: se importa al benchmark una sola vez; ids con rutas se rechazan.
	os.MkdirAll(filepath.Join(s.Dir, "sessions"), 0o755)
	ses := `{"id":"20261013-220000","at":"x","label":"opt","game":"cs2.exe","run":{"label":"opt","avg":240,"low1":150,"src":"fpsforge","frames":1000},"spikes":3,"frames":[4,4,30]}`
	os.WriteFile(filepath.Join(s.Dir, "sessions", "20261013-220000.json"), []byte(ses), 0o644)
	if c, b := call(t, s, base, "/api/sessions", ``, nil); c != 200 || !strings.Contains(b, "20261013-220000") || strings.Contains(b, `"frames":[4`) {
		t.Fatalf("lista: %d %s", c, b)
	}
	if c, b := call(t, s, base, "/api/sessions/get?id=20261013-220000", ``, nil); c != 200 || !strings.Contains(b, `"frames":[4,4,30]`) {
		t.Fatalf("get: %d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/sessions/get?id=../../x", ``, nil); c != 404 {
		t.Fatalf("path traversal debe ser 404, fue %d", c)
	}
	if c, b := call(t, s, base, "/api/sessions/import", `{"id":"20261013-220000","label":"optimizado"}`, nil); c != 200 || !strings.Contains(b, `"optimizado"`) {
		t.Fatalf("import: %d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/sessions/import", `{"id":"20261013-220000"}`, nil); c != 409 {
		t.Fatalf("segunda importación debe ser 409, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/sessions/delete", `{"id":"20261013-220000"}`, nil); c != 200 {
		t.Fatal(c)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "sessions", "20261013-220000.json")); err == nil {
		t.Fatal("no se borró")
	}
}

func TestBisectFlowAndTools(t *testing.T) {
	s, base, f := newTestServer(t)
	f.PSHook = func(string) (string, error) { return "", nil }
	for _, id := range []string{"fps.gamebar", "fps.gamedvr", "fps.fso", "lat.mouseaccel"} {
		if _, ok := tweaksByID(id); !ok {
			continue
		}
		call(t, s, base, "/api/apply", `{"id":"`+id+`"}`, nil)
	}
	n := len(s.Eng.AppliedIDs())
	if n < 2 {
		t.Fatalf("necesito al menos 2 aplicados, hay %d", n)
	}
	c, b := call(t, s, base, "/api/bisect/start", `{}`, nil)
	if c != 200 || !strings.Contains(b, `"active":true`) {
		t.Fatalf("start: %d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/bisect/start", `{}`, nil); c != 409 {
		t.Fatalf("segunda búsqueda debe ser 409, fue %d", c)
	}
	if got := len(s.Eng.AppliedIDs()); got >= n {
		t.Fatalf("debió revertir la mitad: antes %d, ahora %d", n, got)
	}
	if c, b := call(t, s, base, "/api/bisect/answer", `{"cancel":true}`, nil); c != 200 || !strings.Contains(b, `"active":false`) {
		t.Fatalf("cancel: %d %s", c, b)
	}
	if got := len(s.Eng.AppliedIDs()); got != n {
		t.Fatalf("cancelar debe volver a aplicar todo: %d de %d", got, n)
	}
	if c, _ := call(t, s, base, "/api/bisect/answer", `{"good":true}`, nil); c != 409 {
		t.Fatalf("responder sin búsqueda debe ser 409, fue %d", c)
	}
	if c, b := call(t, s, base, "/api/report", ``, nil); c != 200 || !strings.Contains(b, "Informe de OverdriveTuner") {
		t.Fatalf("report: %d", c)
	}
	for _, p := range []string{"/api/disks", "/api/cs2video", "/api/net"} {
		if c, _ := call(t, s, base, p, ``, nil); c != 200 {
			t.Errorf("%s: %d", p, c)
		}
	}
}

func tweaksByID(id string) (tweaks.Tweak, bool) { return tweaks.ByID(id) }

func TestStateDoesNotWaitForScan(t *testing.T) {
	f := system.NewFake()
	release := make(chan struct{})
	s, err := New(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.ScanFn = func(sys system.System) (scan.Hardware, error) { <-release; return scan.Demo(), nil } // escaneo lento
	u, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	base := strings.TrimSuffix(u, "/")
	start := time.Now()
	c, b := call(t, s, base, "/api/state", ``, nil)
	if c != 200 || time.Since(start) > 2*time.Second {
		t.Fatalf("el estado debe responder sin esperar al escaneo: %d en %v", c, time.Since(start))
	}
	if !strings.Contains(b, `"scanning":true`) || !strings.Contains(b, `"suitable":true`) {
		t.Fatalf("durante el escaneo todo se ofrece: %.200s", b)
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, b = call(t, s, base, "/api/state", ``, nil); strings.Contains(b, `"scanning":false`) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("el escaneo no terminó")
}

func TestLogoServed(t *testing.T) {
	s, base, _ := newTestServer(t)
	for _, p := range []string{"/logo.png", "/favicon.png"} {
		req, _ := http.NewRequest("GET", base+p, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" || len(b) < 200 || string(b[1:4]) != "PNG" {
			t.Fatalf("%s: %d %s %d bytes", p, resp.StatusCode, resp.Header.Get("Content-Type"), len(b))
		}
	}
	_ = s
}

func TestV14Endpoints(t *testing.T) {
	s, base, f := newTestServer(t)
	// Historial: aplicar y revertir quedan registrados, el más reciente primero.
	if c, b := call(t, s, base, "/api/history", ``, nil); c != 200 || strings.TrimSpace(b) != "[]" {
		t.Fatalf("historial vacío debe ser []: %d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/apply", `{"id":"net.dns.google"}`, nil); c != 200 {
		t.Fatal(c)
	}
	if c, _ := call(t, s, base, "/api/revert", `{"id":"net.dns.google"}`, nil); c != 200 {
		t.Fatal(c)
	}
	c, b := call(t, s, base, "/api/history", ``, nil)
	if c != 200 || strings.Index(b, `"revert"`) > strings.Index(b, `"apply"`) || !strings.Contains(b, "Google") {
		t.Fatalf("historial: %d %s", c, b)
	}
	if c, b := call(t, s, base, "/api/history", `{}`, nil); c != 200 || strings.TrimSpace(b) != "[]" {
		t.Fatalf("limpiar: %d %s", c, b)
	}
	// Microcortes de una sesión grabada.
	os.MkdirAll(filepath.Join(s.Dir, "sessions"), 0o755)
	fr := make([]string, 300)
	for i := range fr {
		fr[i] = "4"
	}
	for _, i := range []int{20, 80, 140, 200, 260} {
		fr[i] = "40"
	}
	ses := `{"id":"20261014-100000","at":"x","label":"a","game":"cs2.exe","run":{"label":"a","avg":240,"seconds":60,"src":"fpsforge","frames":1000},"spikes":5,"frames":[` + strings.Join(fr, ",") + `]}`
	os.WriteFile(filepath.Join(s.Dir, "sessions", "20261014-100000.json"), []byte(ses), 0o644)
	if c, b := call(t, s, base, "/api/sessions/microcuts?id=20261014-100000", ``, nil); c != 200 || !strings.Contains(b, `"pattern":"periodic"`) || !strings.Contains(b, `"advice":[`) {
		t.Fatalf("microcuts: %d %s", c, b)
	}
	if c, _ := call(t, s, base, "/api/sessions/microcuts?id=../x", ``, nil); c != 404 {
		t.Fatal(c)
	}
	// DNS (sin red real) y monitor de red.
	s.DNSLookup = func(_ context.Context, sv, _ string) (time.Duration, error) { return 12 * time.Millisecond, nil }
	if c, b := call(t, s, base, "/api/dnsbench", ``, nil); c != 200 || !strings.Contains(b, `"best":true`) || !strings.Contains(b, "net.dns.quad9") {
		t.Fatalf("dns: %d %s", c, b)
	}
	f.PSHook = func(script string) (string, error) {
		return `[{"pid":5,"name":"cs2","tcp":2,"udp":4,"kbs":55.5,"remote":["1.2.3.4"]}]`, nil
	}
	if c, b := call(t, s, base, "/api/netmon", ``, nil); c != 200 || !strings.Contains(b, `"cs2"`) {
		t.Fatalf("netmon: %d %s", c, b)
	}
}

func TestSettingsMergeAndHotkeys(t *testing.T) {
	s, base, _ := newTestServer(t)
	if c, b := call(t, s, base, "/api/settings", `{"lang":"en","introVol":40}`, nil); c != 200 || !strings.Contains(b, `"lang":"en"`) || !strings.Contains(b, `"targetHz":240`) {
		t.Fatalf("%d %s", c, b)
	}
	// Un cambio parcial no borra lo anterior.
	if c, b := call(t, s, base, "/api/settings", `{"targetHz":144}`, nil); c != 200 || !strings.Contains(b, `"lang":"en"`) || !strings.Contains(b, `"introVol":40`) || !strings.Contains(b, `"targetHz":144`) {
		t.Fatalf("%d %s", c, b)
	}
	for _, bad := range []string{`{"lang":"fr"}`, `{"introVol":300}`, `{"targetHz":5}`} {
		if c, _ := call(t, s, base, "/api/settings", bad, nil); c != 400 {
			t.Fatalf("%s debe ser 400, fue %d", bad, c)
		}
	}
	if c, b := call(t, s, base, "/api/state", ``, nil); c != 200 || !strings.Contains(b, `"settings":{"targetHz":144,"lang":"en"`) {
		t.Fatalf("state: %d %s", c, b[:200])
	}
	// Atajos: inválidos o repetidos se rechazan; los válidos se guardan normalizados.
	if c, _ := call(t, s, base, "/api/overlay/config", `{"gameExe":"cs2.exe","hkShow":"O","hkRecord":"Ctrl+Alt+R","hkCorner":"Ctrl+Alt+P"}`, nil); c != 400 {
		t.Fatalf("atajo sin modificador debe ser 400, fue %d", c)
	}
	if c, _ := call(t, s, base, "/api/overlay/config", `{"gameExe":"cs2.exe","hkShow":"Ctrl+Alt+K","hkRecord":"alt+ctrl+k","hkCorner":"Ctrl+Alt+P"}`, nil); c != 400 {
		t.Fatalf("atajos repetidos deben ser 400, fue %d", c)
	}
	if c, b := call(t, s, base, "/api/overlay/config", `{"gameExe":"cs2.exe","hotkey":true,"hkShow":"alt+ctrl+h","hkRecord":"Ctrl+Alt+F9","hkCorner":"Ctrl+Alt+P"}`, nil); c != 200 || !strings.Contains(b, `"hkShow":"Ctrl+Alt+H"`) || !strings.Contains(b, `"hkRecord":"Ctrl+Alt+F9"`) {
		t.Fatalf("%d %s", c, b)
	}
}

func TestConfigExportImport(t *testing.T) {
	s, base, _ := newTestServer(t)
	call(t, s, base, "/api/settings", `{"lang":"en","targetHz":165}`, nil)
	if c, _ := call(t, s, base, "/api/apply", `{"id":"net.dns.google"}`, nil); c != 200 {
		t.Fatal(c)
	}
	c, exp := call(t, s, base, "/api/config/export", ``, nil)
	if c != 200 || !strings.Contains(exp, `"app": "OverdriveTuner"`) || !strings.Contains(exp, "net.dns.google") || !strings.Contains(exp, `"targetHz": 165`) {
		t.Fatalf("export: %d %s", c, exp)
	}
	// Importar en un «equipo nuevo»: otro servidor vacío.
	s2, base2, _ := newTestServer(t)
	body := `{"bundle":` + exp + `,"apply":false}`
	c, pv := call(t, s2, base2, "/api/config/import", body, nil)
	if c != 200 || !strings.Contains(pv, `"applied":false`) || !strings.Contains(pv, "net.dns.google") {
		t.Fatalf("vista previa: %d %s", c, pv)
	}
	if _, st := call(t, s2, base2, "/api/settings", ``, nil); strings.Contains(st, `"lang":"en"`) {
		t.Fatal("la vista previa no debe guardar nada")
	}
	c, ap := call(t, s2, base2, "/api/config/import", `{"bundle":`+exp+`,"apply":true}`, nil)
	if c != 200 || !strings.Contains(ap, `"applied":true`) {
		t.Fatalf("aplicar: %d %s", c, ap)
	}
	if _, st := call(t, s2, base2, "/api/settings", ``, nil); !strings.Contains(st, `"lang":"en"`) || !strings.Contains(st, `"targetHz":165`) {
		t.Fatalf("settings tras importar: %s", st)
	}
	// Archivos ajenos o peligrosos.
	if c, _ := call(t, s2, base2, "/api/config/import", `{"bundle":{"app":"otra","format":1},"apply":true}`, nil); c != 400 {
		t.Fatalf("app ajena debe ser 400, fue %d", c)
	}
	if c, _ := call(t, s2, base2, "/api/config/import", `{"bundle":{"app":"OverdriveTuner","format":1,"settings":{"targetHz":1},"tweaks":[]},"apply":true}`, nil); c != 400 {
		t.Fatalf("settings inválidos deben ser 400, fue %d", c)
	}
	// Los ajustes que tocan anti-cheat o son avanzados nunca se aplican solos.
	var ac, adv string
	for _, tw := range tweaks.All() {
		if tw.AC != "" && ac == "" {
			ac = tw.ID
		}
		if tw.Risk >= 2 && tw.AC == "" && adv == "" {
			adv = tw.ID
		}
	}
	if ac == "" {
		t.Skip("no hay ajustes AC")
	}
	_, pv = call(t, s2, base2, "/api/config/import", `{"bundle":{"app":"OverdriveTuner","format":1,"tweaks":["`+ac+`","`+adv+`","no.existe"]},"apply":false}`, nil)
	if strings.Contains(pv, `"toApply":[{`) || strings.Count(pv, `"reason"`) != 3 {
		t.Fatalf("AC/avanzados/desconocidos deben quedar omitidos: %s", pv)
	}
}

func TestStateEnglish(t *testing.T) {
	s, base, _ := newTestServer(t)
	_, es := call(t, s, base, "/api/state", ``, nil)
	_, en := call(t, s, base, "/api/state?lang=en", ``, nil)
	if !strings.Contains(es, "Activar Modo Juego de Windows") || strings.Contains(es, "Turn on Windows Game Mode") {
		t.Fatal("el español debe ser el idioma por defecto")
	}
	if !strings.Contains(en, "Turn on Windows Game Mode") || strings.Contains(en, "Activar Modo Juego de Windows") || !strings.Contains(en, "FPS and games") {
		t.Fatal("falta el catálogo en inglés")
	}
	call(t, s, base, "/api/settings", `{"lang":"en"}`, nil)
	if _, b := call(t, s, base, "/api/state", ``, nil); !strings.Contains(b, "Turn on Windows Game Mode") {
		t.Fatal("el idioma guardado debe aplicarse sin parámetro")
	}
}
