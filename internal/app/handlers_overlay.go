package app

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"fpsforge/internal/autostart"
	"fpsforge/internal/bench"
	"fpsforge/internal/memory"
	"fpsforge/internal/overlay"
	"fpsforge/internal/procs"
	"fpsforge/internal/system"
)

var sessionIDRe = regexp.MustCompile(`^\d{8}-\d{6}$`)

func (s *Server) overlayRunning() (overlay.State, bool) {
	st, ok := overlay.LoadState(s.Dir)
	return st, ok && st.Alive(time.Now())
}

// spawnOverlay abre el overlay como proceso aparte (hereda los permisos de administrador).
func (s *Server) spawnOverlay() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"--overlay"}
	if _, demo := s.Sys.(*system.Fake); demo {
		args = append(args, "--demo")
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

type plan struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

var planRe = regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\s+\(([^)]*)\)`)

func (s *Server) powerPlans() []plan {
	out, _, err := s.Sys.Exec(10*time.Second, "powercfg.exe", "/list")
	if err != nil {
		return []plan{}
	}
	ps := []plan{}
	for _, m := range planRe.FindAllStringSubmatch(out, -1) {
		ps = append(ps, plan{m[1], m[2]})
	}
	return ps
}

func (s *Server) handleOverlay(w http.ResponseWriter, r *http.Request) {
	st, alive := s.overlayRunning()
	writeJSON(w, map[string]any{"config": overlay.LoadConfig(s.Dir), "running": alive, "state": st, "plans": s.powerPlans()})
}

func (s *Server) handleOverlayConfig(w http.ResponseWriter, r *http.Request) {
	var c overlay.Config
	if r.Method != http.MethodPost || readJSON(r, &c) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	if !overlay.ValidExe(c.GameExe) {
		fail(w, 400, "El juego debe ser un nombre de programa como cs2.exe")
		return
	}
	for _, n := range c.CloseList {
		if procs.Protected(n) {
			fail(w, 400, "«"+n+"» es parte de Windows o de OverdriveTuner y no se puede cerrar desde el Modo juego.")
			return
		}
	}
	if msg := c.HotkeyConflict(); msg != "" {
		fail(w, 400, msg)
		return
	}
	if err := overlay.SaveConfig(s.Dir, c); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, overlay.LoadConfig(s.Dir))
}

func (s *Server) handleOverlayStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	if _, alive := s.overlayRunning(); !alive {
		if err := s.SpawnOverlay(); err != nil {
			fail(w, 500, "No se pudo abrir el overlay: "+err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleOverlayStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	if err := overlay.SendCmd(s.Dir, "quit", ""); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleOverlayRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		Label  string `json:"label"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || (req.Action != "start" && req.Action != "stop") {
		fail(w, 400, "petición no válida")
		return
	}
	if _, alive := s.overlayRunning(); !alive {
		fail(w, 409, "Abrí primero el overlay.")
		return
	}
	if err := overlay.SendCmd(s.Dir, "record_"+req.Action, strings.TrimSpace(req.Label)); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) readSession(id string) (bench.Session, error) {
	var ses bench.Session
	if !sessionIDRe.MatchString(id) {
		return ses, os.ErrInvalid
	}
	b, err := os.ReadFile(filepath.Join(s.Dir, "sessions", id+".json"))
	if err != nil {
		return ses, err
	}
	return ses, json.Unmarshal(b, &ses)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	ents, _ := os.ReadDir(filepath.Join(s.Dir, "sessions"))
	list := []bench.Session{}
	for _, e := range ents {
		id := strings.TrimSuffix(e.Name(), ".json")
		if ses, err := s.readSession(id); err == nil {
			ses.Frames = nil
			list = append(list, ses)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID > list[j].ID })
	writeJSON(w, list)
}

func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	ses, err := s.readSession(r.URL.Query().Get("id"))
	if err != nil {
		fail(w, 404, "sesión no encontrada")
		return
	}
	writeJSON(w, ses)
}

func (s *Server) handleSessionImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	ses, err := s.readSession(req.ID)
	if err != nil {
		fail(w, 404, "sesión no encontrada")
		return
	}
	if ses.Imported {
		fail(w, 409, "Esa sesión ya está en el benchmark.")
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = ses.Label
	}
	if len(label) > 60 {
		label = label[:60]
	}
	if _, err := s.Bench.AddRun(ses.Run, label); err != nil {
		fail(w, 500, err.Error())
		return
	}
	ses.Imported = true
	_ = overlayWrite(filepath.Join(s.Dir, "sessions", ses.ID+".json"), ses)
	writeJSON(w, s.benchView())
}

func (s *Server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || !sessionIDRe.MatchString(req.ID) {
		fail(w, 400, "petición no válida")
		return
	}
	_ = os.Remove(filepath.Join(s.Dir, "sessions", req.ID+".json"))
	writeJSON(w, map[string]any{"ok": true})
}

func overlayWrite(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AutoOpenOverlay abre el overlay al iniciar el programa si así está configurado y no hay uno abierto.
func (s *Server) AutoOpenOverlay() {
	if !overlay.LoadConfig(s.Dir).AutoOpen || s.SpawnOverlay == nil {
		return
	}
	if _, alive := s.overlayRunning(); alive {
		return
	}
	_ = s.SpawnOverlay()
}

// handleMemoryFree libera memoria RAM (vacía el conjunto de trabajo de los programas y la lista en espera).
func (s *Server) handleMemoryFree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	game := overlay.FindGame(s.Sys, nil)
	res, err := memory.Free(memory.Options{Game: game})
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, res)
}

// handleAutostart consulta o cambia el inicio del overlay con Windows (tarea programada).
func (s *Server) handleAutostart(w http.ResponseWriter, r *http.Request) {
	_, demo := s.Sys.(*system.Fake)
	if r.Method == http.MethodPost {
		var req struct {
			Enabled *bool `json:"enabled"`
			Run     bool  `json:"run"`
		}
		if readJSON(r, &req) != nil {
			fail(w, 400, "petición no válida")
			return
		}
		if !s.requireAdmin(w) {
			return
		}
		var err error
		switch {
		case req.Run:
			if !demo {
				err = autostart.RunNow(s.Sys)
			}
		case req.Enabled != nil && *req.Enabled:
			var exe string
			if exe, err = os.Executable(); err == nil {
				err = autostart.Enable(s.Sys, exe)
			}
			s.autostartDemo = true
		case req.Enabled != nil:
			err = autostart.Disable(s.Sys)
			s.autostartDemo = false
		}
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	info := autostart.Status(s.Sys)
	if demo {
		info = autostart.Info{Registered: s.autostartDemo, State: "Ready", LastResult: 267011}
	}
	_, running := s.overlayRunning()
	writeJSON(w, map[string]any{"enabled": info.Registered, "state": info.State, "lastRun": info.LastRun, "lastResult": info.LastResult, "overlayRunning": running})
}
