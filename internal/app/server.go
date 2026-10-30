// Package app implementa el servidor local que alimenta la interfaz.
package app

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fpsforge/internal/bench"
	"fpsforge/internal/history"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/icon"
	"fpsforge/internal/scan"
	"fpsforge/internal/system"
	"fpsforge/internal/tools"
	"fpsforge/internal/tweaks"
)

//go:embed assets
var assets embed.FS

// Version del programa.
const Version = "2.8.1"

// Server es el backend de la interfaz.
type Server struct {
	Sys     system.System
	Eng     *tweaks.Engine
	Token   string
	Addr    string
	Closed  chan struct{}
	Bench   *bench.Store
	History *history.Log
	// DNSLookup mide el DNS (se reemplaza en las pruebas).
	DNSLookup tools.LookupFn
	// SpawnOverlay abre el overlay en otro proceso (se reemplaza en las pruebas).
	SpawnOverlay func() error
	// Up busca e instala versiones nuevas desde GitHub.
	Up *Updates

	autostartDemo bool // en modo demostración no hay Programador de tareas: se recuerda en memoria
	// ScanFn hace el escaneo del equipo (se reemplaza en las pruebas).
	ScanFn func(system.System) (scan.Hardware, error)
	Dir    string

	mu       sync.Mutex
	scanMu   sync.Mutex // un solo escaneo a la vez
	hw       *scan.Hardware
	lastBeat time.Time
	beaten   bool
	srv      *http.Server
	closeOne sync.Once
}

// New crea el servidor.
func New(sys system.System, dataDir string) (*Server, error) {
	eng, err := tweaks.NewEngine(sys, dataDir)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	bs, err := bench.Open(dataDir)
	if err != nil {
		return nil, err
	}
	srv := &Server{Sys: sys, Eng: eng, Bench: bs, History: history.Open(dataDir), Dir: dataDir, Token: hex.EncodeToString(b), Closed: make(chan struct{})}
	srv.SpawnOverlay = srv.spawnOverlay
	srv.ScanFn = scan.Run
	srv.Up = NewUpdates(sys, dataDir)
	if _, demo := sys.(*system.Fake); demo {
		srv.Up.Upd = demoUpdater{} // en demostración y en las pruebas no se usa la red
	}
	return srv, nil
}

// hardware devuelve el escaneo del equipo; si todavía no terminó, espera.
func (s *Server) hardware(force bool) scan.Hardware {
	if !force {
		if h, ok := s.hardwareIfReady(); ok {
			return h
		}
	}
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	if !force {
		if h, ok := s.hardwareIfReady(); ok { // otro escaneo terminó mientras esperábamos
			return h
		}
	}
	h, err := s.ScanFn(s.Sys)
	if err != nil {
		if prev, ok := s.hardwareIfReady(); ok && prev.CPU != "No detectado" {
			return prev // falló la actualización: se mantiene el último escaneo bueno
		}
		h = scan.Hardware{Computer: "equipo", CPU: "No detectado", OS: "Windows", Build: 22000, Admin: s.Sys.IsAdmin()}
	}
	s.mu.Lock()
	s.hw = &h
	s.mu.Unlock()
	if err == nil {
		s.saveHardwareCache(h)
	}
	return h
}

// hardwareIfReady devuelve el escaneo solo si ya está listo (no espera).
func (s *Server) hardwareIfReady() (scan.Hardware, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hw == nil {
		return scan.Hardware{}, false
	}
	return *s.hw, true
}

