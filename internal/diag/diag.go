// Package diag revisa el equipo en busca de causas típicas de microcortes y
// de falta de fluidez en monitores de alta frecuencia.
package diag

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/scan"
	"fpsforge/internal/system"
)

// Estados de un chequeo.
const (
	OK   = "ok"
	Warn = "warn"
	Bad  = "bad"
	Info = "info"
)

// Check es el resultado de una revisión.
type Check struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	Action      string `json:"action,omitempty"`
	ActionLabel string `json:"actionLabel,omitempty"`
}

var schemeRe = regexp.MustCompile(`\(([^)]+)\)\s*$`)

// activePlan devuelve el nombre del plan de energía activo.
func activePlan(sys system.System) string {
	out, _, err := sys.Exec(15*time.Second, "powercfg.exe", "/getactivescheme")
	if err != nil {
		return ""
	}
	if m := schemeRe.FindStringSubmatch(strings.TrimSpace(out)); m != nil {
		return m[1]
	}
	return ""
}

func regDword(sys system.System, key, name string) (int, bool) {
	v, err := system.RegGet(sys, key, name)
	if err != nil || !v.Exists {
		return 0, false
	}
	n, err := strconv.Atoi(v.Data)
	return n, err == nil
}

// Run ejecuta todos los chequeos para un monitor de targetHz: los rápidos, los lentos y la guía de BIOS.
func Run(sys system.System, hw scan.Hardware, targetHz int) []Check {
	out := append(Fast(sys, hw, targetHz), Slow(sys, hw)...)
	return append(out, BIOSGuide(hw))
}

// Slow son los chequeos que necesitan PowerShell (anti-cheat de kernel; salud de discos, driver y
// Resizable BAR): tardan unos segundos, así que corren a la vez y la interfaz los guarda un rato.
func Slow(sys system.System, hw scan.Hardware) []Check {
	var ac, hl []Check
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); ac = antiCheat(sys) }()
	go func() { defer wg.Done(); hl = Health(sys, hw) }()
	wg.Wait()
	return append(ac, hl...)
}

