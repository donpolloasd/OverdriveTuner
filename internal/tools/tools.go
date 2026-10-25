// Package tools reúne diagnósticos puntuales: red, disco y la configuración
// de video de CS2.
package tools

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/system"
)

// ---------------------------------------------------------------- red

// PingResult es la medición contra un destino.
type PingResult struct {
	Name   string  `json:"name"`
	Host   string  `json:"host"`
	Sent   int     `json:"sent"`
	LossPc int     `json:"lossPct"`
	Min    int     `json:"min"`
	Avg    int     `json:"avg"`
	Max    int     `json:"max"`
	Jitter float64 `json:"jitter"`
	Status string  `json:"status"` // ok, warn, bad, error
	Detail string  `json:"detail"`
}

var (
	lossRe  = regexp.MustCompile(`\((\d+)%`)
	statRe  = regexp.MustCompile(`=\s*(\d+)\s*ms`)
	replyRe = regexp.MustCompile(`(?i)(?:time|tiempo)\s*[=<]\s*(\d+)\s*ms`)
)

// ParsePing interpreta la salida de ping.exe (en inglés o español).
func ParsePing(out string) (loss, min, max, avg int, jitter float64, ok bool) {
	m := lossRe.FindStringSubmatch(out)
	if m == nil {
		return
	}
	loss, _ = strconv.Atoi(m[1])
	st := statRe.FindAllStringSubmatch(out, -1)
	if len(st) >= 3 {
		min, _ = strconv.Atoi(st[len(st)-3][1])
		max, _ = strconv.Atoi(st[len(st)-2][1])
		avg, _ = strconv.Atoi(st[len(st)-1][1])
	}
	var ts []float64
	for _, r := range replyRe.FindAllStringSubmatch(out, -1) {
		v, _ := strconv.ParseFloat(r[1], 64)
		ts = append(ts, v)
	}
	for i := 1; i < len(ts); i++ {
		jitter += math.Abs(ts[i] - ts[i-1])
	}
	if len(ts) > 1 {
		jitter /= float64(len(ts) - 1)
	}
	return loss, min, max, avg, jitter, true
}

func judge(r *PingResult) {
	switch {
	case r.LossPc >= 5:
		r.Status, r.Detail = "bad", "Pierde paquetes: se nota como tirones y desincronización. Probá con cable de red en vez de Wi-Fi."
	case r.LossPc > 0 || r.Avg > 80 || r.Jitter > 15:
		r.Status, r.Detail = "warn", "Conexión irregular: hay pérdida, ping alto o variaciones."
	default:
		r.Status, r.Detail = "ok", "Estable."
	}
}

// NetTest hace 15 pings a cada destino (en paralelo) y los evalúa.
func NetTest(sys system.System) []PingResult {
	hosts := []PingResult{{Name: "Google DNS", Host: "8.8.8.8"}, {Name: "Cloudflare", Host: "1.1.1.1"}, {Name: "Quad9", Host: "9.9.9.9"}}
	var wg sync.WaitGroup
	for i := range hosts {
		wg.Add(1)
		go func(r *PingResult) {
			defer wg.Done()
			out, _, _ := sys.Exec(60*time.Second, "ping.exe", "-n", "15", r.Host)
			loss, mn, mx, avg, jit, ok := ParsePing(out)
			if !ok {
				r.Status, r.Detail = "error", "No se pudo medir (sin conexión o ping bloqueado)."
				return
			}
			r.Sent, r.LossPc, r.Min, r.Max, r.Avg, r.Jitter = 15, loss, mn, mx, avg, math.Round(jit*10)/10
			judge(r)
		}(&hosts[i])
	}
	wg.Wait()
	return hosts
}

// ---------------------------------------------------------------- discos

// Volume es una unidad (letra) de un disco.
type Volume struct {
	Letter string  `json:"letter"`
	FreeGB float64 `json:"freeGB"`
	SizeGB float64 `json:"sizeGB"`
}

// Disk es el estado de un disco.
type Disk struct {
	Name    string   `json:"name"`
	Media   string   `json:"media"`
	Bus     string   `json:"bus"`
	Health  string   `json:"health"`
	SizeGB  int      `json:"sizeGB"`
	Status  string   `json:"status"` // ok, warn, bad
	Detail  string   `json:"detail"`
	Temp    int      `json:"temp"`  // °C, -1 si no se sabe
	Wear    int      `json:"wear"`  // % de vida usado, -1 si no se sabe
	Hours   int      `json:"hours"` // horas encendido, -1 si no se sabe
	Errors  int      `json:"errors"`
	Trim    string   `json:"trim"` // on, off, "" si no se sabe
	Volumes []Volume `json:"volumes"`
	Tips    []string `json:"tips"`
}

