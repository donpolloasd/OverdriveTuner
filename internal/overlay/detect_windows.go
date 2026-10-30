//go:build windows

package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	dUser32          = syscall.NewLazyDLL("user32.dll")
	dKernel32        = syscall.NewLazyDLL("kernel32.dll")
	pGetForeground   = dUser32.NewProc("GetForegroundWindow")
	pGetWindowRect   = dUser32.NewProc("GetWindowRect")
	pGetWindowPID    = dUser32.NewProc("GetWindowThreadProcessId")
	pMonitorFromWnd  = dUser32.NewProc("MonitorFromWindow")
	pGetMonitorInfoW = dUser32.NewProc("GetMonitorInfoW")
	pIsWindowVisible = dUser32.NewProc("IsWindowVisible")
	pIsIconic        = dUser32.NewProc("IsIconic")
	pOpenProcess     = dKernel32.NewProc("OpenProcess")
	pQueryImageName  = dKernel32.NewProc("QueryFullProcessImageNameW")
	pCloseHandle     = dKernel32.NewProc("CloseHandle")
	pQueryUNS        = syscall.NewLazyDLL("shell32.dll").NewProc("SHQueryUserNotificationState")
)

type dRect struct{ L, T, R, B int32 }

// exclusiveFullscreen dice si Windows informa que hay un programa Direct3D en pantalla completa exclusiva
// (QUNS_RUNNING_D3D_FULL_SCREEN). En ese modo ninguna ventana externa, tampoco el overlay, puede dibujarse
// encima. Solo consulta el estado del escritorio: no toca el juego.
func exclusiveFullscreen() (on bool) {
	defer func() {
		if recover() != nil {
			on = false
		}
	}()
	var st uint32
	if r, _, _ := pQueryUNS.Call(uintptr(unsafe.Pointer(&st))); r != 0 { // HRESULT distinto de S_OK
		return false
	}
	return st == 3
}

// runningExes lista los programas abiertos (minúsculas → nombre original) leyendo la lista de procesos de
// Windows directamente, sin lanzar tasklist.exe (que tarda cientos de milisegundos). ok=false si falló.
func runningExes() (m map[string]string, ok bool) {
	defer func() {
		if recover() != nil {
			m, ok = nil, false
		}
	}()
	snap, _, _ := pSnapshot.Call(2, 0) // TH32CS_SNAPPROCESS
	if snap == ^uintptr(0) || snap == 0 {
		return nil, false
	}
	defer pCloseHandle.Call(snap)
	m = map[string]string{}
	var pe procEntry
	pe.Size = uint32(unsafe.Sizeof(pe))
	r, _, _ := pProcFirst.Call(snap, uintptr(unsafe.Pointer(&pe)))
	for r != 0 {
		if name := syscall.UTF16ToString(pe.Exe[:]); name != "" {
			m[strings.ToLower(name)] = name
		}
		r, _, _ = pProcNext.Call(snap, uintptr(unsafe.Pointer(&pe)))
	}
	return m, len(m) > 0
}

// foregroundFullscreenExe devuelve el programa de la ventana activa si ocupa toda la pantalla
// (pantalla completa o sin bordes) y no es del sistema. Solo consulta ventanas: no toca el juego.
func foregroundFullscreenExe() string { return foregroundExe(true) }

// foregroundAnyExe devuelve el programa de la ventana activa aunque esté en ventana (no del sistema).
func foregroundAnyExe() string { return foregroundExe(false) }