// Fast son los chequeos que se resuelven al instante (hardware ya escaneado, registro, plan de energía).
func Fast(sys system.System, hw scan.Hardware, targetHz int) []Check {
	var out []Check
	add := func(c Check) { out = append(out, c) }

	// 1) ¿A qué gráfica está conectado el monitor?
	var active *scan.GPU
	hasDiscrete := false
	for i := range hw.GPUs {
		if hw.GPUs[i].Active && active == nil {
			active = &hw.GPUs[i]
		}
		if !hw.GPUs[i].Integrated() {
			hasDiscrete = true
		}
	}
	switch {
	case active == nil:
		add(Check{ID: "display_gpu", Title: "Gráfica que maneja el monitor", Status: Info,
			Detail: "No se pudo determinar qué gráfica tiene el monitor conectado."})
	case active.Integrated() && hasDiscrete:
		add(Check{ID: "display_gpu", Title: "Gráfica que maneja el monitor", Status: Bad,
			Detail: fmt.Sprintf("Tu monitor está conectado a la gráfica integrada (%s), no a la dedicada. Los juegos se dibujan en la placa de video pero la imagen pasa por la integrada: eso agrega latencia y microcortes y puede limitar los Hz. Pasá el cable del monitor a una salida DisplayPort/HDMI de la placa de video (en la parte baja de la parte trasera del PC, no la de la placa madre).", active.Name),
		})
	default:
		add(Check{ID: "display_gpu", Title: "Gráfica que maneja el monitor", Status: OK,
			Detail: fmt.Sprintf("El monitor está conectado a %s.", active.Name)})
	}

	// 2) Frecuencia actual contra la que debería tener.
	if active != nil && active.Refresh > 0 {
		switch {
		case targetHz > 0 && active.Refresh < targetHz-3:
			add(Check{ID: "refresh", Title: "Frecuencia del monitor", Status: Bad,
				Detail: fmt.Sprintf("Windows está usando %d Hz y tu monitor es de %d Hz. Con menos Hz la imagen se ve menos fluida aunque el juego dé más FPS. Cambialo en Pantalla avanzada > Frecuencia de actualización. Si no aparece %d Hz, revisá el cable (DisplayPort o HDMI 2.0 o superior) y que el monitor tenga el modo activado en su menú.", active.Refresh, targetHz, targetHz),
				Action: "open_display", ActionLabel: "Abrir configuración de pantalla"})
		default:
			add(Check{ID: "refresh", Title: "Frecuencia del monitor", Status: OK,
				Detail: fmt.Sprintf("Windows está usando %d Hz.", active.Refresh)})
		}
	}

	// 3) Memoria: perfil DOCP/XMP y canales.
	switch {
	case hw.RAMRated > 0 && hw.RAMMTs > 0 && float64(hw.RAMMTs) < float64(hw.RAMRated)*0.95:
		add(Check{ID: "ram_xmp", Title: "Velocidad de la memoria", Status: Bad,
			Detail: fmt.Sprintf("Tu RAM funciona a %d MT/s pero está preparada para %d MT/s. Activá el perfil %s en la BIOS: es de lo que más FPS parejos da en CS2.", hw.RAMMTs, hw.RAMRated, memProfile(hw))})
	case hw.RAMMTs > 0:
		add(Check{ID: "ram_xmp", Title: "Velocidad de la memoria", Status: OK,
			Detail: fmt.Sprintf("La RAM funciona a %d MT/s.", hw.RAMMTs)})
	}
	switch {
	case hw.RAMSticks == 1:
		add(Check{ID: "ram_channels", Title: "Canales de memoria", Status: Warn,
			Detail: "Hay un solo módulo de RAM: funciona en canal simple, que rinde bastante menos. Con dos módulos iguales en las ranuras que indica el manual de tu placa (suelen ser la 2.ª y la 4.ª contando desde el procesador) pasa a dual channel."})
	case hw.RAMSticks >= 2:
		add(Check{ID: "ram_channels", Title: "Canales de memoria", Status: OK,
			Detail: fmt.Sprintf("%d módulos. Para que sea dual channel deben estar en las ranuras que indica el manual de la placa (normalmente la 2.ª y la 4.ª contando desde el procesador).", hw.RAMSticks)})
	}

	// 4) Plan de energía.
	if plan := activePlan(sys); plan != "" {
		low := strings.ToLower(plan)
		switch {
		case strings.Contains(low, "econom") || strings.Contains(low, "saver") || strings.Contains(low, "ahorro"):
			add(Check{ID: "power", Title: "Plan de energía", Status: Bad,
				Detail: fmt.Sprintf("El plan activo es «%s», que limita el procesador. Cambialo a Equilibrado o Alto rendimiento.", plan)})
		default:
			add(Check{ID: "power", Title: "Plan de energía", Status: OK, Detail: fmt.Sprintf("Plan activo: %s.", plan)})
		}
	}

	// 5) Captura y modo juego de Windows.
	if n, ok := regDword(sys, `HKCU\System\GameConfigStore`, "GameDVR_Enabled"); ok && n == 1 {
		add(Check{ID: "dvr", Title: "Grabación en segundo plano (Game DVR)", Status: Warn,
			Detail: "Windows puede estar grabando en segundo plano. Se desactiva en «FPS y juegos»."})
	}
	if n, ok := regDword(sys, `HKCU\Software\Microsoft\GameBar`, "AutoGameModeEnabled"); ok && n == 0 {
		add(Check{ID: "gamemode", Title: "Modo Juego de Windows", Status: Warn,
			Detail: "El Modo Juego está desactivado. Se activa en «FPS y juegos»."})
	}

	// 6) Varios monitores.
	if hw.Monitors > 1 {
		add(Check{ID: "monitors", Title: "Varios monitores", Status: Info,
			Detail: fmt.Sprintf("Hay %d monitores activos. Si el segundo tiene otra frecuencia (por ejemplo 60 Hz), el escritorio de Windows puede entrecortar los juegos en ventana o sin bordes. Jugá en pantalla completa o apagá el segundo monitor al jugar.", hw.Monitors)})
	}

	// 7) Programación de GPU por hardware (informativo).
	if n, ok := regDword(sys, `HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, "HwSchMode"); ok {
		state := "desactivada"
		if n == 2 {
			state = "activada"
		}
		add(Check{ID: "hags", Title: "Programación de GPU acelerada por hardware", Status: Info,
			Detail: fmt.Sprintf("Está %s. En algunos juegos mejora la fluidez y en otros la empeora: si la cambiás, medilo con el benchmark.", state)})
	}
	return out
}

const acScript = `$ErrorActionPreference='SilentlyContinue'
$r=@{}
try{$r.secureboot=[bool](Confirm-SecureBootUEFI)}catch{$r.secureboot=$null}
$t=Get-Tpm; if($t){$r.tpm=[bool]$t.TpmPresent;$r.tpmready=[bool]$t.TpmReady}
$h=(Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity').Enabled
$r.hvci=($h -eq 1)
$dg=Get-CimInstance -Namespace root\Microsoft\Windows\DeviceGuard -ClassName Win32_DeviceGuard
if($dg){$r.dma=[bool](@($dg.AvailableSecurityProperties) -contains 3)}
$r.reboot=[bool](Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired')
$r | ConvertTo-Json -Compress`

// antiCheat revisa lo que piden los anti-cheat de kernel y avisa si falta algo.
func antiCheat(sys system.System) []Check {
	raw, err := sys.PowerShell(acScript, 30*time.Second)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var r struct {
		SecureBoot *bool `json:"secureboot"`
		TPM        *bool `json:"tpm"`
		TPMReady   *bool `json:"tpmready"`
		HVCI       bool  `json:"hvci"`
		DMA        *bool `json:"dma"`
		Reboot     bool  `json:"reboot"`
	}
	if json.Unmarshal([]byte(raw), &r) != nil {
		return nil
	}
	var miss []string
	if r.SecureBoot != nil && !*r.SecureBoot {
		miss = append(miss, "Secure Boot")
	}
	if r.TPM != nil && (!*r.TPM || (r.TPMReady != nil && !*r.TPMReady)) {
		miss = append(miss, "TPM 2.0")
	}
	if !r.HVCI {
		miss = append(miss, "Integridad de memoria (HVCI)")
	}
	if r.DMA != nil && !*r.DMA {
		miss = append(miss, "IOMMU / protección DMA (se activa en la BIOS: IOMMU o SVM)")
	}
	if r.Reboot {
		miss = append(miss, "Windows tiene actualizaciones pendientes: reiniciá para completarlas")
	}
	if len(miss) > 0 {
		return []Check{{ID: "anticheat", Title: "Requisitos de anti-cheat de kernel", Status: Warn,
			Detail: "No están activos: " + strings.Join(miss, ", ") + ". Los anti-cheat de kernel los piden para jugar. OverdriveTuner nunca los desactiva sin doble confirmación."}}
	}
	return []Check{{ID: "anticheat", Title: "Requisitos de anti-cheat de kernel", Status: OK,
		Detail: "Secure Boot, TPM e integridad de memoria están activos. OverdriveTuner no los toca."}}
}

// Summary cuenta avisos por gravedad.
func Summary(checks []Check) (bad, warn int) {
	for _, c := range checks {
		switch c.Status {
		case Bad:
			bad++
		case Warn:
			warn++
		}
	}
	return
}

// OpenDisplay abre la pantalla avanzada de Configuración.
func OpenDisplay(sys system.System) error {
	_, _, err := sys.Exec(10*time.Second, "explorer.exe", "ms-settings:display-advanced")
	return err
}

func memProfile(hw scan.Hardware) string {
	switch hw.CPUVendor {
	case "amd":
		return "DOCP/EXPO"
	case "intel":
		return "XMP"
	}
	return "DOCP/XMP"
}
