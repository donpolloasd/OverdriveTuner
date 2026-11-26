package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/changelog"
	"fpsforge/internal/diag"
	"fpsforge/internal/drvstore"
	"fpsforge/internal/oldapps"
	"fpsforge/internal/overlay"
	"fpsforge/internal/system"
	"fpsforge/internal/tools"
	"fpsforge/internal/tweaks"
)

// settingsMu ordena las escrituras de settings.json que hace el programa por su cuenta (compilación de
// Windows, limpiezas, recordatorios), para que no se pisen entre sí.
var settingsMu sync.Mutex

// updateSettings lee settings.json, aplica f y lo guarda.
func updateSettings(dir string, f func(*settings)) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	st := readSettings(dir)
	f(&st)
	return writeSettings(dir, st)
}

// ---------------------------------------------------------------- actualización de Windows

// noteBuild anota la compilación de Windows. Si cambió desde la última vez, Windows se actualizó: se
// recuerda cuándo, para explicar por qué algunos ajustes pueden haberse deshecho.
func (s *Server) noteBuild(build int) {
	if build <= 0 {
		return
	}
	if st := s.loadSettings(); st.LastBuild == build {
		return
	}
	_ = updateSettings(s.Dir, func(st *settings) {
		if st.LastBuild != 0 && st.LastBuild != build {
			st.PrevBuild, st.BuildAt = st.LastBuild, time.Now()
		}
		st.LastBuild = build
	})
}

// winUpdateInfo dice si Windows se actualizó hace poco (30 días) y de qué compilación a cuál.
func winUpdateInfo(st settings, now time.Time) map[string]any {
	if st.PrevBuild == 0 || st.BuildAt.IsZero() || now.Sub(st.BuildAt) > 30*24*time.Hour {
		return nil
	}
	return map[string]any{"from": st.PrevBuild, "to": st.LastBuild, "at": st.BuildAt.Format("2006-01-02")}
}

// handleReapply vuelve a aplicar los ajustes que Windows deshizo (los que la interfaz marca con drift).
func (s *Server) handleReapply(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || len(req.IDs) == 0 {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	out := make([]tweaks.Result, 0, len(req.IDs))
	for _, id := range req.IDs {
		res := s.Eng.Reapply(id)
		s.logChange(id, "reapply", res)
		out = append(out, res)
	}
	writeJSON(w, out)
}

// ---------------------------------------------------------------- novedades al actualizar

// handleWhatsNew dice qué trajo la versión nueva la primera vez que se abre (POST: ya se mostró).
func (s *Server) handleWhatsNew(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = updateSettings(s.Dir, func(st *settings) { st.SeenVersion = Version })
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	st := s.loadSettings()
	lang := r.URL.Query().Get("lang")
	var list []changelog.Release
	switch {
	case st.SeenVersion == "" && !st.TermsOK:
		// Instalación nueva: no hay nada «nuevo» que contar.
		_ = updateSettings(s.Dir, func(st *settings) { st.SeenVersion = Version })
	case st.SeenVersion == "":
		// Venía de una versión anterior a esta ventana: se muestra solo la versión actual.
		for _, rel := range changelog.Releases {
			if rel.Version == Version {
				list = append(list, rel)
			}
		}
	case changelog.Newer(Version, st.SeenVersion):
		list = changelog.Since(st.SeenVersion)
	}
	if len(list) > 3 {
		list = list[:3]
	}
	for i := range list {
		list[i] = changelog.Localized(list[i], lang)
	}
	writeJSON(w, map[string]any{"version": Version, "releases": list})
}

// ---------------------------------------------------------------- reportar un problema

const issuesURL = "https://github.com/donpolloasd/OverdriveTuner/issues/new"

// issueDraft arma el texto de un reporte con el equipo y el diagnóstico. No incluye el nombre del equipo,
// del usuario ni rutas de carpetas: el usuario lo revisa antes de abrir nada.
func (s *Server) issueDraft(lang string) string {
	en := lang == "en"
	L := func(es, eng string) string {
		if en {
			return eng
		}
		return es
	}
	hw := s.hardware(false)
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", L("**Qué pasó:** (describilo acá: qué hiciste, qué esperabas y qué pasó)", "**What happened:** (describe it here: what you did, what you expected and what happened)"))
	fmt.Fprintf(&b, "### %s\n", L("Equipo", "System"))
	fmt.Fprintf(&b, "- OverdriveTuner %s\n- %s (%s %d)\n- CPU: %s\n", Version, hw.OS, L("compilación", "build"), hw.Build, hw.CPU)
	for _, g := range hw.GPUs {
		fmt.Fprintf(&b, "- GPU: %s (driver %s)\n", g.Name, g.Driver)
	}
	fmt.Fprintf(&b, "- RAM: %d GB @ %d MT/s\n", hw.RAMGB, hw.RAMMTs)
	if hw.Laptop {
		fmt.Fprintf(&b, "- %s\n", L("Notebook", "Laptop"))
	}
	views := s.views(hw, false, lang)
	act, drift := 0, 0
	for _, v := range views {
		if v.Applied {
			act++
		}
		if v.Drift {
			drift++
		}
	}
	fmt.Fprintf(&b, "- %s: %d", L("Ajustes aplicados", "Tweaks applied"), act)
	if drift > 0 {
		fmt.Fprintf(&b, " (%s: %d)", L("deshechos por Windows", "undone by Windows"), drift)
	}
	b.WriteString("\n")
	var probs []string
	for _, c := range s.diagChecks(hw, s.loadSettings().targetHz(hw), false) {
		if c.Status == diag.Bad || c.Status == diag.Warn {
			probs = append(probs, fmt.Sprintf("- [%s] %s", c.Status, c.Title))
		}
	}
	if len(probs) > 0 {
		fmt.Fprintf(&b, "\n### %s\n%s\n", L("Avisos del Diagnóstico", "Diagnostics warnings"), strings.Join(probs, "\n"))
	}
	return b.String()
}

// handleIssue devuelve el borrador del reporte (GET) o abre GitHub con el texto que revisó el usuario (POST).
func (s *Server) handleIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]any{"title": "", "body": s.issueDraft(r.URL.Query().Get("lang"))})
		return
	}
	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if readJSON(r, &req) != nil || strings.TrimSpace(req.Title) == "" {
		fail(w, 400, "Poné un título que resuma el problema.")
		return
	}
	u := issueURL(req.Title, req.Body)
	if _, demo := s.Sys.(*system.Fake); !demo {
		if _, _, err := s.Sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", u); err != nil {
			fail(w, 500, "No se pudo abrir el navegador: "+err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true, "url": u})
}

