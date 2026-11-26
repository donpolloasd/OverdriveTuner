package perf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry es una prueba guardada en el ranking local (solo en esta PC).
type Entry struct {
	ID     int64     `json:"id"`
	At     time.Time `json:"at"`
	Kind   string    `json:"kind"`
	Value  float64   `json:"value"`            // el número que ordena el ranking
	Extra  []float64 `json:"extra,omitempty"`  // ram: escritura y copia; disco: IOPS; stab: errores, caída %, temp. máx., segundos
	Target string    `json:"target,omitempty"` // disco: unidad
	Rank   int       `json:"rank"`             // puesto en el ranking (se calcula al leer)
}

const maxPerKind = 30

// Store guarda las pruebas en pruebas.json.
type Store struct {
	mu   sync.Mutex
	path string
	list []Entry
}

// OpenStore lee el ranking (si no existe o está dañado, empieza vacío).
func OpenStore(dir string) *Store {
	s := &Store{path: filepath.Join(dir, "pruebas.json")}
	if b, err := os.ReadFile(s.path); err == nil && json.Unmarshal(b, &s.list) != nil {
		_ = os.Rename(s.path, s.path+".corrupto")
		s.list = nil
	}
	return s
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.list, "", " ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// better dice si a va antes que b en el ranking.
func better(a, b Entry) bool {
	if a.Value != b.Value {
		return (a.Value < b.Value) == LowerBetter(a.Kind)
	}
	return a.At.Before(b.At)
}

// group separa los rankings: los discos se comparan por unidad.
func group(e Entry) string { return e.Kind + "|" + e.Target }

// Add guarda una prueba y devuelve la entrada con su puesto. De cada ranking quedan las mejores, pero la
// recién hecha nunca se descarta.
func (s *Store) Add(e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e.ID = time.Now().UnixNano() / 1000
	for _, x := range s.list {
		if x.ID >= e.ID {
			e.ID = x.ID + 1
		}
	}
	if e.At.IsZero() {
		e.At = time.Now()
	}
	s.list = append(s.list, e)
	same := s.ranked(group(e))
	if len(same) > maxPerKind {
		drop := same[len(same)-1].ID
		if drop == e.ID {
			drop = same[len(same)-2].ID
		}
		s.remove(drop)
	}
	for i, x := range s.ranked(group(e)) {
		if x.ID == e.ID {
			e.Rank = i + 1
		}
	}
	return e, s.save()
}

func (s *Store) ranked(g string) []Entry {
	var out []Entry
	for _, x := range s.list {
		if group(x) == g {
			out = append(out, x)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return better(out[i], out[j]) })
	return out
}

func (s *Store) remove(id int64) {
	out := s.list[:0:0]
	for _, x := range s.list {
		if x.ID != id {
			out = append(out, x)
		}
	}
	s.list = out
}

// List devuelve todas las pruebas, ordenadas por ranking.
func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, seen := []Entry{}, map[string]bool{}
	for _, x := range s.list {
		g := group(x)
		if seen[g] {
			continue
		}
		seen[g] = true
		for i, e := range s.ranked(g) {
			e.Rank = i + 1
			out = append(out, e)
		}
	}
	return out
}

// Delete borra una prueba; id 0 borra todas.
func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == 0 {
		s.list = nil
	} else {
		s.remove(id)
	}
	return s.save()
}
