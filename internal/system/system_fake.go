package system

import (
	"strings"
	"sync"
	"time"
)

// Fake es una implementación en memoria: sirve para pruebas y para ejecutar
// la interfaz en Linux/macOS sin tocar nada real.
type Fake struct {
	mu       sync.Mutex
	vals     map[string]RegValue
	keys     map[string]bool
	ExecLog  []string
	PSLog    []string
	ExecHook func(name string, args []string) (string, int, error)
	PSHook   func(script string) (string, error)
	Admin    bool
	// FailSet hace fallar RegSet para claves que contengan alguna de estas cadenas.
	FailSet []string
}

// NewFake crea un sistema simulado vacío.
func NewFake() *Fake {
	return &Fake{vals: map[string]RegValue{}, keys: map[string]bool{}, Admin: true}
}

func (f *Fake) Name() string  { return "simulado" }
func (f *Fake) IsAdmin() bool { return f.Admin }

func fkey(key, name string) string { return strings.ToUpper(key) + "|" + strings.ToLower(name) }

// Seed precarga un valor.
func (f *Fake) Seed(key, name, typ, data string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vals[fkey(key, name)] = RegValue{Key: key, Name: name, Exists: true, Type: typ, Data: NormalizeData(typ, data)}
	f.keys[strings.ToUpper(key)] = true
}

func (f *Fake) RegGetMany(refs []RegRef) ([]RegValue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]RegValue, 0, len(refs))
	for _, r := range refs {
		if v, ok := f.vals[fkey(r.Key, r.Name)]; ok {
			out = append(out, v)
		} else {
			out = append(out, RegValue{Key: r.Key, Name: r.Name})
		}
	}
	return out, nil
}

func (f *Fake) RegSet(key, name, typ, data string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.FailSet {
		if strings.Contains(strings.ToUpper(key), strings.ToUpper(s)) {
			return errFake("acceso denegado (simulado): " + key)
		}
	}
	f.vals[fkey(key, name)] = RegValue{Key: key, Name: name, Exists: true, Type: strings.ToUpper(typ), Data: NormalizeData(typ, data)}
	f.keys[strings.ToUpper(key)] = true
	return nil
}

type errFake string

func (e errFake) Error() string { return string(e) }

func (f *Fake) RegDeleteValue(key, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.vals[fkey(key, name)]; !ok {
		return errFake("valor inexistente")
	}
	delete(f.vals, fkey(key, name))
	return nil
}

func (f *Fake) RegDeleteKey(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	up := strings.ToUpper(key)
	if !f.keys[up] {
		return errFake("clave inexistente")
	}
	for k := range f.vals {
		if strings.HasPrefix(k, up+"|") || strings.HasPrefix(k, up+"\\") {
			delete(f.vals, k)
		}
	}
	for k := range f.keys {
		if k == up || strings.HasPrefix(k, up+"\\") {
			delete(f.keys, k)
		}
	}
	return nil
}

func (f *Fake) RegKeyExists(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.keys[strings.ToUpper(key)]
}

func (f *Fake) RegEnumValues(key string) ([]RegValue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []RegValue
	for _, v := range f.vals {
		if strings.EqualFold(v.Key, key) {
			out = append(out, v)
		}
	}
	return out, nil
}

func (f *Fake) Exec(timeout time.Duration, name string, args ...string) (string, int, error) {
	f.mu.Lock()
	f.ExecLog = append(f.ExecLog, name+" "+strings.Join(args, " "))
	hook := f.ExecHook
	f.mu.Unlock()
	if hook != nil {
		if out, code, err := hook(name, args); out != "" || code != 0 || err != nil {
			return out, code, err
		}
	}
	low := strings.ToLower(name + " " + strings.Join(args, " "))
	switch {
	case strings.Contains(low, "powercfg") && strings.Contains(low, "duplicatescheme"):
		return "GUID del esquema de energía: 11111111-2222-3333-4444-555555555555  (Copia)\n", 0, nil
	case strings.Contains(low, "powercfg") && strings.Contains(low, "getactivescheme"):
		return "GUID del esquema de energía: 381b4222-f694-41f0-9685-ff5bb260df2e  (Equilibrado)\n", 0, nil
	}
	return "", 0, nil
}

func (f *Fake) PowerShell(script string, timeout time.Duration) (string, error) {
	f.mu.Lock()
	f.PSLog = append(f.PSLog, script)
	hook := f.PSHook
	f.mu.Unlock()
	if hook != nil {
		return hook(script)
	}
	return "", nil
}

// Snapshot devuelve una copia de todos los valores (para comparar estados en pruebas).
func (f *Fake) Snapshot() map[string]RegValue {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]RegValue, len(f.vals))
	for k, v := range f.vals {
		out[k] = v
	}
	return out
}

// KeyCount devuelve cuántas claves existen.
func (f *Fake) KeyCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.keys)
}
