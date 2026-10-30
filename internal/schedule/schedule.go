// Package schedule registra tareas programadas de Windows que abren OverdriveTuner sin ventana para hacer
// mantenimiento: limpieza de temporales y puntos de restauración. Corren con permisos de administrador, solo
// con la sesión iniciada, y si la PC estaba apagada a esa hora se ejecutan al encenderla.
package schedule

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fpsforge/internal/extras"
	"fpsforge/internal/system"
)

// Job es una tarea de mantenimiento.
type Job struct {
	ID   string `json:"id"`
	Name string `json:"name"` // nombre en el Programador de tareas
	Arg  string `json:"-"`
}

// Jobs son las tareas disponibles.
var Jobs = []Job{
	{ID: "cleanup", Name: "OverdriveTuner Limpieza semanal", Arg: "--task cleanup"},
	{ID: "restore", Name: "OverdriveTuner Punto de restauración", Arg: "--task restore"},
}

// ByID busca una tarea.
func ByID(id string) (Job, bool) {
	for _, j := range Jobs {
		if j.ID == id {
			return j, true
		}
	}
	return Job{}, false
}

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

var dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Script devuelve el PowerShell que crea la tarea. day: 0 = domingo … 6 = sábado; hhmm: «HH:MM».
func Script(j Job, exe string, day int, hhmm string) string {
	return `$ErrorActionPreference='Stop'
$u = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$a = New-ScheduledTaskAction -Execute ` + psq(exe) + ` -Argument ` + psq(j.Arg) + ` -WorkingDirectory (Split-Path -Parent ` + psq(exe) + `)
$t = New-ScheduledTaskTrigger -Weekly -DaysOfWeek ` + dayNames[day] + ` -At ` + psq(hhmm) + `
$p = New-ScheduledTaskPrincipal -UserId $u -LogonType Interactive -RunLevel Highest
$s = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Hours 1)
Register-ScheduledTask -TaskName ` + psq(j.Name) + ` -Action $a -Trigger $t -Principal $p -Settings $s -Force | Out-Null`
}

// Enable crea o reemplaza la tarea semanal.
func Enable(sys system.System, id, exe string, day int, hhmm string) error {
	j, ok := ByID(id)
	if !ok {
		return fmt.Errorf("tarea desconocida")
	}
	if day < 0 || day > 6 || !timeRe.MatchString(hhmm) {
		return fmt.Errorf("día u hora no válidos")
	}
	if strings.TrimSpace(exe) == "" {
		return fmt.Errorf("no se conoce la ruta del programa")
	}
	if out, err := sys.PowerShell(Script(j, exe, day, hhmm), 45*time.Second); err != nil {
		return fmt.Errorf("no se pudo programar la tarea: %v %s", err, strings.TrimSpace(out))
	}
	return nil
}

// Disable quita la tarea (no es error que no exista).
func Disable(sys system.System, id string) error {
	j, ok := ByID(id)
	if !ok {
		return fmt.Errorf("tarea desconocida")
	}
	if _, err := sys.PowerShell(`Unregister-ScheduledTask -TaskName `+psq(j.Name)+` -Confirm:$false -ErrorAction SilentlyContinue`, 30*time.Second); err != nil {
		return fmt.Errorf("no se pudo quitar la tarea: %w", err)
	}
	return nil
}

// RunNow ejecuta la tarea ya mismo (debe estar programada).
func RunNow(sys system.System, id string) error {
	j, ok := ByID(id)
	if !ok {
		return fmt.Errorf("tarea desconocida")
	}
	if _, err := sys.PowerShell(`Start-ScheduledTask -TaskName `+psq(j.Name)+` -ErrorAction Stop`, 25*time.Second); err != nil {
		return fmt.Errorf("no se pudo ejecutar la tarea: %w", err)
	}
	return nil
}

// Info es el estado de una tarea para la interfaz.
type Info struct {
	ID         string `json:"id"`
	Registered bool   `json:"registered"`
	Day        int    `json:"day"`
	Time       string `json:"time"`
	LastRun    string `json:"lastRun"`
	NextRun    string `json:"nextRun"`
	LastResult int64  `json:"lastResult"`
	Last       string `json:"last"` // qué hizo la última vez que corrió
}

