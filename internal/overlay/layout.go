package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// RowKeys son las filas de datos del panel, en su orden original (debajo del número de FPS y el gráfico).
var RowKeys = []string{"cpu", "gpu", "vram", "ram", "frame", "clock", "game"}

// normalizeOrder deja cada fila conocida una sola vez, en el orden elegido; las que falten van al final.
func normalizeOrder(in []string) []string {
	known := map[string]bool{}
	for _, k := range RowKeys {
		known[k] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, k := range in {
		if known[k] && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for _, k := range RowKeys {
		if !seen[k] {
			out = append(out, k)
		}
	}
	return out
}

// Preset es un estilo guardado con nombre: colores, forma y letra, más diseño, tamaño y transparencia.
type Preset struct {
	Name    string  `json:"name"`
	Layout  string  `json:"layout"`
	Scale   float64 `json:"scale"`
	Opacity float64 `json:"opacity"`
	Style   Style   `json:"style"`
}

// normalize corrige el estilo guardado; devuelve false si no tiene un nombre usable.
func (p *Preset) normalize() bool {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len([]rune(p.Name)) > 30 || strings.ContainsAny(p.Name, "<>\"`") {
		return false
	}
	d := Default()
	if p.Layout != "card" && p.Layout != "line" {
		p.Layout = d.Layout
	}
	if p.Scale < 0.8 || p.Scale > 1.8 {
		p.Scale = d.Scale
	}
	if p.Opacity < 0.3 || p.Opacity > 1 {
		p.Opacity = d.Opacity
	}
	p.Style.normalize()
	return true
}

// ---- idioma de los textos que se dibujan sobre el juego ----

// UILang devuelve el idioma de la interfaz («es» o «en»): el que se eligió en Configuración o, si no se
// eligió ninguno, el de Windows.
func UILang(dir string) string {
	var st struct {
		Lang string `json:"lang"`
	}
	if b, err := os.ReadFile(filepath.Join(dir, "settings.json")); err == nil && json.Unmarshal(b, &st) == nil && (st.Lang == "es" || st.Lang == "en") {
		return st.Lang
	}
	return systemLang()
}

// overlayEN son los textos del panel en inglés (las etiquetas CPU, GPU, RAM, VRAM y FPS son iguales).
var overlayEN = map[string]string{"Aviso": "Alert", "CPU caliente": "CPU hot", "GPU caliente": "GPU hot", "Cuadro": "Frame", "Hora": "Time", "Juego": "Game"}

// tr traduce un texto del panel al idioma de la interfaz.
func (sn Snapshot) tr(s string) string {
	if sn.Lang == "en" {
		if t, ok := overlayEN[s]; ok {
			return t
		}
	}
	return s
}

// Monitor es una pantalla donde puede aparecer el panel.
type Monitor struct {
	Index   int    `json:"index"` // 0 = el principal
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Primary bool   `json:"primary"`
	Name    string `json:"name"`
	L, T    int    `json:"-"` // esquina superior izquierda (coordenadas del escritorio)
}
