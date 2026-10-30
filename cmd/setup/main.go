// Instalador de OverdriveTuner: un único .exe que lleva el programa dentro.
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/app"
	"fpsforge/internal/apps"
	"fpsforge/internal/autostart"
	"fpsforge/internal/icon"
	"fpsforge/internal/installer"
	"fpsforge/internal/system"
	"fpsforge/internal/winhost"
)

//go:embed assets payload
var content embed.FS

// patchMode vale "1" en OverdriveTuner-Patch.exe (se fija al compilar con -ldflags -X).
var patchMode string

type setup struct {
	sys   system.System
	token string
	addr  string
	demo  bool

	mu      sync.Mutex
	running bool
	pct     int
	msg     string
	done    bool
	errMsg  string
	note    string
	exe     string
	last    time.Time
	beaten  bool

	closed chan struct{}
	once   sync.Once
	srv    *http.Server
}

func payload() ([]byte, error) {
	b, err := fs.ReadFile(content, "payload/OverdriveTuner.exe")
	if err != nil || len(b) < 1024 {
		return nil, fmt.Errorf("este instalador está incompleto (no contiene el programa). Descárgalo de nuevo")
	}
	return b, nil
}

func (s *setup) close() {
	s.once.Do(func() {
		close(s.closed)
		if s.srv != nil {
			go s.srv.Close()
		}
	})
}

