//go:build windows

package overlay

import (
	"syscall"
	"unsafe"
)

// regSubKeys lista las subclaves de una clave de HKLM (en la vista de 64 y de 32 bits).
func regSubKeys(path string, view uint32) []string {
	var h syscall.Handle
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil || syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, p, 0, syscall.KEY_READ|syscall.KEY_ENUMERATE_SUB_KEYS|view, &h) != nil {
		return nil
	}
	defer syscall.RegCloseKey(h)
	var out []string
	for i := uint32(0); i < 2000; i++ {
		buf := make([]uint16, 256)
		n := uint32(len(buf))
		if syscall.RegEnumKeyEx(h, i, &buf[0], &n, nil, nil, nil, nil) != nil {
			break
		}
		out = append(out, syscall.UTF16ToString(buf[:n]))
	}
	return out
}

// regString lee un valor de texto de una clave de HKLM.
func regString(path, name string, view uint32) string {
	var h syscall.Handle
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil || syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, p, 0, syscall.KEY_READ|view, &h) != nil {
		return ""
	}
	defer syscall.RegCloseKey(h)
	n, _ := syscall.UTF16PtrFromString(name)
	var typ, size uint32
	if syscall.RegQueryValueEx(h, n, nil, &typ, nil, &size) != nil || size == 0 || size > 2048 || (typ != syscall.REG_SZ && typ != syscall.REG_EXPAND_SZ) {
		return ""
	}
	buf := make([]uint16, size/2+1)
	size = uint32(len(buf) * 2)
	if syscall.RegQueryValueEx(h, n, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &size) != nil {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// registryGameDirs devuelve las carpetas de juegos que EA (y Origin), GOG y Ubisoft guardan en el registro.
func registryGameDirs() []string {
	const wow64, wow32 = 0x100, 0x200
	var out []string
	add := func(d string) {
		if len(d) >= 4 {
			out = append(out, d)
		}
	}
	for _, view := range []uint32{wow64, wow32} {
		for _, k := range regSubKeys(`SOFTWARE\GOG.com\Games`, view) {
			add(regString(`SOFTWARE\GOG.com\Games\`+k, "path", view))
		}
		for _, root := range []string{`SOFTWARE\EA Games`, `SOFTWARE\Origin Games`, `SOFTWARE\Electronic Arts`} {
			for _, k := range regSubKeys(root, view) {
				d := regString(root+`\`+k, "Install Dir", view)
				if d == "" {
					d = regString(root+`\`+k, "InstallDir", view)
				}
				add(d)
			}
		}
		for _, k := range regSubKeys(`SOFTWARE\Ubisoft\Launcher\Installs`, view) {
			add(regString(`SOFTWARE\Ubisoft\Launcher\Installs\`+k, "InstallDir", view))
		}
	}
	return out
}
