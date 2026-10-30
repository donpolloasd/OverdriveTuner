package diag

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fpsforge/internal/scan"
	"fpsforge/internal/system"
)

const healthScript = `$ErrorActionPreference='SilentlyContinue'
$r=[ordered]@{}
$smi = $null
foreach ($c in @('nvidia-smi.exe', (Join-Path $env:SystemRoot 'System32\nvidia-smi.exe'), (Join-Path $env:ProgramFiles 'NVIDIA Corporation\NVSMI\nvidia-smi.exe'))) {
  $g = Get-Command $c -ErrorAction SilentlyContinue
  if ($g) { $smi = $g.Source; break }
}
if ($smi) {
  $t = (& $smi -q -d MEMORY 2>$null) -join "` + "`n" + `"
  if ($t -match '(?s)BAR1 Memory Usage.*?Total\s*:\s*(\d+)\s*MiB') { $r.bar1MiB = [int]$Matches[1] }
}
$gp = @()
foreach ($g in @(Get-CimInstance Win32_VideoController)) {
  if ([string]$g.Name -match 'Basic|Remote|Virtual|Mirror|Meta') { continue }
  $d = ''; if ($g.DriverDate) { $d = ([datetime]$g.DriverDate).ToString('yyyy-MM-dd') }
  $gp += [ordered]@{ name=[string]$g.Name; version=[string]$g.DriverVersion; date=$d }
}
$r.gpus = @($gp)
$dk = @()
foreach ($d in @(Get-PhysicalDisk)) {
  $rc = $d | Get-StorageReliabilityCounter
  $w = $null; $tp = $null
  if ($rc) { if ($null -ne $rc.Wear) { $w = [int]$rc.Wear }; if ($null -ne $rc.Temperature) { $tp = [int]$rc.Temperature } }
  $dk += [ordered]@{ name=([string]$d.FriendlyName).Trim(); health=[string]$d.HealthStatus; wear=$w; temp=$tp }
}
$r.disks = @($dk)
$sd = Get-PSDrive -Name ($env:SystemDrive.Substring(0,1))
if ($sd -and ($sd.Used + $sd.Free) -gt 0) { $r.sysFreeGB = [math]::Round($sd.Free/1GB,1); $r.sysFreePct = [int][math]::Round(100*$sd.Free/($sd.Used+$sd.Free)) }
ConvertTo-Json -InputObject $r -Compress -Depth 4`

type healthData struct {
	Bar1MiB *int `json:"bar1MiB"`
	GPUs    []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Date    string `json:"date"`
	} `json:"gpus"`
	Disks []struct {
		Name   string `json:"name"`
		Health string `json:"health"`
		Wear   *int   `json:"wear"`
		Temp   *int   `json:"temp"`
	} `json:"disks"`
	SysFreeGB  float64 `json:"sysFreeGB"`
	SysFreePct *int    `json:"sysFreePct"`
}

func vendorOf(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "nvidia") || strings.Contains(n, "geforce") || strings.Contains(n, "rtx") || strings.Contains(n, "gtx"):
		return "nvidia"
	case strings.Contains(n, "amd") || strings.Contains(n, "radeon"):
		return "amd"
	case strings.Contains(n, "intel"):
		return "intel"
	}
	return ""
}

// DriverURLs son las páginas oficiales de drivers (fijas: la interfaz solo puede pedir abrir una de estas).
var DriverURLs = map[string]string{
	"nvidia": "https://www.nvidia.com/Download/index.aspx",
	"amd":    "https://www.amd.com/en/support/download/drivers.html",
	"intel":  "https://www.intel.com/content/www/us/en/download-center/home.html",
}

// OpenDriverPage abre la página oficial de drivers de un fabricante.
func OpenDriverPage(sys system.System, vendor string) error {
	u, ok := DriverURLs[vendor]
	if !ok {
		return fmt.Errorf("fabricante desconocido")
	}
	_, _, err := sys.Exec(15*time.Second, "rundll32.exe", "url.dll,FileProtocolHandler", u)
	return err
}

// monthsSince devuelve los meses completos desde una fecha yyyy-MM-dd (-1 si no se entiende).
func monthsSince(date string, now time.Time) int {
	t, err := time.Parse("2006-01-02", date)
	if err != nil || t.Year() < 2000 {
		return -1
	}
	m := (now.Year()-t.Year())*12 + int(now.Month()) - int(t.Month())
	if now.Day() < t.Day() {
		m--
	}
	if m < 0 {
		m = 0
	}
	return m
}

