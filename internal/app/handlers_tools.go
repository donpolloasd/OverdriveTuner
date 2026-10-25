package app

import (
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
	"time"

	"fpsforge/internal/bench"
	"fpsforge/internal/bisect"
	"fpsforge/internal/diag"
	"fpsforge/internal/tools"
	"fpsforge/internal/tweaks"
)

func (s *Server) handleNetTest(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, tools.NetTest(s.Sys))
}

func (s *Server) handleDisks(w http.ResponseWriter, r *http.Request) {
	ds, err := tools.Disks(s.Sys)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, ds)
}

func (s *Server) handleVideo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, tools.Video(s.Sys, s.loadSettings().targetHz(s.hardware(false))))
}

// ---- búsqueda del ajuste culpable ----

func (s *Server) bisectCandidates() []string {
	var ids []string
	for _, id := range s.Eng.AppliedIDs() {
		if tw, ok := tweaks.ByID(id); ok && tw.AC == "" && tw.Cat != "apps" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *Server) runActions(a bisect.Actions) []string {
	var errs []string
	for _, id := range a.Revert {
		res := s.Eng.Revert(id)
		s.logChange(id, "revert", res)
		if !res.OK {
			errs = append(errs, id+": "+strings.Join(res.Errors, "; "))
		}
	}
	for _, id := range a.Apply {
		res := s.Eng.Apply(id)
		s.logChange(id, "apply", res)
		if !res.OK {
			errs = append(errs, id+": "+strings.Join(res.Errors, "; "))
		}
	}
	return errs
}

func (s *Server) bisectView(st bisect.State, errs []string) map[string]any {
	names := map[string]string{}
	for _, id := range append(append([]string{}, st.Suspects...), st.Tested...) {
		if tw, ok := tweaks.ByID(id); ok {
			names[id] = tw.Name
		}
	}
	if st.Culprit != "" {
		if tw, ok := tweaks.ByID(st.Culprit); ok {
			names[st.Culprit] = tw.Name
		}
	}
	return map[string]any{"state": st, "names": names, "errors": errs, "available": len(s.bisectCandidates())}
}

func (s *Server) handleBisect(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.bisectView(bisect.Load(s.Dir), nil))
}

func (s *Server) handleBisectStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireAdmin(w) {
		return
	}
	if old := bisect.Load(s.Dir); old.Active {
		fail(w, 409, "Ya hay una búsqueda en curso.")
		return
	}
	st, act, err := bisect.Start(s.bisectCandidates())
	if err != nil {
		fail(w, 422, err.Error())
		return
	}
	errs := s.runActions(act)
	_ = bisect.Save(s.Dir, st)
	writeJSON(w, s.bisectView(st, errs))
}

func (s *Server) handleBisectAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Good   bool `json:"good"`
		Cancel bool `json:"cancel"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || !s.requireAdmin(w) {
		return
	}
	st := bisect.Load(s.Dir)
	var act bisect.Actions
	var err error
	if req.Cancel {
		st, act = st.Cancel()
	} else if st, act, err = st.Answer(req.Good); err != nil {
		fail(w, 409, err.Error())
		return
	}
	errs := s.runActions(act)
	_ = bisect.Save(s.Dir, st)
	writeJSON(w, s.bisectView(st, errs))
}

// ---- informe ----

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	hw := s.hardware(false)
	var b strings.Builder
	e := html.EscapeString
	fmt.Fprintf(&b, `<!doctype html><html lang="es"><meta charset="utf-8"><title>Informe OverdriveTuner</title>
<style>body{font:15px/1.5 Segoe UI,sans-serif;max-width:860px;margin:30px auto;padding:0 16px;color:#1b1d2a}h1{margin-bottom:0}h2{margin-top:28px;border-bottom:1px solid #ddd;padding-bottom:4px}table{border-collapse:collapse;width:100%%}td,th{border-bottom:1px solid #eee;padding:6px 8px;text-align:left}.bad{color:#c0243f}.warn{color:#a86a00}.ok{color:#127a58}small{color:#666}</style>
<h1>Informe de OverdriveTuner %s</h1><small>%s · %s</small>`, e(Version), e(hw.Computer), time.Now().Format("02/01/2006 15:04"))
	fmt.Fprintf(&b, `<h2>Equipo</h2><table><tr><th>Procesador</th><td>%s</td></tr><tr><th>Memoria</th><td>%d GB a %d MT/s (nominal %d)</td></tr><tr><th>Sistema</th><td>%s (compilación %d)</td></tr>`,
		e(hw.CPU), hw.RAMGB, hw.RAMMTs, hw.RAMRated, e(hw.OS), hw.Build)
	for _, g := range hw.GPUs {
		act := ""
		if g.Active {
			act = " — monitor conectado acá"
		}
		fmt.Fprintf(&b, `<tr><th>Gráficos</th><td>%s%s</td></tr>`, e(g.Name), act)
	}
	b.WriteString("</table>")
	cs := diag.Run(s.Sys, hw, s.loadSettings().targetHz(hw))
	b.WriteString(`<h2>Diagnóstico de fluidez</h2><table>`)
	for _, c := range cs {
		fmt.Fprintf(&b, `<tr><td class="%s"><b>%s</b></td><td>%s</td></tr>`, e(c.Status), e(c.Title), e(c.Detail))
	}
	b.WriteString("</table><h2>Ajustes aplicados</h2>")
	ids := s.Eng.AppliedIDs()
	sort.Strings(ids)
	if len(ids) == 0 {
		b.WriteString("<p>Ninguno.</p>")
	} else {
		b.WriteString("<ul>")
		for _, id := range ids {
			if tw, ok := tweaks.ByID(id); ok {
				fmt.Fprintf(&b, "<li>%s</li>", e(tw.Name))
			}
		}
		b.WriteString("</ul>")
	}
	gs := bench.Groups(s.Bench.List())
	if len(gs) > 0 {
		b.WriteString(`<h2>Benchmark</h2><table><tr><th>Estado</th><th>Corridas</th><th>FPS prom.</th><th>Desvío</th><th>1% low</th></tr>`)
		for _, g := range gs {
			fmt.Fprintf(&b, "<tr><td>%s</td><td>%d</td><td>%.1f</td><td>±%.1f</td><td>%.1f</td></tr>", e(g.Label), g.N, g.Mean, g.SD, g.Low1)
		}
		b.WriteString("</table>")
		for _, v := range bench.Compares(gs) {
			fmt.Fprintf(&b, "<p><b>%s → %s: %+.1f%%</b><br>%s</p>", e(v.A), e(v.B), v.DiffPct, e(v.Text))
		}
	}
	b.WriteString(`<p><small>Las mediciones propias del overlay y las pegadas de otros programas usan métodos distintos para el «1% low»; compará solo corridas del mismo origen.</small></p></html>`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="informe-overdrivetuner.html"`)
	_, _ = w.Write([]byte(b.String()))
}