// issueURL arma la dirección de un issue nuevo con el texto ya cargado (recortado para que entre en la URL).
func issueURL(title, body string) string {
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if r := []rune(title); len(r) > 120 {
		title = string(r[:120])
	}
	if r := []rune(body); len(r) > 5000 {
		body = string(r[:5000]) + "\n…"
	}
	return issuesURL + "?title=" + url.QueryEscape(title) + "&body=" + url.QueryEscape(body)
}

// ---------------------------------------------------------------- recordatorios de mantenimiento

// MarkCleaned anota que se limpiaron los temporales (lo usa también la limpieza programada).
func MarkCleaned(dir string) {
	_ = updateSettings(dir, func(st *settings) { st.LastClean = time.Now() })
}

func (s *Server) markCleaned(ids []string) {
	for _, id := range ids {
		if id == "usertemp" || id == "wintemp" {
			MarkCleaned(s.Dir)
			return
		}
	}
}

// Reminder es un recordatorio para mostrar en la bandeja.
type Reminder struct{ Title, Text string }

// dueReminders decide qué recordatorios tocan ahora y anota que se mostraron. Cada uno se repite como
// mucho cada dos semanas (el de driver, cada mes) y nunca si el usuario los desactivó.
func (s *Server) dueReminders(now time.Time, drivers []diag.Check, up time.Duration) []Reminder {
	st := s.loadSettings()
	if st.RemindOff {
		return nil
	}
	en := overlay.UILang(s.Dir) == "en"
	L := func(es, eng string) string {
		if en {
			return eng
		}
		return es
	}
	var out []Reminder
	_ = updateSettings(s.Dir, func(st *settings) {
		if st.LastClean.IsZero() {
			st.LastClean = now // se empieza a contar desde ahora: nada de avisos el primer día
		}
		if now.Sub(st.LastClean) > 30*24*time.Hour && now.Sub(st.RemindClean) > 14*24*time.Hour {
			st.RemindClean = now
			out = append(out, Reminder{L("Hace más de un mes que no limpiás", "It has been over a month since your last cleanup"),
				L("Abrí OverdriveTuner → Limpieza para borrar los archivos temporales. Tarda unos segundos.", "Open OverdriveTuner → Cleanup to delete temporary files. It takes a few seconds.")})
		}
		for _, c := range drivers {
			if strings.HasPrefix(c.ID, "driver_") && c.Status == diag.Warn && now.Sub(st.RemindDrv) > 30*24*time.Hour {
				st.RemindDrv = now
				out = append(out, Reminder{L("Tu driver de video es viejo", "Your graphics driver is old"),
					L("Tiene más de 9 meses. Bajá el más reciente desde OverdriveTuner → DRIVERS.", "It is over 9 months old. Get the latest one from OverdriveTuner → DRIVERS.")})
				break
			}
		}
		if days := int(up.Hours() / 24); days >= 7 && now.Sub(st.RemindBoot) > 3*24*time.Hour {
			st.RemindBoot = now
			out = append(out, Reminder{L(fmt.Sprintf("Hace %d días que no reiniciás la PC", days), fmt.Sprintf("Your PC has not restarted in %d days", days)),
				L("Reiniciar libera memoria y termina de instalar actualizaciones. Ojo: «Apagar» no cuenta si está activado el Inicio rápido de Windows.", "Restarting frees memory and finishes installing updates. Note: «Shut down» does not count if Windows Fast Startup is on.")})
		}
	})
	return out
}

