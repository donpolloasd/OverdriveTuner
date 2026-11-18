// Package oldapps encuentra programas y juegos que no se usan hace tiempo, con lo que ocupan, para que el
// usuario decida si los desinstala. No desinstala nada solo: abre el desinstalador oficial de cada uno.
package oldapps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// Item es un programa o un juego instalado.
type Item struct {
	ID        string `json:"id"`   // «reg:<clave>» o «steam:<appid>»
	Kind      string `json:"kind"` // program, game
	Name      string `json:"name"`
	Publisher string `json:"publisher,omitempty"`
	SizeMB    int    `json:"sizeMB"`
	LastUsed  string `json:"lastUsed,omitempty"` // yyyy-mm-dd; vacío si Windows no lo registró
	Days      int    `json:"days"`               // días sin usarlo; -1 si no se sabe
}

// programsScript lista los programas instalados (las mismas claves que «Aplicaciones instaladas») y, para
// cada uno, la última vez que se abrió alguno de sus .exe según la carpeta Prefetch de Windows.
const programsScript = `$ErrorActionPreference='SilentlyContinue'
$pf=@{}
foreach($f in Get-ChildItem "$env:WINDIR\Prefetch\*.pf"){ $n=($f.Name -replace '-[0-9A-F]{8}\.pf$','').ToLower(); if(-not $pf[$n] -or $pf[$n] -lt $f.LastWriteTime){ $pf[$n]=$f.LastWriteTime } }
$roots=@('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall','HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall','HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
$out=foreach($r in $roots){ foreach($k in Get-ChildItem $r){
  $p=Get-ItemProperty $k.PSPath
  if(-not $p.DisplayName -or -not $p.UninstallString -or $p.SystemComponent -eq 1 -or $p.ParentKeyName -or $p.ReleaseType -match 'Update|Hotfix'){ continue }
  $last=$null; $exes=@()
  if($p.InstallLocation -and (Test-Path -LiteralPath $p.InstallLocation)){ $exes=@(Get-ChildItem -LiteralPath $p.InstallLocation -Filter *.exe -Recurse -Depth 2 | Select-Object -First 80 -ExpandProperty Name) }
  if($p.DisplayIcon){ $exes+=[IO.Path]::GetFileName(($p.DisplayIcon -split ',')[0].Trim('"')) }
  foreach($e in $exes){ $t=$pf[$e.ToLower()]; if($t -and (-not $last -or $t -gt $last)){ $last=$t } }
  $mb=0; if($p.EstimatedSize){ $mb=[int]([double]$p.EstimatedSize/1024) }
  [pscustomobject]@{ key=($k.Name -replace '^HKEY_LOCAL_MACHINE','HKLM' -replace '^HKEY_CURRENT_USER','HKCU'); name=[string]$p.DisplayName; pub=[string]$p.Publisher; mb=$mb; last=$(if($last){$last.ToString('yyyy-MM-dd')}else{''}) }
}}
ConvertTo-Json -Compress -InputObject @($out)`

// keyRe valida las claves de desinstalación que se aceptan (las de «Aplicaciones instaladas»).
var keyRe = regexp.MustCompile(`(?i)^(HKLM|HKCU)\\SOFTWARE\\(WOW6432Node\\)?Microsoft\\Windows\\CurrentVersion\\Uninstall\\[^\\]+$`)

// List devuelve programas y juegos de Steam, de los menos usados a los más usados.
func List(sys system.System, now time.Time) ([]Item, error) {
	out, err := sys.PowerShell(programsScript, 2*time.Minute)
	if err != nil {
		return nil, err
	}
	items, err := parsePrograms(out, now)
	if err != nil {
		return nil, err
	}
	if v, err := system.RegGet(sys, `HKCU\Software\Valve\Steam`, "SteamPath"); err == nil && v.Exists {
		items = append(items, SteamGames(filepath.FromSlash(v.Data), now)...)
	}
	Sort(items)
	return items, nil
}

func parsePrograms(out string, now time.Time) ([]Item, error) {
	var raw []struct {
		Key, Name, Pub, Last string
		MB                   int
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("no se pudo leer la lista de programas: %w", err)
	}
	seen := map[string]bool{}
	var items []Item
	for _, r := range raw {
		if !keyRe.MatchString(r.Key) || seen[strings.ToLower(r.Name)] {
			continue
		}
		seen[strings.ToLower(r.Name)] = true
		items = append(items, Item{ID: "reg:" + r.Key, Kind: "program", Name: r.Name, Publisher: r.Pub, SizeMB: r.MB,
			LastUsed: r.Last, Days: daysSince(r.Last, now)})
	}
	return items, nil
}

func daysSince(date string, now time.Time) int {
	t, err := time.ParseInLocation("2006-01-02", date, now.Location())
	if err != nil {
		return -1
	}
	return int(now.Sub(t).Hours() / 24)
}

var vdfRe = regexp.MustCompile(`"(\w+)"\s+"((?:[^"\\]|\\.)*)"`)

