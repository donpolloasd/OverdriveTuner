// Package winopts lista todas las opciones del Panel de control de Windows (la carpeta «God Mode») para
// buscarlas y abrirlas desde el programa. No cambia ninguna configuración.
package winopts

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// Folder es la carpeta especial de Windows con todas las opciones.
const Folder = "shell:::{ED7BA470-8E54-465E-825C-99712043E01C}"

// Option es una opción de Windows. Los nombres vienen en el idioma de Windows.
type Option struct {
	ID    string `json:"id"` // GUID de la tarea
	Name  string `json:"name"`
	Group string `json:"group"`
}

var idRe = regexp.MustCompile(`^\{[0-9A-Fa-f]{8}(-[0-9A-Fa-f]{4}){3}-[0-9A-Fa-f]{12}\}$`)

// ValidID dice si el identificador es un GUID (se usa dentro de un comando).
func ValidID(id string) bool { return idRe.MatchString(id) }

const listScript = `$f = (New-Object -ComObject Shell.Application).Namespace('` + Folder + `')
$r = @(foreach ($i in $f.Items()) { if ($i.Path -match '(\{[0-9A-Fa-f-]{36}\})$') { @{ id = $Matches[1]; name = [string]$i.Name; group = [string]$f.GetDetailsOf($i, 1) } } })
'JSON' + (ConvertTo-Json -Compress -InputObject $r)`

// List lee las opciones de Windows.
func List(sys system.System) ([]Option, error) {
	out, err := sys.PowerShell(listScript, time.Minute)
	if err != nil {
		return nil, err
	}
	return parse(out)
}

func parse(out string) ([]Option, error) {
	i := strings.Index(out, "JSON[")
	if i < 0 {
		return nil, errors.New("Windows no devolvió la lista de opciones")
	}
	var raw, res []Option
	if err := json.NewDecoder(strings.NewReader(out[i+4:])).Decode(&raw); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, o := range raw {
		o.Name, o.Group = strings.TrimSpace(o.Name), strings.TrimSpace(o.Group)
		if o.Name == "" || !ValidID(o.ID) || seen[o.ID] {
			continue
		}
		seen[o.ID] = true
		res = append(res, o)
	}
	sort.SliceStable(res, func(i, j int) bool {
		a, b := strings.ToLower(res[i].Group), strings.ToLower(res[j].Group)
		if a != b {
			return a < b
		}
		return strings.ToLower(res[i].Name) < strings.ToLower(res[j].Name)
	})
	return res, nil
}

// Open abre una opción (el id tiene que venir de List).
func Open(sys system.System, id string) error {
	if !ValidID(id) {
		return errors.New("opción no válida")
	}
	out, err := sys.PowerShell(`$f = (New-Object -ComObject Shell.Application).Namespace('`+Folder+`')
$i = $f.Items() | Where-Object { $_.Path.EndsWith('`+id+`') } | Select-Object -First 1
if ($i) { $i.InvokeVerb(); 'ABIERTO'; Start-Sleep -Milliseconds 1500 }`, time.Minute)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "ABIERTO") {
		return errors.New("Windows ya no tiene esa opción")
	}
	return nil
}

// Demo son opciones de ejemplo para el modo demostración.
func Demo() []Option {
	mk := func(n int, name, group string) Option {
		return Option{"{00000000-0000-0000-0000-00000000000" + string(rune('0'+n)) + "}", name, group}
	}
	return []Option{mk(1, "Cambiar el funcionamiento del mouse", "Centro de accesibilidad"), mk(2, "Administrador de dispositivos", "Dispositivos e impresoras"),
		mk(3, "Editar plan de energía", "Opciones de energía"), mk(4, "Administrar dispositivos de audio", "Sonido")}
}
