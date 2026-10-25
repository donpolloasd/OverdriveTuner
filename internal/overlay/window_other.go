//go:build !windows

package overlay

import (
	"fmt"
	"time"
)

// RunWindow en otros sistemas imprime el texto del overlay (solo para probar).
func RunWindow(e *Engine, setQuit func(func())) error {
	done := make(chan struct{})
	setQuit(func() { close(done) })
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for {
		select {
		case <-done:
			return nil
		case <-tk.C:
			fmt.Println(BuildModel(e.Snapshot()).Line)
		}
	}
}
