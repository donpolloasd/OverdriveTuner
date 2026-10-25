package overlay

import "testing"

func TestParseHotkey(t *testing.T) {
	ok := map[string]string{
		"ctrl+alt+o": "Ctrl+Alt+O", "Alt+Ctrl+7": "Ctrl+Alt+7", "Ctrl+F9": "Ctrl+F9",
		"Ctrl+Shift+Insert": "Ctrl+Shift+Insert", "win+alt+pagedown": "Alt+Win+PageDown", "Shift+F12": "Shift+F12",
	}
	for in, want := range ok {
		h, err := ParseHotkey(in)
		if err != nil || FormatHotkey(h) != want {
			t.Fatalf("%q -> %q %v (quería %q)", in, FormatHotkey(h), err, want)
		}
	}
	for _, bad := range []string{"", "O", "Ctrl+O", "Shift+A", "Alt+X", "Ctrl+Alt", "Ctrl+Ctrl+O", "Ctrl+Alt+F13", "Ctrl+Alt+ñ", "Ctrl+Alt+Delete", "Hyper+Alt+O", "Ctrl+Alt+Enter"} {
		if _, err := ParseHotkey(bad); err == nil {
			t.Fatalf("%q debería rechazarse", bad)
		}
	}
	if NormalizeHotkey("nada", "Ctrl+Alt+O") != "Ctrl+Alt+O" || NormalizeHotkey("alt+ctrl+p", "x") != "Ctrl+Alt+P" {
		t.Fatal("NormalizeHotkey")
	}
	h, _ := ParseHotkey("Ctrl+Alt+O")
	if h.Mod != modCtrl|modAlt || h.VK != 'O' {
		t.Fatalf("%+v", h)
	}
}
