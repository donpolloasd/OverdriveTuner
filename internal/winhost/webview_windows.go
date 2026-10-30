//go:build windows

package winhost

// Ventana propia con WebView2 (el motor web que trae Windows 10/11) adentro. Reemplaza a la ventana de
// Edge en modo --app: el programa tiene su propia ventana, ícono y menú en la barra de tareas, sin menú
// contextual de navegador ni atajos como F5, Ctrl+P o F12. Está escrito con llamadas COM a mano para no
// depender de CGO ni de WebView2Loader.dll: se carga directamente el runtime instalado en Windows.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	ole32  = syscall.NewLazyDLL("ole32.dll")
	dwmapi = syscall.NewLazyDLL("dwmapi.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
)

// AppOptions describe la ventana propia.
type AppOptions struct {
	Title         string
	Maximized     bool
	Width, Height int
	HideOnClose   bool   // la X esconde la ventana (el programa sigue en la bandeja) en vez de cerrarla
	OnClose       func() // se llama cuando el usuario aprieta la X
}

// AppWindow es la ventana propia del programa con la interfaz adentro.
type AppWindow struct {
	hwnd     uintptr
	ctrl     uintptr // ICoreWebView2Controller
	opts     AppOptions
	done     chan struct{}
	quitting bool
}

const (
	wmAppShow = 0x8000 + 41 // WM_APP: mostrar y traer al frente
	wmAppQuit = 0x8000 + 42 // WM_APP: cerrar de verdad
)

var (
	curApp   *AppWindow // una sola ventana por proceso
	appClass sync.Once
	appCB    uintptr
)

// ptr convierte una dirección que entregó Windows (no es memoria de Go) en un puntero.
func ptr(a uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&a)) }

