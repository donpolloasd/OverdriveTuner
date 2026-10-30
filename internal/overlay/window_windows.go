package overlay

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi    = syscall.NewLazyDLL("gdi32.dll")
	kern   = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassEx = user32.NewProc("RegisterClassExW")
	pCreateWindowEx  = user32.NewProc("CreateWindowExW")
	pDefWindowProc   = user32.NewProc("DefWindowProcW")
	pGetMessage      = user32.NewProc("GetMessageW")
	pTranslateMsg    = user32.NewProc("TranslateMessage")
	pDispatchMsg     = user32.NewProc("DispatchMessageW")
	pPostMessage     = user32.NewProc("PostMessageW")
	pPostQuit        = user32.NewProc("PostQuitMessage")
	pDestroyWindow   = user32.NewProc("DestroyWindow")
	pSetLayeredAttr  = user32.NewProc("SetLayeredWindowAttributes")
	pSetWindowPos    = user32.NewProc("SetWindowPos")
	pSetWindowRgn    = user32.NewProc("SetWindowRgn")
	pSetTimer        = user32.NewProc("SetTimer")
	pInvalidate      = user32.NewProc("InvalidateRect")
	pBeginPaint      = user32.NewProc("BeginPaint")
	pEndPaint        = user32.NewProc("EndPaint")
	pGetSysMetrics   = user32.NewProc("GetSystemMetrics")
	pRegisterHotKey  = user32.NewProc("RegisterHotKey")
	pUnregHotKey     = user32.NewProc("UnregisterHotKey")
	pShowWindow      = user32.NewProc("ShowWindow")
	pSetDPIAware     = user32.NewProc("SetProcessDPIAware")
	pFillRect        = user32.NewProc("FillRect")

	pCreateFont     = gdi.NewProc("CreateFontW")
	pSelectObject   = gdi.NewProc("SelectObject")
	pDeleteObject   = gdi.NewProc("DeleteObject")
	pSetTextColor   = gdi.NewProc("SetTextColor")
	pSetBkMode      = gdi.NewProc("SetBkMode")
	pTextOut        = gdi.NewProc("TextOutW")
	pGetTextExtent  = gdi.NewProc("GetTextExtentPoint32W")
	pCreateCompDC   = gdi.NewProc("CreateCompatibleDC")
	pCreateCompBmp  = gdi.NewProc("CreateCompatibleBitmap")
	pDeleteDC       = gdi.NewProc("DeleteDC")
	pBitBlt         = gdi.NewProc("BitBlt")
	pCreateSolidBr  = gdi.NewProc("CreateSolidBrush")
	pCreateRoundRgn = gdi.NewProc("CreateRoundRectRgn")
	pFrameRgn       = gdi.NewProc("FrameRgn")
	pGetModule      = kern.NewProc("GetModuleHandleW")
)

type wndClassEx struct {
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

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
	Private uint32
}

type rect struct{ L, T, R, B int32 }
type size struct{ CX, CY int32 }

const (
	wsPopup         = 0x80000000
	wsExTopmost     = 0x00000008
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000
	swpNoActivate   = 0x0010
	swpShow         = 0x0040
	wmDestroy       = 0x0002
	wmClose         = 0x0010
	wmPaint         = 0x000F
	wmEraseBkgnd    = 0x0014
	wmNCHitTest     = 0x0084
	wmTimer         = 0x0113
	wmHotkey        = 0x0312
	lwaAlpha        = 0x2
	lwaColorKey     = 0x1
	// keyColor es el «color invisible» del fondo transparente: ningún tema lo usa y el texto lleva contorno negro.
	keyColor = 0x010101
	srcCopy  = 0x00CC0020
)

type win struct {
	e        *Engine
	hkOn     [4]string // atajo registrado por id (1-3)
	hkTried  [4]string // último atajo intentado (aunque haya fallado)
	hwnd     uintptr
	scale    float64
	fonts    [3]uintptr // grande, normal, pequeña
	fontKey  string
	rgnKey   string // tamaño y forma con los que se armó la región de la ventana
	layKey   string // opacidad y transparencia aplicadas
	outline  bool   // fondo transparente: el texto lleva contorno para leerse sobre el juego
	w, h     int
	x, y     int
	alpha    byte
	visible  bool
	lastLine string
	timerMs  int
}

