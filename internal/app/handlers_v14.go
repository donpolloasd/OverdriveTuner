package app

import (
	"net/http"
	"strings"

	"fpsforge/internal/bench"
	"fpsforge/internal/history"
	"fpsforge/internal/tools"
	"fpsforge/internal/tweaks"
)

// logChange anota un cambio en el historial.
func (s *Server) logChange(id, action string, res tweaks.Result) {
	if s.History == nil {
		return
	}
	name := id
	if tw, ok := tweaks.ByID(id); ok {
		name = tw.Name
	}
	s.History.Add(history.Entry{ID: id, Name: name, Action: action, OK: res.OK, Error: strings.Join(res.Errors, "; ")})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.History.Clear()
	}
	writeJSON(w, s.History.List())
}

func (s *Server) handleMicrocuts(w http.ResponseWriter, r *http.Request) {
	ses, err := s.readSession(r.URL.Query().Get("id"))
	if err != nil {
		fail(w, 404, "sesión no encontrada")
		return
	}
	writeJSON(w, bench.AnalyzeSpikes(ses.Frames, ses.Run.Seconds, ses.Spikes))
}

func (s *Server) handleDNSBench(w http.ResponseWriter, r *http.Request) {
	res := tools.DNSBench(s.DNSLookup)
	for i := range res {
		if tw, ok := tweaks.ByID(res[i].TweakID); ok {
			res[i].Applied = s.Eng.Applied(tw.ID)
		}
	}
	writeJSON(w, res)
}

func (s *Server) handleNetMon(w http.ResponseWriter, r *http.Request) {
	ps, err := tools.NetMon(s.Sys)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, ps)
}
