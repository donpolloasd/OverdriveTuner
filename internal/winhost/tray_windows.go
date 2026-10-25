//go:build windows

package winhost

import (
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Ícono en la bandeja del sistema: al cerrar la ventana con la X el programa
// sigue activo acá (overlay incluido) y se vuelve a abrir con un clic.

const (
	wmApp        = 0x8000
	wmTray       = wmApp + 1
	wmNull       = 0x0000
	wmLButtonUp  = 0x0202
	wmLDblClk    = 0x0203
	wmRButtonUp  = 0x0205
	nimAdd       = 0
	nimModify    = 1
	nimDelete    = 2
	nifMessage   = 0x1
	nifIcon      = 0x2
	nifTip       = 0x4
	nifInfo      = 0x10
	niifInfo     = 0x1
	mfString     = 0
	tpmReturnCmd = 0x100
	tpmNoNotify  = 0x80
	tpmRightBtn  = 0x2
	menuOpen     = 1
	menuQuit     = 2
)

type notifyIconData struct {
	Size       uint32
	Wnd        uintptr
	ID         uint32
	Flags      uint32
	CallbackMs uint32
	Icon       uintptr
	Tip        [128]uint16
	State      uint32
	StateMask  uint32
	Info       [256]uint16
	Timeout    uint32
	InfoTitle  [64]uint16
	InfoFlags  uint32
	GUID       [16]byte
	BalloonIco uintptr
}

type trayWndClass struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type trayPoint struct{ X, Y int32 }
type trayMsg struct {
	Wnd     uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      trayPoint
}

var (
	trayHwnd    uintptr
	trayIcon    uintptr
	trayOnOpen  func()
	trayOnQuit  func()
	trayReady   = make(chan struct{})
	trayStarted bool
	trayCreated uint32 // mensaje «TaskbarCreated»: el Explorador se reinició y hay que volver a poner el ícono
	trayTip     string
)

func copyU16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = append(u[:len(dst)-1], 0)
	}
	copy(dst, u)
}

func trayData(flags uint32) *notifyIconData {
	d := &notifyIconData{Wnd: trayHwnd, ID: 1, Flags: flags, CallbackMs: wmTray, Icon: trayIcon}
	d.Size = uint32(unsafe.Sizeof(*d))
	return d
}

func trayProc(hwnd, msg, wp, lp uintptr) uintptr {
	if trayCreated != 0 && uint32(msg) == trayCreated {
		d := trayData(nifMessage | nifIcon | nifTip)
		copyU16(d.Tip[:], trayTip)
		shell32.NewProc("Shell_NotifyIconW").Call(nimAdd, uintptr(unsafe.Pointer(d)))
		return 0
	}
	if msg == wmTray {
		switch lp & 0xFFFF {
		case wmLButtonUp, wmLDblClk:
			if trayOnOpen != nil {
				go trayOnOpen()
			}
		case wmRButtonUp:
			trayMenu(hwnd)
		}
		return 0
	}
	r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, msg, wp, lp)
	return r
}

func trayMenu(hwnd uintptr) {
	menu, _, _ := user32.NewProc("CreatePopupMenu").Call()
	if menu == 0 {
		return
	}
	defer user32.NewProc("DestroyMenu").Call(menu)
	user32.NewProc("AppendMenuW").Call(menu, mfString, menuOpen, uintptr(unsafe.Pointer(utf16(trayOpenLabel))))
	user32.NewProc("AppendMenuW").Call(menu, mfString, menuQuit, uintptr(unsafe.Pointer(utf16(trayQuitLabel))))
	var pt trayPoint
	user32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&pt)))
	user32.NewProc("SetForegroundWindow").Call(hwnd)
	cmd, _, _ := user32.NewProc("TrackPopupMenu").Call(menu, tpmReturnCmd|tpmNoNotify|tpmRightBtn, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	user32.NewProc("PostMessageW").Call(hwnd, wmNull, 0, 0)
	switch cmd {
	case menuOpen:
		if trayOnOpen != nil {
			go trayOnOpen()
		}
	case menuQuit:
		if trayOnQuit != nil {
			go trayOnQuit()
		}
	}
}

var trayOpenLabel, trayQuitLabel = "Abrir OverdriveTuner", "Salir por completo"

// SetTrayLabels cambia los textos del menú del ícono (hay que llamarlo antes de StartTray).
func SetTrayLabels(open, quit string) { trayOpenLabel, trayQuitLabel = open, quit }