const diskScript = `$ErrorActionPreference='SilentlyContinue'
$trim = ''
try { $t = (& fsutil.exe behavior query DisableDeleteNotify) -join ' '; if ($t -match 'DisableDeleteNotify\s*=\s*(\d)') { $trim = $(if ($matches[1] -eq '0') { 'on' } else { 'off' }) } } catch {}
$res = foreach ($d in @(Get-PhysicalDisk)) {
  $rc = $d | Get-StorageReliabilityCounter
  $vols = @()
  try {
    $vols = @(Get-Partition -DiskNumber ([int]$d.DeviceId) | Where-Object { $_.DriveLetter } | ForEach-Object {
      $v = Get-Volume -DriveLetter $_.DriveLetter
      [pscustomobject]@{ letter=[string]$_.DriveLetter; freeGB=[math]::Round($v.SizeRemaining/1GB,1); sizeGB=[math]::Round($v.Size/1GB,1) } })
  } catch {}
  $tmp = -1; if ($rc -and $rc.Temperature) { $tmp = [int]$rc.Temperature }
  $wr = -1; if ($rc -and $null -ne $rc.Wear) { $wr = [int]$rc.Wear }
  $ph = -1; if ($rc -and $null -ne $rc.PowerOnHours) { $ph = [int]$rc.PowerOnHours }
  $er = 0; if ($rc) { $er = [int]($rc.ReadErrorsUncorrected + $rc.WriteErrorsUncorrected) }
  [pscustomobject]@{ name=$d.FriendlyName; media="$($d.MediaType)"; bus="$($d.BusType)"; health="$($d.HealthStatus)"; size=[int]($d.Size/1GB); temp=$tmp; wear=$wr; hours=$ph; errors=$er; trim=$trim; volumes=@($vols) }
}
ConvertTo-Json -InputObject @($res) -Depth 4 -Compress`

// Disks lee la salud de los discos con Windows (sin programas externos).
func Disks(sys system.System) ([]Disk, error) {
	out, err := sys.PowerShell(diskScript, 40*time.Second)
	if err != nil {
		return nil, err
	}
	return parseDisks(out)
}

func parseDisks(out string) ([]Disk, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return []Disk{}, nil
	}
	type raw struct {
		Name, Media, Bus, Health string
		Size, Temp, Wear, Hours  int
		Errors                   int
		Trim                     string
		Volumes                  []Volume
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
	ds := []Disk{}
	for _, r := range rs {
		d := Disk{Name: r.Name, Media: r.Media, Bus: r.Bus, Health: r.Health, SizeGB: r.Size, Status: "ok", Detail: "Sin problemas.",
			Temp: r.Temp, Wear: r.Wear, Hours: r.Hours, Errors: r.Errors, Trim: r.Trim, Volumes: r.Volumes, Tips: []string{}}
		if d.Volumes == nil {
			d.Volumes = []Volume{}
		}
		bump := func(st, msg string) {
			if rank(st) < rank(d.Status) {
				d.Status, d.Detail = st, msg
			}
		}
		switch strings.ToLower(r.Health) {
		case "healthy":
		case "warning":
			bump("warn", "Windows reporta una advertencia en este disco. Hacé una copia de lo importante.")
		default:
			bump("bad", "Windows reporta que este disco puede fallar. Copiá tus archivos ya.")
		}
		ssd := strings.EqualFold(r.Media, "SSD") || strings.EqualFold(r.Bus, "NVMe")
		hot := 55
		if ssd {
			hot = 70
		}
		switch {
		case r.Wear >= 95:
			bump("bad", "El SSD usó casi toda su vida útil. Hacé copia de tus archivos y planificá reemplazarlo.")
		case r.Wear >= 80:
			bump("warn", "El SSD ya usó buena parte de su vida útil.")
		}
		if r.Errors > 0 {
			bump("warn", "El disco registró errores de lectura o escritura sin corregir. Hacé una copia de lo importante.")
		}
		if r.Temp >= hot {
			bump("warn", "El disco está muy caliente. Revisá la ventilación de la caja.")
		}
		for _, v := range d.Volumes {
			if v.SizeGB > 0 && v.FreeGB/v.SizeGB < 0.10 {
				bump("warn", "La unidad "+v.Letter+": tiene menos del 10 % libre. Con poco espacio libre los juegos cargan peor y el SSD se desgasta más.")
				break
			}
		}
		if ssd && r.Trim == "off" {
			d.Tips = append(d.Tips, "TRIM está desactivado en Windows: activalo con «fsutil behavior set DisableDeleteNotify 0» desde un CMD de administrador.")
		}
		if ssd {
			d.Tips = append(d.Tips, "Ideal para instalar los juegos y Windows.")
		} else if strings.EqualFold(r.Media, "HDD") {
			d.Tips = append(d.Tips, "Disco mecánico: mejor para archivos y copias; los juegos cargan más lento y pueden dar tirones.")
		}
		ds = append(ds, d)
	}
	return ds, nil
}

