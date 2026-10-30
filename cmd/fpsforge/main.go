// OverdriveTuner: optimizador de PC para gaming en Windows 10/11.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"fpsforge/internal/app"
	"fpsforge/internal/installer"
	"fpsforge/internal/overlay"
	"fpsforge/internal/schedule"
	"fpsforge/internal/system"
	"fpsforge/internal/uninst"
	"fpsforge/internal/winhost"
)

func dataDirPath() string {
	if d := os.Getenv("ProgramData"); d != "" {
		return filepath.Join(d, "FPSForge")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".fpsforge")
}

func main() {
	demo := flag.Bool("demo", false, "modo demostración (sin tocar el sistema)")
	noWindow := flag.Bool("no-window", false, "no abrir ventana; solo imprimir la URL")
	uninstall := flag.Bool("uninstall", false, "desinstalar OverdriveTuner")
	ovl := flag.Bool("overlay", false, "abrir solo el overlay de rendimiento")
	task := flag.String("task", "", "tarea de mantenimiento sin ventana: cleanup o restore (la usa el Programador de tareas)")
	boot := flag.Bool("boot", false, "con --overlay: arrancó con Windows (muestra un ícono en la bandeja)")
	flag.Parse()
	if *uninstall {
		runUninstall(*demo, *noWindow)
		return
	}
	if *task != "" {
		runTask(*task)
		return
	}

	dir := dataDirPath()
	if *demo {
		dir = filepath.Join(os.TempDir(), "overdrivetuner-demo")
	}
	_ = os.MkdirAll(dir, 0o755)
	if f, err := os.OpenFile(filepath.Join(dir, "overdrivetuner.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(f)
		defer f.Close()
	}

	if *ovl {
		if !*demo {
			if !system.New().IsAdmin() && winhost.EnsureElevated() {
				return
			}
			if !winhost.SingleInstance("OverdriveTunerOverlaySingleInstance") {
				return
			}
		}
		log.Printf("overlay iniciado (arranque automático: %v, administrador: %v)", *boot, system.New().IsAdmin())
		if *boot && !*demo {
			// Sin ventana hasta que se abre un juego: el ícono de la bandeja deja ver que está activo.
			exe, _ := os.Executable()
			winhost.SetTrayLabels("Abrir OverdriveTuner", "Cerrar solo el overlay")
			winhost.StartTray("Overlay de OverdriveTuner (activo)", func() {
				c := exec.Command(exe)
				c.Dir = filepath.Dir(exe)
				_ = c.Start()
			}, func() {
				_ = overlay.SendCmd(dir, "quit", "")
				time.Sleep(1500 * time.Millisecond)
				winhost.TrayRemove()
				os.Exit(0)
			})
			winhost.TrayNotify("Overlay activo", "Aparece solo cuando abrís un juego. Clic en el ícono para abrir OverdriveTuner.")
			defer winhost.TrayRemove()
			// Si el usuario dejó activadas las actualizaciones automáticas, se buscan también desde acá: así llegan
			// aunque nunca abra la ventana. Si las desactivó, no se consulta nada.
			up := app.NewUpdates(system.New(), dir)
			up.OnApplied = func(v string) {
				winhost.TrayNotify("OverdriveTuner se actualizó", "La versión "+v+" ya está instalada. Se usa la próxima vez que abras el programa o reinicies la PC.")
			}
			up.Watch(nil, 90*time.Second, 6*time.Hour)
		}
		if err := overlay.Main(dir, *demo); err != nil {
			log.Println("overlay:", err)
			winhost.Fatal("El overlay no pudo iniciar: " + err.Error())
		}
		return
	}

	var sys system.System
	if *demo {
		sys = system.NewFake()
	} else {
		sys = system.New()
		if !sys.IsAdmin() && winhost.EnsureElevated() {
			return // se relanzó con permisos de administrador
		}
		if winhost.ActivateExisting() {
			return // ya hay una copia (en la bandeja o con ventana): se le avisa que se muestre
		}
		if !winhost.SingleInstance("OverdriveTunerSingleInstance") {
			return
		}
	}

	srv, err := app.New(sys, dir)
	if err != nil {
		winhost.Fatal("No se pudo iniciar: " + err.Error())
	}
	url, err := srv.Start()
	if err != nil {
		winhost.Fatal("No se pudo abrir el puerto local: " + err.Error())
	}
	log.Println("servidor en", url)
	if !*demo {
		go srv.AutoOpenOverlay()
		srv.Up.OnApplied = func(v string) {
			winhost.TrayNotify("OverdriveTuner se actualizó", "La versión "+v+" ya está instalada. Reiniciá el programa para usarla (Configuración → Actualizaciones).")
		}
		srv.Up.Watch(srv.Closed, 25*time.Second, 6*time.Hour) // solo actúa si las actualizaciones automáticas están activadas
	}
	if *noWindow {
		fmt.Println(url)
		select {
		case <-srv.Closed:
		}
		return
	}
	// La X de la ventana no cierra el programa: queda en la bandeja (con el overlay y todo lo demás
	// activo) y un clic en el ícono vuelve a abrir la ventana. «Salir por completo» cierra todo.
	var uiMu sync.Mutex
	notified := false
	windowGone := func() {
		uiMu.Lock()
		first := !notified
		notified = true
		uiMu.Unlock()
		if first {
			winhost.TrayNotify("OverdriveTuner sigue activo", "Quedó en la bandeja junto al reloj. Hacé clic en el ícono para abrirlo o clic derecho para salir.")
		}
	}
	// watch vigila el proceso de la ventana: cuando termina, es que se cerró con la X.
	watch := func(proc *exec.Cmd) {
		if proc == nil {
			return
		}
		go func() {
			started := time.Now()
			proc.Wait()
			// Si el proceso termina enseguida, Edge delegó en otra instancia: dependemos de los latidos.
			if time.Since(started) > 5*time.Second {
				windowGone()
			}
		}()
	}
	openUI := func() {
		uiMu.Lock()
		defer uiMu.Unlock()
		if winhost.FocusWindow() {
			return
		}
		winhost.Cleanup("window", dir) // restos de la ventana anterior, para abrir una limpia
		watch(winhost.OpenWindowMax(url+"?nointro=1", dir))
	}
	winhost.StartTray("OverdriveTuner", openUI, srv.Close)
	winhost.ListenActivate(openUI)
	srv.WatchWindow(40*time.Second, windowGone)
	watch(winhost.OpenWindowMax(url, dir))
	<-srv.Closed
	time.Sleep(300 * time.Millisecond)
	winhost.TrayRemove()
	// Cierre total: overlay y la ventana, para poder volver a abrir sin conflictos.
	winhost.Cleanup("all", dir)
	os.Exit(0)
}

// runTask hace una tarea de mantenimiento programada, sin ventana, y termina.
func runTask(id string) {
	dir := dataDirPath()
	_ = os.MkdirAll(dir, 0o755)
	if f, err := os.OpenFile(filepath.Join(dir, "overdrivetuner.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(f)
		defer f.Close()
	}
	msg, err := schedule.Run(system.New(), dir, id)
	if err != nil {
		log.Printf("tarea %s: %v", id, err)
		os.Exit(1)
	}
	log.Printf("tarea %s: %s", id, msg)
}

// runUninstall abre el desinstalador (una ventana que explica qué se quita y qué se conserva).
// Solo actúa si esta copia es la instalada, para no borrar por error otra carpeta.
func runUninstall(demo, noWindow bool) {
	var sys system.System
	dir := filepath.Join(os.TempDir(), "overdrivetuner-demo-uninst")
	dataDir := filepath.Join(os.TempDir(), "overdrivetuner-demo")
	if demo {
		sys = system.NewFake()
		_ = os.MkdirAll(dataDir, 0o755)
	} else {
		sys = system.New()
		if !sys.IsAdmin() && winhost.EnsureElevated() {
			return
		}
		exe, err := os.Executable()
		if err != nil {
			winhost.Fatal("No se pudo localizar el programa: " + err.Error())
		}
		dir = filepath.Dir(exe)
		dataDir = dataDirPath()
		if inst := installer.InstalledDir(sys); inst == "" || !installer.SamePath(inst, dir) {
			winhost.Fatal("Esta copia no está instalada con el instalador, así que no se desinstala desde aquí.")
		}
		if !winhost.SingleInstance("OverdriveTunerUninstallSingleInstance") {
			return
		}
	}
	logDir := filepath.Join(os.TempDir(), "OverdriveTunerUninstall")
	_ = os.MkdirAll(logDir, 0o755)
	if f, err := os.OpenFile(filepath.Join(logDir, "uninstall.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(f)
		defer f.Close()
	}
	uninst.Run(&uninst.Server{Sys: sys, Dir: dir, DataDir: dataDir, Demo: demo}, noWindow, logDir)
}
