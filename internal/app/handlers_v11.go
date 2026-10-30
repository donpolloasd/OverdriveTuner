package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fpsforge/internal/bench"
	"fpsforge/internal/diag"
	"fpsforge/internal/procs"
	"fpsforge/internal/scan"
)

type settings struct {
	TargetHz int    `json:"targetHz"`
	Lang     string `json:"lang"`     // es, en (vacío = según el sistema)
	IntroOff bool   `json:"introOff"` // desactiva la animación de inicio
	IntroVol int    `json:"introVol"` // 0-100
	TourSeen bool   `json:"tourSeen"` // ya vio (o descartó) el recorrido de la primera vez
	HzManual bool   `json:"hzManual"` // la frecuencia la fijó el usuario; si no, se usa la máxima que detecta Windows
	// AutoUpdate: instala solo las versiones nuevas que salen en GitHub. Lo elige el usuario en el Setup;
	// si falta (instalaciones anteriores) queda desactivado: ahí solo se actualiza con el Parche.
	AutoUpdate bool `json:"autoUpdate"`
}

// targetHz devuelve los Hz que debería tener el monitor: el valor fijado a mano o, si no hay, el máximo
// que Windows ofrece (0 = no se sabe, no se compara).
func (st settings) targetHz(hw scan.Hardware) int {
	if st.HzManual {
		return st.TargetHz
	}
	if hw.MaxHz >= 30 {
		return hw.MaxHz
	}
	return 0
}

func defaultSettings() settings { return settings{TargetHz: 240, IntroVol: 70} }

func (st *settings) normalize() {
	if st.TargetHz < 30 || st.TargetHz > 1000 {
		st.TargetHz = 240
	}
	if st.Lang != "es" && st.Lang != "en" {
		st.Lang = ""
	}
	if st.IntroVol < 0 || st.IntroVol > 100 {
		st.IntroVol = 70
	}
}

// readSettings lee settings.json de la carpeta de datos (con los valores por defecto si no existe).
func readSettings(dir string) settings {
	st := defaultSettings()
	if b, err := os.ReadFile(filepath.Join(dir, "settings.json")); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	st.normalize()
	return st
}

func writeSettings(dir string, st settings) error {
	st.normalize()
	b, _ := json.Marshal(st)
	_ = os.MkdirAll(dir, 0o755)
	return os.WriteFile(filepath.Join(dir, "settings.json"), b, 0o644)
}

func (s *Server) loadSettings() settings { return readSettings(s.Dir) }

func (s *Server) saveSettings(st settings) error { return writeSettings(s.Dir, st) }

// DefaultDataDir es la carpeta de datos del programa (la misma que usa la aplicación).
func DefaultDataDir() string {
	if d := os.Getenv("ProgramData"); d != "" {
		return filepath.Join(d, "FPSForge")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".fpsforge")
}

// AutoUpdateOn dice si las actualizaciones automáticas están activadas en la carpeta de datos (lo usa el Setup).
func AutoUpdateOn(dir string) bool { return readSettings(dir).AutoUpdate }

// SetAutoUpdate guarda la elección de actualizaciones automáticas sin tocar el resto de los ajustes.
func SetAutoUpdate(dir string, on bool) error {
	st := readSettings(dir)
	st.AutoUpdate = on
	return writeSettings(dir, st)
}

// settingsPatch solo cambia los campos que vienen en la petición.
type settingsPatch struct {
	TargetHz *int    `json:"targetHz"`
	Lang     *string `json:"lang"`
	IntroOff *bool   `json:"introOff"`
	IntroVol *int    `json:"introVol"`
	TourSeen *bool   `json:"tourSeen"`
	HzAuto   *bool   `json:"hzAuto"` // true: vuelve a usar la frecuencia máxima detectada
	// AutoUpdate activa o desactiva las actualizaciones automáticas.
	AutoUpdate *bool `json:"autoUpdate"`
}

