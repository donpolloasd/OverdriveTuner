// Package scan detecta el hardware y el sistema para recomendar tweaks.
package scan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// GPU describe una tarjeta gráfica.
type GPU struct {
	Name    string `json:"name"`
	Vendor  string `json:"vendor"` // nvidia, amd, intel, otro
	VRAMMB  int    `json:"vramMB"`
	Driver  string `json:"driver"`
	Refresh int    `json:"refresh"`
	Active  bool   `json:"active"` // esta gráfica está manejando un monitor
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// Integrated indica si es una gráfica integrada en el procesador/placa.
func (g GPU) Integrated() bool {
	n := strings.ToLower(g.Name)
	if strings.Contains(n, " rx ") || strings.Contains(n, "rtx") || strings.Contains(n, "gtx") || strings.Contains(n, "arc a") || strings.Contains(n, "radeon pro") {
		return false
	}
	for _, k := range []string{"radeon(tm) graphics", "radeon graphics", "vega", "uhd", "iris", "hd graphics", "intel(r) graphics"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// Disk describe una unidad física.
type Disk struct {
	Model  string `json:"model"`
	Type   string `json:"type"` // SSD, HDD, NVMe, desconocido
	SizeGB int    `json:"sizeGB"`
	System bool   `json:"system"`
}

// Hardware es el resultado del escaneo.
type Hardware struct {
	Computer  string `json:"computer"`
	CPU       string `json:"cpu"`
	CPUVendor string `json:"cpuVendor"`
	Cores     int    `json:"cores"`
	Threads   int    `json:"threads"`
	MaxMHz    int    `json:"maxMHz"`
	RAMGB     int    `json:"ramGB"`
	RAMMTs    int    `json:"ramMTs"`
	RAMRated  int    `json:"ramRatedMTs"`
	Monitors  int    `json:"monitors"`
	RAMSticks int    `json:"ramSticks"`
	Board     string `json:"board"`
	GPUs      []GPU  `json:"gpus"`
	MaxHz     int    `json:"maxHz"` // frecuencia máxima que Windows ofrece al monitor, a su resolución actual
	Disks     []Disk `json:"disks"`
	OS        string `json:"os"`
	Build     int    `json:"build"`
	Laptop    bool   `json:"laptop"`
	Admin     bool   `json:"admin"`
	Demo      bool   `json:"demo"`
}

const psScan = `
function Vendor([string]$n) {
  if ($n -match 'NVIDIA|GeForce|RTX|GTX|Quadro') { return 'nvidia' }
  if ($n -match 'AMD|Radeon|ATI') { return 'amd' }
  if ($n -match 'Intel') { return 'intel' }
  return 'otro'
}
$r = [ordered]@{}
$cs = Get-CimInstance Win32_ComputerSystem
$r.computer = [string]$cs.Name
$r.ramGB = [int][math]::Round($cs.TotalPhysicalMemory / 1GB)
$cpu = @(Get-CimInstance Win32_Processor)[0]
$r.cpu = ([string]$cpu.Name).Trim()
$r.cpuVendor = $(if ($cpu.Name -match 'AMD|Ryzen|EPYC|Athlon') { 'amd' } elseif ($cpu.Name -match 'Intel') { 'intel' } else { 'otro' })
$r.cores = [int]((Get-CimInstance Win32_Processor | Measure-Object NumberOfCores -Sum).Sum)
$r.threads = [int]((Get-CimInstance Win32_Processor | Measure-Object NumberOfLogicalProcessors -Sum).Sum)
$r.maxMHz = [int]$cpu.MaxClockSpeed
$mem = @(Get-CimInstance Win32_PhysicalMemory)
$r.ramSticks = $mem.Count
$spd = 0
foreach ($m in $mem) { $v = [int]$m.ConfiguredClockSpeed; if ($v -le 0) { $v = [int]$m.Speed }; if ($v -gt $spd) { $spd = $v } }
$r.ramMTs = $spd
$rated = 0
foreach ($m in $mem) { $v = [int]$m.Speed; if ($v -gt $rated) { $rated = $v } }
$r.ramRatedMTs = $rated
$r.monitors = @(Get-CimInstance -Namespace root\wmi -ClassName WmiMonitorBasicDisplayParams -ErrorAction SilentlyContinue | Where-Object { $_.Active }).Count
$bb = Get-CimInstance Win32_BaseBoard
$r.board = (([string]$bb.Manufacturer + ' ' + [string]$bb.Product).Trim())
$os = Get-CimInstance Win32_OperatingSystem
$r.os = [string]$os.Caption
$r.build = [int]$os.BuildNumber
$bat = @(Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue)
$enc = @((Get-CimInstance Win32_SystemEnclosure).ChassisTypes)
$r.laptop = ($bat.Count -gt 0) -or (($enc | Where-Object { $_ -in 8,9,10,11,12,14,18,21,30,31,32 }).Count -gt 0)
$gpus = New-Object System.Collections.ArrayList
$cls = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11cd-bfc1-08002be10318}'
$vram = @{}
function ToMem($v) {
  if ($null -eq $v) { return [int64]0 }
  if ($v -is [byte[]]) { $b = New-Object byte[] 8; [Array]::Copy($v, $b, [Math]::Min(8, $v.Length)); return [int64][BitConverter]::ToUInt64($b, 0) }
  try { return [int64]$v } catch { return [int64]0 }
}
Get-ChildItem -LiteralPath $cls -ErrorAction SilentlyContinue | ForEach-Object {
  try {
    $p = Get-ItemProperty -LiteralPath $_.PSPath -ErrorAction Stop
    $m = ToMem $p.'HardwareInformation.qwMemorySize'
    if ($m -le 0) { $m = ToMem $p.'HardwareInformation.MemorySize' }
    if ($m -gt 0 -and $p.DriverDesc) {
      $k = [string]$p.DriverDesc
      if (-not $vram.ContainsKey($k) -or $vram[$k] -lt $m) { $vram[$k] = $m }
    }
  } catch {}
}
foreach ($g in @(Get-CimInstance Win32_VideoController)) {
  $n = [string]$g.Name
  if ($n -match 'Basic|Remote|Virtual|Mirror|Meta') { continue }
  $mb = 0
  if ($vram.ContainsKey($n)) { $mb = [int]($vram[$n] / 1MB) }
  else {
    $key = $vram.Keys | Where-Object { $_ -like "*$n*" -or $n -like "*$_*" } | Select-Object -First 1
    if ($key) { $mb = [int]($vram[$key] / 1MB) }
    elseif ($g.AdapterRAM -and ([int64]$g.AdapterRAM -lt 4000MB)) { $mb = [int]([int64]$g.AdapterRAM / 1MB) }
  }
  [void]$gpus.Add([ordered]@{ name=$n; vendor=(Vendor $n); vramMB=$mb; driver=[string]$g.DriverVersion; refresh=[int]$g.CurrentRefreshRate; active=([int]$g.CurrentHorizontalResolution -gt 0); width=[int]$g.CurrentHorizontalResolution; height=[int]$g.CurrentVerticalResolution })
}
# Si un escaneo anterior ya obtuvo la memoria de esa gráfica, se reutiliza (dxdiag tarda unos 5 segundos).
if ($knownVram) { foreach ($g2 in $gpus) { if ($g2.vramMB -le 0 -and $knownVram.ContainsKey($g2.name)) { $g2.vramMB = [int]$knownVram[$g2.name] } } }
# Si el registro no dio la memoria de alguna gráfica, se la pide a DirectX (dxdiag): es lo que muestra Windows.
if (@($gpus | Where-Object { $_.vramMB -le 0 }).Count -gt 0) {
  try {
    $f = Join-Path $env:TEMP 'ot-dxdiag.txt'
    Remove-Item -LiteralPath $f -ErrorAction SilentlyContinue
    Start-Process -FilePath dxdiag.exe -ArgumentList '/whql:off','/t',$f -WindowStyle Hidden
    $t0 = Get-Date; $last = -1
    while (((Get-Date) - $t0).TotalSeconds -lt 40) {
      Start-Sleep -Milliseconds 700
      if (Test-Path -LiteralPath $f) { $sz = (Get-Item -LiteralPath $f).Length; if ($sz -gt 2000 -and $sz -eq $last) { break }; $last = $sz }
    }
    $dx = @{}; $cur = $null
    foreach ($ln in (Get-Content -LiteralPath $f -ErrorAction Stop)) {
      if ($ln -match '^\s*(Card name|Nombre de tarjeta)\s*:\s*(.+?)\s*$') { $cur = $Matches[2] }
      elseif ($cur -and $ln -match '^\s*(Dedicated Memory|Memoria dedicada)\s*:\s*(\d+)') { $dx[$cur] = [int]$Matches[2]; $cur = $null }
    }
    foreach ($g2 in $gpus) {
      if ($g2.vramMB -le 0) {
        $k2 = $dx.Keys | Where-Object { $_ -eq $g2.name -or $_ -like "*$($g2.name)*" -or $g2.name -like "*$_*" } | Select-Object -First 1
        if ($k2) { $g2.vramMB = [int]$dx[$k2] }
      }
    }
    Remove-Item -LiteralPath $f -ErrorAction SilentlyContinue
  } catch {}
}
# Frecuencia máxima que Windows ofrece al monitor principal con su resolución actual.
$r.maxHz = 0
try {
  Add-Type -Namespace OT -Name Dm -MemberDefinition '
  [StructLayout(LayoutKind.Sequential, CharSet=CharSet.Ansi)] public struct DEVMODE { [MarshalAs(UnmanagedType.ByValTStr, SizeConst=32)] public string dmDeviceName; public short dmSpecVersion; public short dmDriverVersion; public short dmSize; public short dmDriverExtra; public int dmFields; public int dmPositionX; public int dmPositionY; public int dmDisplayOrientation; public int dmDisplayFixedOutput; public short dmColor; public short dmDuplex; public short dmYResolution; public short dmTTOption; public short dmCollate; [MarshalAs(UnmanagedType.ByValTStr, SizeConst=32)] public string dmFormName; public short dmLogPixels; public int dmBitsPerPel; public int dmPelsWidth; public int dmPelsHeight; public int dmDisplayFlags; public int dmDisplayFrequency; public int dmICMMethod; public int dmICMIntent; public int dmMediaType; public int dmDitherType; public int dmReserved1; public int dmReserved2; public int dmPanningWidth; public int dmPanningHeight; }
  [DllImport("user32.dll", CharSet=CharSet.Ansi)] public static extern bool EnumDisplaySettings(string dev, int mode, ref DEVMODE dm);'
  $act = @($gpus | Where-Object { $_.active }) | Select-Object -First 1
  $w = 0; $h = 0
  if ($act) { $w = [int]$act.width; $h = [int]$act.height }
  $dm = New-Object OT.Dm+DEVMODE
  $dm.dmSize = [int16][Runtime.InteropServices.Marshal]::SizeOf($dm)
  $mx = 0; $i = 0
  while ([OT.Dm]::EnumDisplaySettings($null, $i, [ref]$dm) -and $i -lt 4000) {
    if (($w -le 0 -or ($dm.dmPelsWidth -eq $w -and $dm.dmPelsHeight -eq $h)) -and $dm.dmDisplayFrequency -gt $mx) { $mx = $dm.dmDisplayFrequency }
    $i++
  }
  $r.maxHz = [int]$mx
} catch {}
$r.gpus = @($gpus)
$sysDisk = -1
try { $sysDisk = (Get-Partition -DriveLetter ($env:SystemDrive.Substring(0,1)) -ErrorAction Stop | Get-Disk).Number } catch {}
$disks = New-Object System.Collections.ArrayList
foreach ($d in @(Get-PhysicalDisk -ErrorAction SilentlyContinue)) {
  $t = [string]$d.MediaType
  if ($d.BusType -eq 'NVMe') { $t = 'NVMe' } elseif ($t -notin 'SSD','HDD') { $t = 'desconocido' }
  [void]$disks.Add([ordered]@{ model=([string]$d.FriendlyName).Trim(); type=$t; sizeGB=[int][math]::Round($d.Size/1GB); system=([int]$d.DeviceId -eq $sysDisk) })
}
$r.disks = @($disks)
ConvertTo-Json -InputObject $r -Compress -Depth 5
`

// Run escanea el equipo. En sistemas simulados devuelve datos de demostración.
func Run(sys system.System) (Hardware, error) { return RunKnown(sys, nil) }

// RunKnown es como Run, pero reutiliza la memoria de video que ya se conocía de cada gráfica (por nombre)
// en vez de volver a pedírsela a dxdiag.
func RunKnown(sys system.System, knownVRAM map[string]int) (Hardware, error) {
	if sys.Name() != "windows" {
		h := Demo()
		h.Admin = sys.IsAdmin()
		return h, nil
	}
	out, err := sys.PowerShell(knownVRAMScript(knownVRAM)+psScan, 90*time.Second)
	if err != nil {
		return Hardware{}, fmt.Errorf("no se pudo escanear el hardware: %w", err)
	}
	h, err := Parse(out)
	if err != nil {
		return Hardware{}, err
	}
	h.Admin = sys.IsAdmin()
	return h, nil
}

// KnownVRAM devuelve la memoria de video ya detectada de cada gráfica (por nombre), para RunKnown.
func (h Hardware) KnownVRAM() map[string]int {
	m := map[string]int{}
	for _, g := range h.GPUs {
		if g.VRAMMB > 0 {
			m[g.Name] = g.VRAMMB
		}
	}
	return m
}

// knownVRAMScript arma la línea de PowerShell con la memoria de video ya conocida ($knownVram).
func knownVRAMScript(known map[string]int) string {
	if len(known) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("$knownVram = @{")
	for name, mb := range known {
		if mb > 0 && !strings.ContainsAny(name, "\r\n") {
			fmt.Fprintf(&b, "'%s'=%d;", strings.ReplaceAll(name, "'", "''"), mb)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// Parse interpreta la salida JSON del escáner.
func Parse(out string) (Hardware, error) {
	out = lastJSONLine(out, '{')
	var h Hardware
	if err := json.Unmarshal([]byte(out), &h); err != nil {
		return Hardware{}, fmt.Errorf("respuesta del escáner no válida: %w", err)
	}
	return h, nil
}

// Demo devuelve un equipo de ejemplo para la vista previa fuera de Windows.
func Demo() Hardware {
	return Hardware{
		Computer: "PC-Demo", CPU: "AMD Ryzen 5 5600G with Radeon Graphics", CPUVendor: "amd",
		Cores: 6, Threads: 12, MaxMHz: 3900, RAMGB: 32, RAMMTs: 3200, RAMSticks: 2,
		Board: "ASRock A520M-K V2", OS: "Microsoft Windows 11 Home", Build: 26100,
		RAMRated: 3200, Monitors: 1, MaxHz: 240,
		GPUs: []GPU{
			{Name: "AMD Radeon RX 7600", Vendor: "amd", VRAMMB: 8192, Driver: "32.0.21025.10016", Refresh: 240, Active: true, Width: 1920, Height: 1080},
			{Name: "AMD Radeon(TM) Graphics", Vendor: "amd", VRAMMB: 512, Driver: "31.0.21924.61"},
		},
		Disks: []Disk{
			{Model: "SSD 480GB", Type: "SSD", SizeGB: 447, System: true},
			{Model: "HDD 960GB", Type: "HDD", SizeGB: 894},
		},
		Demo: true,
	}
}

// Has evalúa una condición de hardware de un tweak.
func (h Hardware) Has(cond string) bool {
	switch cond {
	case "":
		return true
	case "ssd":
		for _, d := range h.Disks {
			if d.System {
				return d.Type == "SSD" || d.Type == "NVMe"
			}
		}
		return false // sin certeza, no se asume SSD
	case "ram16":
		return h.RAMGB >= 15
	case "desktop":
		return !h.Laptop
	case "laptop":
		return h.Laptop
	case "amd", "nvidia", "intel":
		for _, g := range h.GPUs {
			if g.Vendor == cond {
				return true
			}
		}
		return false
	case "win11":
		return h.Build >= 22000
	}
	return true
}

// CondText explica una condición en lenguaje natural.
func CondText(cond string) string {
	switch cond {
	case "ssd":
		return "requiere que Windows esté en un SSD"
	case "ram16":
		return "requiere 16 GB de RAM o más"
	case "desktop":
		return "pensado para PC de escritorio"
	case "laptop":
		return "pensado para portátiles"
	case "amd":
		return "requiere una GPU AMD"
	case "nvidia":
		return "requiere una GPU NVIDIA"
	case "intel":
		return "requiere una GPU Intel"
	case "win11":
		return "requiere Windows 11"
	}
	return ""
}

// lastJSONLine devuelve la última línea que empieza con el carácter dado:
// ignora avisos o errores que PowerShell pueda haber escrito antes.
func lastJSONLine(out string, open byte) string {
	lines := strings.Split(strings.ReplaceAll(out, "\r", ""), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" && l[0] == open {
			return l
		}
	}
	return strings.TrimSpace(out)
}