// Start abre el puerto local y atiende peticiones. Devuelve la URL.
func (s *Server) Start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.Addr = ln.Addr().String()
	sub, _ := fs.Sub(assets, "assets")
	mux := http.NewServeMux()
	mux.HandleFunc("/logo.png", s.logo(128))
	mux.HandleFunc("/favicon.png", s.logo(64))
	mux.HandleFunc("/", s.index(sub))
	mux.HandleFunc("/api/state", s.guard(s.handleState))
	mux.HandleFunc("/api/rescan", s.guard(s.handleRescan))
	mux.HandleFunc("/api/apply", s.guard(s.handleApply))
	mux.HandleFunc("/api/revert", s.guard(s.handleRevert))
	mux.HandleFunc("/api/profile", s.guard(s.handleProfile))
	mux.HandleFunc("/api/restorepoint", s.guard(s.handleRestorePoint))
	mux.HandleFunc("/api/clean/scan", s.guard(s.handleCleanScan))
	mux.HandleFunc("/api/clean/run", s.guard(s.handleCleanRun))
	mux.HandleFunc("/api/startup", s.guard(s.handleStartup))
	mux.HandleFunc("/api/startup/set", s.guard(s.handleStartupSet))
	mux.HandleFunc("/api/diag", s.guard(s.handleDiag))
	mux.HandleFunc("/api/diag/action", s.guard(s.handleDiagAction))
	mux.HandleFunc("/api/procs", s.guard(s.handleProcs))
	mux.HandleFunc("/api/procs/kill", s.guard(s.handleProcKill))
	mux.HandleFunc("/api/bench", s.guard(s.handleBench))
	mux.HandleFunc("/api/bench/add", s.guard(s.handleBenchAdd))
	mux.HandleFunc("/api/bench/delete", s.guard(s.handleBenchDelete))
	mux.HandleFunc("/api/revertall", s.guard(s.handleRevertAll))
	mux.HandleFunc("/api/settings", s.guard(s.handleSettings))
	mux.HandleFunc("/api/overlay", s.guard(s.handleOverlay))
	mux.HandleFunc("/api/autostart", s.guard(s.handleAutostart))
	mux.HandleFunc("/api/overlay/config", s.guard(s.handleOverlayConfig))
	mux.HandleFunc("/api/overlay/start", s.guard(s.handleOverlayStart))
	mux.HandleFunc("/api/memory/free", s.guard(s.handleMemoryFree))
	mux.HandleFunc("/api/overlay/stop", s.guard(s.handleOverlayStop))
	mux.HandleFunc("/api/overlay/record", s.guard(s.handleOverlayRecord))
	mux.HandleFunc("/api/sessions", s.guard(s.handleSessions))
	mux.HandleFunc("/api/sessions/get", s.guard(s.handleSessionGet))
	mux.HandleFunc("/api/sessions/import", s.guard(s.handleSessionImport))
	mux.HandleFunc("/api/sessions/delete", s.guard(s.handleSessionDelete))
	mux.HandleFunc("/api/net", s.guard(s.handleNetTest))
	mux.HandleFunc("/api/schedule", s.guard(s.handleSchedule))
	mux.HandleFunc("/api/gameping", s.guard(s.handleGamePing))
	mux.HandleFunc("/api/disks", s.guard(s.handleDisks))
	mux.HandleFunc("/api/cs2video", s.guard(s.handleVideo))
	mux.HandleFunc("/api/bisect", s.guard(s.handleBisect))
	mux.HandleFunc("/api/bisect/start", s.guard(s.handleBisectStart))
	mux.HandleFunc("/api/bisect/answer", s.guard(s.handleBisectAnswer))
	mux.HandleFunc("/api/report", s.guard(s.handleReport))
	mux.HandleFunc("/api/config/export", s.guard(s.handleConfigExport))
	mux.HandleFunc("/api/config/import", s.guard(s.handleConfigImport))
	mux.HandleFunc("/api/history", s.guard(s.handleHistory))
	mux.HandleFunc("/api/sessions/microcuts", s.guard(s.handleMicrocuts))
	mux.HandleFunc("/api/dnsbench", s.guard(s.handleDNSBench))
	mux.HandleFunc("/api/netmon", s.guard(s.handleNetMon))
	mux.HandleFunc("/api/apps", s.guard(s.handleApps))
	mux.HandleFunc("/api/apps/installed", s.guard(s.handleAppsInstalled))
	mux.HandleFunc("/api/apps/install", s.guard(s.handleAppsInstall))
	mux.HandleFunc("/api/apps/upgrade", s.guard(s.handleAppsUpgrade))
	mux.HandleFunc("/api/apps/winget", s.guard(s.handleAppsWinget))
	mux.HandleFunc("/api/apps/open", s.guard(s.handleAppsOpen))
	mux.HandleFunc("/api/apps/drivers", s.guard(s.handleAppsDrivers))
	mux.HandleFunc("/api/update", s.guard(s.handleUpdate))
	mux.HandleFunc("/api/update/check", s.guard(s.handleUpdateCheck))
	mux.HandleFunc("/api/update/install", s.guard(s.handleUpdateInstall))
	mux.HandleFunc("/api/update/restart", s.guard(s.handleUpdateRestart))
	mux.HandleFunc("/api/update/page", s.guard(s.handleUpdatePage))
	mux.HandleFunc("/api/heartbeat", s.guard(s.handleHeartbeat))
	mux.HandleFunc("/api/quit", s.guard(s.handleQuit))
	if h, ok := s.loadHardwareCache(); ok {
		// Con el último escaneo guardado la interfaz aparece completa al instante; se actualiza en segundo plano.
		s.mu.Lock()
		s.hw = &h
		s.mu.Unlock()
		go s.hardware(true)
	} else {
		go s.hardware(false) // el escaneo arranca ya, mientras se abre la ventana
	}
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return "http://" + s.Addr + "/", nil
}

