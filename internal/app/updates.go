package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/installer"
	"fpsforge/internal/system"
	"fpsforge/internal/update"
)

// Updater es lo que necesita la parte de actualizaciones (se reemplaza en las pruebas y en el modo demostración).
type Updater interface {
	Installed() bool
	Check(ctx context.Context) (update.Release, bool, error)
	Install(ctx context.Context, r update.Release) (bool, error)
}

// UpdateStatus es lo que ve la interfaz sobre las actualizaciones.
type UpdateStatus struct {
	Auto      bool   `json:"auto"`      // actualizaciones automáticas activadas
	Installed bool   `json:"installed"` // esta copia está instalada con el Setup (puede reemplazarse sola)
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	Notes     string `json:"notes"`
	Page      string `json:"page"`
	Busy      string `json:"busy"`    // "", "checking" o "installing"
	Applied   string `json:"applied"` // versión ya instalada en el disco que falta reiniciar
	Error     string `json:"error"`
	CheckedAt int64  `json:"checkedAt"`
}

// Updates busca versiones nuevas en GitHub y las instala. Lo usan la ventana y el overlay de arranque.
type Updates struct {
	Upd Updater
	Dir string // carpeta de datos (ahí está settings.json)
	// OnApplied se llama una vez, cuando se deja instalada una versión nueva.
	OnApplied func(version string)
	// ExePath es el programa instalado, tal como estaba al arrancar ("" si esta copia no está instalada).
	// Después de actualizar, el programa en ejecución queda apartado como «.old»: para volver a abrirlo
	// se usa esta ruta, que ya apunta a la versión nueva.
	ExePath string

	mu       sync.Mutex
	st       UpdateStatus
	release  update.Release
	notified string
}

// NewUpdates crea el actualizador real. Solo reemplaza el programa si esta copia es la que instaló el Setup.
func NewUpdates(sys system.System, dir string) *Updates {
	exe := installedExe(sys)
	if exe != "" {
		go update.CleanOld(exe) // restos de la actualización anterior
	}
	return &Updates{Upd: &update.Updater{Client: update.NewClient(), Current: Version, ExePath: exe}, Dir: dir, ExePath: exe}
}

// demoUpdater simula una versión nueva para probar la interfaz en modo demostración (no usa la red).
type demoUpdater struct{}

func (demoUpdater) Installed() bool { return true }
func (demoUpdater) Check(context.Context) (update.Release, bool, error) {
	time.Sleep(700 * time.Millisecond)
	return update.Release{Version: "99.0.0", Notes: "Versión de demostración: no se descarga nada.", Page: "https://github.com/" + update.Repo + "/releases"}, true, nil
}
func (demoUpdater) Install(context.Context, update.Release) (bool, error) {
	time.Sleep(1500 * time.Millisecond)
	return true, nil
}

// installedExe devuelve la ruta de este programa si está en la carpeta que registró el instalador.
func installedExe(sys system.System) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	inst := installer.InstalledDir(sys)
	if inst == "" || !installer.SamePath(inst, filepath.Dir(exe)) {
		return ""
	}
	return exe
}

// Status devuelve el estado actual (no consulta la red).
func (u *Updates) Status() UpdateStatus {
	u.mu.Lock()
	st := u.st
	u.mu.Unlock()
	st.Current = Version
	st.Auto = readSettings(u.Dir).AutoUpdate
	st.Installed = u.Upd.Installed()
	return st
}

// Applied devuelve la versión ya instalada en el disco que falta reiniciar ("" si no hay).
func (u *Updates) Applied() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.st.Applied
}

func (u *Updates) begin(what string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.st.Busy != "" {
		return false
	}
	u.st.Busy = what
	return true
}

func (u *Updates) end() {
	u.mu.Lock()
	u.st.Busy = ""
	u.mu.Unlock()
}

func friendlyUpdateError(err error) string {
	var se *update.StatusError
	switch {
	case errors.Is(err, update.ErrNoRelease):
		return "Todavía no hay ninguna versión publicada en GitHub."
	case errors.Is(err, update.ErrBusy):
		return "Ya se está actualizando desde otra ventana. Probá en un minuto."
	case errors.Is(err, update.ErrNotInstalled):
		return "Esta copia no está instalada con el instalador: descargá la versión nueva desde GitHub."
	case errors.As(err, &se) && (se.Code == http.StatusForbidden || se.Code == http.StatusTooManyRequests):
		return "GitHub limitó las consultas por ahora. Probá más tarde."
	case errors.As(err, &se):
		return "GitHub respondió con un error (" + http.StatusText(se.Code) + "). Probá más tarde."
	case errors.Is(err, context.DeadlineExceeded):
		return "GitHub tardó demasiado en responder. Probá más tarde."
	}
	return "No se pudo buscar actualizaciones: " + err.Error()
}

