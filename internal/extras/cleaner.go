// Package extras contiene utilidades además de los tweaks: limpieza de
// archivos temporales, gestor de inicio y puntos de restauración.
package extras

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Target es una ubicación limpiable.
type Target struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Desc  string   `json:"desc"`
	Paths []string `json:"-"`
	Safe  bool     `json:"safe"` // incluido por defecto
	Deep  bool     `json:"deep"` // limpieza profunda: caché de navegadores y apps
	// Confirm: si no está vacío, borrarlo pide confirmación explícita del usuario (este texto la explica).
	Confirm string `json:"confirm,omitempty"`
}

// NeedsConfirm devuelve los objetivos pedidos que exigen confirmación del usuario antes de borrarse.
func NeedsConfirm(ids []string) []Target {
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []Target
	for _, t := range Targets() {
		if want[t.ID] && t.Confirm != "" {
			out = append(out, t)
		}
	}
	return out
}

// CleanInfo es el tamaño estimado de un objetivo.
type CleanInfo struct {
	Target
	Bytes int64 `json:"bytes"`
	Files int   `json:"files"`
}

// CleanResult es el resultado de limpiar un objetivo.
type CleanResult struct {
	ID      string `json:"id"`
	Freed   int64  `json:"freed"`
	Removed int    `json:"removed"`
	Skipped int    `json:"skipped"`
}

func env(k string) string { return os.Getenv(k) }

func join(base string, parts ...string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(append([]string{base}, parts...)...)
}

func clean(paths ...string) []string {
	var out []string
	for _, p := range paths {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Targets devuelve los objetivos de limpieza para el equipo actual.
func Targets() []Target {
	if runtime.GOOS != "windows" {
		return nil // la limpieza real solo existe en Windows
	}
	local := env("LOCALAPPDATA")
	win := env("SystemRoot")
	if win == "" {
		win = `C:\Windows`
	}
	base := []Target{
		{ID: "usertemp", Name: "Temporales del usuario", Desc: "Archivos en %TEMP%", Safe: true,
			Paths: clean(os.TempDir())},
		{ID: "wintemp", Name: "Temporales de Windows", Desc: "C:\\Windows\\Temp", Safe: true,
			Paths: clean(join(win, "Temp"))},
		// Los shaders no van marcados por defecto ni en la limpieza semanal: borrarlos sin querer provoca tirones.
		{ID: "shaders", Name: "Caché de shaders (DirectX, NVIDIA, AMD)", Desc: "Se regenera al abrir los juegos; las primeras partidas después pueden tener tirones. Pide confirmación.",
			Confirm: "Vas a borrar la caché de shaders de DirectX, NVIDIA y AMD.\n\nLos juegos la vuelven a crear solos, pero las primeras partidas después pueden tardar más en cargar y tener tirones (microcortes) mientras se compilan de nuevo. Conviene hacerlo solo si un juego tiene errores gráficos o después de cambiar el driver de la placa de video, y no justo antes de una partida competitiva.\n\nLo hacés bajo tu responsabilidad: los archivos borrados no se pueden recuperar.",
			Paths: clean(join(local, "D3DSCache"), join(local, "NVIDIA", "DXCache"), join(local, "NVIDIA", "GLCache"),
				join(local, "AMD", "DxCache"), join(local, "AMD", "GLCache"), join(local, "AMD", "VkCache"))},
		{ID: "crash", Name: "Volcados y reportes de errores", Desc: "Minidumps y reportes de WER", Safe: true,
			Paths: clean(join(win, "Minidump"), join(local, "CrashDumps"), join(env("ProgramData"), "Microsoft", "Windows", "WER", "ReportQueue"),
				join(env("ProgramData"), "Microsoft", "Windows", "WER", "ReportArchive"))},
		{ID: "thumbs", Name: "Miniaturas e iconos en caché", Desc: "Se vuelven a generar solos", Safe: true,
			Paths: nil},
		{ID: "wu", Name: "Descargas de Windows Update", Desc: "SoftwareDistribution\\Download", Safe: false,
			Paths: clean(join(win, "SoftwareDistribution", "Download"))},
		{ID: "logs", Name: "Registros de Windows", Desc: "Logs antiguos de CBS y Panther", Safe: false,
			Paths: clean(join(win, "Logs", "CBS"), join(win, "Panther"))},
	}
	return append(base, deepTargets(local, env("APPDATA"), win)...)
}

// deepTargets son cachés de navegadores y apps: se regeneran solas, pero
// pueden hacer que la primera carga sea más lenta o cerrar sesiones de páginas.
func deepTargets(local, roaming, win string) []Target {
	browser := func(root string) []string {
		return clean(join(root, "Default", "Cache"), join(root, "Default", "Code Cache"), join(root, "Default", "GPUCache"),
			join(root, "ShaderCache"), join(root, "GrShaderCache"))
	}
	return []Target{
		{ID: "edgecache", Name: "Caché de Microsoft Edge", Desc: "Páginas e imágenes guardadas. No borra contraseñas, cookies ni historial.", Deep: true,
			Paths: browser(join(local, "Microsoft", "Edge", "User Data"))},
		{ID: "chromecache", Name: "Caché de Google Chrome", Desc: "Páginas e imágenes guardadas. No borra contraseñas, cookies ni historial.", Deep: true,
			Paths: browser(join(local, "Google", "Chrome", "User Data"))},
		{ID: "bravecache", Name: "Caché de Brave", Desc: "Páginas e imágenes guardadas. No borra contraseñas, cookies ni historial.", Deep: true,
			Paths: browser(join(local, "BraveSoftware", "Brave-Browser", "User Data"))},
		{ID: "discordcache", Name: "Caché de Discord", Desc: "Imágenes y archivos temporales. No cierra tu sesión.", Deep: true,
			Paths: clean(join(roaming, "discord", "Cache"), join(roaming, "discord", "Code Cache"), join(roaming, "discord", "GPUCache"))},
		{ID: "steamweb", Name: "Caché web de Steam", Desc: "Tienda y comunidad dentro de Steam. Se vuelve a descargar.", Deep: true,
			Paths: clean(join(local, "Steam", "htmlcache"))},
		{ID: "spotifycache", Name: "Caché de Spotify", Desc: "Canciones guardadas para escuchar sin conexión rápida. Se vuelve a llenar.", Deep: true,
			Paths: clean(join(local, "Spotify", "Data"), join(local, "Spotify", "Storage"))},
		{ID: "delivopt", Name: "Caché de Delivery Optimization", Desc: "Archivos de actualizaciones compartidas entre equipos.", Deep: true,
			Paths: clean(join(win, "ServiceProfiles", "NetworkService", "AppData", "Local", "Microsoft", "Windows", "DeliveryOptimization", "Cache"))},
	}
}

// dirSize recorre un directorio sumando tamaño y cantidad de archivos.
func dirSize(root string) (int64, int) {
	var total int64
	var n int
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
			n++
		}
		return nil
	})
	return total, n
}