// WatchHeartbeat cierra el servidor cuando la ventana deja de avisar.
func (s *Server) WatchHeartbeat(grace time.Duration) {
	go func() {
		start := time.Now()
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-s.Closed:
				return
			case <-t.C:
				s.mu.Lock()
				beaten, last := s.beaten, s.lastBeat
				s.mu.Unlock()
				if !beaten && time.Since(start) > 2*time.Minute {
					s.Close()
					return
				}
				if beaten && time.Since(last) > grace {
					s.Close()
					return
				}
			}
		}
	}()
}

// WatchWindow avisa (onGone) cuando la ventana deja de dar señales de vida, sin cerrar el servidor,
// y vuelve a armarse cuando la ventana se abre de nuevo.
func (s *Server) WatchWindow(grace time.Duration, onGone func()) {
	go func() {
		start := time.Now()
		gone := false
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-s.Closed:
				return
			case <-t.C:
				s.mu.Lock()
				beaten, last := s.beaten, s.lastBeat
				s.mu.Unlock()
				silent := (!beaten && time.Since(start) > 2*time.Minute) || (beaten && time.Since(last) > grace)
				if silent && !gone {
					gone = true
					onGone()
				} else if !silent && gone && beaten {
					gone = false
				}
			}
		}
	}()
}

// Close detiene el servidor.
func (s *Server) Close() {
	s.closeOne.Do(func() {
		close(s.Closed)
		if s.srv != nil {
			go s.srv.Close()
		}
	})
}

// guard exige el token y un Host local: impide que una web cualquiera
// llame a la API del programa (CSRF / DNS rebinding).
func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.hostOK(r) || r.Header.Get("X-Token") != s.Token {
			http.Error(w, "no autorizado", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+s.Addr {
			http.Error(w, "origen no permitido", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		h(w, r)
	}
}

// logo sirve el logo como PNG (también es el ícono de la ventana y de la barra de tareas).
func (s *Server) logo(size int) http.HandlerFunc {
	b := icon.PNG(size)
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.hostOK(r) {
			http.Error(w, "host no permitido", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=86400")
		_, _ = w.Write(b)
	}
}

func (s *Server) hostOK(r *http.Request) bool { return r.Host == s.Addr }

func (s *Server) index(sub fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(sub))
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.hostOK(r) {
			http.Error(w, "host no permitido", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			b, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.Error(w, "falta index.html", 500)
				return
			}
			// El puerto cambia en cada apertura, y con él el «origen» del navegador: el almacenamiento local
			// no sobrevive. Por eso los ajustes iniciales viajan dentro de la página.
			boot, _ := json.Marshal(s.loadSettings())
			page := strings.Replace(string(b), "__TOKEN__", s.Token, 1)
			page = strings.Replace(page, "__BOOT__", strings.ReplaceAll(string(boot), "<", `\u003c`), 1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Write([]byte(page))
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	return json.NewDecoder(r.Body).Decode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
