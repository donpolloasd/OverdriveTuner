package overlay

import (
	"errors"
	"strings"
)

// Modificadores de RegisterHotKey.
const (
	modAlt   = 0x1
	modCtrl  = 0x2
	modShift = 0x4
	modWin   = 0x8
)

// Hotkey es un atajo global ya interpretado.
type Hotkey struct {
	Mod uint32
	VK  uint32
}

var namedKeys = map[string]uint32{
	"SPACE": 0x20, "PAGEUP": 0x21, "PAGEDOWN": 0x22, "END": 0x23, "HOME": 0x24,
	"LEFT": 0x25, "UP": 0x26, "RIGHT": 0x27, "DOWN": 0x28, "INSERT": 0x2D, "DELETE": 0x2E,
}

var keyNames = func() map[uint32]string {
	m := map[uint32]string{}
	for k, v := range namedKeys {
		m[v] = strings.ToUpper(k[:1]) + strings.ToLower(k[1:])
	}
	m[0x21], m[0x22] = "PageUp", "PageDown"
	return m
}()

// ParseHotkey interpreta textos como «Ctrl+Alt+O». Para no pisar el tipeo
// normal exige al menos dos modificadores, o uno solo con una tecla F1-F12.
func ParseHotkey(s string) (Hotkey, error) {
	var h Hotkey
	parts := strings.Split(strings.TrimSpace(s), "+")
	if len(parts) < 2 {
		return h, errors.New("el atajo necesita al menos un modificador (Ctrl, Alt, Shift o Win) y una tecla")
	}
	nmods := 0
	for _, p := range parts[:len(parts)-1] {
		var bit uint32
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "ctrl", "control":
			bit = modCtrl
		case "alt":
			bit = modAlt
		case "shift", "mayús", "mayus":
			bit = modShift
		case "win", "windows", "meta":
			bit = modWin
		default:
			return h, errors.New("modificador no reconocido: " + p)
		}
		if h.Mod&bit != 0 {
			return h, errors.New("modificador repetido: " + p)
		}
		h.Mod |= bit
		nmods++
	}
	key := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
	switch {
	case len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z', len(key) == 1 && key[0] >= '0' && key[0] <= '9':
		h.VK = uint32(key[0])
	case len(key) >= 2 && key[0] == 'F':
		n := 0
		for _, c := range key[1:] {
			if c < '0' || c > '9' {
				n = -1
				break
			}
			n = n*10 + int(c-'0')
		}
		if n < 1 || n > 12 {
			return h, errors.New("tecla no válida: " + parts[len(parts)-1])
		}
		h.VK = uint32(0x70 + n - 1)
	default:
		vk, ok := namedKeys[key]
		if !ok {
			return h, errors.New("tecla no válida: " + parts[len(parts)-1])
		}
		h.VK = vk
	}
	isF := h.VK >= 0x70 && h.VK <= 0x7B
	if nmods < 2 && !isF {
		return h, errors.New("usá al menos dos modificadores (por ejemplo Ctrl+Alt) o una tecla F1-F12, para no pisar el tipeo normal")
	}
	if h.Mod == modCtrl|modAlt && h.VK == 0x2E {
		return h, errors.New("Ctrl+Alt+Supr está reservado por Windows")
	}
	return h, nil
}

// FormatHotkey devuelve el texto canónico de un atajo («Ctrl+Alt+O»).
func FormatHotkey(h Hotkey) string {
	var parts []string
	if h.Mod&modCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if h.Mod&modAlt != 0 {
		parts = append(parts, "Alt")
	}
	if h.Mod&modShift != 0 {
		parts = append(parts, "Shift")
	}
	if h.Mod&modWin != 0 {
		parts = append(parts, "Win")
	}
	switch {
	case h.VK >= 'A' && h.VK <= 'Z', h.VK >= '0' && h.VK <= '9':
		parts = append(parts, string(rune(h.VK)))
	case h.VK >= 0x70 && h.VK <= 0x7B:
		parts = append(parts, "F"+itoa(int(h.VK-0x70)+1))
	default:
		parts = append(parts, keyNames[h.VK])
	}
	return strings.Join(parts, "+")
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// NormalizeHotkey devuelve el atajo canónico, o def si s no es válido.
func NormalizeHotkey(s, def string) string {
	h, err := ParseHotkey(s)
	if err != nil {
		return def
	}
	return FormatHotkey(h)
}
