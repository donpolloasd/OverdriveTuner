// Package overlay implementa el overlay minimalista de FPS y rendimiento.
// Corre como un proceso aparte (OverdriveTuner.exe --overlay) con una ventana
// propia siempre visible y transparente a los clics; no se inyecta en el
// juego, así que no afecta a los anti-cheat.
package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Game es un perfil por juego.
type Game struct {
	Exe       string   `json:"exe"`
	Priority  bool     `json:"priority"`  // prioridad Alta al proceso del juego
	PowerPlan string   `json:"powerPlan"` // GUID del plan de energía mientras se juega (vacío = no cambiar)
	Close     []string `json:"close"`     // programas a cerrar al abrir el juego (los eligió el usuario)
}

// Config es la configuración del overlay.
type Config struct {
	Show      bool     `json:"show"`
	Corner    string   `json:"corner"`  // tl, tr, bl, br
	Layout    string   `json:"layout"`  // card o line
	Scale     float64  `json:"scale"`   // 0.8 a 1.8
	Opacity   float64  `json:"opacity"` // 0.3 a 1
	ShowFPS   bool     `json:"showFps"`
	ShowLow   bool     `json:"showLow"`
	ShowGrph  bool     `json:"showGraph"`
	ShowCPU   bool     `json:"showCpu"`
	ShowGPU   bool     `json:"showGpu"`
	ShowRAM   bool     `json:"showRam"`
	ShowTemp  bool     `json:"showTemps"`
	ShowVRAM  bool     `json:"showVram"`
	Hotkey    bool     `json:"hotkey"`     // activa los atajos globales
	HkShow    string   `json:"hkShow"`     // muestra/oculta el overlay
	HkRecord  string   `json:"hkRecord"`   // empieza/detiene una grabación
	HkCorner  string   `json:"hkCorner"`   // cambia de esquina
	FpsSource string   `json:"fpsSource"`  // siempre rtss: los FPS salen de RTSS
	GameDirs  []string `json:"gameDirs"`   // carpetas de juegos adicionales (ej. D:\\Juegos)
	UpdateMs  int      `json:"updateMs"`   // cada cuántos milisegundos se redibuja el panel (50 a 500)
	GameExe   string   `json:"gameExe"`    // juego a medir (ej. cs2.exe) cuando la detección automática está apagada
	AutoGame  bool     `json:"autoGame"`   // detecta solo qué juego está abierto
	OnlyGame  bool     `json:"onlyInGame"` // el panel aparece solo mientras hay un juego abierto
	AutoOpen  bool     `json:"autoOpen"`   // el overlay se abre solo al iniciar OverdriveTuner
	AlertOn   bool     `json:"alertOn"`    // avisa en el overlay cuando la PC se calienta de más
	AlertCPU  int      `json:"alertCpu"`   // °C a partir de los que se avisa del procesador (60 a 110)
	AlertGPU  int      `json:"alertGpu"`   // °C a partir de los que se avisa de la gráfica (60 a 110)
	CloseList []string `json:"closeList"`  // Modo juego: programas que se cierran siempre al abrir un juego
	Games     []Game   `json:"games"`
	Style              // aspecto del panel: colores, forma y letra (se guarda en el mismo nivel del JSON)
}

// Fuentes de FPS.
const (
	SrcRTSS = "rtss"
)

// Default devuelve la configuración inicial.
func Default() Config {
	return Config{Show: true, Corner: "tl", Layout: "card", Scale: 1, Opacity: 0.82,
		ShowFPS: true, ShowLow: true, ShowGrph: true, ShowCPU: true, ShowGPU: true, ShowRAM: true,
		ShowTemp: true, Hotkey: true, HkShow: "Ctrl+Alt+O", HkRecord: "Ctrl+Alt+R", HkCorner: "Ctrl+Alt+P", UpdateMs: 100, FpsSource: SrcRTSS, GameExe: "cs2.exe", AutoGame: true, OnlyGame: true, AutoOpen: true, AlertOn: true, AlertCPU: 90, AlertGPU: 85,
		Games: []Game{{Exe: "cs2.exe", Priority: false}}, Style: DefaultStyle()}
}

var corners = map[string]bool{"tl": true, "tr": true, "bl": true, "br": true}

// ValidExe acepta solo nombres de programa simples: sin rutas, espacios, comillas ni caracteres que
// PowerShell interprete (el nombre se usa dentro de comandos).
func ValidExe(s string) bool {
	if len(s) < 5 || len(s) > 64 || !strings.HasSuffix(strings.ToLower(s), ".exe") {
		return false
	}
	return !strings.ContainsAny(s, `\/:*?"<>| ,;&'`+"`$")
}

