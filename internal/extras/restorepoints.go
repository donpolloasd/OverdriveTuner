package extras

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// RestoreInfo son los puntos de restauración del disco del sistema y lo que ocupan.
type RestoreInfo struct {
	Points  []RestorePoint `json:"points"`
	Shadows int            `json:"shadows"` // instantáneas del disco del sistema (cada punto usa una)
	Used    int64          `json:"used"`    // bytes que ocupan
}

// RestorePoint es un punto de restauración de Windows.
type RestorePoint struct {
	Seq  int    `json:"seq"`
	Desc string `json:"desc"`
	At   string `json:"at"`
}

const restoreInfoScript = `$ErrorActionPreference='SilentlyContinue'
$vol = (Get-CimInstance Win32_Volume -Filter ("DriveLetter='" + $env:SystemDrive + "'")).DeviceID
$rp = @(Get-ComputerRestorePoint | ForEach-Object { [pscustomobject]@{ seq=[int]$_.SequenceNumber; desc=[string]$_.Description; at=([Management.ManagementDateTimeConverter]::ToDateTime($_.CreationTime)).ToString('yyyy-MM-dd HH:mm') } })
$sh = @(Get-CimInstance Win32_ShadowCopy | Where-Object { $_.VolumeName -eq $vol })
$used = 0; foreach ($s in @(Get-CimInstance Win32_ShadowStorage)) { if ($s.Volume.DeviceID -eq $vol) { $used = [int64]$s.UsedSpace } }
ConvertTo-Json -Compress -Depth 3 -InputObject @{ points = @($rp); shadows = $sh.Count; used = $used }`

// RestorePoints lista los puntos de restauración (necesita permisos de administrador).
func RestorePoints(sys system.System) (RestoreInfo, error) {
	out, err := sys.PowerShell(restoreInfoScript, 2*time.Minute)
	if err != nil {
		return RestoreInfo{}, err
	}
	var ri RestoreInfo
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &ri); err != nil {
		return RestoreInfo{}, fmt.Errorf("no se pudo leer la lista de puntos de restauración")
	}
	if ri.Points == nil {
		ri.Points = []RestorePoint{}
	}
	return ri, nil
}

const oldShadowsScript = `$ErrorActionPreference='SilentlyContinue'
$vol = (Get-CimInstance Win32_Volume -Filter ("DriveLetter='" + $env:SystemDrive + "'")).DeviceID
$sh = @(Get-CimInstance Win32_ShadowCopy | Where-Object { $_.VolumeName -eq $vol } | Sort-Object InstallDate)
if ($sh.Count -gt 1) { ($sh | Select-Object -First ($sh.Count - 1) | ForEach-Object { [string]$_.ID }) -join "\n" }`

var guidRe = regexp.MustCompile(`^\{[0-9A-Fa-f-]{36}\}$`)

// DeleteOldRestorePoints borra todos los puntos de restauración del disco del sistema menos el más
// reciente. Devuelve cuántos borró.
func DeleteOldRestorePoints(sys system.System) (int, error) {
	out, err := sys.PowerShell(oldShadowsScript, 2*time.Minute)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, id := range strings.Fields(strings.ReplaceAll(out, `\n`, " ")) {
		if !guidRe.MatchString(id) {
			continue
		}
		if _, code, err := sys.Exec(2*time.Minute, "vssadmin.exe", "delete", "shadows", "/shadow="+id, "/quiet"); err == nil && code == 0 {
			n++
		}
	}
	return n, nil
}
