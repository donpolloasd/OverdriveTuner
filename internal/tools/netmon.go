package tools

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// NetProc es la actividad de red de un proceso.
type NetProc struct {
	PID    int      `json:"pid"`
	Name   string   `json:"name"`
	TCP    int      `json:"tcp"`
	UDP    int      `json:"udp"`
	KBs    float64  `json:"kbs"` // E/S «otra» por segundo: incluye la red, es una estimación
	Remote []string `json:"remote"`
}

const netMonScript = `$ErrorActionPreference='SilentlyContinue'
$tcp = @(Get-NetTCPConnection -State Established)
$udp = @(Get-NetUDPEndpoint)
$io = @{}
foreach ($p in @(Get-CimInstance Win32_PerfFormattedData_PerfProc_Process)) { $io[[int]$p.IDProcess] = [double]$p.IOOtherBytesPersec }
$ids = @($tcp.OwningProcess) + @($udp.OwningProcess) | Where-Object { $_ -gt 4 } | Sort-Object -Unique
$res = foreach ($id in $ids) {
  $pr = Get-Process -Id $id
  if (-not $pr) { continue }
  $t = @($tcp | Where-Object { $_.OwningProcess -eq $id })
  $r = @($t.RemoteAddress | Where-Object { $_ -and $_ -notmatch '^(127\.|::1|0\.0\.0\.0|fe80)' } | Sort-Object -Unique | Select-Object -First 3)
  [pscustomobject]@{ pid=[int]$id; name=[string]$pr.ProcessName; tcp=$t.Count; udp=@($udp | Where-Object { $_.OwningProcess -eq $id }).Count; kbs=[math]::Round($io[[int]$id]/1KB,1); remote=@($r) }
}
ConvertTo-Json -InputObject @($res) -Depth 3 -Compress`

// NetMon lista los procesos con conexiones de red, de mayor a menor actividad.
func NetMon(sys system.System) ([]NetProc, error) {
	out, err := sys.PowerShell(netMonScript, 30*time.Second)
	if err != nil {
		return nil, err
	}
	return ParseNetMon(out)
}

// ParseNetMon interpreta la salida del script.
func ParseNetMon(out string) ([]NetProc, error) {
	out = strings.TrimSpace(out)
	if out == "" || out == "null" {
		return []NetProc{}, nil
	}
	var ps []NetProc
	if strings.HasPrefix(out, "{") {
		var p NetProc
		if err := json.Unmarshal([]byte(out), &p); err != nil {
			return nil, err
		}
		ps = []NetProc{p}
	} else if err := json.Unmarshal([]byte(out), &ps); err != nil {
		return nil, err
	}
	res := make([]NetProc, 0, len(ps))
	for _, p := range ps {
		if p.Name == "" {
			continue
		}
		if p.Remote == nil {
			p.Remote = []string{}
		}
		res = append(res, p)
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].KBs != res[j].KBs {
			return res[i].KBs > res[j].KBs
		}
		return res[i].TCP+res[i].UDP > res[j].TCP+res[j].UDP
	})
	if len(res) > 20 {
		res = res[:20]
	}
	return res, nil
}