// StartTray crea el ícono. onOpen se llama con el clic; onQuit con «Salir por completo».
func StartTray(tip string, onOpen, onQuit func()) {
	if trayStarted {
		return
	}
	trayStarted = true
	trayOnOpen, trayOnQuit, trayTip = onOpen, onQuit, tip
	go func() {
		runtime.LockOSThread()
		inst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
		ic, _, _ := user32.NewProc("LoadIconW").Call(inst, 1) // grupo de iconos 1: el del .exe
		if ic == 0 {
			ic, _, _ = user32.NewProc("LoadIconW").Call(0, 32512)
		}
		trayIcon = ic
		if m, _, _ := user32.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(utf16("TaskbarCreated")))); m != 0 {
			trayCreated = uint32(m)
		}
		cls := utf16("OverdriveTunerTray")
		wc := trayWndClass{Instance: inst, ClassName: cls, WndProc: syscall.NewCallback(trayProc)}
		wc.Size = uint32(unsafe.Sizeof(wc))
		user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
		h, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("OverdriveTuner Tray"))), 0, 0, 0, 0, 0, 0, 0, inst, 0)
		if h == 0 {
			close(trayReady)
			return
		}
		trayHwnd = h
		d := trayData(nifMessage | nifIcon | nifTip)
		copyU16(d.Tip[:], tip)
		shell32.NewProc("Shell_NotifyIconW").Call(nimAdd, uintptr(unsafe.Pointer(d)))
		close(trayReady)
		var m trayMsg
		for {
			r, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
			user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
			user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
		}
	}()
	select {
	case <-trayReady:
	case <-time.After(2 * time.Second):
	}
}

// TrayNotify muestra un aviso breve junto al ícono.
func TrayNotify(title, text string) {
	if trayHwnd == 0 {
		return
	}
	d := trayData(nifInfo)
	copyU16(d.Info[:], text)
	copyU16(d.InfoTitle[:], title)
	d.InfoFlags = niifInfo
	shell32.NewProc("Shell_NotifyIconW").Call(nimModify, uintptr(unsafe.Pointer(d)))
}

// TrayRemove quita el ícono (hay que llamarlo antes de salir, si no queda un fantasma).
func TrayRemove() {
	if trayHwnd == 0 {
		return
	}
	d := trayData(0)
	shell32.NewProc("Shell_NotifyIconW").Call(nimDelete, uintptr(unsafe.Pointer(d)))
	trayHwnd = 0
}

const showEventName = `Global\OverdriveTunerShow`

// ListenActivate espera el aviso de una segunda copia que se intentó abrir y llama a onShow.
func ListenActivate(onShow func()) {
	h, _, _ := kernel32.NewProc("CreateEventW").Call(0, 0, 0, uintptr(unsafe.Pointer(utf16(showEventName))))
	if h == 0 {
		return
	}
	go func() {
		for {
			r, _, _ := kernel32.NewProc("WaitForSingleObject").Call(h, 0xFFFFFFFF)
			if r != 0 {
				return
			}
			onShow()
		}
	}()
}

// ActivateExisting avisa a la copia que ya está abierta (en la bandeja o con ventana) y devuelve true si la encontró.
func ActivateExisting() bool {
	h, _, _ := kernel32.NewProc("OpenEventW").Call(0x0002, 0, uintptr(unsafe.Pointer(utf16(showEventName)))) // EVENT_MODIFY_STATE
	if h == 0 {
		return false
	}
	kernel32.NewProc("SetEvent").Call(h)
	kernel32.NewProc("CloseHandle").Call(h)
	return true
}

// FocusWindow trae al frente la ventana de la interfaz si existe (aunque esté minimizada).
func FocusWindow() bool {
	var found uintptr
	cb := syscall.NewCallback(func(h, _ uintptr) uintptr {
		cls := make([]uint16, 64)
		user32.NewProc("GetClassNameW").Call(h, uintptr(unsafe.Pointer(&cls[0])), 64)
		if syscall.UTF16ToString(cls) != "Chrome_WidgetWin_1" {
			return 1
		}
		t := make([]uint16, 128)
		user32.NewProc("GetWindowTextW").Call(h, uintptr(unsafe.Pointer(&t[0])), 128)
		if strings.HasPrefix(syscall.UTF16ToString(t), "OverdriveTuner") {
			if v, _, _ := user32.NewProc("IsWindowVisible").Call(h); v != 0 {
				found = h
				return 0
			}
		}
		return 1
	})
	user32.NewProc("EnumWindows").Call(cb, 0)
	if found == 0 {
		return false
	}
	if ic, _, _ := user32.NewProc("IsIconic").Call(found); ic != 0 {
		user32.NewProc("ShowWindow").Call(found, 9) // SW_RESTORE
	}
	user32.NewProc("SetForegroundWindow").Call(found)
	return true
}