// Normalize corrige valores fuera de rango y rechaza lo no válido.
func (c *Config) Normalize() {
	d := Default()
	if !corners[c.Corner] {
		c.Corner = d.Corner
	}
	if c.Layout != "card" && c.Layout != "line" {
		c.Layout = d.Layout
	}
	var dirs []string
	for _, d := range c.GameDirs {
		if d = strings.TrimSpace(d); len(d) >= 4 && len(d) <= 260 && len(dirs) < 30 {
			dirs = append(dirs, d)
		}
	}
	c.GameDirs = dirs
	c.FpsSource = SrcRTSS // los FPS salen solo de RTSS (ya no hay otra fuente)
	switch c.UpdateMs {
	case 50, 100, 250, 500:
	default:
		c.UpdateMs = d.UpdateMs
	}
	if c.Scale < 0.8 || c.Scale > 1.8 {
		c.Scale = d.Scale
	}
	if c.Opacity < 0.3 || c.Opacity > 1 {
		c.Opacity = d.Opacity
	}
	if c.AlertCPU < 60 || c.AlertCPU > 110 {
		c.AlertCPU = d.AlertCPU
	}
	if c.AlertGPU < 60 || c.AlertGPU > 110 {
		c.AlertGPU = d.AlertGPU
	}
	var cl []string
	seen := map[string]bool{}
	for _, n := range c.CloseList {
		n = strings.TrimSpace(n)
		k := strings.ToLower(n)
		if ValidExe(n) && !seen[k] && len(cl) < 40 {
			seen[k] = true
			cl = append(cl, n)
		}
	}
	c.CloseList = cl
	if !ValidExe(c.GameExe) {
		c.GameExe = d.GameExe
	}
	c.HkShow = NormalizeHotkey(c.HkShow, d.HkShow)
	c.HkRecord = NormalizeHotkey(c.HkRecord, d.HkRecord)
	c.HkCorner = NormalizeHotkey(c.HkCorner, d.HkCorner)
	// Dos acciones no pueden compartir atajo: la repetida vuelve a su valor de fábrica.
	if c.HkRecord == c.HkShow {
		c.HkRecord = d.HkRecord
	}
	if c.HkCorner == c.HkShow || c.HkCorner == c.HkRecord {
		c.HkCorner = d.HkCorner
	}
	var gs []Game
	for _, g := range c.Games {
		if !ValidExe(g.Exe) {
			continue
		}
		var cl []string
		for _, n := range g.Close {
			if ValidExe(n) && len(cl) < 20 {
				cl = append(cl, n)
			}
		}
		g.Close = cl
		if g.PowerPlan != "" && !guidRe.MatchString(g.PowerPlan) {
			g.PowerPlan = ""
		}
		if len(gs) < 20 {
			gs = append(gs, g)
		}
	}
	c.Games = gs
	c.Style.normalize()
}

// LoadConfig lee overlay.json (o devuelve los valores por defecto).
func LoadConfig(dir string) Config {
	c := Default()
	if b, err := os.ReadFile(filepath.Join(dir, "overlay.json")); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	c.Normalize()
	return c
}

// SaveConfig escribe overlay.json de forma atómica.
func SaveConfig(dir string, c Config) error {
	c.Normalize()
	return writeJSON(filepath.Join(dir, "overlay.json"), c)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GameFor devuelve el perfil del juego exe (si existe).
func (c Config) GameFor(exe string) (Game, bool) {
	for _, g := range c.Games {
		if strings.EqualFold(g.Exe, exe) {
			return g, true
		}
	}
	return Game{}, false
}

// HotkeyConflict devuelve un mensaje si el texto de algún atajo no es válido o se repite.
func (c Config) HotkeyConflict() string {
	names := []struct{ label, v string }{{"Mostrar/ocultar", c.HkShow}, {"Grabar", c.HkRecord}, {"Cambiar esquina", c.HkCorner}}
	seen := map[string]string{}
	for _, n := range names {
		if n.v == "" {
			continue
		}
		h, err := ParseHotkey(n.v)
		if err != nil {
			return n.label + ": " + err.Error()
		}
		k := FormatHotkey(h)
		if other, dup := seen[k]; dup {
			return "El atajo " + k + " está repetido en «" + other + "» y «" + n.label + "»"
		}
		seen[k] = n.label
	}
	return ""
}
