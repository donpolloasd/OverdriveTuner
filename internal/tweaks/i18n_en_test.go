package tweaks

import "testing"

func TestEnglishCoversEveryTweak(t *testing.T) {
	for _, tw := range All() {
		e, ok := English(tw.ID)
		if !ok || e.Name == "" || e.Desc == "" {
			t.Errorf("falta la traducción de %s", tw.ID)
			continue
		}
		if (tw.Warn != "") != (e.Warn != "") {
			t.Errorf("%s: la advertencia y su traducción no coinciden", tw.ID)
		}
		if (tw.AC != "") != (e.AC != "") {
			t.Errorf("%s: el texto de anti-cheat y su traducción no coinciden", tw.ID)
		}
	}
	for _, c := range Categories() {
		if e, ok := EnglishCategory(c.ID); !ok || e.Name == "" || e.Desc == "" {
			t.Errorf("falta la categoría %s", c.ID)
		}
	}
}
