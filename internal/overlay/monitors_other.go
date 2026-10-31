//go:build !windows

package overlay

// Monitors fuera de Windows devuelve una pantalla de ejemplo (modo demostración).
func Monitors() []Monitor {
	return []Monitor{{Index: 0, Width: 1920, Height: 1080, Primary: true, Name: "Monitor 1 · 1920×1080 (principal)"}}
}

func systemLang() string { return "es" }
