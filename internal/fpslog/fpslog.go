// Package fpslog anota en un archivo de texto qué hace el overlay (juego detectado, fuente de FPS,
// avisos de RTSS). Sirve para diagnosticar cuando los FPS no aparecen.
package fpslog

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// DiagPath, si no está vacío, es el archivo donde se anotan los eventos.
var DiagPath string

var (
	diagMu    sync.Mutex
	diagReset bool
)

// Diag anota una línea en el archivo de diagnóstico.
func Diag(format string, a ...any) {
	if DiagPath == "" {
		return
	}
	diagMu.Lock()
	defer diagMu.Unlock()
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if !diagReset { // la primera vez de cada ejecución del overlay empieza un archivo nuevo
		flags |= os.O_TRUNC
		diagReset = true
	}
	f, err := os.OpenFile(DiagPath, flags, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, a...))
}
