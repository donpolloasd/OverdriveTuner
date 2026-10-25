// Package bench interpreta resultados de benchmark de FPS (formato de MSI
// Afterburner / RTSS) y los compara contra una medición de referencia.
package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Run es una corrida de benchmark.
type Run struct {
	Label   string  `json:"label"`
	App     string  `json:"app"`
	Date    string  `json:"date"`
	Frames  int     `json:"frames"`
	Seconds float64 `json:"seconds"`
	Avg     float64 `json:"avg"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Low1    float64 `json:"low1"`
	Low01   float64 `json:"low01"`
	Src     string  `json:"src,omitempty"` // «fpsforge» si la midió el overlay; vacío si se pegó de otro programa
}

var (
	reHead = regexp.MustCompile(`(?i)^\s*(?:(\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4}),?\s+\d{1,2}:\d{2}(?::\d{2})?\s+)?(\S+?)\s+benchmark completed,\s*([\d.,]+)\s+frames rendered in\s*([\d.,]+)\s*s`)
	reAvg  = regexp.MustCompile(`(?i)average framerate\s*:\s*([\d.,]+)`)
	reMin  = regexp.MustCompile(`(?i)minimum framerate\s*:\s*([\d.,]+)`)
	reMax  = regexp.MustCompile(`(?i)maximum framerate\s*:\s*([\d.,]+)`)
	re1    = regexp.MustCompile(`(?i)^\s*1%\s*low framerate\s*:\s*([\d.,]+)`)
	re01   = regexp.MustCompile(`(?i)0[.,]1%\s*low framerate\s*:\s*([\d.,]+)`)
)

func num(s string) float64 {
	s = strings.TrimSpace(s)
	// «1.234,5» (miles con punto, decimal con coma) y «1234,5» → 1234.5
	if strings.Contains(s, ",") {
		if strings.Contains(s, ".") && strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
		}
		s = strings.ReplaceAll(s, ",", ".")
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// Parse extrae todas las corridas de un texto pegado. La etiqueta de cada
// corrida es la última línea de texto libre que la precede («Try 1:», etc.).
func Parse(text string) []Run {
	var runs []Run
	var cur *Run
	label := ""
	flush := func() {
		if cur != nil && cur.Avg > 0 {
			runs = append(runs, *cur)
		}
		cur = nil
	}
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if m := reHead.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Run{Label: label, Date: m[1], App: m[2], Frames: int(num(m[3])), Seconds: num(m[4])}
			continue
		}
		if cur != nil {
			switch {
			case reAvg.MatchString(line):
				cur.Avg = num(reAvg.FindStringSubmatch(line)[1])
				continue
			case reMin.MatchString(line):
				cur.Min = num(reMin.FindStringSubmatch(line)[1])
				continue
			case reMax.MatchString(line):
				cur.Max = num(reMax.FindStringSubmatch(line)[1])
				continue
			case re01.MatchString(line):
				cur.Low01 = num(re01.FindStringSubmatch(line)[1])
				continue
			case re1.MatchString(line):
				cur.Low1 = num(re1.FindStringSubmatch(line)[1])
				continue
			}
		}
		// Texto libre: pasa a ser la etiqueta de la próxima corrida.
		flush()
		label = strings.TrimSpace(strings.TrimRight(line, ":"))
	}
	flush()
	return runs
}

// Pct devuelve la variación porcentual de cur respecto de base.
func Pct(base, cur float64) float64 {
	if base == 0 {
		return 0
	}
	return (cur - base) / base * 100
}

// Recovered indica si cur está dentro de la tolerancia (en %) de la
// referencia en promedio y en 1% low (este último con el doble de margen,
// porque es más ruidoso).
func Recovered(base, cur Run, tolPct float64) bool {
	return Pct(base.Avg, cur.Avg) >= -tolPct && Pct(base.Low1, cur.Low1) >= -2*tolPct
}

// Entry es una corrida guardada.
type Entry struct {
	ID       int       `json:"id"`
	At       time.Time `json:"at"`
	Baseline bool      `json:"baseline"`
	Note     string    `json:"note"`
	Run      Run       `json:"run"`
}

// Store guarda las corridas en disco.
type Store struct {
	mu      sync.Mutex
	path    string
	Entries []Entry
	next    int
}

// Open carga (o crea) el almacén en dir.
func Open(dir string) (*Store, error) {
	s := &Store{path: filepath.Join(dir, "benchmarks.json"), next: 1}
	b, err := os.ReadFile(s.path)
	if err == nil {
		if json.Unmarshal(b, &s.Entries) != nil {
			_ = os.Rename(s.path, s.path+".corrupto")
			s.Entries = nil
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range s.Entries {
		if e.ID >= s.next {
			s.next = e.ID + 1
		}
	}
	return s, nil
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.Entries, "", " ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// List devuelve una copia de las corridas.
func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry{}, s.Entries...)
}

// Add interpreta el texto y guarda cada corrida. baseline marca la primera
// como referencia («antes de optimizar»), quitando la marca anterior.
func (s *Store) Add(text, note string, baseline bool) ([]Entry, error) {
	runs := Parse(text)
	if len(runs) == 0 {
		return nil, fmt.Errorf("no se encontró ningún resultado. Pega el texto completo del benchmark (con «Average framerate», «1%% low framerate», etc.)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var added []Entry
	for i, r := range runs {
		if note != "" {
			r.Label = note
		}
		e := Entry{ID: s.next, At: time.Now(), Note: note, Run: r}
		s.next++
		if baseline && i == 0 {
			for j := range s.Entries {
				s.Entries[j].Baseline = false
			}
			e.Baseline = true
		}
		s.Entries = append(s.Entries, e)
		added = append(added, e)
	}
	return added, s.save()
}

// Delete quita una corrida.
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.Entries[:0:0]
	for _, e := range s.Entries {
		if e.ID != id {
			out = append(out, e)
		}
	}
	s.Entries = out
	return s.save()
}

// Clear borra todas las corridas.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Entries = nil
	return s.save()
}

// Baseline devuelve la referencia: la marcada o, si no hay, la primera.
func (s *Store) Baseline() (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.Entries {
		if e.Baseline {
			return e, true
		}
	}
	if len(s.Entries) > 0 {
		return s.Entries[0], true
	}
	return Entry{}, false
}

// AddRun guarda una corrida ya calculada (por ejemplo, de una sesión del overlay).
func (s *Store) AddRun(r Run, label string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if label != "" {
		r.Label = label
	}
	e := Entry{ID: s.next, At: time.Now(), Run: r}
	s.next++
	s.Entries = append(s.Entries, e)
	return e, s.save()
}
