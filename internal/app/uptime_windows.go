package app

import (
	"syscall"
	"time"
)

var procTick64 = syscall.NewLazyDLL("kernel32.dll").NewProc("GetTickCount64")

// uptime es cuánto hace que se encendió la PC. Con el «Inicio rápido» de Windows, apagar no reinicia:
// solo «Reiniciar» pone este contador en cero.
func uptime() time.Duration {
	ms, _, _ := procTick64.Call()
	return time.Duration(ms) * time.Millisecond
}
