//go:build windows

package overlay

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	pEnumMonitors  = dUser32.NewProc("EnumDisplayMonitors")
	pMonitorInfo   = dUser32.NewProc("GetMonitorInfoW")
	pGetUILanguage = dKernel32.NewProc("GetUserDefaultUILanguage")
	monitorsCB     = syscall.NewCallback(monitorEnumProc)
	monitorsFound  []Monitor
)

func monitorEnumProc(h, _, _, _ uintptr) uintptr {
	var mi struct {
		Size    uint32
		Monitor dRect
		Work    dRect
		Flags   uint32
	}
	mi.Size = uint32(unsafe.Sizeof(mi))
	if r, _, _ := pMonitorInfo.Call(h, uintptr(unsafe.Pointer(&mi))); r != 0 {
		monitorsFound = append(monitorsFound, Monitor{Width: int(mi.Monitor.R - mi.Monitor.L), Height: int(mi.Monitor.B - mi.Monitor.T),
			Primary: mi.Flags&1 != 0, L: int(mi.Monitor.L), T: int(mi.Monitor.T)})
	}
	return 1
}

// Monitors lista las pantallas: primero la principal y después las demás, en el orden de Windows.
func Monitors() (out []Monitor) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	monitorsFound = nil
	pEnumMonitors.Call(0, 0, monitorsCB, 0)
	var rest []Monitor
	for _, m := range monitorsFound {
		if m.Primary {
			out = append(out, m)
		} else {
			rest = append(rest, m)
		}
	}
	out = append(out, rest...)
	for i := range out {
		out[i].Index = i
		out[i].Name = fmt.Sprintf("Monitor %d · %d×%d", i+1, out[i].Width, out[i].Height)
		if out[i].Primary {
			out[i].Name += " (principal)"
		}
	}
	return out
}

// systemLang es el idioma de Windows: español si el idioma principal es español; si no, inglés.
func systemLang() string {
	id, _, _ := pGetUILanguage.Call()
	if id&0x3FF == 0x0A { // LANG_SPANISH
		return "es"
	}
	return "en"
}
