//go:build windows

package winhost

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
)

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func Fatal(msg string) {
	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(utf16(msg))), uintptr(unsafe.Pointer(utf16("OverdriveTuner"))), 0x10)
	os.Exit(1)
}

func quoteArg(a string) string {
	if a != "" && !strings.ContainsAny(a, " \t\"") {
		return a
	}
	return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
}

// EnsureElevated relanza el programa pidiendo permisos de administrador (UAC).
// Devuelve true si se lanzó la copia elevada y este proceso debe terminar.
func EnsureElevated() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	var args []string
	for _, a := range os.Args[1:] {
		args = append(args, quoteArg(a))
	}
	r, _, _ := shell32.NewProc("ShellExecuteW").Call(0, uintptr(unsafe.Pointer(utf16("runas"))),
		uintptr(unsafe.Pointer(utf16(exe))), uintptr(unsafe.Pointer(utf16(strings.Join(args, " ")))), 0, 1)
	return r > 32 // >32 = éxito; si el usuario rechaza UAC, se sigue sin permisos (modo lectura)
}

var singleMutex uintptr

func acquire(name string) (held bool, ok bool) {
	h, _, err := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(utf16(`Global\`+name))))
	if errno, isErr := err.(syscall.Errno); isErr && errno == 183 {
		if h != 0 {
			kernel32.NewProc("CloseHandle").Call(h)
		}
		return true, false
	}
	singleMutex = h
	return false, h != 0
}

// SingleInstance evita abrir dos copias a la vez. Si quedó una copia colgada
// (por ejemplo, tras cerrar la ventana con la X), la cierra y sigue.
func SingleInstance(name string) bool {
	mode := "stale-main"
	if strings.Contains(name, "Overlay") {
		mode = "stale-overlay"
	}
	for i := 0; i < 4; i++ {
		held, ok := acquire(name)
		if !held {
			return ok
		}
		if i == 0 || i == 2 {
			Cleanup(mode, "")
		}
		time.Sleep(700 * time.Millisecond)
	}
	Fatal("Ya hay otra copia abierta y no se pudo cerrar. Cerrala desde el Administrador de tareas (OverdriveTuner) e intentá de nuevo.")
	return false
}

// Cleanup termina los procesos de OverdriveTuner (y sus hijos) distintos del actual.
// mode: "stale-main" (otras copias de la app), "stale-overlay" (otros overlays) o
// "window" (solo la ventana de la interfaz), "all" (todo: app, overlay, PresentMon propio y la ventana de la interfaz).
func Cleanup(mode, dataDir string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	q := func(x string) string { return strings.ReplaceAll(x, "'", "''") }
	script := `$ErrorActionPreference='SilentlyContinue'
$self=` + strconv.Itoa(os.Getpid()) + `
$mode='` + mode + `'
$exe='` + q(filepath.Base(exe)) + `'
$dir='` + q(dataDir) + `'
$kill = @()
foreach ($p in @(Get-CimInstance Win32_Process -Filter "Name='$exe'")) {
  if ($p.ProcessId -eq $self) { continue }
  $ov = [string]$p.CommandLine -match '--overlay'
  if ($mode -eq 'all' -or ($mode -eq 'stale-main' -and -not $ov) -or ($mode -eq 'stale-overlay' -and $ov)) { $kill += $p.ProcessId }
}
if ($mode -ne 'stale-overlay' -and $dir -ne '') {
  if ($mode -eq 'all') { foreach ($p in @(Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'PresentMon.exe' -and [string]$_.ExecutablePath -like ($dir + '*') })) { $kill += $p.ProcessId } }
  foreach ($p in @(Get-CimInstance Win32_Process | Where-Object { $_.Name -in 'msedge.exe','chrome.exe' -and [string]$_.CommandLine -like ('*' + $dir + '*ventana*') })) { $kill += $p.ProcessId }
}
foreach ($id in ($kill | Select-Object -Unique)) { & taskkill.exe /F /T /PID $id | Out-Null }
`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	done := make(chan struct{})
	go func() { _ = cmd.Run(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}

func findBrowser() string {
	pf86 := os.Getenv("ProgramFiles(x86)")
	pf := os.Getenv("ProgramFiles")
	local := os.Getenv("LOCALAPPDATA")
	for _, p := range []string{
		filepath.Join(pf86, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(pf86, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(local, `Google\Chrome\Application\chrome.exe`),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// OpenWindow abre la interfaz como ventana de aplicación (Edge/Chrome en modo --app).
func OpenWindow(url, dir string) *exec.Cmd { return openWindow(url, dir, false) }

// OpenWindowMax es como OpenWindow pero la ventana se abre maximizada (ocupa toda la pantalla, con su barra de título).
func OpenWindowMax(url, dir string) *exec.Cmd { return openWindow(url, dir, true) }

func openWindow(url, dir string, maximized bool) *exec.Cmd {
	size := "--window-size=1240,800"
	if maximized {
		size = "--start-maximized"
	}
	if b := findBrowser(); b != "" {
		cmd := exec.Command(b, "--app="+url, "--user-data-dir="+filepath.Join(dir, "ventana"),
			size, "--no-first-run", "--no-default-browser-check", "--disable-features=Translate,msEdgeStartupBoost", "--disable-background-mode", "--autoplay-policy=no-user-gesture-required", "--disable-background-timer-throttling", "--disable-renderer-backgrounding", "--disable-backgrounding-occluded-windows")
		if err := cmd.Start(); err == nil {
			return cmd
		}
	}
	// Último recurso: navegador predeterminado.
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	return nil
}

func messageBox(msg string, flags uintptr) int {
	r, _, _ := user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(utf16(msg))), uintptr(unsafe.Pointer(utf16("OverdriveTuner"))), flags)
	return int(r)
}

func AskYesNo(msg string) bool { return messageBox(msg, 0x24) == 6 } // MB_YESNO | MB_ICONQUESTION

func Info(msg string) { messageBox(msg, 0x40) }

// KillByName cierra todos los procesos con ese nombre de programa (y sus hijos), salvo el actual.
func KillByName(name string) {
	cmd := exec.Command("taskkill.exe", "/F", "/T", "/FI", "PID ne "+strconv.Itoa(os.Getpid()), "/IM", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if cmd.Run() == nil { // solo se espera si había algo que cerrar
		time.Sleep(500 * time.Millisecond)
	}
}
