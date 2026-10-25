// Package procs lista los procesos que más CPU usan y permite cerrar los
// que no son del sistema.
package procs

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// Proc es un proceso con su uso de CPU medido en 2 segundos.
type Proc struct {
	PID  int     `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"` // % de un núcleo
	MB   int     `json:"mb"`
	Safe bool    `json:"safe"` // se puede cerrar desde el programa
}

const script = `$ErrorActionPreference='SilentlyContinue'
$a=@{}; Get-Process | ForEach-Object { $a[$_.Id]=$_.CPU }
Start-Sleep -Seconds 2
$o=@(); Get-Process | ForEach-Object { if($a.ContainsKey($_.Id) -and $_.CPU -ne $null -and $a[$_.Id] -ne $null){ $d=($_.CPU-$a[$_.Id])/2*100; if($d -gt 0.5){ $o+=[pscustomobject]@{pid=$_.Id;name=$_.ProcessName;cpu=[math]::Round($d,1);mb=[int]($_.WorkingSet64/1MB);path=$_.Path} } } }
$o | Sort-Object cpu -Descending | Select-Object -First 25 | ConvertTo-Json -Compress`

var protected = map[string]bool{
	"system": true, "idle": true, "registry": true, "smss": true, "csrss": true, "wininit": true,
	"winlogon": true, "services": true, "lsass": true, "svchost": true, "dwm": true, "explorer": true,
	"fontdrvhost": true, "sihost": true, "taskhostw": true, "ctfmon": true, "audiodg": true,
	"searchindexer": true, "runtimebroker": true, "msmpeng": true, "memory compression": true,
	"fpsforge": true, "fpsforge-setup": true, "overdrivetuner": true, "overdrivetuner-setup": true, "cs2": true, "dllhost": true, "wudfhost": true,
	"spoolsv": true, "lsaiso": true, "securityhealthservice": true, "startmenuexperiencehost": true,
	"shellexperiencehost": true, "textinputhost": true, "conhost": true, "applicationframehost": true,
}

// Protected dice si un proceso no debe cerrarse (nombre sin .exe).
func Protected(name string) bool {
	return protected[strings.ToLower(strings.TrimSuffix(name, ".exe"))]
}

func underWindows(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "/", `\`))
	win := strings.ToLower(os.Getenv("SystemRoot"))
	if win == "" {
		win = `c:\windows`
	}
	return strings.HasPrefix(p, win+`\`)
}

// Top mide el uso de CPU durante 2 segundos y devuelve los 15 mayores.
func Top(sys system.System) ([]Proc, error) {
	out, err := sys.PowerShell(script, 40*time.Second)
	if err != nil {
		return nil, err
	}
	return parse(out)
}

func parse(out string) ([]Proc, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return []Proc{}, nil
	}
	type raw struct {
		PID  int     `json:"pid"`
		Name string  `json:"name"`
		CPU  float64 `json:"cpu"`
		MB   int     `json:"mb"`
		Path string  `json:"path"`
	}
	var rs []raw
	if strings.HasPrefix(out, "{") {
		var r raw
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			return nil, err
		}
		rs = []raw{r}
	} else if err := json.Unmarshal([]byte(out), &rs); err != nil {
		return nil, err
	}
	var ps []Proc
	for _, r := range rs {
		if r.PID == os.Getpid() {
			continue
		}
		ps = append(ps, Proc{PID: r.PID, Name: r.Name, CPU: r.CPU, MB: r.MB,
			Safe: !Protected(r.Name) && !(r.Path == "" || underWindows(r.Path))})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].CPU > ps[j].CPU })
	if len(ps) > 15 {
		ps = ps[:15]
	}
	if ps == nil {
		ps = []Proc{}
	}
	return ps, nil
}

// Kill cierra un proceso solo si sigue siendo seguro cerrarlo y se llama como
// se esperaba (evita matar otro proceso que reutilizó el PID).
func Kill(sys system.System, pid int, name string) error {
	if pid <= 4 || pid == os.Getpid() {
		return fmt.Errorf("proceso no permitido")
	}
	if name == "" || Protected(name) {
		return fmt.Errorf("«%s» es un proceso del sistema o del juego y no se cierra desde acá", name)
	}
	chk := fmt.Sprintf(`$p=Get-Process -Id %d -ErrorAction SilentlyContinue
if(-not $p){ 'GONE'; exit 0 }
if($p.ProcessName -ne '%s'){ 'DIFF'; exit 0 }
if(-not $p.Path -or $p.Path.ToLower().StartsWith($env:SystemRoot.ToLower()+'\')){ 'SYS'; exit 0 }
Stop-Process -Id %d -Force; 'OK'`, pid, strings.ReplaceAll(name, "'", "''"), pid)
	out, err := sys.PowerShell(chk, 20*time.Second)
	if err != nil {
		return err
	}
	switch strings.TrimSpace(out) {
	case "OK":
		return nil
	case "GONE":
		return fmt.Errorf("el proceso ya no existe")
	case "DIFF":
		return fmt.Errorf("el PID cambió de programa; actualizá la lista")
	default:
		return fmt.Errorf("es un proceso de Windows y no se cierra desde acá")
	}
}

// KillByName cierra todos los procesos con ese nombre, con las mismas
// protecciones que Kill (nunca procesos de Windows ni el propio juego).
func KillByName(sys system.System, name string) error {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".exe")
	if name == "" || Protected(name) || strings.ContainsAny(name, `\/:*?"<>|'`) {
		return fmt.Errorf("«%s» no se puede cerrar desde acá", name)
	}
	_, err := sys.PowerShell(fmt.Sprintf(`$w=$env:SystemRoot.ToLower()+'\'
Get-Process -Name '%s' -ErrorAction SilentlyContinue | Where-Object { $_.Path -and -not $_.Path.ToLower().StartsWith($w) } | Stop-Process -Force -ErrorAction SilentlyContinue`, name), 20*time.Second)
	return err
}
