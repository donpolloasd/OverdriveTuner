package app

import (
	"encoding/json"
	"net/http"
	"time"

	"fpsforge/internal/overlay"
	"fpsforge/internal/tweaks"
)

const configFormat = 1

// configBundle es el archivo de exportar/importar configuración.
type configBundle struct {
	App        string          `json:"app"`
	Format     int             `json:"format"`
	Version    string          `json:"version"`
	ExportedAt string          `json:"exportedAt"`
	Settings   settingsPatch   `json:"settings"`
	Overlay    *overlay.Config `json:"overlay,omitempty"`
	Tweaks     []string        `json:"tweaks"`
}

type importItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

// importPlan es lo que pasaría al importar (vista previa) o lo que pasó.
type importPlan struct {
	Applied  bool         `json:"applied"` // ajustes y overlay ya guardados
	Settings bool         `json:"settings"`
	Overlay  bool         `json:"overlay"`
	ToApply  []importItem `json:"toApply"`
	Already  []importItem `json:"already"`
	Skipped  []importItem `json:"skipped"`
}

func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	st := s.loadSettings()
	oc := overlay.LoadConfig(s.Dir)
	ids := s.Eng.AppliedIDs()
	if ids == nil {
		ids = []string{}
	}
	var hz *int
	if st.HzManual {
		hz = &st.TargetHz // si la frecuencia es automática no se exporta: cada PC detecta la suya
	}
	b := configBundle{App: "OverdriveTuner", Format: configFormat, Version: Version, ExportedAt: time.Now().Format(time.RFC3339),
		Settings: settingsPatch{TargetHz: hz, Lang: &st.Lang, IntroOff: &st.IntroOff, IntroVol: &st.IntroVol},
		Overlay:  &oc, Tweaks: ids}
	w.Header().Set("Content-Disposition", `attachment; filename="overdrivetuner-config.json"`)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	_ = enc.Encode(b)
}

// planImport clasifica los ajustes del archivo según este equipo.
func (s *Server) planImport(b configBundle) importPlan {
	hw := s.hardware(false)
	p := importPlan{ToApply: []importItem{}, Already: []importItem{}, Skipped: []importItem{}}
	seen := map[string]bool{}
	for _, id := range b.Tweaks {
		if seen[id] || len(p.ToApply)+len(p.Already)+len(p.Skipped) >= 500 {
			continue
		}
		seen[id] = true
		tw, ok := tweaks.ByID(id)
		if !ok {
			p.Skipped = append(p.Skipped, importItem{ID: id, Name: id, Reason: "No existe en esta versión"})
			continue
		}
		it := importItem{ID: id, Name: tw.Name}
		switch {
		case tw.AC != "":
			it.Reason = "Afecta a los anti-cheat de kernel: aplicalo a mano, con doble confirmación"
			p.Skipped = append(p.Skipped, it)
		case tw.Risk >= 2:
			it.Reason = "Ajuste avanzado: aplicalo a mano"
			p.Skipped = append(p.Skipped, it)
		case !hw.Has(tw.Cond):
			it.Reason = "No compatible con este equipo"
			p.Skipped = append(p.Skipped, it)
		case s.Eng.Applied(id):
			p.Already = append(p.Already, it)
		default:
			p.ToApply = append(p.ToApply, it)
		}
	}
	return p
}

func (s *Server) handleConfigImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bundle configBundle `json:"bundle"`
		Apply  bool         `json:"apply"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	b := req.Bundle
	if b.App != "OverdriveTuner" || b.Format != configFormat {
		fail(w, 400, "Este archivo no es una configuración de OverdriveTuner (o es de una versión incompatible).")
		return
	}
	st := s.loadSettings()
	if msg := b.Settings.apply(&st); msg != "" {
		fail(w, 400, "Configuración no válida: "+msg)
		return
	}
	var oc overlay.Config
	if b.Overlay != nil {
		oc = *b.Overlay
		if !overlay.ValidExe(oc.GameExe) {
			fail(w, 400, "Configuración no válida: el juego del overlay debe ser un programa como cs2.exe")
			return
		}
		if msg := oc.HotkeyConflict(); msg != "" {
			fail(w, 400, "Configuración no válida: "+msg)
			return
		}
	}
	p := s.planImport(b)
	p.Settings = true
	p.Overlay = b.Overlay != nil
	if req.Apply {
		if err := s.saveSettings(st); err != nil {
			fail(w, 500, err.Error())
			return
		}
		if b.Overlay != nil {
			if err := overlay.SaveConfig(s.Dir, oc); err != nil {
				fail(w, 500, err.Error())
				return
			}
		}
		p.Applied = true
	}
	writeJSON(w, p)
}
