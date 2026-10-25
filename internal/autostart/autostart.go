// Package autostart registra (o quita) una tarea programada de Windows que abre el overlay de
// OverdriveTuner al iniciar sesión, en segundo plano y con permisos de administrador, para que no
// aparezca el aviso de UAC cada vez.
package autostart

import (
	"fmt"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// TaskName es el nombre de la tarea en el Programador de tareas.
const TaskName = "OverdriveTuner Overlay"

func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Script devuelve el comando de PowerShell que crea la tarea para el programa exe.
func Script(exe string) string {
	return `$ErrorActionPreference='Stop'
$u = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$a = New-ScheduledTaskAction -Execute ` + psq(exe) + ` -Argument '--overlay --boot' -WorkingDirectory (Split-Path -Parent ` + psq(exe) + `)
$t = New-ScheduledTaskTrigger -AtLogOn -User $u
$t.Delay = 'PT15S'
$p = New-ScheduledTaskPrincipal -UserId $u -LogonType Interactive -RunLevel Highest
$s = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero) -StartWhenAvailable
Register-ScheduledTask -TaskName ` + psq(TaskName) + ` -Action $a -Trigger $t -Principal $p -Settings $s -Force | Out-Null`
}

// Enable crea o reemplaza la tarea.
func Enable(sys system.System, exe string) error {
	if strings.TrimSpace(exe) == "" {
		return fmt.Errorf("no se conoce la ruta del programa")
	}
	if out, err := sys.PowerShell(Script(exe), 45*time.Second); err != nil {
		return fmt.Errorf("no se pudo activar el inicio con Windows: %v %s", err, strings.TrimSpace(out))
	}
	return nil
}

// Disable quita la tarea (no es error que no exista).
func Disable(sys system.System) error {
	_, err := sys.PowerShell(`Unregister-ScheduledTask -TaskName `+psq(TaskName)+` -Confirm:$false -ErrorAction SilentlyContinue`, 30*time.Second)
	if err != nil {
		return fmt.Errorf("no se pudo desactivar el inicio con Windows: %w", err)
	}
	return nil
}

// Enabled dice si la tarea existe.
func Enabled(sys system.System) bool {
	out, err := sys.PowerShell(`if (Get-ScheduledTask -TaskName `+psq(TaskName)+` -ErrorAction SilentlyContinue) { 'si' }`, 20*time.Second)
	return err == nil && strings.Contains(out, "si")
}

// Info es el estado de la tarea programada.
type Info struct {
	Registered bool   `json:"registered"`
	State      string `json:"state"`      // Ready, Running, Disabled…
	LastRun    string `json:"lastRun"`    // fecha de la última ejecución (vacía si nunca corrió)
	LastResult int64  `json:"lastResult"` // código de la última ejecución (0 = bien, 267011 = todavía no corrió)
}

// Status consulta la tarea: si existe, su estado y cómo terminó la última vez.
func Status(sys system.System) Info {
	out, err := sys.PowerShell(`$t = Get-ScheduledTask -TaskName `+psq(TaskName)+` -ErrorAction SilentlyContinue
if ($t) { $i = Get-ScheduledTaskInfo -TaskName `+psq(TaskName)+`
  $d = ''; if ($i.LastRunTime -and $i.LastRunTime.Year -gt 2000) { $d = $i.LastRunTime.ToString('yyyy-MM-dd HH:mm') }
  'T|' + $t.State + '|' + $d + '|' + $i.LastTaskResult }`, 25*time.Second)
	if err != nil {
		return Info{}
	}
	return parseInfo(out)
}

func parseInfo(out string) Info {
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "T|") {
			continue
		}
		f := strings.Split(ln, "|")
		if len(f) < 4 {
			continue
		}
		var code int64
		fmt.Sscan(f[3], &code)
		return Info{Registered: true, State: f[1], LastRun: f[2], LastResult: code}
	}
	return Info{}
}

// RunNow ejecuta la tarea ahora, para comprobar que anda sin reiniciar.
func RunNow(sys system.System) error {
	_, err := sys.PowerShell(`Start-ScheduledTask -TaskName `+psq(TaskName)+` -ErrorAction Stop`, 25*time.Second)
	if err != nil {
		return fmt.Errorf("no se pudo ejecutar la tarea: %w", err)
	}
	return nil
}
