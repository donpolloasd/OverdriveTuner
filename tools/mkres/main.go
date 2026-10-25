// mkres genera los recursos de Windows (icono + manifiesto) como .syso.
package main

import (
	"fmt"
	"os"

	"fpsforge/internal/icon"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "uso: mkres <destino.syso> <admin|invoker> <nombre> [icono.ico]")
		os.Exit(2)
	}
	if err := os.WriteFile(os.Args[1], icon.Syso(os.Args[2] == "admin", os.Args[3]), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) > 4 {
		if err := os.WriteFile(os.Args[4], icon.ICO(), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
