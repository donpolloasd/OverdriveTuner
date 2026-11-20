package app

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/diskmap"
	"fpsforge/internal/extras"
)

// ---------------------------------------------------------------- espacio en disco (estilo WizTree)

// spaceState es el análisis de disco en curso o el último terminado. Uno a la vez.
type spaceState struct {
	mu      sync.Mutex
	running bool
	root    string
	started time.Time
	prog    *diskmap.Progress
	cancel  context.CancelFunc
	res     *diskmap.Result
	err     string
	dups    []diskmap.DupGroup
	dupsAll bool
	dupsAt  time.Time
}

var space spaceState

func (s *Server) handleSpace(w http.ResponseWriter, r *http.Request) {
	space.mu.Lock()
	defer space.mu.Unlock()
	out := map[string]any{"running": space.running, "root": space.root, "volumes": diskmap.Volumes(), "error": space.err}
	if space.prog != nil {
		out["files"], out["bytes"] = space.prog.Files.Load(), space.prog.Bytes.Load()
		out["elapsedMs"] = time.Since(space.started).Milliseconds()
	}
	if res := space.res; res != nil && !space.running {
		out["done"] = map[string]any{"root": res.RootPath, "files": res.Files, "dirs": res.Dirs, "bytes": res.Bytes, "skipped": res.Skipped, "tookMs": res.Took.Milliseconds()}
	}
	writeJSON(w, out)
}

// handleSpaceScan empieza a analizar una unidad o carpeta (POST {root}); {cancel:true} lo detiene.
func (s *Server) handleSpaceScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root   string `json:"root"`
		Cancel bool   `json:"cancel"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	space.mu.Lock()
	defer space.mu.Unlock()
	if req.Cancel {
		if space.cancel != nil {
			space.cancel()
		}
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	if space.running {
		fail(w, 409, "Ya hay un análisis en curso.")
		return
	}
	root := filepath.Clean(req.Root)
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() || !filepath.IsAbs(root) {
		fail(w, 400, "Elegí una unidad o carpeta que exista.")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	prog := &diskmap.Progress{}
	space.running, space.root, space.started, space.prog, space.cancel, space.err = true, root, time.Now(), prog, cancel, ""
	space.res, space.dups = nil, nil
	go func() {
		res, err := diskmap.Scan(ctx, root, prog)
		space.mu.Lock()
		defer space.mu.Unlock()
		space.running, space.cancel = false, nil
		if err != nil {
			if ctx.Err() != nil {
				space.err = "Análisis cancelado."
			} else {
				space.err = err.Error()
			}
			return
		}
		space.res = res
	}()
	writeJSON(w, map[string]any{"ok": true})
}

func spaceResult(w http.ResponseWriter) *diskmap.Result {
	space.mu.Lock()
	res := space.res
	space.mu.Unlock()
	if res == nil {
		fail(w, 409, "Primero analizá una unidad.")
	}
	return res
}

// handleSpaceDir devuelve una carpeta del análisis con sus subcarpetas más grandes.
func (s *Server) handleSpaceDir(w http.ResponseWriter, r *http.Request) {
	res := spaceResult(w)
	if res == nil {
		return
	}
	p := r.URL.Query().Get("path")
	if p == "" {
		p = res.RootPath
	}
	v, ok := res.ViewOf(p, 120)
	if !ok {
		fail(w, 404, "Esa carpeta no está en el análisis.")
		return
	}
	writeJSON(w, map[string]any{"view": v, "root": res.RootPath, "protected": diskmap.Protected(v.Path)})
}

// handleSpaceTop devuelve los archivos más grandes y el espacio por tipo.
func (s *Server) handleSpaceTop(w http.ResponseWriter, r *http.Request) {
	res := spaceResult(w)
	if res == nil {
		return
	}
	files := res.Largest(100)
	type f struct {
		diskmap.File
		Protected bool `json:"protected"`
	}
	out := make([]f, len(files))
	for i, x := range files {
		out[i] = f{x, diskmap.Protected(x.Path)}
	}
	writeJSON(w, map[string]any{"files": out, "types": res.Types(30)})
}

// handleSpaceDups busca archivos repetidos (hasta un minuto). El resultado se guarda hasta el próximo análisis.
func (s *Server) handleSpaceDups(w http.ResponseWriter, r *http.Request) {
	res := spaceResult(w)
	if res == nil {
		return
	}
	space.mu.Lock()
	cached, all := space.dups, space.dupsAll
	space.mu.Unlock()
	if cached == nil || r.Method == http.MethodPost {
		cached, all = res.Duplicates(r.Context(), time.Minute, 200)
		if cached == nil {
			cached = []diskmap.DupGroup{}
		}
		space.mu.Lock()
		space.dups, space.dupsAll, space.dupsAt = cached, all, time.Now()
		space.mu.Unlock()
	}
	writeJSON(w, map[string]any{"groups": cached, "complete": all})
}

// handleSpaceRecycle manda un archivo o carpeta a la Papelera (pide confirmación del usuario).
func (s *Server) handleSpaceRecycle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Confirm int    `json:"confirm"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || req.Path == "" {
		fail(w, 400, "petición no válida")
		return
	}
	if req.Confirm < 1 {
		fail(w, 428, "Mandar a la Papelera necesita la confirmación del usuario.")
		return
	}
	if diskmap.Protected(req.Path) {
		fail(w, 403, "«"+req.Path+"» es una carpeta del sistema o de programas instalados: no se borra desde acá.")
		return
	}
	if !s.isDemo() {
		if err := diskmap.Recycle(s.Sys, req.Path); err != nil {
			fail(w, 500, "No se pudo mandar a la Papelera: "+err.Error())
			return
		}
	}
	// Se descuenta del análisis para que la vista quede al día sin volver a recorrer el disco.
	space.mu.Lock()
	if res := space.res; res != nil {
		res.Forget(req.Path)
		for i := range space.dups {
			space.dups[i].Files = removeStr(space.dups[i].Files, req.Path)
		}
	}
	space.mu.Unlock()
	writeJSON(w, map[string]any{"ok": true})
}

