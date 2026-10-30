package overlay

import "testing"

func TestDefaultStyleIsTheOriginalLook(t *testing.T) {
	p := Default().Style.Palette()
	if p.BG != ColBG || p.Value != ColText || p.Label != ColLabel || p.FPS != ColText || p.Warn != ColWarn || p.Bad != ColBad || p.Graph != ColGood {
		t.Fatalf("el tema clásico debe verse igual que antes: %+v", p)
	}
	if colorRef("#FF5F78") != ColBad || colorRef("#0A0A0A") != ColBG {
		t.Fatal("conversión de #RRGGBB a COLORREF")
	}
}

func TestStyleNormalize(t *testing.T) {
	c := Default()
	c.Theme, c.ColorFPS, c.ColorBG = "custom", "#00ff88", "rojo"
	c.FpsLow, c.FpsMid, c.Radius, c.Border, c.Font, c.Weight, c.FpsSize = 200, 100, 99, 7, "comic", "x", 5
	c.Normalize()
	if c.ColorFPS != "#00FF88" || c.ColorBG != DefaultStyle().ColorBG {
		t.Fatalf("colores propios: válido se conserva, inválido vuelve al original: %q %q", c.ColorFPS, c.ColorBG)
	}
	if c.FpsLow != 60 || c.FpsMid != 144 || c.Radius != 13 || c.Border != 0 || c.Font != "segoe" || c.Weight != "semibold" || c.FpsSize != 1 {
		t.Fatalf("valores fuera de rango: %+v", c.Style)
	}
	c.Theme, c.ColorFPS = "neon", "#123456"
	c.Normalize()
	if c.ColorFPS != "#39FF14" {
		t.Fatal("un tema usa siempre sus colores")
	}
	c.Theme = "inventado"
	c.Normalize()
	if c.Theme != "custom" {
		t.Fatal("un tema desconocido pasa a personalizado")
	}
}

func TestFPSColorByValue(t *testing.T) {
	s := DefaultStyle()
	p := s.Palette()
	if s.fpsColor(p, 30) != p.FPS {
		t.Fatal("sin «según el valor» el color es fijo")
	}
	s.FpsByValue = true
	if s.fpsColor(p, 30) != p.Bad || s.fpsColor(p, 100) != p.Warn || s.fpsColor(p, 240) != p.Good {
		t.Fatal("colores por valor de FPS")
	}
	sn := snap()
	sn.Cfg.Style.FpsByValue = true
	if m := BuildModel(sn); m.FPSColor != p.Good {
		t.Fatalf("238 FPS con objetivo 144 es «bien»: %x", m.FPSColor)
	}
}