// Scan calcula cuánto se puede liberar en cada objetivo.
func Scan() []CleanInfo {
	out := []CleanInfo{}
	for _, t := range Targets() {
		ci := CleanInfo{Target: t}
		for _, p := range t.Paths {
			b, n := dirSize(p)
			ci.Bytes += b
			ci.Files += n
		}
		if t.ID == "thumbs" {
			b, n := thumbsSize()
			ci.Bytes += b
			ci.Files += n
		}
		out = append(out, ci)
	}
	return out
}

func thumbCaches() []string {
	dir := join(env("LOCALAPPDATA"), "Microsoft", "Windows", "Explorer")
	m, _ := filepath.Glob(filepath.Join(dir, "thumbcache_*.db"))
	i, _ := filepath.Glob(filepath.Join(dir, "iconcache_*.db"))
	return append(m, i...)
}

func thumbsSize() (int64, int) {
	var t int64
	var n int
	for _, f := range thumbCaches() {
		if st, err := os.Stat(f); err == nil {
			t += st.Size()
			n++
		}
	}
	return t, n
}

// safeRoot impide borrar rutas peligrosas aunque una variable de entorno sea rara.
func safeRoot(p string) bool {
	c := filepath.Clean(p)
	if len(c) < 8 || c == filepath.VolumeName(c)+string(filepath.Separator) {
		return false
	}
	low := strings.ToLower(c)
	for _, bad := range []string{`c:\windows`, `c:\users`, `c:\program files`, `c:\programdata`} {
		if low == bad {
			return false
		}
	}
	return true
}

// removeContents borra el contenido de un directorio sin borrar el directorio.
// Lo que está en uso se omite sin error.
func removeContents(root string) (freed int64, removed, skipped int) {
	if !safeRoot(root) {
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		p := filepath.Join(root, e.Name())
		size, n := int64(0), 0
		if e.IsDir() {
			size, n = dirSize(p)
		} else if info, err := e.Info(); err == nil {
			size, n = info.Size(), 1
		}
		if err := os.RemoveAll(p); err != nil {
			// Puede haberse borrado parcialmente: se recalcula lo que queda.
			left, ln := dirSize(p)
			freed += size - left
			removed += n - ln
			skipped++
			continue
		}
		freed += size
		removed += n
	}
	return
}

// Run limpia los objetivos indicados.
func Run(ids []string) []CleanResult {
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []CleanResult
	for _, t := range Targets() {
		if !want[t.ID] {
			continue
		}
		r := CleanResult{ID: t.ID}
		for _, p := range t.Paths {
			f, n, s := removeContents(p)
			r.Freed += f
			r.Removed += n
			r.Skipped += s
		}
		if t.ID == "thumbs" {
			for _, f := range thumbCaches() {
				if st, err := os.Stat(f); err == nil {
					if os.Remove(f) == nil {
						r.Freed += st.Size()
						r.Removed++
					} else {
						r.Skipped++
					}
				}
			}
		}
		out = append(out, r)
	}
	return out
}