func (s *setup) guard(h http.HandlerFunc) http.HandlerFunc {
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

func (s *setup) start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.addr = ln.Addr().String()
	page, _ := fs.ReadFile(content, "assets/index.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.addr || r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(strings.Replace(string(page), "__TOKEN__", s.token, 1)))
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=3600")
		_, _ = w.Write(icon.PNG(128))
	})
	mux.HandleFunc("/api/info", s.guard(func(w http.ResponseWriter, r *http.Request) {
		inst := installer.InstalledDir(s.sys)
		iv := installer.InstalledVersion(s.sys)
		writeJSON(w, map[string]any{"dir": installer.DefaultDir(), "installed": inst != "", "installedDir": inst, "version": app.Version,
			"patch": patchMode == "1", "installedVersion": iv, "changes": releasesSince(iv),
			"autostart": iv == "" || newer("2.6.0", iv) || autostart.Enabled(s.sys),
			// instalación nueva: activado de entrada; si ya había una, se respeta lo que el usuario tenía
			"autoUpdate": iv == "" || app.AutoUpdateOn(app.DefaultDataDir())})
	}))
	mux.HandleFunc("/api/patch", s.guard(s.handlePatch))
	mux.HandleFunc("/api/install", s.guard(s.handleInstall))
	mux.HandleFunc("/api/progress", s.guard(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, map[string]any{"pct": s.pct, "msg": s.msg, "done": s.done, "err": s.errMsg, "note": s.note})
	}))
	mux.HandleFunc("/api/heartbeat", s.guard(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.last, s.beaten = time.Now(), true
		s.mu.Unlock()
		writeJSON(w, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/api/finish", s.guard(s.handleFinish))
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return "http://" + s.addr + "/", nil
}

// installRTSS instala RTSS (solo la medición de FPS) con winget. Un fallo no cancela la instalación.
func (s *setup) installRTSS() string {
	return s.installApp("rtss", "RTSS", "Instalando RTSS (medición de FPS)…", 97)
}

// installCoreTemp instala Core Temp (temperatura del procesador para el overlay).
func (s *setup) installCoreTemp() string {
	return s.installApp("coretemp", "Core Temp", "Instalando Core Temp (temperatura del procesador)…", 99)
}

func (s *setup) installApp(id, name, msg string, pct int) string {
	s.mu.Lock()
	s.pct, s.msg = pct, msg
	s.mu.Unlock()
	it, ok := apps.ByID(id)
	if !ok {
		return ""
	}
	if !apps.Available(s.sys) {
		if err := apps.InstallWinget(s.sys); err != nil {
			return "No se pudo instalar " + name + " (winget no está disponible). Podés instalarlo después desde «Instalar programas»."
		}
	}
	if r := apps.Install(s.sys, it); !r.OK {
		log.Println(name+":", r.Notes)
		return "No se pudo instalar " + name + ". Podés instalarlo después desde «Instalar programas»."
	}
	return ""
}

// setAutostart activa o desactiva el inicio del overlay con Windows. Devuelve un aviso si falla.
func (s *setup) setAutostart(on bool, exe string) string {
	if !on {
		return ""
	}
	if err := autostart.Enable(s.sys, exe); err != nil {
		log.Println("inicio con Windows:", err)
		return "No se pudo activar el inicio con Windows; podés activarlo en Configuración."
	}
	return ""
}

func (s *setup) handleInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir       string `json:"dir"`
		Desktop   bool   `json:"desktop"`
		StartMenu bool   `json:"startMenu"`
		Rtss      bool   `json:"rtss"`
		CoreTemp  bool   `json:"coretemp"`
		Autostart bool   `json:"autostart"`
		// AutoUpdate: instalar solas las versiones nuevas que salgan en GitHub.
		AutoUpdate bool `json:"autoUpdate"`
	}
	if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	req.Dir = strings.TrimSpace(req.Dir)
	if err := installer.ValidateDir(req.Dir); err != nil {
		fail(w, 400, err.Error())
		return
	}
	exe, err := payload()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		fail(w, 409, "ya se está instalando")
		return
	}
	s.running, s.done, s.errMsg, s.note, s.pct, s.msg = true, false, "", "", 0, "Preparando…"
	s.mu.Unlock()
	go func() {
		o := installer.Options{Dir: req.Dir, Desktop: req.Desktop, StartMenu: req.StartMenu, Version: app.Version, Exe: exe, Icon: icon.ICO()}
		err := installer.Install(s.sys, o, func(p int, m string) {
			s.mu.Lock()
			s.pct, s.msg = p, m
			s.mu.Unlock()
		})
		note := ""
		if err == nil {
			if serr := app.SetAutoUpdate(app.DefaultDataDir(), req.AutoUpdate); serr != nil {
				log.Println("actualizaciones automáticas:", serr)
				note = "No se pudo guardar la elección de actualizaciones automáticas; podés cambiarla en Configuración."
			}
		}
		if err == nil && req.Autostart {
			note = strings.TrimSpace(note + " " + s.setAutostart(true, filepath.Join(req.Dir, installer.ExeName)))
		}
		if err == nil && req.Rtss {
			note = strings.TrimSpace(note + " " + s.installRTSS())
		}
		if err == nil && req.CoreTemp {
			note = strings.TrimSpace(note + " " + s.installCoreTemp())
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = false
		if err != nil {
			log.Println("error de instalación:", err)
			s.errMsg = err.Error()
			return
		}
		s.note = note
		s.exe = filepath.Join(req.Dir, installer.ExeName)
		s.done = true
	}()
	writeJSON(w, map[string]any{"ok": true})
}

func (s *setup) handlePatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rtss      bool `json:"rtss"`
		CoreTemp  bool `json:"coretemp"`
		Autostart bool `json:"autostart"`
	}
	if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	exe, err := payload()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		fail(w, 409, "ya se está aplicando")
		return
	}
	s.running, s.done, s.errMsg, s.note, s.pct, s.msg = true, false, "", "", 0, "Preparando…"
	s.mu.Unlock()
	go func() {
		dir, err := installer.Patch(s.sys, exe, icon.ICO(), app.Version, func(p int, m string) {
			s.mu.Lock()
			s.pct, s.msg = p, m
			s.mu.Unlock()
		})
		note := ""
		if err == nil {
			if req.Autostart {
				note = s.setAutostart(true, filepath.Join(dir, installer.ExeName))
			} else if iv := installer.InstalledVersion(s.sys); iv == "" || !newer("2.6.0", iv) {
				_ = autostart.Disable(s.sys) // quedó apagado a propósito
			}
		}
		if err == nil && req.Rtss {
			note = strings.TrimSpace(note + " " + s.installRTSS())
		}
		if err == nil && req.CoreTemp {
			note = strings.TrimSpace(note + " " + s.installCoreTemp())
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = false
		if err != nil {
			log.Println("error del parche:", err)
			s.errMsg = err.Error()
			return
		}
		s.note = note
		s.exe = filepath.Join(dir, installer.ExeName)
		s.done = true
	}()
	writeJSON(w, map[string]any{"ok": true})
}

