package overlay

import (
	"strings"
	"time"

	"fpsforge/internal/fpslog"
	"fpsforge/internal/memory"
	"fpsforge/internal/procs"
	"fpsforge/internal/system"
)

type profileState struct {
	prevPlan string
}

func planGUID(out string) string {
	for _, f := range strings.Fields(out) {
		f = strings.Trim(f, "()")
		if guidRe.MatchString(f) {
			return f
		}
	}
	return ""
}

// applyProfile aplica el perfil del juego: plan de energía, prioridad y
// cierre de los programas que el usuario eligió. Se deshace al cerrar el juego
// (salvo los programas cerrados).
func (e *Engine) applyProfile(g Game) {
	sys := e.d.Sys
	if sys == nil {
		return
	}
	ps := &profileState{}
	var notes []string
	if g.PowerPlan != "" {
		if out, _, err := sys.Exec(10*time.Second, "powercfg.exe", "/getactivescheme"); err == nil {
			ps.prevPlan = planGUID(out)
		}
		if _, _, err := sys.Exec(10*time.Second, "powercfg.exe", "/setactive", g.PowerPlan); err != nil {
			ps.prevPlan = ""
			notes = append(notes, "no se pudo cambiar el plan de energía")
		}
	}
	for _, n := range g.Close {
		if err := procs.KillByName(sys, n); err != nil {
			notes = append(notes, err.Error())
		}
	}
	if _, demo := sys.(*system.Fake); g.FreeRAM && !demo { // en demostración y en las pruebas no se toca la memoria real
		go func() { // tarda unos segundos: no se espera; el juego no se toca
			if _, err := memory.Free(memory.Options{Game: g.Exe}); err != nil {
				fpslog.Diag("perfil de %s: no se pudo liberar RAM: %v", g.Exe, err)
			}
		}()
	}
	if g.Priority {
		name := strings.TrimSuffix(g.Exe, ".exe")
		_, _ = sys.PowerShell("Get-Process -Name '"+strings.ReplaceAll(name, "'", "''")+"' -ErrorAction SilentlyContinue | ForEach-Object { $_.PriorityClass='High' }", 15*time.Second)
	}
	e.mu.Lock()
	e.profile = ps
	if len(notes) > 0 {
		e.msg = "Perfil: " + strings.Join(notes, "; ")
	}
	e.mu.Unlock()
}

// restoreProfile devuelve el plan de energía anterior.
func (e *Engine) restoreProfile() {
	e.mu.Lock()
	ps := e.profile
	e.profile = nil
	e.mu.Unlock()
	if ps != nil && ps.prevPlan != "" && e.d.Sys != nil {
		_, _, _ = e.d.Sys.Exec(10*time.Second, "powercfg.exe", "/setactive", ps.prevPlan)
	}
}
