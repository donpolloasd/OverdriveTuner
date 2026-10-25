// Package system abstrae todo lo que toca Windows (registro, procesos,
// PowerShell) para poder probar la lógica del optimizador en cualquier SO.
package system

import (
	"strconv"
	"strings"
	"time"
)

// RegRef identifica un valor del registro.
type RegRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// RegValue es el contenido leído de un valor del registro.
// Los DWORD/QWORD se normalizan a decimal; los binarios a hex sin separadores.
type RegValue struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
	Type   string `json:"type"`
	Data   string `json:"data"`
}

// System es la interfaz con el sistema operativo.
type System interface {
	Name() string
	IsAdmin() bool
	RegGetMany(refs []RegRef) ([]RegValue, error)
	RegSet(key, name, typ, data string) error
	RegDeleteValue(key, name string) error
	RegDeleteKey(key string) error
	RegKeyExists(key string) bool
	RegEnumValues(key string) ([]RegValue, error)
	Exec(timeout time.Duration, name string, args ...string) (out string, code int, err error)
	PowerShell(script string, timeout time.Duration) (out string, err error)
}

// RegGet lee un único valor.
func RegGet(s System, key, name string) (RegValue, error) {
	r, err := s.RegGetMany([]RegRef{{Key: key, Name: name}})
	if err != nil || len(r) == 0 {
		return RegValue{Key: key, Name: name}, err
	}
	return r[0], nil
}

// NormalizeData deja los datos en una forma comparable.
func NormalizeData(typ, data string) string {
	t := strings.ToUpper(typ)
	d := strings.TrimSpace(data)
	if t == "REG_DWORD" || t == "REG_QWORD" {
		if strings.HasPrefix(strings.ToLower(d), "0x") {
			if n, err := strconv.ParseUint(d[2:], 16, 64); err == nil {
				return strconv.FormatUint(n, 10)
			}
		}
		if n, err := strconv.ParseUint(d, 10, 64); err == nil {
			return strconv.FormatUint(n, 10)
		}
		return d
	}
	if t == "REG_BINARY" {
		d = strings.ReplaceAll(strings.ToLower(d), " ", "")
		return strings.ReplaceAll(d, ",", "")
	}
	return data
}

// Matches indica si el valor leído equivale a (typ, data).
func (v RegValue) Matches(typ, data string) bool {
	if !v.Exists {
		return false
	}
	if !strings.EqualFold(v.Type, typ) {
		return false
	}
	return NormalizeData(typ, v.Data) == NormalizeData(typ, data)
}