var cur *win // una sola ventana por proceso

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// makeFonts arma las tres letras (grande, normal, pequeña) según el tamaño y el estilo elegidos.
func (w *win) makeFonts(c Config) {
	st := c.Style
	key := fmt.Sprint(c.Scale, st.Font, st.Weight, st.FpsSize, st.Transparent)
	if w.fontKey == key && w.fonts[0] != 0 {
		return
	}
	for _, f := range w.fonts {
		if f != 0 {
			pDeleteObject.Call(f)
		}
	}
	quality := uintptr(5) // CLEARTYPE_QUALITY
	if st.Transparent {
		quality = 4 // ANTIALIASED_QUALITY: sin bordes de colores sobre el fondo invisible
	}
	face := Fonts[st.Font]
	mk := func(px float64, weight int, face string) uintptr {
		h := -int32(px * c.Scale)
		r, _, _ := pCreateFont.Call(uintptr(h), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, quality, 0, uintptr(unsafe.Pointer(u16(face))))
		return r
	}
	strong, light := 600, 400
	switch st.Weight {
	case "normal":
		strong = 400
	case "bold":
		strong, light = 700, 600
	}
	sface := face
	if st.Font == "segoe" && st.Weight == "semibold" {
		sface = "Segoe UI Semibold" // el aspecto original
	}
	w.fonts = [3]uintptr{mk(30*st.FpsSize, strong, sface), mk(14, strong, sface), mk(12, light, face)}
	w.fontKey = key
}

func (w *win) extent(hdc, font uintptr, s string) int {
	pSelectObject.Call(hdc, font)
	u, _ := syscall.UTF16FromString(s)
	var sz size
	pGetTextExtent.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&sz)))
	return int(sz.CX)
}

func (w *win) text(hdc, font uintptr, x, y int, s string, col uint32) int {
	pSelectObject.Call(hdc, font)
	u, _ := syscall.UTF16FromString(s)
	if w.outline { // contorno negro alrededor de cada letra: se lee sobre cualquier fondo del juego
		pSetTextColor.Call(hdc, 0)
		for _, d := range [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
			pTextOut.Call(hdc, uintptr(x+d[0]), uintptr(y+d[1]), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1))
		}
	}
	pSetTextColor.Call(hdc, uintptr(col))
	pTextOut.Call(hdc, uintptr(x), uintptr(y), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1))
	return w.extent(hdc, font, s)
}

// bg es el color del fondo: el del estilo o el «color invisible» si el fondo es transparente.
func bg(sn Snapshot, p Palette) uint32 {
	if sn.Cfg.Style.Transparent {
		return keyColor
	}
	return p.BG
}

func (w *win) fill(hdc uintptr, l, t, r, b int, col uint32) {
	br, _, _ := pCreateSolidBr.Call(uintptr(col))
	rc := rect{int32(l), int32(t), int32(r), int32(b)}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), br)
	pDeleteObject.Call(br)
}

// layout mide (draw=false) o dibuja (draw=true) el overlay y devuelve su tamaño.
func (w *win) layout(hdc uintptr, sn Snapshot, draw bool) (int, int) {
	s := sn.Cfg.Scale
	m := BuildModel(sn)
	px := func(v float64) int { return int(v*s + 0.5) }
	pad := px(12)
	big, norm, small := w.fonts[0], w.fonts[1], w.fonts[2]
	if sn.Cfg.Layout == "line" {
		wd := 0
		for _, p := range m.LineCols {
			wd += w.extent(hdc, norm, p.Text)
		}
		W, H := wd+2*pad, px(30)
		if draw {
			w.fill(hdc, 0, 0, W, H, bg(sn, m.Pal))
			x := pad
			for _, p := range m.LineCols {
				x += w.text(hdc, norm, x, px(5), p.Text, p.Color)
			}
		}
		return W, H
	}
	W := px(180)
	y := pad
	if sn.Cfg.ShowFPS {
		f := sn.Cfg.Style.FpsSize // el número de FPS puede ser más grande o más chico que el resto
		if draw {
			fw := w.text(hdc, big, pad, y-px(4*f), m.FPS, m.FPSColor)
			w.text(hdc, small, pad+fw+px(5), y+px(30*f)-px(16), "FPS", m.Pal.Label)
			if m.Low != "" {
				lw := w.extent(hdc, small, m.Low)
				w.text(hdc, small, W-pad-lw, y+px(30*f)-px(16), m.Low, m.Pal.Label)
			}
		}
		y += px(34 * f)
	}
	if len(m.Spark) > 1 {
		if draw {
			med, top := Median(m.Spark), SparkScale(m.Spark)
			gh := px(20)
			iw := W - 2*pad
			n := len(m.Spark)
			for i, ft := range m.Spark {
				x0 := pad + i*iw/n
				x1 := pad + (i+1)*iw/n
				if x1 <= x0 {
					x1 = x0 + 1
				}
				v := ft
				if v > top {
					v = top
				}
				bh := int(float32(gh) * v / top)
				if bh < 2 {
					bh = 2
				}
				w.fill(hdc, x0, y+gh-bh, x1, y+gh, SparkColor(m.Pal, ft, med))
			}
		}
		y += px(20) + px(8)
	}
	for _, r := range m.Rows {
		if draw {
			w.text(hdc, small, pad, y+px(2), r.Label, m.Pal.Label)
			vx := pad + px(40)
			w.text(hdc, norm, vx, y-px(1), r.Parts[0].Text, r.Parts[0].Color)
			if len(r.Parts) > 1 {
				tw := w.extent(hdc, norm, r.Parts[1].Text)
				w.text(hdc, norm, W-pad-tw, y-px(1), r.Parts[1].Text, r.Parts[1].Color)
			}
		}
		y += px(21)
	}
	H := y + pad - px(4)
	if draw {
		_ = H
	}
	return W, H
}

