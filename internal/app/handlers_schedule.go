package app

import (
	"net/http"
	"os"
	"sync"

	"fpsforge/internal/schedule"
	"fpsforge/internal/system"
)

var (
	schedDemoMu sync.Mutex
	schedDemo   = map[string]schedule.Info{} // en demostración no hay Programador de tareas: se recuerda en memoria
)

func (s *Server) scheduleExe() string {
	if s.Up != nil && s.Up.ExePath != "" {
		return s.Up.ExePath
	}
	exe, _ := os.Executable()
	return exe
}

func (s *Server) scheduleInfos() []schedule.Info {
	_, demo := s.Sys.(*system.Fake)
	out := make([]schedule.Info, 0, len(schedule.Jobs))
	for _, j := range schedule.Jobs {
		var i schedule.Info
		if demo {
			schedDemoMu.Lock()
			i = schedDemo[j.ID]
			schedDemoMu.Unlock()
			i.ID = j.ID
			if i.Time == "" {
				i.Time = "20:00"
			}
		} else {
			i = schedule.Status(s.Sys, j.ID)
		}
		i.Last = schedule.LastMessage(s.Dir, j.ID)
		out = append(out, i)
	}
	return out
}

// handleSchedule consulta (GET) o cambia (POST) las tareas programadas de mantenimiento.
func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			ID      string `json:"id"`
			Enabled *bool  `json:"enabled"`
			Day     int    `json:"day"`
			Time    string `json:"time"`
			Run     bool   `json:"run"`
		}
		if readJSON(r, &req) != nil {
			fail(w, 400, "petición no válida")
			return
		}
		if _, ok := schedule.ByID(req.ID); !ok {
			fail(w, 404, "tarea desconocida")
			return
		}
		_, demo := s.Sys.(*system.Fake)
		var err error
		switch {
		case req.Run:
			if !demo {
				err = schedule.RunNow(s.Sys, req.ID)
			}
		case req.Enabled != nil && *req.Enabled:
			if demo {
				schedDemoMu.Lock()
				schedDemo[req.ID] = schedule.Info{ID: req.ID, Registered: true, Day: req.Day, Time: req.Time}
				schedDemoMu.Unlock()
			} else {
				err = schedule.Enable(s.Sys, req.ID, s.scheduleExe(), req.Day, req.Time)
			}
		case req.Enabled != nil:
			if demo {
				schedDemoMu.Lock()
				delete(schedDemo, req.ID)
				schedDemoMu.Unlock()
			} else {
				err = schedule.Disable(s.Sys, req.ID)
			}
		}
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, s.scheduleInfos())
}