// ParseHealth convierte la salida del script en chequeos.
func ParseHealth(raw string, hw scan.Hardware, now time.Time) []Check {
	var h healthData
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &h) != nil {
		return nil
	}
	var out []Check

	// Resizable BAR
	switch {
	case h.Bar1MiB != nil && *h.Bar1MiB >= 1024:
		out = append(out, Check{ID: "rebar", Title: "Resizable BAR", Status: OK,
			Detail: fmt.Sprintf("Activo: la GPU puede acceder a toda su memoria de video a la vez (ventana de %d MiB).", *h.Bar1MiB)})
	case h.Bar1MiB != nil:
		out = append(out, Check{ID: "rebar", Title: "Resizable BAR", Status: Warn,
			Detail: "Está desactivado (la GPU accede a su memoria de a 256 MiB). Activalo en la BIOS: «Above 4G Decoding» y «Resizable BAR» (en AMD a veces se llama «Smart Access Memory»). Suele dar unos FPS extra en varios juegos. Antes actualizá la BIOS y el driver de la GPU."})
	default:
		for _, g := range hw.GPUs {
			if g.Vendor == "amd" && !g.Integrated() {
				out = append(out, Check{ID: "rebar", Title: "Resizable BAR (Smart Access Memory)", Status: Info,
					Detail: "Windows no permite saber si está activo en una Radeon. Se ve en el software AMD Adrenalin (Rendimiento > Ajuste) y se activa en la BIOS con «Above 4G Decoding» y «Resizable BAR»."})
				break
			}
		}
	}

	// Driver de la gráfica
	for _, g := range h.GPUs {
		v := vendorOf(g.Name)
		months := monthsSince(g.Date, now)
		if months < 0 || v == "" {
			continue
		}
		c := Check{ID: "driver_" + v, Title: "Driver de " + g.Name}
		switch {
		case months >= 9:
			c.Status = Warn
			c.Detail = fmt.Sprintf("Versión %s del %s (hace %d meses). Un driver viejo puede perder rendimiento y correcciones en juegos nuevos. Bajá el más reciente de la web oficial.", g.Version, g.Date, months)
			c.Action, c.ActionLabel = "open_driver_"+v, "Abrir página de drivers"
		default:
			c.Status = OK
			c.Detail = fmt.Sprintf("Versión %s del %s: es reciente.", g.Version, g.Date)
		}
		out = append(out, c)
	}

	// Discos
	for _, d := range h.Disks {
		var probs []string
		if d.Health != "" && !strings.EqualFold(d.Health, "Healthy") {
			probs = append(probs, "Windows lo marca como «"+d.Health+"»: hacé una copia de tus archivos importantes ya")
		}
		if d.Wear != nil && *d.Wear >= 80 {
			probs = append(probs, fmt.Sprintf("desgaste del %d%% de su vida útil", *d.Wear))
		}
		if d.Temp != nil && *d.Temp >= 70 {
			probs = append(probs, fmt.Sprintf("temperatura de %d °C", *d.Temp))
		}
		c := Check{ID: "disk_health", Title: "Estado del disco " + d.Name}
		if len(probs) > 0 {
			c.Status = Warn
			if d.Health != "" && !strings.EqualFold(d.Health, "Healthy") {
				c.Status = Bad
			}
			c.Detail = strings.ToUpper(probs[0][:1]) + probs[0][1:] + "."
			if len(probs) > 1 {
				c.Detail = strings.ToUpper(probs[0][:1]) + probs[0][1:] + "; " + strings.Join(probs[1:], "; ") + "."
			}
		} else {
			c.Status = OK
			c.Detail = "Saludable."
			if d.Wear != nil {
				c.Detail += fmt.Sprintf(" Desgaste: %d%%.", *d.Wear)
			}
			if d.Temp != nil {
				c.Detail += fmt.Sprintf(" Temperatura: %d °C.", *d.Temp)
			}
		}
		out = append(out, c)
	}

	// Espacio libre en el disco de Windows
	if h.SysFreePct != nil {
		c := Check{ID: "disk_space", Title: "Espacio libre en el disco de Windows"}
		switch {
		case *h.SysFreePct < 10:
			c.Status = Bad
			c.Detail = fmt.Sprintf("Quedan %.0f GB libres (%d%%). Con menos del 10%% Windows y los juegos se vuelven más lentos. Usá «Limpieza» para liberar espacio.", h.SysFreeGB, *h.SysFreePct)
		case *h.SysFreePct < 20:
			c.Status = Warn
			c.Detail = fmt.Sprintf("Quedan %.0f GB libres (%d%%). Conviene dejar al menos un 20%%, sobre todo si es un SSD.", h.SysFreeGB, *h.SysFreePct)
		default:
			c.Status = OK
			c.Detail = fmt.Sprintf("%.0f GB libres (%d%%).", h.SysFreeGB, *h.SysFreePct)
		}
		out = append(out, c)
	}
	return out
}

// Health revisa Resizable BAR, driver de la gráfica, discos y espacio libre.
func Health(sys system.System, hw scan.Hardware) []Check {
	raw, err := sys.PowerShell(healthScript, 45*time.Second)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	return ParseHealth(raw, hw, time.Now())
}

// BIOSGuide es la lista de ajustes de BIOS que conviene revisar (Windows no puede cambiarlos).
func BIOSGuide(hw scan.Hardware) Check {
	mem := memProfile(hw)
	virt := "SVM Mode"
	if hw.CPUVendor == "intel" {
		virt = "Intel VT-x / VT-d"
	}
	return Check{ID: "bios_guide", Title: "Guía de ajustes de la BIOS", Status: Info,
		Detail: "Esto no se cambia desde Windows: entrá a la BIOS (tecla Supr o F2 al encender) y revisá estos puntos. " +
			"Perfil de memoria " + mem + " activado. " +
			"«Above 4G Decoding» y «Resizable BAR» activados (con una GPU reciente). " +
			"Si un anti-cheat lo pide: Secure Boot, TPM (fTPM/PTT) y " + virt + " / IOMMU activados. " +
			"Modo de arranque UEFI, no Legacy. " +
			"Actualizá la BIOS solo si tu placa lo recomienda para tu procesador o memoria, y nunca con el equipo en riesgo de apagarse."}
}