const (
	hkIDShow   = 1
	hkIDRecord = 2
	hkIDCorner = 3
)

// syncHotkeys registra (o vuelve a registrar) los atajos globales según la configuración.
func (w *win) syncHotkeys() {
	cfg := w.e.Snapshot().Cfg
	want := [4]string{"", cfg.HkShow, cfg.HkRecord, cfg.HkCorner}
	if !cfg.Hotkey {
		want = [4]string{}
	}
	changed := false
	for id := 1; id <= 3; id++ {
		if want[id] != w.hkTried[id] {
			changed = true
		}
	}
	if !changed {
		return
	}
	var errs []string
	for id := 1; id <= 3; id++ {
		pUnregHotKey.Call(w.hwnd, uintptr(id))
		w.hkOn[id], w.hkTried[id] = "", want[id]
		if want[id] == "" {
			continue
		}
		h, err := ParseHotkey(want[id])
		if err != nil {
			errs = append(errs, want[id]+": "+err.Error())
			continue
		}
		// MOD_NOREPEAT evita que mantener apretado repita la acción.
		if r, _, _ := pRegisterHotKey.Call(w.hwnd, uintptr(id), uintptr(h.Mod|0x4000), uintptr(h.VK)); r == 0 {
			errs = append(errs, want[id]+": otro programa ya usa este atajo")
			continue
		}
		w.hkOn[id] = want[id]
	}
	w.e.SetHotkeyErrors(errs)
}

func (w *win) refresh() {
	w.syncHotkeys()
	sn := w.e.Snapshot()
	if ms := sn.Cfg.UpdateMs; ms >= 50 && ms != w.timerMs { // el usuario cambió la frecuencia de actualización
		w.timerMs = ms
		pSetTimer.Call(w.hwnd, 1, uintptr(ms), 0)
	}
	w.makeFonts(sn.Cfg)
	hdc, _, _ := pCreateCompDC.Call(0)
	W, H := w.layout(hdc, sn, false)
	pDeleteDC.Call(hdc)

	sw, _, _ := pGetSysMetrics.Call(0)
	sh, _, _ := pGetSysMetrics.Call(1)
	mg := int(14 * sn.Cfg.Scale)
	x, y := mg, mg
	switch sn.Cfg.Corner {
	case "tr":
		x = int(sw) - W - mg
	case "bl":
		y = int(sh) - H - mg
	case "br":
		x, y = int(sw)-W-mg, int(sh)-H-mg
	}
	st := sn.Cfg.Style
	if k := fmt.Sprint(W, H, st.Radius, st.Transparent, sn.Cfg.Scale); k != w.rgnKey {
		w.rgnKey = k
		pSetWindowRgn.Call(w.hwnd, w.region(W, H, sn.Cfg), 1) // Windows se queda con la región
	}
	a := byte(sn.Cfg.Opacity*255 + 0.5)
	if k := fmt.Sprint(a, st.Transparent); k != w.layKey {
		w.layKey = k
		if st.Transparent { // el fondo («color invisible») desaparece y queda solo el texto
			pSetLayeredAttr.Call(w.hwnd, keyColor, uintptr(a), lwaAlpha|lwaColorKey)
		} else {
			pSetLayeredAttr.Call(w.hwnd, 0, uintptr(a), lwaAlpha)
		}
		w.alpha = a
	}
	w.w, w.h, w.x, w.y = W, H, x, y
	vis := sn.Cfg.Show && sn.Visible && (!sn.Cfg.OnlyGame || sn.GameRunning || sn.Recording)
	if vis {
		// HWND_TOPMOST = -1: se vuelve a pedir para quedar por encima del juego.
		pSetWindowPos.Call(w.hwnd, ^uintptr(0), uintptr(x), uintptr(y), uintptr(W), uintptr(H), swpNoActivate|swpShow)
		pInvalidate.Call(w.hwnd, 0, 0)
	} else if w.visible {
		pShowWindow.Call(w.hwnd, 0)
	}
	w.visible = vis
}

