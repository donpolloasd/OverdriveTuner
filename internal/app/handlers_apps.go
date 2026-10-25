package app

import (
	"net/http"
	"sync"
	"time"

	"fpsforge/internal/apps"
)

var (
	appsMu     sync.Mutex
	appsCache  map[string]bool
	appsCached time.Time
)

func appsLang(r *http.Request, def string) string {
	if q := r.URL.Query().Get("lang"); q == "es" || q == "en" {
		return q
	}
	return def
}

// handleApps devuelve el catálogo de programas (sin consultar qué está instalado, que es lento).
func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	lang := appsLang(r, s.loadSettings().Lang)
	hw, ready := s.hardwareIfReady() // sin esperar al escaneo: si no terminó, se ofrecen los tres fabricantes
	vendors := []string{}
	var gpuV []string
	for _, g := range hw.GPUs {
		gpuV = append(gpuV, g.Vendor)
		if _, ok := apps.DriverPage[g.Vendor]; ok && !contains(vendors, g.Vendor) {
			vendors = append(vendors, g.Vendor)
		}
	}
	items := make([]apps.Item, 0)
	for _, it := range apps.Catalog() {
		if ready && !it.Matches(hw.CPUVendor, gpuV) {
			continue // programas de un fabricante que esta PC no tiene
		}
		items = append(items, it.Localize(lang))
	}
	writeJSON(w, map[string]any{
		"items": items, "categories": apps.LocalizeCategories(lang),
		"winget": apps.Available(s.Sys), "gpuVendors": vendors, "hwReady": ready,
	})
}

func contains(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// handleAppsInstalled consulta (con caché de un minuto) qué programas del catálogo ya están instalados.
func (s *Server) handleAppsInstalled(w http.ResponseWriter, r *http.Request) {
	fresh := r.URL.Query().Get("fresh") == "1"
	appsMu.Lock()
	defer appsMu.Unlock()
	if fresh || appsCache == nil || time.Since(appsCached) > time.Minute {
		have, err := apps.Installed(s.Sys)
		if err != nil {
			fail(w, 500, "No se pudo consultar qué está instalado: "+err.Error())
			return
		}
		appsCache, appsCached = have, time.Now()
	}
	out := map[string]bool{}
	for _, it := range apps.Catalog() {
		out[it.ID] = apps.IsInstalled(it, appsCache)
	}
	writeJSON(w, out)
}

func (s *Server) handleAppsInstall(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	it, ok := apps.ByID(req.ID)
	if !ok {
		fail(w, 404, "Programa desconocido")
		return
	}
	if it.URL != "" {
		fail(w, 400, "Este programa no se instala con winget: abrí su página oficial")
		return
	}
	res := apps.Install(s.Sys, it)
	appsMu.Lock()
	appsCache = nil // se vuelve a consultar
	appsMu.Unlock()
	writeJSON(w, res)
}

func (s *Server) handleAppsUpgrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	out, err := apps.UpgradeAll(s.Sys)
	appsMu.Lock()
	appsCache = nil
	appsMu.Unlock()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "output": out})
}

func (s *Server) handleAppsWinget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	if err := apps.InstallWinget(s.Sys); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleAppsDrivers abre la página oficial de drivers del fabricante (solo direcciones fijas).
func (s *Server) handleAppsDrivers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Vendor string `json:"vendor"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	u, ok := apps.DriverPage[req.Vendor]
	if !ok {
		fail(w, 400, "Fabricante desconocido")
		return
	}
	if _, _, err := s.Sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", u); err != nil {
		fail(w, 500, "No se pudo abrir el navegador: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "url": u})
}

// handleAppsOpen abre la página oficial de un programa del catálogo que no se instala con winget.
func (s *Server) handleAppsOpen(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	it, ok := apps.ByID(req.ID)
	if !ok || it.Site == "" {
		fail(w, 404, "Programa desconocido")
		return
	}
	if _, _, err := s.Sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", it.Site); err != nil {
		fail(w, 500, "No se pudo abrir el navegador: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "url": it.Site})
}
