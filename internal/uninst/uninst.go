// Package uninst es el desinstalador de OverdriveTuner: una ventana limpia (la misma técnica que el
// instalador: servidor local + ventana de aplicación) que explica qué se quita y qué se conserva.
package uninst

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/app"
	"fpsforge/internal/autostart"
	"fpsforge/internal/icon"
	"fpsforge/internal/installer"
	"fpsforge/internal/system"
	"fpsforge/internal/winhost"
)

//go:embed page.html
var page string

// Server atiende la ventana del desinstalador.
type Server struct {
	Sys     system.System
	Dir     string // carpeta de instalación
	DataDir string // carpeta de datos (ajustes, copias de seguridad, historial)
	Demo    bool

	token, addr string
	mu          sync.Mutex
	running     bool
	done        bool
	pct         int
	msg, errMsg string
	last        time.Time
	beaten      bool
	Closed      chan struct{}
	once        sync.Once
	srv         *http.Server
}

// SafeDataDir dice si path es exactamente «<ProgramData>\FPSForge», lo único que el desinstalador borra.
func SafeDataDir(path string) bool {
	pd := os.Getenv("ProgramData")
	if pd == "" || path == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(path), filepath.Join(pd, "FPSForge"))
}

// AppliedCount cuenta los ajustes que OverdriveTuner tiene aplicados (state.json de la carpeta de datos).
func AppliedCount(dataDir string) int {
	b, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	if err != nil {
		return 0
	}
	var l []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(b, &l) != nil {
		var l2 []map[string]any
		if json.Unmarshal(b, &l2) != nil {
			return 0
		}
		return len(l2)
	}
	return len(l)
}

func (s *Server) close() {
	s.once.Do(func() {
		close(s.Closed)
		if s.srv != nil {
			go s.srv.Close()
		}
	})
}

func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.addr || r.Header.Get("X-Token") != s.token {
			http.Error(w, "no autorizado", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+s.addr {
			http.Error(w, "origen no permitido", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Start abre el puerto local y devuelve la dirección de la ventana.
func (s *Server) Start() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s.token = hex.EncodeToString(b)
	s.Closed = make(chan struct{})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.addr = ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.addr || r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(strings.Replace(page, "__TOKEN__", s.token, 1)))
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=3600")
		_, _ = w.Write(icon.PNG(128))
	})
	mux.HandleFunc("/api/info", s.guard(func(w http.ResponseWriter, r *http.Request) {
		_, derr := os.Stat(s.DataDir)
		writeJSON(w, map[string]any{"dir": s.Dir, "dataDir": s.DataDir, "version": installer.InstalledVersion(s.Sys),
			"applied": AppliedCount(s.DataDir), "hasData": derr == nil, "canDeleteData": s.Demo || SafeDataDir(s.DataDir), "demo": s.Demo, "current": app.Version})
	}))
	mux.HandleFunc("/api/uninstall", s.guard(s.handleUninstall))
	mux.HandleFunc("/api/progress", s.guard(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, map[string]any{"pct": s.pct, "msg": s.msg, "done": s.done, "err": s.errMsg})
	}))
	mux.HandleFunc("/api/heartbeat", s.guard(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.last, s.beaten = time.Now(), true
		s.mu.Unlock()
		writeJSON(w, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/api/finish", s.guard(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); s.close() }()
	}))
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return "http://" + s.addr + "/", nil
}

func (s *Server) set(p int, m string) {
	s.mu.Lock()
	s.pct, s.msg = p, m
	s.mu.Unlock()
}

func (s *Server) handleUninstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeleteData bool `json:"deleteData"`
	}
	if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		fail(w, 409, "ya se está desinstalando")
		return
	}
	s.running, s.done, s.errMsg, s.pct, s.msg = true, false, "", 0, "Preparando…"
	s.mu.Unlock()
	go func() {
		err := s.run(req.DeleteData)
		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = false
		if err != nil {
			log.Println("error al desinstalar:", err)
			s.errMsg = err.Error()
			return
		}
		s.pct, s.msg, s.done = 100, "Listo", true
	}()
	writeJSON(w, map[string]any{"ok": true})
}

// run hace el trabajo. Cada paso informa su avance; la carpeta del programa se borra al final,
// unos segundos después de que este proceso salga.
func (s *Server) run(deleteData bool) error {
	s.set(8, "Cerrando OverdriveTuner y el overlay…")
	if !s.Demo {
		winhost.Cleanup("all", s.DataDir)
		winhost.KillByName(installer.ExeName)
	}
	s.set(30, "Quitando el inicio con Windows…")
	_ = autostart.Disable(s.Sys)
	s.set(50, "Quitando accesos directos y el registro de Windows…")
	if err := installer.Uninstall(s.Sys, s.Dir); err != nil {
		return err
	}
	if deleteData {
		s.set(75, "Borrando tus datos…")
		if !s.Demo {
			if !SafeDataDir(s.DataDir) {
				return fmt.Errorf("no se borró la carpeta de datos porque su ruta no es la esperada: %s", s.DataDir)
			}
			if err := os.RemoveAll(s.DataDir); err != nil {
				return fmt.Errorf("no se pudieron borrar todos tus datos: %w", err)
			}
		}
	}
	s.set(92, "Programando el borrado de la carpeta del programa…")
	if !s.Demo {
		if err := installer.ScheduleDelete(s.Sys, s.Dir); err != nil {
			log.Println("no se pudo programar el borrado:", err)
		}
	}
	return nil
}

// Run abre la ventana y espera a que termine.
func Run(s *Server, noWindow bool, profileDir string) {
	url, err := s.Start()
	if err != nil {
		winhost.Fatal("No se pudo iniciar el desinstalador: " + err.Error())
	}
	if noWindow {
		fmt.Println(url)
		<-s.Closed
		return
	}
	proc := winhost.OpenWindow(url, profileDir)
	go func() {
		if proc == nil {
			return
		}
		started := time.Now()
		proc.Wait()
		if time.Since(started) > 5*time.Second {
			for { // si cierran la ventana a mitad de la desinstalación, se espera a que termine
				s.mu.Lock()
				r := s.running
				s.mu.Unlock()
				if !r {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			s.close()
		}
	}()
	go func() { // sin latidos durante 3 minutos (o nunca hubo): salir
		start := time.Now()
		for {
			select {
			case <-s.Closed:
				return
			case <-time.After(2 * time.Second):
			}
			s.mu.Lock()
			beaten, last, running := s.beaten, s.last, s.running
			s.mu.Unlock()
			if running {
				continue
			}
			if (!beaten && time.Since(start) > 2*time.Minute) || (beaten && time.Since(last) > 3*time.Minute) {
				s.close()
				return
			}
		}
	}()
	<-s.Closed
	time.Sleep(300 * time.Millisecond)
}