// ---------------------------------------------------------------- CS2

// Finding es una observación sobre la configuración de video.
type Finding struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Status string `json:"status"` // ok, warn, bad, info
	Advice string `json:"advice"`
}

// VideoReview es el resultado de revisar cs2_video.txt.
type VideoReview struct {
	Found    bool      `json:"found"`
	Path     string    `json:"path"`
	Message  string    `json:"message"`
	Findings []Finding `json:"findings"`
}

var kvRe = regexp.MustCompile(`"([^"]+)"\s+"([^"]*)"`)

// ParseVideo extrae pares clave/valor del archivo de video de CS2.
func ParseVideo(text string) map[string]string {
	m := map[string]string{}
	for _, x := range kvRe.FindAllStringSubmatch(text, -1) {
		m[strings.ToLower(x[1])] = x[2]
	}
	return m
}

// ReviewVideo revisa solo opciones conocidas; lo demás se ignora.
func ReviewVideo(kv map[string]string, targetHz int) []Finding {
	var f []Finding
	if v, ok := kv["setting.mat_vsync"]; ok {
		if v != "0" {
			f = append(f, Finding{"V-Sync", v, "warn", "V-Sync activado: agrega latencia y limita los FPS. Para competir conviene desactivarlo."})
		} else {
			f = append(f, Finding{"V-Sync", v, "ok", "Desactivado."})
		}
	}
	num, _ := strconv.ParseFloat(kv["setting.refreshrate_numerator"], 64)
	den, _ := strconv.ParseFloat(kv["setting.refreshrate_denominator"], 64)
	if num > 0 && den > 0 {
		hz := num / den
		val := fmt.Sprintf("%.0f Hz", hz)
		if targetHz > 0 && hz < float64(targetHz)-3 {
			f = append(f, Finding{"Frecuencia en el juego", val, "bad", fmt.Sprintf("El juego está configurado a %.0f Hz y tu monitor es de %d Hz. Cambialo en Configuración > Video.", hz, targetHz)})
		} else {
			f = append(f, Finding{"Frecuencia en el juego", val, "ok", "Correcta."})
		}
	}
	if v, ok := kv["setting.msaa_samples"]; ok {
		if n, _ := strconv.Atoi(v); n >= 4 {
			f = append(f, Finding{"Antialiasing MSAA", v + "x", "warn", "MSAA 4x o más cuesta bastante rendimiento. CMAA2 o 2x suelen dar buena imagen con menos costo."})
		} else {
			f = append(f, Finding{"Antialiasing MSAA", v + "x", "ok", "Costo bajo."})
		}
	}
	if v, ok := kv["setting.fullscreen"]; ok && v == "0" {
		f = append(f, Finding{"Modo de pantalla", "ventana", "info", "En ventana puede sumar algo de latencia; probá pantalla completa."})
	}
	sort.SliceStable(f, func(i, j int) bool { return rank(f[i].Status) < rank(f[j].Status) })
	return f
}

func rank(s string) int {
	switch s {
	case "bad":
		return 0
	case "warn":
		return 1
	case "info":
		return 2
	}
	return 3
}

// FindVideoFile busca cs2_video.txt en la carpeta de Steam del usuario.
func FindVideoFile(sys system.System) string {
	v, err := system.RegGet(sys, `HKCU\Software\Valve\Steam`, "SteamPath")
	if err != nil || !v.Exists || v.Data == "" {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(v.Data, "userdata", "*", "730", "local", "cfg", "cs2_video.txt"))
	best, bt := "", time.Time{}
	for _, m := range matches {
		if st, err := os.Stat(m); err == nil && st.ModTime().After(bt) {
			best, bt = m, st.ModTime()
		}
	}
	return best
}

// Video revisa la configuración de video de CS2.
func Video(sys system.System, targetHz int) VideoReview {
	p := FindVideoFile(sys)
	if p == "" {
		return VideoReview{Message: "No encontré cs2_video.txt. Abrí CS2 al menos una vez (y cerralo) para que Steam lo cree.", Findings: []Finding{}}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return VideoReview{Path: p, Message: "No se pudo leer el archivo: " + err.Error(), Findings: []Finding{}}
	}
	f := ReviewVideo(ParseVideo(string(b)), targetHz)
	if f == nil {
		f = []Finding{}
	}
	return VideoReview{Found: true, Path: p, Findings: f,
		Message: "Solo se revisan las opciones que OverdriveTuner reconoce; el resto del archivo no se toca."}
}
