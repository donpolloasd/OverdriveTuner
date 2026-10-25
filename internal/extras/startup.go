package extras

import (
	"fmt"
	"sort"
	"strings"

	"fpsforge/internal/system"
)

// StartupItem es un programa que se inicia con Windows.
type StartupItem struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Scope   string `json:"scope"` // usuario | sistema
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"`
}

type startSrc struct {
	run, approved, scope string
}

var startSources = []startSrc{
	{`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`, "usuario"},
	{`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`, "sistema"},
	{`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32`, "sistema"},
}

// disabledMarker: Windows marca un inicio como deshabilitado con el primer byte
// impar (03 = deshabilitado por el usuario) y habilitado con par (02).
func isEnabledBlob(hex string) bool {
	hex = strings.ToLower(strings.TrimSpace(hex))
	if len(hex) < 2 {
		return true
	}
	switch hex[:2] {
	case "03", "01", "07":
		return false
	}
	return true
}

// ListStartup lista los programas de inicio con su estado.
func ListStartup(sys system.System) ([]StartupItem, error) {
	var out []StartupItem
	for _, s := range startSources {
		runs, err := sys.RegEnumValues(s.run)
		if err != nil {
			return nil, err
		}
		appr, _ := sys.RegEnumValues(s.approved)
		state := map[string]bool{}
		for _, a := range appr {
			state[strings.ToLower(a.Name)] = isEnabledBlob(a.Data)
		}
		for _, r := range runs {
			if r.Name == "" {
				continue
			}
			en, ok := state[strings.ToLower(r.Name)]
			if !ok {
				en = true
			}
			out = append(out, StartupItem{Name: r.Name, Command: r.Data, Scope: s.scope, Enabled: en, Source: s.run})
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// SetStartup habilita o deshabilita un programa de inicio sin borrarlo.
func SetStartup(sys system.System, source, name string, enabled bool) error {
	var s *startSrc
	for i := range startSources {
		if startSources[i].run == source {
			s = &startSources[i]
		}
	}
	if s == nil {
		return fmt.Errorf("origen de inicio no válido")
	}
	if s.scope == "sistema" && !sys.IsAdmin() {
		return fmt.Errorf("se necesitan permisos de administrador")
	}
	// 12 bytes: estado (02 habilitado / 03 deshabilitado) + 11 bytes de ceros.
	blob := "020000000000000000000000"
	if !enabled {
		blob = "030000000000000000000000"
	}
	return sys.RegSet(s.approved, name, "REG_BINARY", blob)
}
