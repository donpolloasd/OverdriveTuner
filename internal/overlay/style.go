package overlay

import (
	"regexp"
	"strconv"
	"strings"
)

// Style es el aspecto del panel: colores, forma y letra. Va dentro de Config (overlay.json).
type Style struct {
	Theme string `json:"theme"` // clasico, neon, cian, ambar, rojo, contraste o custom (colores propios)
	// Colores en #RRGGBB. Con un tema que no es «custom» se usan los del tema.
	ColorBG     string  `json:"colorBg"`     // fondo de la tarjeta
	ColorValue  string  `json:"colorValue"`  // valores (41%, 11.2 GB…)
	ColorLabel  string  `json:"colorLabel"`  // etiquetas (CPU, GPU, «FPS»)
	ColorFPS    string  `json:"colorFps"`    // número de FPS
	ColorGood   string  `json:"colorGood"`   // bien (FPS por encima del objetivo, con «color según el valor»)
	ColorWarn   string  `json:"colorWarn"`   // aviso (temperatura alta, FPS justos, picos medianos)
	ColorBad    string  `json:"colorBad"`    // problema (sobrecalentamiento, FPS bajos, picos grandes)
	ColorGraph  string  `json:"colorGraph"`  // barras normales del gráfico
	ColorBorder string  `json:"colorBorder"` // borde de la tarjeta
	FpsByValue  bool    `json:"fpsByValue"`  // el número de FPS cambia de color según el valor
	FpsLow      int     `json:"fpsLow"`      // por debajo: color de problema
	FpsMid      int     `json:"fpsMid"`      // por debajo: color de aviso; desde acá: color «bien»
	Transparent bool    `json:"transparent"` // sin tarjeta: solo el texto (con contorno) sobre el juego
	Radius      int     `json:"radius"`      // redondeo de las esquinas, 0 a 20
	Border      int     `json:"border"`      // grosor del borde, 0 a 3 (0 = sin borde)
	Font        string  `json:"font"`        // segoe, bahnschrift o consolas
	Weight      string  `json:"weight"`      // normal, semibold o bold
	FpsSize     float64 `json:"fpsSize"`     // tamaño del número de FPS, 0.7 a 1.6
}

// Theme es un juego de colores listo para usar.
type Theme struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	BG     string `json:"colorBg"`
	Value  string `json:"colorValue"`
	Label  string `json:"colorLabel"`
	FPS    string `json:"colorFps"`
	Good   string `json:"colorGood"`
	Warn   string `json:"colorWarn"`
	Bad    string `json:"colorBad"`
	Graph  string `json:"colorGraph"`
	Border string `json:"colorBorder"`
}

// Themes son los temas de colores, en el orden en que se muestran. «clasico» es el aspecto original.
var Themes = []Theme{
	{"clasico", "Clásico", "#0A0A0A", "#F5F5F5", "#8A8A8A", "#F5F5F5", "#57D9A3", "#FFC857", "#FF5F78", "#C4C4C4", "#333333"},
	{"neon", "Verde neón", "#040A04", "#B8FFB0", "#3FAA4F", "#39FF14", "#39FF14", "#FFD400", "#FF3B3B", "#2ECC40", "#39FF14"},
	{"cian", "Cian", "#04121A", "#E6FBFF", "#4FB3C8", "#00E5FF", "#00E5FF", "#FFC857", "#FF5F78", "#00B8D4", "#00E5FF"},
	{"ambar", "Ámbar", "#140D02", "#FFE3B0", "#B5823A", "#FFB000", "#9BE15D", "#FF8C00", "#FF4D4D", "#E09A2B", "#FFB000"},
	{"rojo", "Rojo gamer", "#120406", "#FFFFFF", "#B0585E", "#FF2E4D", "#57D9A3", "#FFC857", "#FF2E4D", "#E0475E", "#FF2E4D"},
	{"contraste", "Alto contraste", "#000000", "#FFFFFF", "#FFFF00", "#FFFFFF", "#00FF00", "#FFFF00", "#FF0000", "#FFFFFF", "#FFFFFF"},
}

// Fonts son las letras disponibles (todas vienen con Windows 10/11): id → nombre de la fuente.
var Fonts = map[string]string{"segoe": "Segoe UI", "bahnschrift": "Bahnschrift", "consolas": "Consolas"}