// WatchReminders revisa cada 6 horas si toca algún recordatorio y lo muestra con notify.
func (s *Server) WatchReminders(stop <-chan struct{}, notify func(title, text string)) {
	if _, demo := s.Sys.(*system.Fake); demo {
		return
	}
	wait := 3 * time.Minute // deja que el programa termine de arrancar
	for {
		select {
		case <-stop:
			return
		case <-time.After(wait):
		}
		wait = 6 * time.Hour
		if s.loadSettings().RemindOff {
			continue
		}
		hw, ok := s.hardwareIfReady()
		var drv []diag.Check
		if ok {
			drv = s.slowDiag(hw, false)
		}
		for _, rm := range s.dueReminders(time.Now(), drv, uptime()) {
			notify(rm.Title, rm.Text)
		}
	}
}

// ---------------------------------------------------------------- programas y juegos sin usar

func (s *Server) isDemo() bool { _, demo := s.Sys.(*system.Fake); return demo }

func (s *Server) handleOldApps(w http.ResponseWriter, r *http.Request) {
	if s.isDemo() {
		writeJSON(w, oldapps.Demo())
		return
	}
	items, err := oldapps.List(s.Sys, time.Now())
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if items == nil {
		items = []oldapps.Item{}
	}
	writeJSON(w, items)
}

// handleOldAppsUninstall abre el desinstalador oficial (no desinstala en silencio).
func (s *Server) handleOldAppsUninstall(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || req.ID == "" {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.isDemo() {
		if err := oldapps.Uninstall(s.Sys, req.ID); err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}

// ---------------------------------------------------------------- drivers viejos

func (s *Server) handleOldDrivers(w http.ResponseWriter, r *http.Request) {
	if s.isDemo() {
		writeJSON(w, map[string]any{"drivers": drvstore.Demo()})
		return
	}
	list, err := drvstore.Old(s.Sys)
	if err == drvstore.ErrUnsupported {
		writeJSON(w, map[string]any{"drivers": []drvstore.Pkg{}, "unsupported": true})
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if list == nil {
		list = []drvstore.Pkg{}
	}
	writeJSON(w, map[string]any{"drivers": list})
}

// handleDriversDelete borra los paquetes elegidos. Pide la confirmación del usuario.
func (s *Server) handleDriversDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Names   []string `json:"names"`
		Confirm int      `json:"confirm"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || len(req.Names) == 0 {
		fail(w, 400, "petición no válida")
		return
	}
	if req.Confirm < 1 {
		fail(w, 428, "Borrar drivers necesita la confirmación del usuario.")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	type res struct {
		Name  string `json:"name"`
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	out := []res{}
	for _, n := range req.Names {
		if s.isDemo() {
			out = append(out, res{Name: n, OK: true})
			continue
		}
		if err := drvstore.Delete(s.Sys, n); err != nil {
			out = append(out, res{Name: n, Error: err.Error()})
		} else {
			out = append(out, res{Name: n, OK: true})
		}
	}
	writeJSON(w, out)
}

// ---------------------------------------------------------------- latencia del sistema (DPC)

func (s *Server) handleDPC(w http.ResponseWriter, r *http.Request) {
	if s.isDemo() {
		time.Sleep(time.Second)
		writeJSON(w, tools.DPCDemo())
		return
	}
	res, err := tools.DPCTest(s.Sys, 10)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, res)
}

// settingsPages son las páginas de Configuración de Windows que la interfaz puede abrir (fijas).
var settingsPages = map[string]string{"winupdate": "ms-settings:windowsupdate", "notifications": "ms-settings:notifications", "micprivacy": "ms-settings:privacy-microphone", "sound": "ms-settings:sound"}

func (s *Server) handleOpenSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Page string `json:"page"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	u, ok := settingsPages[req.Page]
	if !ok {
		fail(w, 400, "página desconocida")
		return
	}
	if !s.isDemo() {
		if _, _, err := s.Sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", u); err != nil {
			fail(w, 500, "No se pudo abrir la Configuración de Windows: "+err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}
