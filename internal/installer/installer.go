// Package installer instala y desinstala OverdriveTuner en Windows.
package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fpsforge/internal/autostart"
	"fpsforge/internal/system"
	"fpsforge/internal/winhost"
)

const (
	AppName   = "OverdriveTuner"
	ExeName   = "OverdriveTuner.exe"
	Publisher = "OverdriveTuner"
	uninstKey = `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\OverdriveTuner`

	// Instalación anterior, cuando el programa se llamaba FPS Forge.
	legacyKey = `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\FPSForge`
	legacyApp = "FPS Forge"
)

// Options configura la instalación.
type Options struct {
	Dir       string
	Desktop   bool
	StartMenu bool
	Version   string
	Exe       []byte // contenido del ejecutable a instalar
	Icon      []byte // contenido del .ico (opcional)
}

// Progress informa el avance (0-100) y un mensaje.
type Progress func(pct int, msg string)

// DefaultDir devuelve la carpeta de instalación predeterminada.
func DefaultDir() string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	return filepath.Join(pf, AppName)
}

// ValidateDir rechaza carpetas peligrosas o no válidas.
func ValidateDir(dir string) error {
	d := strings.TrimSpace(dir)
	if d == "" {
		return fmt.Errorf("elige una carpeta de instalación")
	}
	if !filepath.IsAbs(d) {
		return fmt.Errorf("la carpeta debe ser una ruta completa, como C:\\Program Files\\OverdriveTuner")
	}
	c := filepath.Clean(d)
	if filepath.Dir(c) == c {
		return fmt.Errorf("no se puede instalar en la raíz de una unidad")
	}
	low := strings.ToLower(c)
	sd := os.Getenv("SystemDrive")
	if sd == "" {
		sd = "C:"
	}
	protected := []string{sd + `\Windows`, sd + `\Users`, sd + `\Program Files`, sd + `\Program Files (x86)`, sd + `\ProgramData`,
		os.Getenv("SystemRoot"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramData")}
	for _, bad := range protected {
		if bad == "" {
			continue
		}
		b := strings.ToLower(filepath.Clean(bad))
		if low == b {
			return fmt.Errorf("elige una subcarpeta propia, por ejemplo %s", DefaultDir())
		}
	}
	if strings.HasPrefix(low, strings.ToLower(sd)+`\windows\`) {
		return fmt.Errorf("no se puede instalar dentro de la carpeta de Windows")
	}
	return nil
}

func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func shortcutScript(folderKind, lnkName, target, dir, icon string) string {
	return `$d = [Environment]::GetFolderPath(` + psq(folderKind) + `)
if (-not (Test-Path -LiteralPath $d)) { New-Item -ItemType Directory -Force -Path $d | Out-Null }
$s = (New-Object -ComObject WScript.Shell).CreateShortcut((Join-Path $d ` + psq(lnkName) + `))
$s.TargetPath = ` + psq(target) + `
$s.WorkingDirectory = ` + psq(dir) + `
$s.Description = 'Optimizador de PC para gaming'
if (` + psq(icon) + ` -ne '') { $s.IconLocation = ` + psq(icon) + ` }
$s.Save()`
}

func dirSizeKB(b []byte) int { return len(b)/1024 + 1 }

// Install copia el programa, crea accesos directos y registra la desinstalación.
func Install(sys system.System, o Options, progress Progress) error {
	if progress == nil {
		progress = func(int, string) {}
	}
	if err := ValidateDir(o.Dir); err != nil {
		return err
	}
	if len(o.Exe) < 1024 {
		return fmt.Errorf("el instalador no contiene el programa (archivo dañado)")
	}
	progress(5, "Preparando carpeta de instalación…")
	if err := os.MkdirAll(o.Dir, 0o755); err != nil {
		return fmt.Errorf("no se pudo crear %s: %w", o.Dir, err)
	}
	exe := filepath.Join(o.Dir, ExeName)
	// Con la X el programa queda en la bandeja: se cierra antes de reemplazar el archivo.
	progress(12, "Cerrando OverdriveTuner si estaba abierto…")
	winhost.KillByName(ExeName)
	progress(20, "Copiando archivos…")
	// Escritura atómica: si OverdriveTuner está abierto, el renombrado falla con un mensaje claro.
	tmp := exe + ".nuevo"
	if err := os.WriteFile(tmp, o.Exe, 0o755); err != nil {
		return fmt.Errorf("no se pudo escribir el programa: %w", err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("no se pudo reemplazar %s. Cierra OverdriveTuner e inténtalo de nuevo (%v)", ExeName, err)
	}
	icon := ""
	if len(o.Icon) > 0 {
		icon = filepath.Join(o.Dir, "overdrivetuner.ico")
		if err := os.WriteFile(icon, o.Icon, 0o644); err != nil {
			icon = ""
		}
	}
	// Los accesos directos se crean en una sola llamada a PowerShell (cada llamada tarda más de un segundo).
	var lnks []string
	if o.StartMenu {
		lnks = append(lnks, shortcutScript("CommonPrograms", AppName+".lnk", exe, o.Dir, icon))
	}
	if o.Desktop {
		lnks = append(lnks, shortcutScript("CommonDesktopDirectory", AppName+".lnk", exe, o.Dir, icon))
	}
	if len(lnks) > 0 {
		progress(55, "Creando accesos directos…")
		if _, err := sys.PowerShell(strings.Join(lnks, "\n"), time.Minute); err != nil {
			return fmt.Errorf("no se pudieron crear los accesos directos: %w", err)
		}
	}
	progress(80, "Registrando en Aplicaciones instaladas…")
	disp := exe
	if icon != "" {
		disp = icon
	}
	reg := []struct{ name, typ, data string }{
		{"DisplayName", "REG_SZ", AppName},
		{"DisplayVersion", "REG_SZ", o.Version},
		{"Publisher", "REG_SZ", Publisher},
		{"InstallLocation", "REG_SZ", o.Dir},
		{"DisplayIcon", "REG_SZ", disp},
		{"UninstallString", "REG_SZ", `"` + exe + `" --uninstall`},
		{"NoModify", "REG_DWORD", "1"},
		{"NoRepair", "REG_DWORD", "1"},
		{"EstimatedSize", "REG_DWORD", fmt.Sprint(dirSizeKB(o.Exe))},
	}
	for _, r := range reg {
		if err := sys.RegSet(uninstKey, r.name, r.typ, r.data); err != nil {
			return fmt.Errorf("no se pudo registrar la desinstalación: %w", err)
		}
	}
	removeLegacy(sys, o.Dir)
	progress(100, "Instalación completa.")
	return nil
}

// LegacyInstalled dice si hay una instalación de la versión anterior (FPS Forge).
func LegacyInstalled(sys system.System) bool { return sys.RegKeyExists(legacyKey) }

// removeLegacy quita la instalación anterior (accesos, registro y carpeta).
// Los datos (copias de seguridad y estado) viven en ProgramData y no se tocan.
func removeLegacy(sys system.System, newDir string) {
	if !sys.RegKeyExists(legacyKey) {
		return
	}
	old := ""
	if v, err := system.RegGet(sys, legacyKey, "InstallLocation"); err == nil && v.Exists {
		old = v.Data
	}
	_, _ = sys.PowerShell(`foreach ($k in 'CommonPrograms','CommonDesktopDirectory') {
  Remove-Item -LiteralPath (Join-Path ([Environment]::GetFolderPath($k)) `+psq(legacyApp+".lnk")+`) -Force -ErrorAction SilentlyContinue
}`, time.Minute)
	_ = sys.RegDeleteKey(legacyKey)
	if old != "" && !SamePath(old, newDir) && ValidateDir(old) == nil {
		_ = ScheduleDelete(sys, old)
	}
}

// InstalledVersion devuelve la versión instalada («» si no se sabe).
func InstalledVersion(sys system.System) string {
	v, err := system.RegGet(sys, uninstKey, "DisplayVersion")
	if err != nil || !v.Exists {
		return ""
	}
	return v.Data
}

// InstalledDir devuelve la carpeta donde está instalado, si lo está.
func InstalledDir(sys system.System) string {
	v, err := system.RegGet(sys, uninstKey, "InstallLocation")
	if err != nil || !v.Exists {
		return ""
	}
	return v.Data
}

// Uninstall quita accesos directos y el registro, y programa el borrado de la
// carpeta (el ejecutable en uso no puede borrarse a sí mismo directamente).
func Uninstall(sys system.System, dir string) error {
	if err := ValidateDir(dir); err != nil {
		return err
	}
	script := `foreach ($k in 'CommonPrograms','CommonDesktopDirectory') {
  $p = Join-Path ([Environment]::GetFolderPath($k)) ` + psq(AppName+".lnk") + `
  Remove-Item -LiteralPath $p -Force -ErrorAction SilentlyContinue
}`
	if _, err := sys.PowerShell(script, time.Minute); err != nil {
		return fmt.Errorf("no se pudieron quitar los accesos directos: %w", err)
	}
	_ = autostart.Disable(sys) // la tarea de inicio con Windows también se quita
	if sys.RegKeyExists(uninstKey) {
		if err := sys.RegDeleteKey(uninstKey); err != nil {
			return err
		}
	}
	return nil
}

// ScheduleDelete borra los archivos de la instalación (y la carpeta si queda vacía) unos segundos después de salir:
// el ejecutable en uso no puede borrarse a sí mismo.
func ScheduleDelete(sys system.System, dir string) error {
	// Solo se borran los archivos que instala OverdriveTuner y, si la carpeta queda vacía, la carpeta.
	// Nunca se borra de forma recursiva: si el usuario instaló dentro de una carpeta que ya tenía
	// otras cosas (Documentos, un disco de juegos), esas cosas no se tocan.
	// También los restos de las actualizaciones automáticas: copias «.old», el candado y la carpeta «update» (con lo
	// que baja el actualizador, que nunca tiene subcarpetas).
	upd := "Join-Path " + psq(dir) + " 'update'"
	inner := "Start-Sleep -Seconds 3; foreach ($n in 'OverdriveTuner.exe','OverdriveTuner.exe.nuevo','overdrivetuner.ico','FPSForge.exe','fpsforge.ico','LEEME.txt','update.lock') { Remove-Item -LiteralPath (Join-Path " + psq(dir) + " $n) -Force -ErrorAction SilentlyContinue }; " +
		"Get-ChildItem -LiteralPath " + psq(dir) + " -Filter 'OverdriveTuner.exe.old*' -File -Force -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue; " +
		"foreach ($n in 'OverdriveTuner.exe','OverdriveTuner.part','meta.json') { Remove-Item -LiteralPath (Join-Path (" + upd + ") $n) -Force -ErrorAction SilentlyContinue }; " +
		"if (-not (Get-ChildItem -LiteralPath (" + upd + ") -Force -ErrorAction SilentlyContinue)) { Remove-Item -LiteralPath (" + upd + ") -Force -ErrorAction SilentlyContinue }; " +
		"if (-not (Get-ChildItem -LiteralPath " + psq(dir) + " -Force -ErrorAction SilentlyContinue)) { Remove-Item -LiteralPath " + psq(dir) + " -Force -ErrorAction SilentlyContinue }"
	script := "Start-Process -WindowStyle Hidden -FilePath powershell.exe -ArgumentList @('-NoProfile','-Command'," + psq(inner) + ")"
	_, err := sys.PowerShell(script, 30*time.Second)
	return err
}

// SamePath compara rutas sin distinguir mayúsculas ni barras finales.
func SamePath(a, b string) bool {
	return strings.EqualFold(strings.TrimRight(filepath.Clean(a), `\/`), strings.TrimRight(filepath.Clean(b), `\/`))
}

// ErrNotInstalled indica que no hay una instalación que actualizar.
var ErrNotInstalled = fmt.Errorf("OverdriveTuner no está instalado en este equipo. Ejecuta primero OverdriveTuner-Setup.exe")

// Patch actualiza el programa ya instalado en su carpeta. No toca los datos
// (copias de seguridad, estado, benchmarks) y solo recrea el acceso del
// escritorio si el usuario todavía lo tiene.
func Patch(sys system.System, exe, ico []byte, version string, progress Progress) (string, error) {
	dir := InstalledDir(sys)
	if dir == "" && LegacyInstalled(sys) {
		dir = DefaultDir() // viene de FPS Forge: se instala con el nombre nuevo
	}
	if dir == "" {
		return "", ErrNotInstalled
	}
	// Un parche no recrea los accesos que ya existen (cada uno cuesta una llamada a PowerShell): solo el del
	// menú Inicio si falta, y todos si viene de la versión anterior (FPS Forge), cuyos accesos apuntan a otro programa.
	legacy := LegacyInstalled(sys)
	exists := func(base, rel string) bool {
		if base == "" {
			return false
		}
		_, err := os.Stat(filepath.Join(base, rel, AppName+".lnk"))
		return err == nil
	}
	desk := legacy && exists(os.Getenv("PUBLIC"), "Desktop")
	start := legacy || !exists(filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu"), "Programs")
	err := Install(sys, Options{Dir: dir, Desktop: desk, StartMenu: start, Version: version, Exe: exe, Icon: ico}, progress)
	return dir, err
}
