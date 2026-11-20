package extras

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32       = syscall.NewLazyDLL("shell32.dll")
	pQueryRecycle = shell32.NewProc("SHQueryRecycleBinW")
	pEmptyRecycle = shell32.NewProc("SHEmptyRecycleBinW")
)

// RecycleBin devuelve lo que ocupa la Papelera (todas las unidades) y cuántos elementos tiene.
func RecycleBin() (bytes, items int64, err error) {
	var q struct {
		Size  uint32
		Bytes int64
		Items int64
	}
	q.Size = uint32(unsafe.Sizeof(q))
	if r, _, _ := pQueryRecycle.Call(0, uintptr(unsafe.Pointer(&q))); r != 0 {
		return 0, 0, fmt.Errorf("Windows no informó el tamaño de la Papelera (0x%x)", r)
	}
	return q.Bytes, q.Items, nil
}

// EmptyRecycleBin vacía la Papelera de todas las unidades, sin la ventana de Windows (la confirmación la
// pide la interfaz antes).
func EmptyRecycleBin() error {
	const noConfirm, noProgress, noSound = 0x1, 0x2, 0x4
	r, _, _ := pEmptyRecycle.Call(0, 0, noConfirm|noProgress|noSound)
	if r != 0 && r != 0x8000FFFF { // E_UNEXPECTED: ya estaba vacía
		return fmt.Errorf("no se pudo vaciar la Papelera (0x%x)", r)
	}
	return nil
}
