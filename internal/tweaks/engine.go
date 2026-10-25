package tweaks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/system"
)

// OpBackup guarda el estado previo de una operación para poder revertirla.
type OpBackup struct {
	Index   int    `json:"i"`
	Kind    OpKind `json:"k"`
	Existed bool   `json:"e"`
	Type    string `json:"t,omitempty"`
	Data    string `json:"d,omitempty"`
	Done    bool   `json:"done"`
}

// Record es el estado persistente de un tweak aplicado.
type Record struct {
	ID      string     `json:"id"`
	Applied time.Time  `json:"applied"`
	Backups []OpBackup `json:"backups"`
	Reboot  bool       `json:"reboot"`
}

// Result es el resultado de aplicar o revertir.
type Result struct {
	ID     string   `json:"id"`
	OK     bool     `json:"ok"`
	Errors []string `json:"errors,omitempty"`
	Notes  []string `json:"notes,omitempty"`
	Reboot bool     `json:"reboot"`
}

// Engine aplica y revierte tweaks guardando copias de seguridad.
type Engine struct {
	Sys  system.System
	Dir  string // carpeta de datos (state.json y registros)
	mu   sync.Mutex
	recs map[string]*Record
}

// NewEngine crea un motor y carga el estado guardado.
func NewEngine(sys system.System, dir string) (*Engine, error) {
	e := &Engine{Sys: sys, Dir: dir, recs: map[string]*Record{}}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := e.load(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) statePath() string { return filepath.Join(e.Dir, "state.json") }

func (e *Engine) load() error {
	b, err := os.ReadFile(e.statePath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []*Record
	if err := json.Unmarshal(b, &list); err != nil {
		// Un estado corrupto no debe impedir abrir el programa: se conserva una copia.
		_ = os.Rename(e.statePath(), e.statePath()+".corrupto")
		return nil
	}
	for _, r := range list {
		e.recs[r.ID] = r
	}
	return nil
}

// save escribe de forma atómica (archivo temporal + renombrado).
func (e *Engine) save() error {
	list := make([]*Record, 0, len(e.recs))
	for _, r := range e.recs {
		list = append(list, r)
	}
	b, err := json.MarshalIndent(list, "", " ")
	if err != nil {
		return err
	}
	tmp := e.statePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, e.statePath())
}

// Applied indica si el tweak fue aplicado por este programa.
func (e *Engine) Applied(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.recs[id]
	return ok
}

// AppliedIDs devuelve los tweaks aplicados.
func (e *Engine) AppliedIDs() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, 0, len(e.recs))
	for id := range e.recs {
		out = append(out, id)
	}
	return out
}

var guidRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

const planName = "OverdriveTuner Gaming"

// Apply aplica un tweak. Si una operación falla, deshace las anteriores.
func (e *Engine) Apply(id string) Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	res := Result{ID: id}
	tw, ok := ByID(id)
	if !ok {
		res.Errors = []string{"tweak desconocido: " + id}
		return res
	}
	if _, already := e.recs[id]; already {
		res.OK = true
		res.Notes = []string{"ya estaba aplicado"}
		return res
	}
	if tw.Group != "" {
		// Opciones excluyentes (p. ej. varios DNS): se revierte la que estuviera aplicada.
		for _, other := range All() {
			if other.Group == tw.Group && other.ID != tw.ID {
				if _, on := e.recs[other.ID]; on {
					if r := e.revertLocked(other.ID); !r.OK {
						res.Errors = append(res.Errors, "no se pudo revertir «"+other.Name+"» antes: "+strings.Join(r.Errors, "; "))
						return res
					}
				}
			}
		}
	}
	rec := &Record{ID: id, Applied: time.Now(), Reboot: tw.Reboot}
	for i, op := range tw.Ops {
		bk, err := e.applyOp(i, op)
		rec.Backups = append(rec.Backups, bk)
		if err != nil {
			if op.Optional {
				res.Notes = append(res.Notes, "omitido: "+err.Error())
				continue
			}
			res.Errors = append(res.Errors, err.Error())
			// Rollback de lo ya hecho.
			e.rollback(tw, rec)
			return res
		}
	}
	e.recs[id] = rec
	if err := e.save(); err != nil {
		res.Notes = append(res.Notes, "no se pudo guardar el estado: "+err.Error())
	}
	res.OK = true
	res.Reboot = tw.Reboot
	return res
}

func (e *Engine) rollback(tw Tweak, rec *Record) {
	for i := len(rec.Backups) - 1; i >= 0; i-- {
		bk := rec.Backups[i]
		if bk.Done {
			_ = e.revertOp(tw.Ops[bk.Index], bk)
		}
	}
}

