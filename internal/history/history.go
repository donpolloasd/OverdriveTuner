// Package history guarda una línea de tiempo de los cambios aplicados y revertidos.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry es un cambio registrado.
type Entry struct {
	At     string `json:"at"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Action string `json:"action"` // apply, revert
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

// Log es el historial persistente (últimas Max entradas).
type Log struct {
	mu   sync.Mutex
	path string
	Max  int
	list []Entry
}

// Open lee el historial de dir (si no existe, empieza vacío).
func Open(dir string) *Log {
	l := &Log{path: filepath.Join(dir, "history.json"), Max: 500}
	if b, err := os.ReadFile(l.path); err == nil {
		_ = json.Unmarshal(b, &l.list)
	}
	return l
}

// Add registra un cambio.
func (l *Log) Add(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.At == "" {
		e.At = time.Now().Format(time.RFC3339)
	}
	l.list = append(l.list, e)
	if len(l.list) > l.Max {
		l.list = append([]Entry{}, l.list[len(l.list)-l.Max:]...)
	}
	l.save()
}

// List devuelve las entradas, la más reciente primero.
func (l *Log) List() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, 0, len(l.list))
	for i := len(l.list) - 1; i >= 0; i-- {
		out = append(out, l.list[i])
	}
	return out
}

// Clear borra el historial.
func (l *Log) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.list = nil
	l.save()
}

func (l *Log) save() {
	b, err := json.MarshalIndent(l.list, "", " ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(l.path), 0o755)
	tmp := l.path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, l.path)
	}
}
