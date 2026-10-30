package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"fpsforge/internal/scan"
)

// El escaneo del equipo tarda varios segundos (consultas a Windows). Como el hardware casi nunca cambia,
// se guarda el último resultado: al abrir el programa la interfaz ya aparece completa y el escaneo se
// repite en segundo plano para ponerla al día.

const hwCacheMaxAge = 14 * 24 * time.Hour

func (s *Server) hwCachePath() string { return filepath.Join(s.Dir, "hardware_cache.json") }

// loadHardwareCache devuelve el último escaneo guardado, si existe, es reciente y es de un equipo real.
func (s *Server) loadHardwareCache() (scan.Hardware, bool) {
	if s.Sys.Name() != "windows" {
		return scan.Hardware{}, false
	}
	p := s.hwCachePath()
	st, err := os.Stat(p)
	if err != nil || time.Since(st.ModTime()) > hwCacheMaxAge {
		return scan.Hardware{}, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return scan.Hardware{}, false
	}
	var h scan.Hardware
	if json.Unmarshal(b, &h) != nil || h.Demo || h.CPU == "" || h.CPU == "No detectado" {
		return scan.Hardware{}, false
	}
	h.Admin = s.Sys.IsAdmin()
	return h, true
}

// saveHardwareCache guarda un escaneo correcto (no los de respaldo ni los de demostración).
func (s *Server) saveHardwareCache(h scan.Hardware) {
	if s.Sys.Name() != "windows" || h.Demo || h.CPU == "" || h.CPU == "No detectado" {
		return
	}
	b, err := json.Marshal(h)
	if err != nil {
		return
	}
	tmp := s.hwCachePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, s.hwCachePath())
	}
}
