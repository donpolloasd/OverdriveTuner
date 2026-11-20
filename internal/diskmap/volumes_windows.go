package diskmap

import (
	"syscall"
	"unsafe"
)

var (
	kernel32   = syscall.NewLazyDLL("kernel32.dll")
	pLogical   = kernel32.NewProc("GetLogicalDrives")
	pFreeSpace = kernel32.NewProc("GetDiskFreeSpaceExW")
	pDriveType = kernel32.NewProc("GetDriveTypeW")
)

// Volumes lista las unidades locales (discos fijos y extraíbles) con su tamaño y espacio libre.
func Volumes() []Volume {
	mask, _, _ := pLogical.Call()
	var out []Volume
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		p, _ := syscall.UTF16PtrFromString(root)
		if t, _, _ := pDriveType.Call(uintptr(unsafe.Pointer(p))); t != 2 && t != 3 { // 2 extraíble, 3 fijo
			continue
		}
		var free, total, totalFree uint64
		if r, _, _ := pFreeSpace.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&totalFree))); r == 0 || total == 0 {
			continue
		}
		out = append(out, Volume{Root: root, Total: int64(total), Free: int64(totalFree)})
	}
	return out
}