func themeByID(id string) (Theme, bool) {
	for _, t := range Themes {
		if t.ID == id {
			return t, true
		}
	}
	return Theme{}, false
}

// DefaultStyle es el aspecto original del panel.
func DefaultStyle() Style {
	s := Style{Theme: "clasico", FpsLow: 60, FpsMid: 144, Radius: 13, Font: "segoe", Weight: "semibold", FpsSize: 1}
	s.setColors(Themes[0])
	return s
}

func (s *Style) setColors(t Theme) {
	s.ColorBG, s.ColorValue, s.ColorLabel, s.ColorFPS = t.BG, t.Value, t.Label, t.FPS
	s.ColorGood, s.ColorWarn, s.ColorBad, s.ColorGraph, s.ColorBorder = t.Good, t.Warn, t.Bad, t.Graph, t.Border
}

var hexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// normalize corrige lo fuera de rango. Un color que no es #RRGGBB vuelve al del tema clásico.
func (s *Style) normalize() {
	d := DefaultStyle()
	if t, ok := themeByID(s.Theme); ok {
		s.setColors(t) // un tema siempre usa sus colores; los propios son los de «custom»
	} else {
		s.Theme = "custom"
		fix := func(v *string, def string) {
			if !hexRe.MatchString(*v) {
				*v = def
			}
			*v = strings.ToUpper(*v)
		}
		fix(&s.ColorBG, d.ColorBG)
		fix(&s.ColorValue, d.ColorValue)
		fix(&s.ColorLabel, d.ColorLabel)
		fix(&s.ColorFPS, d.ColorFPS)
		fix(&s.ColorGood, d.ColorGood)
		fix(&s.ColorWarn, d.ColorWarn)
		fix(&s.ColorBad, d.ColorBad)
		fix(&s.ColorGraph, d.ColorGraph)
		fix(&s.ColorBorder, d.ColorBorder)
	}
	if s.FpsLow < 10 || s.FpsLow > 1000 {
		s.FpsLow = d.FpsLow
	}
	if s.FpsMid < 10 || s.FpsMid > 1000 {
		s.FpsMid = d.FpsMid
	}
	if s.FpsLow >= s.FpsMid {
		s.FpsLow, s.FpsMid = d.FpsLow, d.FpsMid
	}
	if s.Radius < 0 || s.Radius > 20 {
		s.Radius = d.Radius
	}
	if s.Border < 0 || s.Border > 3 {
		s.Border = 0
	}
	if _, ok := Fonts[s.Font]; !ok {
		s.Font = d.Font
	}
	if s.Weight != "normal" && s.Weight != "semibold" && s.Weight != "bold" {
		s.Weight = d.Weight
	}
	if s.FpsSize < 0.7 || s.FpsSize > 1.6 {
		s.FpsSize = d.FpsSize
	}
}

// Palette son los colores ya listos para dibujar (COLORREF 0x00BBGGRR).
type Palette struct {
	BG, Value, Label, FPS, Good, Warn, Bad, Graph, Border uint32
}

// colorRef convierte #RRGGBB en COLORREF (0x00BBGGRR).
func colorRef(hex string) uint32 {
	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return ColText
	}
	r, g, b := uint32(v>>16)&0xFF, uint32(v>>8)&0xFF, uint32(v)&0xFF
	return b<<16 | g<<8 | r
}

// Palette devuelve los colores del estilo para dibujar.
func (s Style) Palette() Palette {
	return Palette{colorRef(s.ColorBG), colorRef(s.ColorValue), colorRef(s.ColorLabel), colorRef(s.ColorFPS),
		colorRef(s.ColorGood), colorRef(s.ColorWarn), colorRef(s.ColorBad), colorRef(s.ColorGraph), colorRef(s.ColorBorder)}
}

// fpsColor es el color del número de FPS: fijo, o según el valor si así se eligió.
func (s Style) fpsColor(p Palette, fps float64) uint32 {
	if !s.FpsByValue {
		return p.FPS
	}
	switch {
	case fps < float64(s.FpsLow):
		return p.Bad
	case fps < float64(s.FpsMid):
		return p.Warn
	}
	return p.Good
}