// region arma la forma de la tarjeta (esquinas redondeadas según el estilo). Con fondo transparente es un
// rectángulo: no hay tarjeta que recortar y el contorno del texto no se corta en las esquinas.
func (w *win) region(W, H int, c Config) uintptr {
	d := 0
	if !c.Style.Transparent {
		d = int(float64(2*c.Style.Radius)*c.Scale + 0.5)
	}
	rgn, _, _ := pCreateRoundRgn.Call(0, 0, uintptr(W+1), uintptr(H+1), uintptr(d), uintptr(d))
	return rgn
}

func (w *win) paint() {
	var ps [128]byte
	hdc, _, _ := pBeginPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps[0])))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps[0])))
	sn := w.e.Snapshot()
	w.makeFonts(sn.Cfg)
	st := sn.Cfg.Style
	pal := st.Palette()
	w.outline = st.Transparent
	mem, _, _ := pCreateCompDC.Call(hdc)
	bmp, _, _ := pCreateCompBmp.Call(hdc, uintptr(w.w), uintptr(w.h))
	old, _, _ := pSelectObject.Call(mem, bmp)
	pSetBkMode.Call(mem, 1)
	w.fill(mem, 0, 0, w.w, w.h, bg(sn, pal))
	w.layout(mem, sn, true)
	if st.Border > 0 && !st.Transparent { // borde siguiendo la forma de la tarjeta
		rgn := w.region(w.w, w.h, sn.Cfg)
		br, _, _ := pCreateSolidBr.Call(uintptr(pal.Border))
		b := uintptr(max(1, int(float64(st.Border)*sn.Cfg.Scale+0.5)))
		pFrameRgn.Call(mem, rgn, br, b, b)
		pDeleteObject.Call(br)
		pDeleteObject.Call(rgn)
	}
	pBitBlt.Call(hdc, 0, 0, uintptr(w.w), uintptr(w.h), mem, 0, 0, srcCopy)
	pSelectObject.Call(mem, old)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mem)
}

func wndProc(hwnd, m, wp, lp uintptr) uintptr {
	w := cur
	switch m {
	case wmPaint:
		if w != nil {
			w.paint()
			return 0
		}
	case wmEraseBkgnd:
		return 1
	case wmNCHitTest:
		return ^uintptr(0) // HTTRANSPARENT: los clics pasan al juego
	case wmTimer:
		if w != nil {
			w.refresh()
		}
		return 0
	case wmHotkey:
		if w != nil {
			switch wp {
			case hkIDShow:
				sn := w.e.Snapshot()
				w.e.SetVisible(!sn.Visible)
			case hkIDRecord:
				w.e.ToggleRecording()
			case hkIDCorner:
				w.e.CycleCorner()
			}
			w.refresh()
		}
		return 0
	case wmClose:
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		for id := uintptr(1); id <= 3; id++ {
			pUnregHotKey.Call(hwnd, id)
		}
		pPostQuit.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, m, wp, lp)
	return r
}

// RunWindow crea la ventana del overlay y atiende mensajes hasta cerrarla.
// setQuit recibe la función que la cierra desde cualquier hilo.
func RunWindow(e *Engine, setQuit func(func())) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("falló la ventana del overlay")
		}
	}()
	runtime.LockOSThread()
	pSetDPIAware.Call()
	inst, _, _ := pGetModule.Call(0)
	cb := syscall.NewCallback(wndProc)
	cls := u16("OverdriveTunerOverlay")
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: cb, Instance: inst, ClassName: cls}
	if a, _, _ := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); a == 0 {
		return errors.New("no se pudo registrar la ventana")
	}
	w := &win{e: e}
	cur = w
	ex := uintptr(wsExTopmost | wsExTransparent | wsExToolWindow | wsExLayered | wsExNoActivate)
	h, _, _ := pCreateWindowEx.Call(ex, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("OverdriveTuner Overlay"))),
		wsPopup, 20, 20, 190, 120, 0, 0, inst, 0)
	if h == 0 {
		return errors.New("no se pudo crear la ventana del overlay")
	}
	w.hwnd = h
	setQuit(func() { pPostMessage.Call(h, wmClose, 0, 0) })
	w.syncHotkeys()
	w.timerMs = 100
	pSetTimer.Call(h, 1, uintptr(w.timerMs), 0)
	w.refresh()
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMsg.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMsg.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}