func (p settingsPatch) apply(st *settings) string {
	if p.TargetHz != nil {
		if *p.TargetHz < 30 || *p.TargetHz > 1000 {
			return "frecuencia no válida"
		}
		st.TargetHz = *p.TargetHz
		st.HzManual = true
	}
	if p.HzAuto != nil && *p.HzAuto {
		st.HzManual = false
	}
	if p.Lang != nil {
		if *p.Lang != "es" && *p.Lang != "en" && *p.Lang != "" {
			return "idioma no válido"
		}
		st.Lang = *p.Lang
	}
	if p.TourSeen != nil {
		st.TourSeen = *p.TourSeen
	}
	if p.IntroOff != nil {
		st.IntroOff = *p.IntroOff
	}
	if p.AutoUpdate != nil {
		st.AutoUpdate = *p.AutoUpdate
	}
	if p.IntroVol != nil {
		if *p.IntroVol < 0 || *p.IntroVol > 100 {
			return "volumen no válido"
		}
		st.IntroVol = *p.IntroVol
	}
	return ""
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	st := s.loadSettings()
	if r.Method == http.MethodPost {
		var in settingsPatch
		if readJSON(r, &in) != nil {
			fail(w, 400, "petición no válida")
			return
		}
		was := st.AutoUpdate
		if msg := in.apply(&st); msg != "" {
			fail(w, 400, msg)
			return
		}
		if err := s.saveSettings(st); err != nil {
			fail(w, 500, err.Error())
			return
		}
		if st.AutoUpdate && !was && s.Up != nil {
			go s.Up.Cycle(true) // recién activadas: se busca de inmediato, sin esperar a la próxima vuelta
		}
	}
	writeJSON(w, st)
}

func (s *Server) handleDiag(w http.ResponseWriter, r *http.Request) {
	hw := s.hardware(r.URL.Query().Get("fresh") == "1")
	st := s.loadSettings()
	hz := st.targetHz(hw)
	cs := diag.Run(s.Sys, hw, hz)
	bad, warn := diag.Summary(cs)
	writeJSON(w, map[string]any{"checks": cs, "bad": bad, "warn": warn, "targetHz": hz, "detectedHz": hw.MaxHz, "manual": st.HzManual})
}

func (s *Server) handleDiagAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "acción no válida")
		return
	}
	var err error
	switch {
	case req.Action == "open_display":
		err = diag.OpenDisplay(s.Sys)
	case strings.HasPrefix(req.Action, "open_driver_"):
		err = diag.OpenDriverPage(s.Sys, strings.TrimPrefix(req.Action, "open_driver_"))
	default:
		fail(w, 400, "acción no válida")
		return
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleProcs(w http.ResponseWriter, r *http.Request) {
	ps, err := procs.Top(s.Sys)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, ps)
}

func (s *Server) handleProcKill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PID  int    `json:"pid"`
		Name string `json:"name"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	if err := procs.Kill(s.Sys, req.PID, req.Name); err != nil {
		fail(w, 409, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) benchView() map[string]any {
	es := s.Bench.List()
	if es == nil {
		es = []bench.Entry{}
	}
	gs := bench.Groups(es)
	if gs == nil {
		gs = []bench.Group{}
	}
	vs := bench.Compares(gs)
	if vs == nil {
		vs = []bench.Verdict{}
	}
	return map[string]any{"entries": es, "groups": gs, "verdicts": vs}
}

func (s *Server) handleBench(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.benchView())
}

func (s *Server) handleBenchAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text     string `json:"text"`
		Label    string `json:"label"`
		Baseline bool   `json:"baseline"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || len(req.Text) > 200000 {
		fail(w, 400, "petición no válida")
		return
	}
	if len(req.Label) > 60 {
		req.Label = req.Label[:60]
	}
	if _, err := s.Bench.Add(req.Text, req.Label, req.Baseline); err != nil {
		fail(w, 422, err.Error())
		return
	}
	writeJSON(w, s.benchView())
}

func (s *Server) handleBenchDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID  int  `json:"id"`
		All bool `json:"all"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	var err error
	if req.All {
		err = s.Bench.Clear()
	} else {
		err = s.Bench.Delete(req.ID)
	}
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, s.benchView())
}

func (s *Server) handleRevertAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	results := s.Eng.RevertAll()
	for _, res := range results {
		s.logChange(res.ID, "revert", res)
	}
	writeJSON(w, results)
}
