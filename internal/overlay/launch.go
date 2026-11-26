package overlay

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// «Arranque optimizado»: el perfil de un juego puede guardar cómo abrirlo (la ruta del programa o el enlace
// de su launcher). Al tocar «Jugar» se abre el overlay si no estaba abierto (es el que aplica el perfil al
// detectar el juego y lo deshace al cerrarlo) y después el juego.

var launchURLRe = regexp.MustCompile(`^(steam://rungameid/\d{1,12}|com\.epicgames\.launcher://apps/[A-Za-z0-9%._:-]{1,200}\?action=launch&silent=true|uplay://launch/\d{1,10}/0|origin2?://launchgame/[A-Za-z0-9._:-]{1,80}|battlenet://[A-Za-z0-9]{1,20})$`)

// ValidLaunch dice si se puede usar para abrir un juego: una ruta completa a un .exe o el enlace de un
// launcher conocido (Steam, Epic, Ubisoft, EA, Battle.net). Nada de comillas ni argumentos.
func ValidLaunch(s string) bool {
	if s == "" || len(s) > 300 {
		return s == ""
	}
	if launchURLRe.MatchString(s) {
		return true
	}
	return filepath.IsAbs(s) && strings.HasSuffix(strings.ToLower(s), ".exe") && !strings.ContainsAny(s, `"*?<>|`+"\n\r\t") &&
		!strings.HasPrefix(s, `\\`) // nada de carpetas de red
}

// Launch abre el juego sin permisos de administrador: el Explorador de Windows lo abre como si el usuario
// le hubiera hecho doble clic (un juego abierto como administrador molesta a algunos anti-cheat).
func Launch(sys system.System, target string) error {
	if target == "" || !ValidLaunch(target) {
		return errors.New("Primero elegí cómo se abre este juego.")
	}
	if !launchURLRe.MatchString(target) {
		if fi, err := os.Stat(target); err != nil || fi.IsDir() {
			return errors.New("Ese programa ya no está en esa carpeta. Buscalo de nuevo.")
		}
	}
	// explorer.exe devuelve un código distinto de cero aunque abra bien, así que no se mira el código.
	_, _, err := sys.Exec(15*time.Second, "explorer.exe", target)
	return err
}

// steamLibraries lee las bibliotecas de Steam (libraryfolders.vdf).
func steamLibraries(root string) []string {
	if root == "" {
		return nil
	}
	libs := []string{root}
	b, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return libs
	}
	re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		p := strings.ReplaceAll(m[1], `\\`, `\`)
		if !strings.EqualFold(filepath.Clean(p), filepath.Clean(root)) {
			libs = append(libs, p)
		}
	}
	return libs
}

var appIDRe = regexp.MustCompile(`"appid"\s+"(\d+)"`)
var installDirRe = regexp.MustCompile(`"installdir"\s+"([^"]+)"`)

// steamAppID busca el número de Steam del juego instalado en esa carpeta de steamapps\common.
func steamAppID(lib, dir string) string {
	files, _ := filepath.Glob(filepath.Join(lib, "steamapps", "appmanifest_*.acf"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		d, id := installDirRe.FindSubmatch(b), appIDRe.FindSubmatch(b)
		if d != nil && id != nil && strings.EqualFold(string(d[1]), dir) {
			return string(id[1])
		}
	}
	return ""
}

// findIn busca el programa exe dentro de root, hasta depth carpetas de profundidad.
func findIn(root, exe string, depth int) string {
	found := ""
	base := strings.Count(filepath.Clean(root), string(filepath.Separator))
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if strings.Count(filepath.Clean(p), string(filepath.Separator))-base >= depth {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), exe) {
			found = p
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// FindLaunch busca cómo abrir un juego por el nombre de su programa (por ejemplo cs2.exe): primero en las
// bibliotecas de Steam (si lo encuentra devuelve el enlace de Steam, que abre el juego como corresponde),
// después en las carpetas de Epic, EA, GOG, Ubisoft y las que agregó el usuario.
func FindLaunch(exe string, extra []string) string {
	if !ValidExe(exe) {
		return ""
	}
	for _, lib := range steamLibraries(steamRoot()) {
		common := filepath.Join(lib, "steamapps", "common")
		entries, _ := os.ReadDir(common)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if p := findIn(filepath.Join(common, e.Name()), exe, 5); p != "" {
				if id := steamAppID(lib, e.Name()); id != "" {
					return "steam://rungameid/" + id
				}
				return p
			}
		}
	}
	for _, d := range append(append(epicDirs(), registryDirs()...), extra...) {
		if p := findIn(d, exe, 5); p != "" && ValidLaunch(p) {
			return p
		}
	}
	return ""
}