func (e *Engine) applyOp(i int, op Op) (OpBackup, error) {
	bk := OpBackup{Index: i, Kind: op.Kind}
	switch op.Kind {
	case OpReg:
		prev, err := system.RegGet(e.Sys, op.Key, op.Name)
		if err != nil {
			return bk, fmt.Errorf("leer %s\\%s: %w", op.Key, op.Name, err)
		}
		bk.Existed, bk.Type, bk.Data = prev.Exists, prev.Type, prev.Data
		if err := e.Sys.RegSet(op.Key, op.Name, op.Type, op.Data); err != nil {
			return bk, err
		}
		bk.Done = true
	case OpSvc:
		key := svcKey + op.Svc
		if !e.Sys.RegKeyExists(key) {
			return bk, nil // el servicio no existe en este equipo: nada que hacer
		}
		prev, err := system.RegGet(e.Sys, key, "Start")
		if err != nil {
			return bk, err
		}
		bk.Existed, bk.Type, bk.Data = prev.Exists, prev.Type, prev.Data
		if err := e.Sys.RegSet(key, "Start", "REG_DWORD", strconv.Itoa(op.Start)); err != nil {
			return bk, err
		}
		if op.Start == 4 {
			// Detenerlo ya mismo; si no estaba corriendo no es un error.
			_, _, _ = e.Sys.Exec(30*time.Second, "sc.exe", "stop", op.Svc)
		}
		bk.Done = true
	case OpPS:
		out, err := e.Sys.PowerShell(op.Apply, 10*time.Minute)
		if err != nil {
			return bk, err
		}
		bk.Done = true
		_ = out
	case OpCmd:
		to := op.Timeout
		if to == 0 {
			to = 2 * time.Minute
		}
		out, code, err := e.Sys.Exec(to, op.Exe, op.Args...)
		if err != nil {
			return bk, err
		}
		if code != 0 {
			return bk, fmt.Errorf("%s terminó con código %d: %s", op.Exe, code, strings.TrimSpace(out))
		}
		bk.Done = true
	case OpPlan:
		return e.applyPlan(bk)
	}
	return bk, nil
}

func (e *Engine) applyPlan(bk OpBackup) (OpBackup, error) {
	out, _, err := e.Sys.Exec(30*time.Second, "powercfg.exe", "/getactivescheme")
	if err != nil {
		return bk, err
	}
	prev := guidRe.FindString(out)
	if prev == "" {
		return bk, fmt.Errorf("no se pudo leer el plan de energía actual")
	}
	bk.Data = prev
	// Duplicar «Máximo rendimiento»; si el esquema no existe (portátiles modernos), se duplica el actual.
	const ultimate = "e9a42b02-d5df-448d-aa00-03f14749eb61"
	out, code, _ := e.Sys.Exec(30*time.Second, "powercfg.exe", "/duplicatescheme", ultimate)
	newGUID := guidRe.FindString(out)
	if code != 0 || newGUID == "" {
		out, code, _ = e.Sys.Exec(30*time.Second, "powercfg.exe", "/duplicatescheme", "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c")
		newGUID = guidRe.FindString(out)
		if code != 0 || newGUID == "" {
			return bk, fmt.Errorf("no se pudo crear el plan de energía: %s", strings.TrimSpace(out))
		}
	}
	bk.Type = newGUID // guardamos el GUID del plan creado para borrarlo al revertir
	steps := [][]string{
		{"/changename", newGUID, planName, "Plan creado por OverdriveTuner"},
		{"/setacvalueindex", newGUID, "SUB_PROCESSOR", "PROCTHROTTLEMIN", "100"},
		{"/setacvalueindex", newGUID, "SUB_PROCESSOR", "PROCTHROTTLEMAX", "100"},
		{"/setacvalueindex", newGUID, "SUB_PROCESSOR", "CPMINCORES", "100"},
		{"/setacvalueindex", newGUID, "SUB_PROCESSOR", "PERFBOOSTMODE", "2"},
		{"/setacvalueindex", newGUID, "SUB_PCIEXPRESS", "ASPM", "0"},
		{"/setacvalueindex", newGUID, "2a737441-1930-4402-8d77-b2bebba308a3", "48e6b7a6-50f5-4782-a5d4-53bb8f07e226", "0"},
		{"/setacvalueindex", newGUID, "SUB_DISK", "DISKIDLE", "0"},
		{"/setacvalueindex", newGUID, "SUB_SLEEP", "STANDBYIDLE", "0"},
		{"/setactive", newGUID},
	}
	for _, a := range steps {
		// Algunos ajustes no existen en todos los equipos: solo /setactive es obligatorio.
		o, c, err := e.Sys.Exec(30*time.Second, "powercfg.exe", a...)
		if err != nil {
			return bk, err
		}
		if c != 0 && a[0] == "/setactive" {
			_, _, _ = e.Sys.Exec(30*time.Second, "powercfg.exe", "/delete", newGUID)
			return bk, fmt.Errorf("no se pudo activar el plan: %s", strings.TrimSpace(o))
		}
	}
	bk.Done = true
	return bk, nil
}