func removeStr(l []string, x string) []string {
	out := l[:0]
	for _, v := range l {
		if !strings.EqualFold(v, x) {
			out = append(out, v)
		}
	}
	return out
}

// handleSpaceOpen muestra el archivo o la carpeta en el Explorador.
func (s *Server) handleSpaceOpen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || req.Path == "" {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.isDemo() {
		if err := diskmap.ShowInExplorer(s.Sys, req.Path); err != nil {
			fail(w, 404, err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}

// ---------------------------------------------------------------- Papelera y puntos de restauración

func (s *Server) handleRecycleBin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Confirm int `json:"confirm"`
		}
		if readJSON(r, &req) != nil || req.Confirm < 1 {
			fail(w, 428, "Vaciar la Papelera necesita la confirmación del usuario.")
			return
		}
		if !s.isDemo() {
			if err := extras.EmptyRecycleBin(); err != nil {
				fail(w, 500, err.Error())
				return
			}
		}
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	if s.isDemo() {
		writeJSON(w, map[string]any{"bytes": 2_480_000_000, "items": 37})
		return
	}
	b, n, err := extras.RecycleBin()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"bytes": b, "items": n})
}

func (s *Server) handleRestorePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Confirm int `json:"confirm"`
		}
		if readJSON(r, &req) != nil || req.Confirm < 2 {
			fail(w, 428, "Borrar puntos de restauración necesita doble confirmación del usuario.")
			return
		}
		if !s.requireAdmin(w) {
			return
		}
		n := 3
		if !s.isDemo() {
			var err error
			if n, err = extras.DeleteOldRestorePoints(s.Sys); err != nil {
				fail(w, 500, err.Error())
				return
			}
		}
		writeJSON(w, map[string]any{"ok": true, "deleted": n})
		return
	}
	if s.isDemo() {
		writeJSON(w, extras.RestoreInfo{Shadows: 4, Used: 9_700_000_000, Points: []extras.RestorePoint{
			{Seq: 41, Desc: "Windows Update", At: "2026-10-02 03:12"}, {Seq: 42, Desc: "OverdriveTuner - antes de optimizar", At: "2026-10-20 21:40"},
			{Seq: 43, Desc: "Instalado Programa X", At: "2026-11-05 18:02"}, {Seq: 44, Desc: "Windows Update", At: "2026-11-18 03:10"}}})
		return
	}
	if !s.requireAdmin(w) {
		return
	}
	ri, err := extras.RestorePoints(s.Sys)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, ri)
}
