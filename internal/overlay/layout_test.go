package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRowOrderAndNewRows(t *testing.T) {
	sn := snap()
	sn.Cfg.Order = []string{"ram", "inventada", "cpu", "ram"}
	sn.Cfg.ShowFrame, sn.Cfg.ShowClock, sn.Cfg.ShowGame = true, true, true
	sn.Game, sn.Now = "Cyberpunk2077_Super_Larguisimo.exe", time.Date(2026, 10, 30, 21, 5, 0, 0, time.Local)
	m := BuildModel(sn)
	var labels []string
	for _, r := range m.Rows {
		labels = append(labels, r.Label)
	}
	if got := strings.Join(labels, ","); got != "RAM,CPU,GPU,Cuadro,Hora,Juego" {
		t.Fatalf("orden: primero el elegido, sin repetidos ni desconocidos, y el resto al final: %s", got)
	}
	if v := m.Rows[3].Parts[0].Text; v != "4.2 ms" {
		t.Fatalf("tiempo de cuadro a 238 FPS: %q", v)
	}
	if v := m.Rows[4].Parts[0].Text; v != "21:05" {
		t.Fatalf("hora: %q", v)
	}
	if v := m.Rows[5].Parts[0].Text; len([]rune(v)) != 16 || !strings.HasSuffix(v, "…") {
		t.Fatalf("nombre de juego largo recortado: %q", v)
	}
}

func TestOverlayEnglish(t *testing.T) {
	sn := snap()
	sn.Lang, sn.AlertGPU, sn.Cfg.ShowFrame = "en", true, true
	m := BuildModel(sn)
	last := m.Rows[len(m.Rows)-1]
	if last.Label != "Alert" || last.Parts[0].Text != "GPU hot" || m.Rows[len(m.Rows)-2].Label != "Frame" {
		t.Fatalf("textos en inglés: %+v", m.Rows)
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"lang":"en"}`), 0o644)
	if UILang(dir) != "en" {
		t.Fatal("el overlay sigue el idioma elegido en Configuración")
	}
}

func TestPresetsAndPosition(t *testing.T) {
	c := Default()
	c.OffsetX, c.OffsetY, c.Monitor = 5000, -3, 42
	c.Presets = []Preset{{Name: "  Competitivo ", Scale: 9, Style: Style{Theme: "neon"}}, {Name: ""}, {Name: "<script>"}}
	c.Normalize()
	if c.OffsetX != 14 || c.OffsetY != 14 || c.Monitor != 0 {
		t.Fatalf("posición fuera de rango: %d %d %d", c.OffsetX, c.OffsetY, c.Monitor)
	}
	if len(c.Presets) != 1 || c.Presets[0].Name != "Competitivo" || c.Presets[0].Scale != 1 || c.Presets[0].Style.ColorFPS != "#39FF14" {
		t.Fatalf("estilos guardados: %+v", c.Presets)
	}
}
