package diskmap

import (
	"io/fs"
	"syscall"
)

// cloudOnly dice si el archivo está solo en la nube (OneDrive y similares): no ocupa lugar en el disco y
// leerlo haría que Windows lo descargue, así que no se cuenta ni se compara.
func cloudOnly(fi fs.FileInfo) bool {
	d, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false
	}
	const offline, recallOpen, recallAccess = 0x1000, 0x40000, 0x400000
	return d.FileAttributes&(offline|recallOpen|recallAccess) != 0
}