func (s *setup) handleFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Launch bool `json:"launch"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req)
	s.mu.Lock()
	exe, ok := s.exe, s.done
	s.mu.Unlock()
	if req.Launch && ok && !s.demo {
		cmd := exec.Command(exe)
		cmd.Dir = filepath.Dir(exe)
		if err := cmd.Start(); err != nil {
			log.Println("no se pudo abrir el programa:", err)
		}
	}
	writeJSON(w, map[string]any{"ok": true})
	go func() { time.Sleep(300 * time.Millisecond); s.close() }()
}

func main() {
	demo := flag.Bool("demo", false, "modo demostración: no toca el sistema")
	noWindow := flag.Bool("no-window", false, "no abrir ventana")
	silent := flag.Bool("silent", false, "instalar sin interfaz con las opciones predeterminadas")
	flag.Parse()
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, "/S") || strings.EqualFold(a, "/silent") {
			*silent = true
		}
	}

	logDir := filepath.Join(os.TempDir(), "OverdriveTunerSetup")
	_ = os.MkdirAll(logDir, 0o755)
	if f, err := os.OpenFile(filepath.Join(logDir, "setup.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(f)
		defer f.Close()
	}

	var sys system.System
	if *demo {
		sys = system.NewFake()
	} else {
		sys = system.New()
		if !sys.IsAdmin() && winhost.EnsureElevated() {
			return
		}
		if !sys.IsAdmin() {
			winhost.Fatal("Instalar OverdriveTuner requiere permisos de administrador.")
		}
		if !winhost.SingleInstance("OverdriveTunerSetupSingleInstance") {
			return
		}
	}

	if patchMode == "1" && *silent {
		exe, err := payload()
		if err != nil {
			winhost.Fatal(err.Error())
		}
		dir, err := installer.Patch(sys, exe, icon.ICO(), app.Version, nil)
		if err != nil {
			log.Println("error del parche:", err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !*demo {
			c := exec.Command(filepath.Join(dir, installer.ExeName))
			c.Dir = dir
			_ = c.Start()
		}
		return
	}

	if *silent && patchMode != "1" {
		exe, err := payload()
		if err != nil {
			winhost.Fatal(err.Error())
		}
		dir := installer.DefaultDir()
		if d := installer.InstalledDir(sys); d != "" {
			dir = d
		}
		err = installer.Install(sys, installer.Options{Dir: dir, Desktop: true, StartMenu: true, Version: app.Version, Exe: exe, Icon: icon.ICO()}, nil)
		if err != nil {
			log.Println("error de instalación:", err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		winhost.Fatal(err.Error())
	}
	s := &setup{sys: sys, token: hex.EncodeToString(b), demo: *demo, closed: make(chan struct{})}
	url, err := s.start()
	if err != nil {
		winhost.Fatal("No se pudo iniciar el instalador: " + err.Error())
	}
	if *noWindow {
		fmt.Println(url)
		<-s.closed
		return
	}
	closed := func() { // si cierran la ventana a mitad de instalación, se espera a que termine
		for {
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
	title := "Instalar OverdriveTuner"
	if patchMode == "1" {
		title = "Actualizar OverdriveTuner"
	}
	// Ventana propia (WebView2); si no se puede, Edge en modo aplicación.
	if win, err := winhost.OpenApp(url, logDir, winhost.AppOptions{Title: title, Maximized: true}); err == nil {
		go func() { <-win.Done(); closed() }()
	} else if proc := winhost.OpenWindowMax(url, filepath.Join(logDir, "ventana")); proc != nil {
		go func() {
			started := time.Now()
			proc.Wait()
			if time.Since(started) > 5*time.Second {
				closed()
			}
		}()
	}
	go func() { // sin latidos durante 3 minutos (o nunca hubo): salir
		start := time.Now()
		for {
			select {
			case <-s.closed:
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
	<-s.closed
	time.Sleep(300 * time.Millisecond)
}
