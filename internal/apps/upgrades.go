package apps

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"fpsforge/internal/system"
)

// Upgrade es un programa instalado con una versión nueva disponible en winget.
type Upgrade struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

var pkgRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+\-]{1,127}$`)

// ValidPackage dice si el texto puede ser un identificador de winget.
func ValidPackage(id string) bool { return pkgRe.MatchString(id) }

// Upgrades consulta qué programas tienen una versión nueva. No cambia nada.
func Upgrades(sys system.System) ([]Upgrade, error) {
	out, code, err := sys.Exec(3*time.Minute, "winget.exe", "upgrade", wingetBase, "--disable-interactivity")
	if err != nil {
		return nil, err
	}
	l := parseUpgrades(out)
	if c := norm(code); len(l) == 0 && c != 0 && c != exitNoUpgrade {
		return nil, fmt.Errorf("winget terminó con código %d", code)
	}
	return l, nil
}

// width es el ancho de una letra en la tabla de winget (las de Asia Oriental ocupan dos columnas).
func width(r rune) int {
	if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) || (r >= 0xFF01 && r <= 0xFF60) {
		return 2
	}
	return 1
}

// column recorta una línea entre dos columnas de pantalla.
func column(line string, from, to int) string {
	var b strings.Builder
	pos := 0
	for _, r := range line {
		if pos >= from && (to < 0 || pos < to) {
			b.WriteRune(r)
		}
		pos += width(r)
	}
	return strings.TrimSpace(b.String())
}

// parseUpgrades lee la tabla de «winget upgrade». Los títulos cambian con el idioma de Windows, así que las
// columnas salen de la posición de cada título: nombre, id, versión, disponible y origen.
func parseUpgrades(out string) []Upgrade {
	res := []Upgrade{}
	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	seen := map[string]bool{}
	var cols []int
	for i, ln := range lines {
		if j := strings.LastIndexByte(ln, '\r'); j >= 0 { // restos de la barra de progreso
			ln = ln[j+1:]
		}
		t := strings.TrimSpace(ln)
		if len(t) >= 20 && strings.Trim(t, "-") == "" && i > 0 {
			head := lines[i-1]
			if j := strings.LastIndexByte(head, '\r'); j >= 0 {
				head = head[j+1:]
			}
			cols = nil
			pos, prev := 0, ' '
			for _, r := range head {
				if r != ' ' && prev == ' ' {
					cols = append(cols, pos)
				}
				prev = r
				pos += width(r)
			}
			continue
		}
		if len(cols) < 4 || t == "" {
			continue
		}
		end := -1
		if len(cols) > 4 {
			end = cols[4]
		}
		u := Upgrade{column(ln, 0, cols[1]), column(ln, cols[1], cols[2]), column(ln, cols[2], cols[3]), column(ln, cols[3], end)}
		// Las líneas de resumen («18 actualizaciones disponibles») y los datos cortados («…») no son filas.
		if u.Name == "" || u.From == "" || u.To == "" || !ValidPackage(u.ID) || strings.ContainsAny(u.To, " …") || seen[u.ID] {
			continue
		}
		seen[u.ID] = true
		res = append(res, u)
	}
	return res
}

// UpgradeOne actualiza un programa.
func UpgradeOne(sys system.System, id string) Result {
	if !ValidPackage(id) {
		return Result{Notes: []string{"identificador no válido"}}
	}
	out, code, err := sys.Exec(30*time.Minute, "winget.exe", "upgrade", "--id", id, "--exact", "--silent",
		"--accept-package-agreements", wingetBase, "--disable-interactivity")
	if err != nil {
		return Result{Notes: []string{err.Error()}}
	}
	if c := norm(code); c != 0 && c != exitNoUpgrade {
		return Result{Notes: []string{fmt.Sprintf("no se pudo actualizar (código %d) %s", code, lastLine(out))}}
	}
	return Result{OK: true}
}
