package apps

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed coretemp.ini
var coreTempINI []byte

// coreTempDir devuelve la carpeta de Core Temp si está instalado ("" si no).
func coreTempDir() string {
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if pf := os.Getenv(env); pf != "" {
			d := filepath.Join(pf, "Core Temp")
			if _, err := os.Stat(filepath.Join(d, "Core Temp.exe")); err == nil {
				return d
			}
		}
	}
	return ""
}

// writeCoreTempConfig crea CoreTemp.ini junto al programa, solo si todavía no existe (nunca pisa uno propio).
func writeCoreTempConfig() error {
	d := coreTempDir()
	if d == "" {
		return nil
	}
	p := filepath.Join(d, "CoreTemp.ini")
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	return os.WriteFile(p, coreTempINI, 0o644)
}
