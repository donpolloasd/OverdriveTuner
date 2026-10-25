//go:build windows

package rtss

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Running dice si RTSS está abierto (existe su memoria compartida).
func Running() bool {
	n, _ := syscall.UTF16PtrFromString("RTSSSharedMemoryV2")
	h, _, _ := pOpenMap.Call(4, 0, uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return false
	}
	pCloseH.Call(h)
	return true
}

// ExePath devuelve la ruta de RTSS.exe si está instalado ("" si no).
func ExePath() string {
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		if pf := os.Getenv(env); pf != "" {
			p := filepath.Join(pf, "RivaTuner Statistics Server", "RTSS.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// EnsureRunning abre RTSS en segundo plano (ventana oculta) si está instalado y cerrado.
// Devuelve true si lo abrió.
func EnsureRunning() bool {
	if Running() {
		return false
	}
	p := ExePath()
	if p == "" {
		return false
	}
	c := exec.Command(p)
	c.Dir = filepath.Dir(p)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return c.Start() == nil
}

// CoreTempRunning dice si Core Temp está abierto (existe su memoria compartida).
func CoreTempRunning() bool {
	n, _ := syscall.UTF16PtrFromString("CoreTempMappingObject")
	h, _, _ := pOpenMap.Call(4, 0, uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return false
	}
	pCloseH.Call(h)
	return true
}

// CoreTempPath devuelve la ruta de Core Temp.exe si está instalado ("" si no).
func CoreTempPath() string {
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if pf := os.Getenv(env); pf != "" {
			p := filepath.Join(pf, "Core Temp", "Core Temp.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// EnsureCoreTemp abre Core Temp (ventana oculta) si está instalado y cerrado. Devuelve true si lo abrió.
func EnsureCoreTemp() bool {
	if CoreTempRunning() {
		return false
	}
	p := CoreTempPath()
	if p == "" {
		return false
	}
	c := exec.Command(p)
	c.Dir = filepath.Dir(p)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return c.Start() == nil
}
