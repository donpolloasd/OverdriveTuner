package app

import (
	"net/http"
	"time"

	"fpsforge/internal/extras"
	"fpsforge/internal/scan"
	"fpsforge/internal/tweaks"
)

// TweakView es lo que ve la interfaz de cada tweak.
type TweakView struct {
	ID       string `json:"id"`
	Cat      string `json:"cat"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Warn     string `json:"warn,omitempty"`
	Risk     int    `json:"risk"`
	Rec      bool   `json:"rec"`
	Comp     bool   `json:"comp"`
	Reboot   bool   `json:"reboot"`
	Test     bool   `json:"test"`
	AC       string `json:"ac,omitempty"`
	Suitable bool   `json:"suitable"`
	WhyNot   string `json:"whyNot,omitempty"`
	Applied  bool   `json:"applied"`         // aplicado por el programa (se puede revertir con copia)
	Active   bool   `json:"active"`          // los valores ya están en ese estado
	Drift    bool   `json:"drift,omitempty"` // lo aplicó el programa, pero Windows (u otro programa) lo deshizo
}

func (s *Server) views(hw scan.Hardware, scanning bool, lang string) []TweakView {
	all := tweaks.All()
	out := make([]TweakView, 0, len(all))
	detected := s.Eng.DetectAll(all)
	for _, t := range all {
		applied := s.Eng.Applied(t.ID)
		if t.Hidden && !applied && !detected[t.ID] {
			continue // ajuste retirado del catálogo visible
		}
		if vendorCond(t.Cond) && !scanning && !hw.Has(t.Cond) {
			continue // ajuste de otro fabricante de GPU: no se muestra
		}
		v := TweakView{ID: t.ID, Cat: t.Cat, Name: t.Name, Desc: t.Desc, Warn: t.Warn, Risk: t.Risk,
			Rec: t.Rec, Comp: t.Comp, Reboot: t.Reboot, Test: t.Test, AC: t.AC, Suitable: scanning || hw.Has(t.Cond), Applied: applied}
		if !v.Suitable && !scanning {
			v.WhyNot = scan.CondText(t.Cond)
		}
		v.Active = applied || detected[t.ID]
		v.Drift = t.Watch && applied && !detected[t.ID]
		if lang == "en" {
			if e, ok := tweaks.English(t.ID); ok {
				v.Name, v.Desc, v.Warn, v.AC = e.Name, e.Desc, e.Warn, e.AC
			}
		}
		out = append(out, v)
	}
	return out
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	// No espera al escaneo: la interfaz aparece ya y se completa cuando termina.
	hw, ready := s.hardwareIfReady()
	st := s.loadSettings()
	lang := st.Lang
	if q := r.URL.Query().Get("lang"); q == "es" || q == "en" {
		lang = q // la interfaz manda el idioma efectivo (el del sistema si no eligió uno)
	}
	cats := tweaks.Categories()
	if lang == "en" {
		for i := range cats {
			if e, ok := tweaks.EnglishCategory(cats[i].ID); ok {
				cats[i].Name, cats[i].Desc = e.Name, e.Desc
			}
		}
	}
	writeJSON(w, map[string]any{
		"version":    Version,
		"scanning":   !ready,
		"hardware":   hw,
		"categories": cats,
		"tweaks":     s.views(hw, !ready, lang),
		"admin":      s.Sys.IsAdmin(),
		"platform":   s.Sys.Name(),
		"settings":   st,
		"lang":       lang,
	})
}

func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.hardware(true))
}

type idReq struct {
	ID  string   `json:"id"`
	IDs []string `json:"ids"`
	// Confirm cuenta las confirmaciones del usuario (2 = doble permiso).
	Confirm int `json:"confirm"`
}

func (s *Server) requireAdmin(w http.ResponseWriter) bool {
	if !s.Sys.IsAdmin() {
		fail(w, http.StatusForbidden, "Se necesitan permisos de administrador. Cierra el programa y ábrelo con «Ejecutar como administrador».")
		return false
	}
	return true
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || req.ID == "" {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	tw, ok := tweaks.ByID(req.ID)
	if !ok {
		fail(w, 404, "tweak desconocido")
		return
	}
	if !s.hardware(false).Has(tw.Cond) {
		fail(w, 409, "No compatible con este equipo: "+scan.CondText(tw.Cond))
		return
	}
	if tw.AC != "" && req.Confirm < 2 {
		fail(w, 428, "Este ajuste desactiva una protección que exigen los anti-cheat (anti-cheat de kernel): "+tw.AC+". Necesita doble confirmación del usuario.")
		return
	}
	res := s.Eng.Apply(req.ID)
	s.logChange(req.ID, "apply", res)
	writeJSON(w, res)
}

func (s *Server) handleRevert(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || req.ID == "" {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	res := s.Eng.Revert(req.ID)
	s.logChange(req.ID, "revert", res)
	writeJSON(w, res)
}

// handleProfile devuelve los IDs que corresponde aplicar para un perfil.
func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	hw := s.hardware(false)
	ids := []string{}
	for _, t := range tweaks.All() {
		if t.Cat == "apps" || t.Risk >= 2 || t.AC != "" || t.Test || !hw.Has(t.Cond) || s.Eng.Applied(t.ID) {
			continue
		}
		switch req.Name {
		case "recommended":
			if t.Rec {
				ids = append(ids, t.ID)
			}
		case "competitive":
			if t.Rec || t.Comp {
				ids = append(ids, t.ID)
			}
		}
	}
	writeJSON(w, map[string]any{"ids": ids})
}

func (s *Server) handleRestorePoint(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w) {
		return
	}
	if err := extras.CreateRestorePoint(s.Sys, "OverdriveTuner - antes de optimizar"); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleCleanScan(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, extras.Scan())
}

func (s *Server) handleCleanRun(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	writeJSON(w, extras.Run(req.IDs))
}

func (s *Server) handleStartup(w http.ResponseWriter, r *http.Request) {
	items, err := extras.ListStartup(s.Sys)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	if items == nil {
		items = []extras.StartupItem{}
	}
	writeJSON(w, items)
}

func (s *Server) handleStartupSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source  string `json:"source"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || req.Name == "" {
		fail(w, 400, "petición no válida")
		return
	}
	if err := extras.SetStartup(s.Sys, req.Source, req.Name, req.Enabled); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.lastBeat = time.Now()
	s.beaten = true
	s.mu.Unlock()
	// «update» avisa a la ventana que ya hay una versión nueva instalada en el disco (para ofrecer reiniciar).
	writeJSON(w, map[string]any{"ok": true, "update": s.Up.Applied()})
}

func (s *Server) handleQuit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true})
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.Close()
	}()
}

// vendorCond dice si la condición es un fabricante de GPU: esos ajustes solo se muestran si hay una.
func vendorCond(c string) bool { return c == "amd" || c == "nvidia" || c == "intel" }
