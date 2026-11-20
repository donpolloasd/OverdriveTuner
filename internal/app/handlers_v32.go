package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------- copias locales de la configuración

// Las copias se guardan como archivos en <carpeta de datos>\copias. Cada una es el mismo archivo que
// «Exportar configuración», con un nombre y la fecha. Restaurar muestra primero qué se activa y qué se
// desactiva, y el usuario decide.

const maxAutoSnapshots = 15 // las automáticas más viejas se borran; las que guarda el usuario, nunca

var snapRe = regexp.MustCompile(`^\d{8}-\d{6}(-[a-z0-9-]{1,40})?\.json$`)

func (s *Server) snapshotsDir() string { return filepath.Join(s.Dir, "copias") }

type snapshotInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	At      string `json:"at"`
	Version string `json:"version"`
	Tweaks  int    `json:"tweaks"`
	Auto    bool   `json:"auto"`
}

func (s *Server) listSnapshots() []snapshotInfo {
	out := []snapshotInfo{}
	es, _ := os.ReadDir(s.snapshotsDir())
	for _, e := range es {
		if e.IsDir() || !snapRe.MatchString(e.Name()) {
			continue
		}
		b, ok := s.readSnapshot(e.Name())
		if !ok {
			continue
		}
		out = append(out, snapshotInfo{ID: e.Name(), Name: b.Name, At: b.ExportedAt, Version: b.Version, Tweaks: len(b.Tweaks), Auto: b.Auto})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID }) // las más nuevas primero
	return out
}

func (s *Server) readSnapshot(id string) (configBundle, bool) {
	var b configBundle
	if !snapRe.MatchString(id) {
		return b, false
	}
	raw, err := os.ReadFile(filepath.Join(s.snapshotsDir(), id))
	if err != nil || json.Unmarshal(raw, &b) != nil || b.App != "OverdriveTuner" {
		return b, false
	}
	return b, true
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// saveSnapshot guarda la configuración actual. Devuelve el nombre del archivo.
func (s *Server) saveSnapshot(name string, auto bool) (string, error) {
	name = strings.TrimSpace(name)
	if r := []rune(name); len(r) > 60 {
		name = string(r[:60])
	}
	if name == "" {
		name = "Copia del " + time.Now().Format("02/01/2006 15:04")
	}
	b := s.exportBundle()
	b.Name, b.Auto = name, auto
	slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(foldAccents(name)), "-"), "-")
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	id := time.Now().Format("20060102-150405")
	if slug != "" {
		id += "-" + slug
	}
	id += ".json"
	if err := os.MkdirAll(s.snapshotsDir(), 0o755); err != nil {
		return "", err
	}
	raw, _ := json.MarshalIndent(b, "", " ")
	if err := os.WriteFile(filepath.Join(s.snapshotsDir(), id), raw, 0o644); err != nil {
		return "", err
	}
	if auto {
		s.pruneAutoSnapshots()
	}
	return id, nil
}

func (s *Server) pruneAutoSnapshots() {
	n := 0
	for _, it := range s.listSnapshots() { // de la más nueva a la más vieja
		if !it.Auto {
			continue
		}
		if n++; n > maxAutoSnapshots {
			_ = os.Remove(filepath.Join(s.snapshotsDir(), it.ID))
		}
	}
}

// foldAccents saca los acentos más comunes para armar el nombre del archivo.
func foldAccents(s string) string {
	return strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n").Replace(s)
}

// handleSnapshots lista las copias (GET) o guarda una nueva (POST {name, auto}).
func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Name string `json:"name"`
			Auto bool   `json:"auto"`
		}
		if readJSON(r, &req) != nil {
			fail(w, 400, "petición no válida")
			return
		}
		id, err := s.saveSnapshot(req.Name, req.Auto)
		if err != nil {
			fail(w, 500, "No se pudo guardar la copia: "+err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "id": id})
		return
	}
	writeJSON(w, map[string]any{"dir": s.snapshotsDir(), "list": s.listSnapshots()})
}

// handleSnapshotGet devuelve una copia (para la vista previa de restaurar o para exportarla).
func (s *Server) handleSnapshotGet(w http.ResponseWriter, r *http.Request) {
	b, ok := s.readSnapshot(r.URL.Query().Get("id"))
	if !ok {
		fail(w, 404, "No se encontró esa copia.")
		return
	}
	writeJSON(w, b)
}

// handleSnapshotDelete borra una copia de la carpeta.
func (s *Server) handleSnapshotDelete(w http.ResponseWriter, r *http.Request) {
	var req idReq
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || !snapRe.MatchString(req.ID) {
		fail(w, 400, "petición no válida")
		return
	}
	if err := os.Remove(filepath.Join(s.snapshotsDir(), req.ID)); err != nil {
		fail(w, 404, "No se encontró esa copia.")
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleSnapshotFolder abre la carpeta de copias en el Explorador.
func (s *Server) handleSnapshotFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "método no permitido")
		return
	}
	_ = os.MkdirAll(s.snapshotsDir(), 0o755)
	if !s.isDemo() {
		_, _, _ = s.Sys.Exec(15*time.Second, "explorer.exe", s.snapshotsDir()) // el Explorador devuelve código 1 aunque abra bien
	}
	writeJSON(w, map[string]any{"ok": true, "dir": s.snapshotsDir()})
}
