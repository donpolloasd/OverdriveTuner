package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fpsforge/internal/telemetry"
)

// State es lo que el overlay informa a la interfaz principal (state.json).
type State struct {
	PID         int               `json:"pid"`
	Updated     time.Time         `json:"updated"`
	Visible     bool              `json:"visible"`
	Game        string            `json:"game"`
	GameRunning bool              `json:"gameRunning"`
	FPS         float64           `json:"fps"`
	Low1        float64           `json:"low1"`
	Recording   bool              `json:"recording"`
	RecLabel    string            `json:"recLabel"`
	RecFrames   int               `json:"recFrames"`
	RecSeconds  float64           `json:"recSeconds"`
	Sample      telemetry.Sample  `json:"sample"`
	Verdict     telemetry.Verdict `json:"verdict"`
	Message     string            `json:"message,omitempty"`
	LastSession string            `json:"lastSession,omitempty"`
	HotkeyErrs  []string          `json:"hotkeyErrors,omitempty"`
	// Exclusive: el juego está en pantalla completa exclusiva y el overlay no puede verse encima.
	Exclusive      bool  `json:"exclusive,omitempty"`
	ExclusiveSince int64 `json:"exclusiveSince,omitempty"` // Unix ms
}

// ClaimExclusiveNotice dice si hay que avisar en la bandeja que el juego está en pantalla completa
// exclusiva. Avisa una sola vez cada vez que pasa, aunque lo consulten la ventana principal y el overlay
// que arrancó con Windows: el primero que lo reclama avisa y el otro no lo repite.
func ClaimExclusiveNotice(dir string, st State) bool {
	if !st.Exclusive || st.ExclusiveSince == 0 {
		return false
	}
	p := filepath.Join(dir, "overlay_exclusive_aviso.txt")
	mark := strconv.FormatInt(st.ExclusiveSince, 10)
	if b, err := os.ReadFile(p); err == nil && string(b) == mark {
		return false
	}
	return os.WriteFile(p, []byte(mark), 0o644) == nil
}

// Cmd es una orden de la interfaz hacia el overlay (cmd.json).
type Cmd struct {
	Seq    int    `json:"seq"`
	Action string `json:"action"` // record_start, record_stop, quit
	Label  string `json:"label"`
}

// Alive dice si el estado es reciente (el overlay sigue corriendo).
func (s State) Alive(now time.Time) bool {
	return s.PID != 0 && now.Sub(s.Updated) < 5*time.Second
}

// LoadState lee state.json.
func LoadState(dir string) (State, bool) {
	var s State
	b, err := os.ReadFile(filepath.Join(dir, "overlay_state.json"))
	if err != nil || json.Unmarshal(b, &s) != nil {
		return State{}, false
	}
	return s, true
}

func saveState(dir string, s State) error {
	return writeJSON(filepath.Join(dir, "overlay_state.json"), s)
}

// SendCmd escribe una orden para el overlay.
func SendCmd(dir string, action, label string) error {
	c := Cmd{Seq: int(time.Now().UnixNano() / 1e6), Action: action, Label: label}
	return writeJSON(filepath.Join(dir, "overlay_cmd.json"), c)
}

func loadCmd(dir string) (Cmd, bool) {
	var c Cmd
	b, err := os.ReadFile(filepath.Join(dir, "overlay_cmd.json"))
	if err != nil || json.Unmarshal(b, &c) != nil {
		return Cmd{}, false
	}
	return c, true
}