// Cycle busca una versión nueva y, si install es verdadero y hay una, la instala.
func (u *Updates) Cycle(install bool) {
	if !u.begin("checking") {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	rel, newer, err := u.Upd.Check(ctx)
	cancel()
	u.mu.Lock()
	u.st.CheckedAt = time.Now().Unix()
	if err != nil {
		u.st.Error = friendlyUpdateError(err)
		u.st.Busy = ""
		u.mu.Unlock()
		return
	}
	u.st.Error = ""
	u.st.Available = newer
	u.st.Latest, u.st.Notes, u.st.Page = rel.Version, rel.Notes, rel.Page
	u.release = rel
	u.st.Busy = ""
	u.mu.Unlock()
	if newer && install && u.Upd.Installed() {
		u.InstallNow()
	}
}

// InstallNow instala la versión nueva encontrada por la última búsqueda.
func (u *Updates) InstallNow() {
	if !u.begin("installing") {
		return
	}
	u.mu.Lock()
	rel := u.release
	u.mu.Unlock()
	if rel.Version == "" {
		u.end()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	fresh, err := u.Upd.Install(ctx, rel)
	cancel()
	u.mu.Lock()
	if err != nil {
		u.st.Error = friendlyUpdateError(err)
		u.st.Busy = ""
		u.mu.Unlock()
		return
	}
	u.st.Error = ""
	u.st.Applied = rel.Version
	u.st.Busy = ""
	notify := u.OnApplied != nil && fresh && u.notified != rel.Version
	if notify {
		u.notified = rel.Version
	}
	cb := u.OnApplied
	u.mu.Unlock()
	if notify {
		cb(rel.Version)
	}
}

// Watch busca actualizaciones por su cuenta: la primera vez después de first y luego cada every.
// Solo actúa si el usuario dejó activadas las actualizaciones automáticas; si no, no consulta nada.
func (u *Updates) Watch(stop <-chan struct{}, first, every time.Duration) {
	go func() {
		t := time.NewTimer(first)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
			}
			if readSettings(u.Dir).AutoUpdate {
				u.Cycle(true)
			}
			t.Reset(every)
		}
	}()
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Up.Status())
}

// handleUpdateCheck busca ahora (lo pidió el usuario). Con las automáticas activas, también instala.
func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "método no permitido")
		return
	}
	s.Up.Cycle(readSettings(s.Dir).AutoUpdate)
	writeJSON(w, s.Up.Status())
}

// handleUpdateInstall instala ya la versión encontrada (para quien tiene las automáticas desactivadas).
func (s *Server) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "método no permitido")
		return
	}
	if st := s.Up.Status(); !st.Available || !st.Installed || st.Busy != "" {
		writeJSON(w, st)
		return
	}
	go s.Up.InstallNow()
	st := s.Up.Status()
	st.Busy = "installing"
	writeJSON(w, st)
}

// releasesPage es la página de versiones de GitHub (fija: no depende de nada que venga de la red).
const releasesPage = "https://github.com/" + update.Repo + "/releases/latest"

// handleUpdatePage abre en el navegador la página de versiones de GitHub.
func (s *Server) handleUpdatePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "método no permitido")
		return
	}
	if _, _, err := s.Sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", releasesPage); err != nil {
		fail(w, 500, "No se pudo abrir el navegador: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "url": releasesPage})
}

// handleUpdateRestart cierra el programa y lo vuelve a abrir (ya con la versión nueva).
func (s *Server) handleUpdateRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "método no permitido")
		return
	}
	st := s.Up.Status()
	if st.Applied == "" {
		fail(w, 400, "no hay ninguna versión nueva instalada para abrir")
		return
	}
	if err := s.Relaunch(); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.Close()
	}()
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Relaunch deja programado que el programa se vuelva a abrir apenas este proceso termine.
func (s *Server) Relaunch() error {
	if _, demo := s.Sys.(*system.Fake); demo {
		return nil
	}
	exe := s.Up.ExePath
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			return err
		}
	}
	inner := fmt.Sprintf("Wait-Process -Id %d -Timeout 40 -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1; Start-Process -FilePath %s -WorkingDirectory %s",
		os.Getpid(), psQuote(exe), psQuote(filepath.Dir(exe)))
	script := "Start-Process -WindowStyle Hidden -FilePath powershell.exe -ArgumentList @('-NoProfile','-Command'," + psQuote(inner) + ")"
	_, err := s.Sys.PowerShell(script, 30*time.Second)
	return err
}