// SteamGames lee los juegos instalados de todas las bibliotecas de Steam (appmanifest_*.acf), con su
// tamaño y la última vez que se jugaron.
func SteamGames(steamDir string, now time.Time) []Item {
	libs := []string{steamDir}
	if b, err := os.ReadFile(filepath.Join(steamDir, "steamapps", "libraryfolders.vdf")); err == nil {
		for _, m := range vdfRe.FindAllStringSubmatch(string(b), -1) {
			if m[1] == "path" {
				libs = append(libs, strings.ReplaceAll(m[2], `\\`, `\`))
			}
		}
	}
	seen := map[string]bool{}
	var out []Item
	for _, lib := range libs {
		files, _ := filepath.Glob(filepath.Join(lib, "steamapps", "appmanifest_*.acf"))
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			kv := map[string]string{}
			for _, m := range vdfRe.FindAllStringSubmatch(string(b), -1) {
				if _, dup := kv[m[1]]; !dup {
					kv[m[1]] = m[2]
				}
			}
			id := kv["appid"]
			if _, err := strconv.Atoi(id); err != nil || seen[id] || id == "228980" { // 228980: Steamworks Common Redistributables
				continue
			}
			seen[id] = true
			it := Item{ID: "steam:" + id, Kind: "game", Name: kv["name"], Publisher: "Steam", Days: -1}
			if sz, err := strconv.ParseInt(kv["SizeOnDisk"], 10, 64); err == nil {
				it.SizeMB = int(sz / (1 << 20))
			}
			if ts, err := strconv.ParseInt(kv["LastPlayed"], 10, 64); err == nil && ts > 0 {
				t := time.Unix(ts, 0).In(now.Location())
				it.LastUsed, it.Days = t.Format("2006-01-02"), int(now.Sub(t).Hours()/24)
			}
			out = append(out, it)
		}
	}
	return out
}

// Sort ordena primero lo que hace más tiempo que no se usa (lo desconocido al final) y, a igualdad, lo
// que más ocupa.
func Sort(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.Days < 0) != (b.Days < 0) {
			return a.Days >= 0
		}
		if a.Days != b.Days {
			return a.Days > b.Days
		}
		return a.SizeMB > b.SizeMB
	})
}

// Uninstall abre el desinstalador oficial del programa o del juego. El usuario sigue los pasos ahí.
func Uninstall(sys system.System, id string) error {
	switch {
	case strings.HasPrefix(id, "steam:"):
		app := strings.TrimPrefix(id, "steam:")
		if _, err := strconv.Atoi(app); err != nil {
			return fmt.Errorf("juego desconocido")
		}
		_, _, err := sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", "steam://uninstall/"+app)
		return err
	case strings.HasPrefix(id, "reg:"):
		key := strings.TrimPrefix(id, "reg:")
		if !keyRe.MatchString(key) {
			return fmt.Errorf("programa desconocido")
		}
		ps := "Registry::" + strings.Replace(strings.Replace(key, "HKLM", "HKEY_LOCAL_MACHINE", 1), "HKCU", "HKEY_CURRENT_USER", 1)
		script := `$ErrorActionPreference='Stop'
$u=(Get-ItemProperty -LiteralPath '` + strings.ReplaceAll(ps, "'", "''") + `').UninstallString
if(-not $u){ throw 'Este programa no tiene desinstalador registrado.' }
if($u -match '^\s*"([^"]+)"\s*(.*)$'){ $exe=$matches[1]; $a=$matches[2] }
elseif($u -match '(?i)^\s*msiexec(\.exe)?\s+(.*)$'){ $exe='msiexec.exe'; $a=($matches[2] -replace '(?i)/I\s*\{','/X{') }
elseif($u -match '(?i)^(.+?\.exe)\s*(.*)$'){ $exe=$matches[1]; $a=$matches[2] }
else { $exe=$u; $a='' }
if($a){ Start-Process -FilePath $exe -ArgumentList $a } else { Start-Process -FilePath $exe }`
		_, err := sys.PowerShell(script, time.Minute)
		return err
	}
	return fmt.Errorf("programa desconocido")
}

// Demo son datos de ejemplo para el modo demostración.
func Demo() []Item {
	return []Item{
		{ID: "steam:550", Kind: "game", Name: "Left 4 Dead 2", Publisher: "Steam", SizeMB: 13900, LastUsed: "2025-03-02", Days: 625},
		{ID: "reg:HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Demo1", Kind: "program", Name: "Editor de video de prueba", Publisher: "Demo", SizeMB: 2100, LastUsed: "2025-11-20", Days: 362},
		{ID: "steam:1172470", Kind: "game", Name: "Apex Legends", Publisher: "Steam", SizeMB: 75300, LastUsed: "2026-05-14", Days: 187},
		{ID: "reg:HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Demo2", Kind: "program", Name: "Herramienta vieja", Publisher: "Demo", SizeMB: 340, Days: -1},
	}
}