func foregroundExe(needFull bool) (exe string) {
	defer func() {
		if recover() != nil {
			exe = ""
		}
	}()
	hwnd, _, _ := pGetForeground.Call()
	if hwnd == 0 {
		return ""
	}
	if v, _, _ := pIsWindowVisible.Call(hwnd); v == 0 {
		return ""
	}
	if v, _, _ := pIsIconic.Call(hwnd); v != 0 {
		return ""
	}
	var wr dRect
	if r, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr))); r == 0 {
		return ""
	}
	mon, _, _ := pMonitorFromWnd.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
	var mi struct {
		Size    uint32
		Monitor dRect
		Work    dRect
		Flags   uint32
	}
	mi.Size = uint32(unsafe.Sizeof(mi))
	if r, _, _ := pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return ""
	}
	m := mi.Monitor
	if needFull && (wr.L > m.L || wr.T > m.T || wr.R < m.R || wr.B < m.B) {
		return ""
	}
	var pid uint32
	pGetWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 || int(pid) == os.Getpid() {
		return ""
	}
	h, _, _ := pOpenProcess.Call(0x1000, 0, uintptr(pid)) // PROCESS_QUERY_LIMITED_INFORMATION
	if h == 0 {
		return ""
	}
	defer pCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	if r, _, _ := pQueryImageName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r == 0 {
		return ""
	}
	full := syscall.UTF16ToString(buf[:n])
	win := strings.ToLower(os.Getenv("SystemRoot"))
	if win != "" && strings.HasPrefix(strings.ToLower(full), win+`\`) {
		return "" // programas de Windows
	}
	return filepath.Base(full)
}

type procEntry struct {
	Size     uint32
	Usage    uint32
	PID      uint32
	HeapID   uintptr
	ModuleID uint32
	Threads  uint32
	Parent   uint32
	PriClass int32
	Flags    uint32
	Exe      [260]uint16
}

var (
	pSnapshot    = dKernel32.NewProc("CreateToolhelp32Snapshot")
	pProcFirst   = dKernel32.NewProc("Process32FirstW")
	pProcNext    = dKernel32.NewProc("Process32NextW")
	pEnumWindows = dUser32.NewProc("EnumWindows")
	pWinPID      = dUser32.NewProc("GetWindowThreadProcessId")
	enumCB       = syscall.NewCallback(enumWindowsProc)
	visiblePIDs  map[uint32]bool
)

func imagePath(pid uint32) string {
	h, _, _ := pOpenProcess.Call(0x1000, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer pCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	if r, _, _ := pQueryImageName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func enumWindowsProc(hwnd, _ uintptr) uintptr {
	if v, _, _ := pIsWindowVisible.Call(hwnd); v == 0 {
		return 1
	}
	if v, _, _ := pIsIconic.Call(hwnd); v != 0 {
		return 1
	}
	var wr dRect
	if r, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr))); r == 0 {
		return 1
	}
	if wr.R-wr.L < 320 || wr.B-wr.T < 240 {
		return 1
	}
	var pid uint32
	pWinPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid != 0 {
		visiblePIDs[pid] = true
	}
	return 1
}

// launcherGame devuelve un programa que está instalado en la carpeta de juegos de un launcher (Steam,
// Epic, EA…) y tiene una ventana visible. Si la ventana activa es de uno de ellos, ese tiene prioridad.
func launcherGame(extra []string) (exe string) {
	defer func() {
		if recover() != nil {
			exe = ""
		}
	}()
	visiblePIDs = map[uint32]bool{}
	pEnumWindows.Call(enumCB, 0)
	snap, _, _ := pSnapshot.Call(2, 0) // TH32CS_SNAPPROCESS
	if snap == ^uintptr(0) || snap == 0 {
		return ""
	}
	defer pCloseHandle.Call(snap)
	var fgPID uint32
	if hwnd, _, _ := pGetForeground.Call(); hwnd != 0 {
		pWinPID.Call(hwnd, uintptr(unsafe.Pointer(&fgPID)))
	}
	var pe procEntry
	pe.Size = uint32(unsafe.Sizeof(pe))
	first := ""
	ok, _, _ := pProcFirst.Call(snap, uintptr(unsafe.Pointer(&pe)))
	for ok != 0 {
		if visiblePIDs[pe.PID] {
			if p := imagePath(pe.PID); p != "" && isLauncherGamePath(p, extra...) {
				name := filepath.Base(p)
				if pe.PID == fgPID {
					return name
				}
				if first == "" {
					first = name
				}
			}
		}
		ok, _, _ = pProcNext.Call(snap, uintptr(unsafe.Pointer(&pe)))
	}
	return first
}
