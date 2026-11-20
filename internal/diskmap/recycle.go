package diskmap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// ErrProtected: la ruta es parte de Windows o de los programas instalados y no se borra desde acá.
var ErrProtected = errors.New("es una carpeta del sistema o de programas instalados: no se borra desde acá")

// Protected dice si una ruta no se puede mandar a la Papelera desde el analizador: la raíz de una unidad,
// Windows, Archivos de programa, ProgramData, la carpeta del usuario y sus carpetas principales, y las
// carpetas internas de Windows en la raíz.
func Protected(p string) bool {
	p = filepath.Clean(p)
	if !filepath.IsAbs(p) || len(p) <= 3 { // «C:\» o relativa
		return true
	}
	under := func(base string) bool {
		if base == "" {
			return false
		}
		base = filepath.Clean(base)
		return strings.EqualFold(p, base) || strings.HasPrefix(strings.ToLower(p), strings.ToLower(base)+`\`)
	}
	for _, v := range []string{"WINDIR", "ProgramFiles", "ProgramFiles(x86)", "ProgramW6432", "ProgramData"} {
		if under(os.Getenv(v)) {
			return true
		}
	}
	root := filepath.VolumeName(p) + `\`
	for _, d := range []string{"$Recycle.Bin", "System Volume Information", "Recovery", "Boot", "$WinREAgent", "pagefile.sys", "hiberfil.sys", "swapfile.sys", "DumpStack.log.tmp"} {
		if under(root + d) {
			return true
		}
	}
	home := os.Getenv("USERPROFILE")
	if home != "" {
		for _, d := range []string{"", "AppData", `AppData\Local`, `AppData\Roaming`, `AppData\LocalLow`, "Desktop", "Documents", "Downloads", "Pictures", "Music", "Videos", "OneDrive"} {
			if strings.EqualFold(p, filepath.Join(home, d)) {
				return true
			}
		}
		if strings.EqualFold(filepath.Dir(p), filepath.Dir(filepath.Clean(home))) { // C:\Users\<otro usuario>
			return true
		}
	}
	return false
}

// Recycle manda un archivo o una carpeta a la Papelera (se puede recuperar desde ahí).
func Recycle(sys system.System, p string) error {
	p = filepath.Clean(p)
	if Protected(p) {
		return ErrProtected
	}
	fi, err := os.Lstat(p)
	if err != nil {
		return errors.New("ya no existe")
	}
	kind := "DeleteFile"
	if fi.IsDir() {
		kind = "DeleteDirectory"
	}
	script := `$ErrorActionPreference='Stop'
Add-Type -AssemblyName Microsoft.VisualBasic
[Microsoft.VisualBasic.FileIO.FileSystem]::` + kind + `('` + strings.ReplaceAll(p, "'", "''") + `', 'OnlyErrorDialogs', 'SendToRecycleBin')`
	_, err = sys.PowerShell(script, 5*time.Minute)
	return err
}

// ShowInExplorer abre el Explorador con el archivo o la carpeta seleccionados.
func ShowInExplorer(sys system.System, p string) error {
	p = filepath.Clean(p)
	if _, err := os.Stat(p); err != nil {
		return errors.New("ya no existe")
	}
	_, _, _ = sys.Exec(15*time.Second, "explorer.exe", "/select,"+p) // el Explorador devuelve 1 aunque abra bien
	return nil
}

// Volume es una unidad con su tamaño y espacio libre.
type Volume struct {
	Root  string `json:"root"`
	Total int64  `json:"total"`
	Free  int64  `json:"free"`
}