// comCall llama al método número idx de la tabla virtual de un objeto COM.
func comCall(obj uintptr, idx int, args ...uintptr) uintptr {
	vt := *(*uintptr)(ptr(obj))
	fn := *(*uintptr)(ptr(vt + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

// ---- objetos COM propios: los «completed handlers» que WebView2 llama al terminar ----

type comHandler struct {
	vtbl   *[4]uintptr
	invoke func(hr, obj uintptr)
}

var (
	handlersMu sync.Mutex
	handlers   = map[uintptr]*comHandler{} // también los mantiene vivos
	handlerVt  *[4]uintptr
)

func newHandler(invoke func(hr, obj uintptr)) uintptr {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	if handlerVt == nil {
		handlerVt = &[4]uintptr{
			syscall.NewCallback(func(this, riid, out uintptr) uintptr { // QueryInterface
				*(*uintptr)(ptr(out)) = this
				return 0
			}),
			syscall.NewCallback(func(this uintptr) uintptr { return 1 }), // AddRef
			syscall.NewCallback(func(this uintptr) uintptr { return 1 }), // Release
			syscall.NewCallback(func(this, hr, obj uintptr) uintptr { // Invoke
				handlersMu.Lock()
				h := handlers[this]
				handlersMu.Unlock()
				if h != nil {
					h.invoke(hr, obj)
				}
				return 0
			}),
		}
	}
	h := &comHandler{vtbl: handlerVt, invoke: invoke}
	p := uintptr(unsafe.Pointer(h))
	handlers[p] = h
	return p
}

// ---- runtime de WebView2 ----

// webviewDLL devuelve la ruta de EmbeddedBrowserWebView.dll del runtime de WebView2 instalado («» si falta).
func webviewDLL() string {
	const client = `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	for _, root := range []syscall.Handle{syscall.HKEY_LOCAL_MACHINE, syscall.HKEY_CURRENT_USER} {
		for _, key := range []string{client, strings.Replace(client, `WOW6432Node\`, "", 1)} {
			loc, ver := regString(root, key, "location"), regString(root, key, "pv")
			if loc == "" || ver == "" || ver == "0.0.0.0" {
				continue
			}
			p := filepath.Join(loc, ver, "EBWebView", "x64", "EmbeddedBrowserWebView.dll")
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

func regString(root syscall.Handle, path, name string) string {
	var k syscall.Handle
	if syscall.RegOpenKeyEx(root, utf16(path), 0, syscall.KEY_READ, &k) != nil {
		return ""
	}
	defer syscall.RegCloseKey(k)
	buf := make([]uint16, 512)
	n := uint32(len(buf) * 2)
	var typ uint32
	if syscall.RegQueryValueEx(k, utf16(name), nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n) != nil || typ != syscall.REG_SZ {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// WebViewAvailable dice si se puede usar la ventana propia (está instalado el runtime de WebView2).
func WebViewAvailable() bool { return webviewDLL() != "" }

// ---- ventana ----

// OpenApp abre la ventana propia con la interfaz (url) adentro y devuelve cuando ya está cargando.
// Si WebView2 no está o falla, devuelve error y no queda ninguna ventana: se usa la de Edge como respaldo.
func OpenApp(url, dataDir string, o AppOptions) (*AppWindow, error) {
	dll := webviewDLL()
	if dll == "" {
		return nil, errors.New("no está instalado el runtime de WebView2")
	}
	if o.Width == 0 {
		o.Width, o.Height = 1240, 800
	}
	w := &AppWindow{opts: o, done: make(chan struct{})}
	ready := make(chan error, 1)
	go w.run(dll, url, filepath.Join(dataDir, "webview"), ready)
	select {
	case err := <-ready:
		if err != nil {
			<-w.done
			return nil, err
		}
		return w, nil
	case <-time.After(30 * time.Second):
		w.Close()
		return nil, errors.New("WebView2 no respondió")
	}
}

// Done se cierra cuando la ventana dejó de existir.
func (w *AppWindow) Done() <-chan struct{} { return w.done }

// Show muestra la ventana (si estaba escondida o minimizada) y la trae al frente.
func (w *AppWindow) Show() { user32.NewProc("PostMessageW").Call(w.hwnd, wmAppShow, 0, 0) }

// Close cierra la ventana de verdad (aunque HideOnClose esté activo).
func (w *AppWindow) Close() { user32.NewProc("PostMessageW").Call(w.hwnd, wmAppQuit, 0, 0) }

type wndClassEx struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}

type rect struct{ L, T, R, B int32 }

type msgT struct {
	Hwnd, Msg, WParam, LParam uintptr
	Time                      uint32
	X, Y                      int32
	_                         uint32
}

func (w *AppWindow) run(dll, url, userData string, ready chan<- error) {
	runtime.LockOSThread()
	defer close(w.done)
	answer := func(err error) { // OpenApp espera una sola respuesta
		select {
		case ready <- err:
		default:
		}
	}
	fail := func(err error) {
		answer(err)
		if w.hwnd != 0 {
			w.quitting = true
			user32.NewProc("DestroyWindow").Call(w.hwnd)
		}
	}
	ole32.NewProc("CoInitializeEx").Call(0, 2) // COINIT_APARTMENTTHREADED
	defer ole32.NewProc("CoUninitialize").Call()
	// Nitidez en pantallas con escala (125 %, 150 %…): la ventana se dibuja según el DPI de cada monitor.
	if p := user32.NewProc("SetThreadDpiAwarenessContext"); p.Find() == nil {
		p.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	}

	inst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	cls := utf16("OverdriveTunerVentana")
	appClass.Do(func() {
		appCB = syscall.NewCallback(appWndProc)
		icon, _, _ := user32.NewProc("LoadIconW").Call(inst, 1) // grupo de iconos 1: el del .exe
		cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
		brush, _, _ := gdi32.NewProc("GetStockObject").Call(4) // negro, mientras carga la interfaz
		wc := wndClassEx{WndProc: appCB, Instance: inst, Icon: icon, Cursor: cursor, Background: brush, ClassName: cls, IconSm: icon}
		wc.Size = uint32(unsafe.Sizeof(wc))
		user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	})
	curApp = w
	const cwDefault = 0x80000000
	h, _, err := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16(w.opts.Title))),
		0x00CF0000, cwDefault, cwDefault, uintptr(w.opts.Width), uintptr(w.opts.Height), 0, 0, inst, 0) // WS_OVERLAPPEDWINDOW
	if h == 0 {
		fail(fmt.Errorf("no se pudo crear la ventana: %v", err))
		return
	}
	w.hwnd = h
	one := int32(1)
	dwmapi.NewProc("DwmSetWindowAttribute").Call(h, 20, uintptr(unsafe.Pointer(&one)), 4) // barra de título oscura
	show := uintptr(1)                                                                    // SW_SHOWNORMAL
	if w.opts.Maximized {
		show = 3 // SW_SHOWMAXIMIZED
	}
	user32.NewProc("ShowWindow").Call(h, show)
	user32.NewProc("UpdateWindow").Call(h)

	// Fondo negro desde el primer cuadro (sin destello blanco) y las mismas opciones que tenía la ventana de Edge.
	os.Setenv("WEBVIEW2_DEFAULT_BACKGROUND_COLOR", "FF000000")
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--autoplay-policy=no-user-gesture-required --disable-background-timer-throttling --disable-renderer-backgrounding --disable-backgrounding-occluded-windows --disable-features=Translate,msSmartScreenProtection")

	onCtrl := newHandler(func(hr, ctrl uintptr) {
		if hr != 0 || ctrl == 0 {
			fail(fmt.Errorf("WebView2 no pudo crear la vista (0x%08X)", uint32(hr)))
			return
		}
		comCall(ctrl, 1) // AddRef: la guardamos
		w.ctrl = ctrl
		var wv, st uintptr
		comCall(ctrl, 25, uintptr(unsafe.Pointer(&wv))) // get_CoreWebView2
		if wv == 0 {
			fail(errors.New("WebView2 no entregó la vista"))
			return
		}
		if comCall(wv, 3, uintptr(unsafe.Pointer(&st))) == 0 && st != 0 { // get_Settings
			comCall(st, 10, 0) // put_IsStatusBarEnabled(false)
			comCall(st, 12, 0) // put_AreDevToolsEnabled(false)
			comCall(st, 14, 0) // put_AreDefaultContextMenusEnabled(false): nada de «Atrás, Actualizar, Inspeccionar…»
			comCall(st, 18, 0) // put_IsZoomControlEnabled(false)
			var st3 uintptr
			if comCall(st, 0, uintptr(unsafe.Pointer(&iidSettings3)), uintptr(unsafe.Pointer(&st3))) == 0 && st3 != 0 {
				comCall(st3, 24, 0) // put_AreBrowserAcceleratorKeysEnabled(false): sin F5, Ctrl+R, Ctrl+P, Ctrl+U, F12…
				comCall(st3, 2)     // Release
			}
			comCall(st, 2)
		}
		// Permisos sin carteles: leer el portapapeles sí («Pegar» del menú propio); cámara, micrófono,
		// ubicación y notificaciones no (la interfaz no los usa).
		var token int64
		comCall(wv, 23, newHandler(func(_, args uintptr) { // add_PermissionRequested
			var kind int32
			comCall(args, 4, uintptr(unsafe.Pointer(&kind))) // get_PermissionKind
			state := uintptr(2)                              // DENY
			if kind == 6 {                                   // CLIPBOARD_READ
				state = 1 // ALLOW
			}
			comCall(args, 7, state) // put_State
		}), uintptr(unsafe.Pointer(&token)))
		w.resize()
		comCall(wv, 5, uintptr(unsafe.Pointer(utf16(url)))) // Navigate
		comCall(wv, 2)
		comCall(ctrl, 12, 0) // MoveFocus: el teclado va a la interfaz
		answer(nil)
	})
	onEnv := newHandler(func(hr, env uintptr) {
		if hr != 0 || env == 0 {
			fail(fmt.Errorf("WebView2 no pudo iniciar (0x%08X)", uint32(hr)))
			return
		}
		if r := comCall(env, 3, w.hwnd, onCtrl); r != 0 { // CreateCoreWebView2Controller
			fail(fmt.Errorf("WebView2 rechazó la ventana (0x%08X)", uint32(r)))
		}
	})
	_ = os.MkdirAll(userData, 0o755)
	create := syscall.NewLazyDLL(dll).NewProc("CreateWebViewEnvironmentWithOptionsInternal")
	if err := create.Find(); err != nil {
		fail(errors.New("el runtime de WebView2 no tiene la función esperada"))
	} else if r, _, _ := create.Call(1, 0, uintptr(unsafe.Pointer(utf16(userData))), 0, onEnv); r != 0 {
		fail(fmt.Errorf("WebView2 no pudo iniciar (0x%08X)", uint32(r)))
	}

	var m msgT
	for {
		r, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
	}
}

// IID_ICoreWebView2Settings3 {FDB5AB74-AF33-4854-84F0-0A631DEB5EBA}
var iidSettings3 = syscall.GUID{Data1: 0xfdb5ab74, Data2: 0xaf33, Data3: 0x4854, Data4: [8]byte{0x84, 0xf0, 0x0a, 0x63, 0x1d, 0xeb, 0x5e, 0xba}}

func (w *AppWindow) resize() {
	if w.ctrl == 0 {
		return
	}
	var rc rect
	user32.NewProc("GetClientRect").Call(w.hwnd, uintptr(unsafe.Pointer(&rc)))
	comCall(w.ctrl, 6, uintptr(unsafe.Pointer(&rc))) // put_Bounds (el RECT va por referencia en x64)
}

func appWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	w := curApp
	if w != nil && w.hwnd == hwnd {
		switch msg {
		case 0x0005: // WM_SIZE
			w.resize()
			return 0
		case 0x0003: // WM_MOVE
			if w.ctrl != 0 {
				comCall(w.ctrl, 23) // NotifyParentWindowPositionChanged (listas desplegables en su lugar)
			}
		case 0x0007: // WM_SETFOCUS
			if w.ctrl != 0 {
				comCall(w.ctrl, 12, 0) // MoveFocus
			}
		case 0x0024: // WM_GETMINMAXINFO: tamaño mínimo usable
			mi := (*[10]int32)(ptr(lp))
			mi[6], mi[7] = 900, 600
			return 0
		case 0x0010: // WM_CLOSE
			if w.opts.HideOnClose && !w.quitting {
				user32.NewProc("ShowWindow").Call(hwnd, 0) // SW_HIDE: el programa sigue en la bandeja
				if w.ctrl != 0 {
					comCall(w.ctrl, 4, 0) // put_IsVisible(false): ahorra recursos mientras está escondida
				}
				if w.opts.OnClose != nil {
					go w.opts.OnClose()
				}
				return 0
			}
			if w.opts.OnClose != nil && !w.quitting {
				go w.opts.OnClose()
			}
			user32.NewProc("DestroyWindow").Call(hwnd)
			return 0
		case wmAppShow:
			if w.ctrl != 0 {
				comCall(w.ctrl, 4, 1) // put_IsVisible(true)
			}
			if ic, _, _ := user32.NewProc("IsIconic").Call(hwnd); ic != 0 {
				user32.NewProc("ShowWindow").Call(hwnd, 9) // SW_RESTORE
			} else if v, _, _ := user32.NewProc("IsWindowVisible").Call(hwnd); v == 0 {
				user32.NewProc("ShowWindow").Call(hwnd, 5) // SW_SHOW
			}
			user32.NewProc("SetForegroundWindow").Call(hwnd)
			if w.ctrl != 0 {
				comCall(w.ctrl, 12, 0)
			}
			return 0
		case wmAppQuit:
			w.quitting = true
			user32.NewProc("DestroyWindow").Call(hwnd)
			return 0
		case 0x0002: // WM_DESTROY
			if w.ctrl != 0 {
				comCall(w.ctrl, 24) // Close
				comCall(w.ctrl, 2)  // Release
				w.ctrl = 0
			}
			user32.NewProc("PostQuitMessage").Call(0)
			return 0
		}
	}
	r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, msg, wp, lp)
	return r
}
