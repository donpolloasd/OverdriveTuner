package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fpsforge/internal/system"
)

const wingetBase = "--accept-source-agreements"

// Códigos de salida de winget (HRESULT sin signo en Windows, por eso se normalizan a int32).
const (
	exitAlreadyInstalled int32 = -1978335135 // 0x8A150061: el paquete ya está instalado
	exitNoUpgrade        int32 = -1978335189 // 0x8A15002B: no hay actualización aplicable
)

func norm(code int) int32 { return int32(uint32(code)) }

// Available dice si winget responde.
func Available(sys system.System) bool {
	_, code, err := sys.Exec(20*time.Second, "winget.exe", "--version")
	return err == nil && code == 0
}

// InstallWinget intenta dejar winget disponible (registrar el Instalador de aplicaciones o bajarlo).
func InstallWinget(sys system.System) error {
	out, err := sys.PowerShell(`$ErrorActionPreference='Stop'
try { Add-AppxPackage -RegisterByFamilyName -MainPackage Microsoft.DesktopAppInstaller_8wekyb3d8bbwe } catch {
  $f = Join-Path $env:TEMP 'winget.msixbundle'
  Invoke-WebRequest -UseBasicParsing -Uri 'https://aka.ms/getwinget' -OutFile $f
  Add-AppxPackage -Path $f
}`, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("no se pudo instalar winget: %v %s", err, strings.TrimSpace(out))
	}
	if !Available(sys) {
		return fmt.Errorf("winget sigue sin responder: reiniciá sesión o instalalo desde Microsoft Store («Instalador de aplicación»)")
	}
	return nil
}

type exportFile struct {
	Sources []struct {
		Packages []struct {
			PackageIdentifier string `json:"PackageIdentifier"`
		} `json:"Packages"`
	} `json:"Sources"`
}

// Installed devuelve los identificadores de winget que ya están instalados (en minúsculas).
func Installed(sys system.System) (map[string]bool, error) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("ot-winget-%d.json", time.Now().UnixNano()))
	defer os.Remove(tmp)
	_, _, err := sys.Exec(2*time.Minute, "winget.exe", "export", "-o", tmp, wingetBase)
	b, rerr := os.ReadFile(tmp)
	if rerr != nil {
		if err != nil {
			return nil, err
		}
		return nil, rerr
	}
	return parseExport(b)
}

func parseExport(b []byte) (map[string]bool, error) {
	b = []byte(strings.TrimPrefix(string(b), "\ufeff"))
	var f exportFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, s := range f.Sources {
		for _, p := range s.Packages {
			out[strings.ToLower(p.PackageIdentifier)] = true
		}
	}
	return out, nil
}

// IsInstalled dice si todos los identificadores del programa están instalados.
func IsInstalled(it Item, have map[string]bool) bool {
	for _, id := range it.IDs {
		if !have[strings.ToLower(id)] {
			return false
		}
	}
	return len(it.IDs) > 0
}

// Result es el resultado de instalar un programa.
type Result struct {
	OK    bool     `json:"ok"`
	Notes []string `json:"notes,omitempty"`
}

// Install instala todos los identificadores del programa.
func Install(sys system.System, it Item) Result {
	var res Result
	res.OK = true
	for _, id := range it.IDs {
		out, code, err := sys.Exec(20*time.Minute, "winget.exe", "install", "--id", id, "--exact", "--silent",
			"--accept-package-agreements", wingetBase, "--disable-interactivity")
		// «Ya instalado» o «sin actualización aplicable» también cuentan como correcto.
		if c := norm(code); err != nil || (c != 0 && c != exitAlreadyInstalled && c != exitNoUpgrade) {
			res.OK = false
			msg := lastLine(out)
			if err != nil {
				msg = err.Error()
			}
			res.Notes = append(res.Notes, fmt.Sprintf("%s: no se pudo instalar (código %d) %s", id, code, msg))
		}
	}
	return res
}

// UpgradeAll actualiza todos los programas que winget puede actualizar.
func UpgradeAll(sys system.System) (string, error) {
	out, code, err := sys.Exec(90*time.Minute, "winget.exe", "upgrade", "--all", "--silent",
		"--accept-package-agreements", wingetBase, "--disable-interactivity")
	if err != nil {
		return out, err
	}
	if c := norm(code); c != 0 && c != exitNoUpgrade { // nada que actualizar no es un error
		return out, fmt.Errorf("winget terminó con código %d", code)
	}
	return out, nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
