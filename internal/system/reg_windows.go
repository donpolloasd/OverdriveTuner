//go:build windows

package system

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	advapi32          = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx  = advapi32.NewProc("RegOpenKeyExW")
	procRegCreateKey  = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueEx = advapi32.NewProc("RegSetValueExW")
	procRegDeleteVal  = advapi32.NewProc("RegDeleteValueW")
	procRegDeleteTree = advapi32.NewProc("RegDeleteTreeW")
	procRegQueryValue = advapi32.NewProc("RegQueryValueExW")
	procRegEnumValue  = advapi32.NewProc("RegEnumValueW")
	procRegCloseKey   = advapi32.NewProc("RegCloseKey")
)

const (
	keyQueryValue = 0x0001
	keySetValue   = 0x0002
	keyWow64_64   = 0x0100

	errFileNotFound = 2
	errMoreData     = 234
	errNoMoreItems  = 259
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// splitKey separa "HKLM\Ruta\Clave" en la raíz y la subclave.
func splitKey(key string) (uintptr, string, error) {
	root, sub := key, ""
	if i := strings.IndexByte(key, '\\'); i >= 0 {
		root, sub = key[:i], key[i+1:]
	}
	var h uintptr
	switch strings.ToUpper(root) {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		h = 0x80000002
	case "HKCU", "HKEY_CURRENT_USER":
		h = 0x80000001
	case "HKCR", "HKEY_CLASSES_ROOT":
		h = 0x80000000
	case "HKU", "HKEY_USERS":
		h = 0x80000003
	default:
		return 0, "", fmt.Errorf("raíz de registro desconocida: %q", root)
	}
	return h, strings.TrimRight(sub, `\`), nil
}

func regErr(op, key, name string, st uintptr) error {
	where := key
	if name != "" {
		where += "\\" + name
	}
	if st == 5 {
		return fmt.Errorf("%s %s: acceso denegado", op, where)
	}
	return fmt.Errorf("%s %s: %w", op, where, syscall.Errno(st))
}

func openKey(key string, access uint32) (uintptr, uintptr, error) {
	root, sub, err := splitKey(key)
	if err != nil {
		return 0, 0, err
	}
	var h uintptr
	st, _, _ := procRegOpenKeyEx.Call(root, uintptr(unsafe.Pointer(u16(sub))), 0, uintptr(access|keyWow64_64), uintptr(unsafe.Pointer(&h)))
	return h, st, nil
}

func closeKey(h uintptr) { procRegCloseKey.Call(h) }

// queryValue lee un valor de una clave abierta. exists=false si no existe.
func queryValue(h uintptr, name string) (kind uint32, raw []byte, exists bool, err error) {
	np := u16(name)
	for try := 0; try < 4; try++ {
		var size uint32
		st, _, _ := procRegQueryValue.Call(h, uintptr(unsafe.Pointer(np)), 0, uintptr(unsafe.Pointer(&kind)), 0, uintptr(unsafe.Pointer(&size)))
		if st == errFileNotFound {
			return 0, nil, false, nil
		}
		if st != 0 {
			return 0, nil, false, syscall.Errno(st)
		}
		buf := make([]byte, size+2)
		got := size
		var ptr uintptr
		if size > 0 {
			ptr = uintptr(unsafe.Pointer(&buf[0]))
		}
		st, _, _ = procRegQueryValue.Call(h, uintptr(unsafe.Pointer(np)), 0, uintptr(unsafe.Pointer(&kind)), ptr, uintptr(unsafe.Pointer(&got)))
		if st == errMoreData {
			continue // el valor creció entre las dos llamadas
		}
		if st != 0 {
			return 0, nil, false, syscall.Errno(st)
		}
		return kind, buf[:got], true, nil
	}
	return 0, nil, false, fmt.Errorf("el valor cambió mientras se leía")
}

func (winSys) RegGetMany(refs []RegRef) ([]RegValue, error) {
	out := make([]RegValue, 0, len(refs))
	type opened struct {
		h  uintptr
		ok bool
	}
	cache := map[string]opened{}
	defer func() {
		for _, o := range cache {
			if o.ok {
				closeKey(o.h)
			}
		}
	}()
	for _, r := range refs {
		v := RegValue{Key: r.Key, Name: r.Name}
		k := strings.ToUpper(r.Key)
		o, seen := cache[k]
		if !seen {
			h, st, err := openKey(r.Key, keyQueryValue)
			if err != nil {
				return nil, err
			}
			o = opened{h: h, ok: st == 0}
			cache[k] = o
		}
		if o.ok {
			kind, raw, exists, err := queryValue(o.h, r.Name)
			if err == nil && exists {
				v.Exists = true
				v.Type, v.Data = DecodeValue(kind, raw)
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func (winSys) RegSet(key, name, typ, data string) error {
	kind, raw, err := EncodeValue(typ, data)
	if err != nil {
		return fmt.Errorf("%s\\%s: %w", key, name, err)
	}
	root, sub, err := splitKey(key)
	if err != nil {
		return err
	}
	var h uintptr
	var disp uint32
	st, _, _ := procRegCreateKey.Call(root, uintptr(unsafe.Pointer(u16(sub))), 0, 0, 0,
		uintptr(keySetValue|keyQueryValue|keyWow64_64), 0, uintptr(unsafe.Pointer(&h)), uintptr(unsafe.Pointer(&disp)))
	if st != 0 {
		return regErr("crear clave", key, "", st)
	}
	defer closeKey(h)
	var ptr uintptr
	if len(raw) > 0 {
		ptr = uintptr(unsafe.Pointer(&raw[0]))
	}
	st, _, _ = procRegSetValueEx.Call(h, uintptr(unsafe.Pointer(u16(name))), 0, uintptr(kind), ptr, uintptr(len(raw)))
	if st != 0 {
		return regErr("escribir", key, name, st)
	}
	return nil
}

func (winSys) RegDeleteValue(key, name string) error {
	h, st, err := openKey(key, keySetValue)
	if err != nil {
		return err
	}
	if st != 0 {
		return regErr("abrir", key, "", st)
	}
	defer closeKey(h)
	st, _, _ = procRegDeleteVal.Call(h, uintptr(unsafe.Pointer(u16(name))))
	if st != 0 {
		return regErr("borrar", key, name, st)
	}
	return nil
}

func (winSys) RegDeleteKey(key string) error {
	root, sub, err := splitKey(key)
	if err != nil {
		return err
	}
	if sub == "" {
		return fmt.Errorf("no se puede borrar una raíz del registro")
	}
	st, _, _ := procRegDeleteTree.Call(root, uintptr(unsafe.Pointer(u16(sub))))
	if st != 0 {
		return regErr("borrar clave", key, "", st)
	}
	return nil
}

func (winSys) RegKeyExists(key string) bool {
	h, st, err := openKey(key, keyQueryValue)
	if err != nil || st != 0 {
		return false
	}
	closeKey(h)
	return true
}

func (winSys) RegEnumValues(key string) ([]RegValue, error) {
	h, st, err := openKey(key, keyQueryValue)
	if err != nil {
		return nil, err
	}
	if st != 0 {
		return nil, nil // la clave no existe: lista vacía
	}
	defer closeKey(h)
	var out []RegValue
	data := make([]byte, 8192)
	name := make([]uint16, 16384)
	for i := uint32(0); ; {
		nameLen := uint32(len(name))
		dataLen := uint32(len(data))
		var kind uint32
		st, _, _ := procRegEnumValue.Call(h, uintptr(i), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameLen)), 0,
			uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataLen)))
		switch st {
		case 0:
			v := RegValue{Key: key, Name: syscall.UTF16ToString(name[:nameLen]), Exists: true}
			v.Type, v.Data = DecodeValue(kind, data[:dataLen])
			out = append(out, v)
			i++
		case errMoreData:
			if dataLen <= uint32(len(data)) || dataLen > 1<<24 {
				return out, fmt.Errorf("valor demasiado grande en %s", key)
			}
			data = make([]byte, dataLen+16)
		case errNoMoreItems:
			return out, nil
		default:
			return out, regErr("enumerar", key, "", st)
		}
	}
}