// Status consulta la tarea.
func Status(sys system.System, id string) Info {
	info := Info{ID: id, Time: "20:00"}
	j, ok := ByID(id)
	if !ok {
		return info
	}
	out, err := sys.PowerShell(`$t = Get-ScheduledTask -TaskName `+psq(j.Name)+` -ErrorAction SilentlyContinue
if ($t) { $i = Get-ScheduledTaskInfo -TaskName `+psq(j.Name)+`
  $l = ''; if ($i.LastRunTime -and $i.LastRunTime.Year -gt 2000) { $l = $i.LastRunTime.ToString('yyyy-MM-dd HH:mm') }
  $n = ''; if ($i.NextRunTime -and $i.NextRunTime.Year -gt 2000) { $n = $i.NextRunTime.ToString('yyyy-MM-dd HH:mm') }
  $tr = $t.Triggers | Select-Object -First 1
  $m = 0; $h = ''; if ($tr) { $m = [int]$tr.DaysOfWeek; $h = ([datetime]$tr.StartBoundary).ToString('HH:mm') }
  'T|' + $l + '|' + $n + '|' + $i.LastTaskResult + '|' + $m + '|' + $h }`, 25*time.Second)
	if err != nil {
		return info
	}
	return parseInfo(id, out)
}

func parseInfo(id, out string) Info {
	info := Info{ID: id, Time: "20:00"}
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "T|") {
			continue
		}
		f := strings.Split(ln, "|")
		if len(f) < 6 {
			continue
		}
		info.Registered = true
		info.LastRun, info.NextRun = f[1], f[2]
		fmt.Sscan(f[3], &info.LastResult)
		var mask int
		fmt.Sscan(f[4], &mask)
		for d := 0; d < 7; d++ { // máscara de Windows: domingo = 1, lunes = 2, martes = 4…
			if mask&(1<<d) != 0 {
				info.Day = d
				break
			}
		}
		if timeRe.MatchString(f[5]) {
			info.Time = f[5]
		}
		return info
	}
	return info
}

// ---- ejecución sin ventana (--task) ----

const logFile = "scheduled.json"

type logEntry struct {
	When string `json:"when"`
	Msg  string `json:"msg"`
}

func readLog(dir string) map[string]logEntry {
	m := map[string]logEntry{}
	if b, err := os.ReadFile(filepath.Join(dir, logFile)); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

// LastMessage devuelve lo que hizo la última ejecución de la tarea (vacío si todavía no corrió).
func LastMessage(dir, id string) string {
	e, ok := readLog(dir)[id]
	if !ok {
		return ""
	}
	return e.When + ": " + e.Msg
}

func writeLog(dir, id, msg string) {
	m := readLog(dir)
	m[id] = logEntry{When: time.Now().Format("2006-01-02 15:04"), Msg: msg}
	b, _ := json.Marshal(m)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, logFile), b, 0o644)
}

// Run hace el trabajo de una tarea (lo llama OverdriveTuner.exe --task <id>) y devuelve un resumen.
func Run(sys system.System, dataDir, id string) (string, error) {
	var msg string
	var err error
	switch id {
	case "cleanup":
		var ids []string
		for _, t := range extras.Targets() {
			if t.Safe && !t.Deep { // solo lo seguro: temporales, shaders, volcados y miniaturas
				ids = append(ids, t.ID)
			}
		}
		var freed int64
		for _, r := range extras.Run(ids) {
			freed += r.Freed
		}
		msg = fmt.Sprintf("se liberaron %.1f MB de archivos temporales.", float64(freed)/(1<<20))
	case "restore":
		if err = extras.CreateRestorePoint(sys, "OverdriveTuner (programado)"); err == nil {
			msg = "punto de restauración creado."
		}
	default:
		return "", fmt.Errorf("tarea desconocida: %s", id)
	}
	if err != nil {
		writeLog(dataDir, id, "falló: "+err.Error())
		return "", err
	}
	writeLog(dataDir, id, msg)
	return msg, nil
}
