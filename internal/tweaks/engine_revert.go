package tweaks

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// restoreReg devuelve un valor a un estado anterior (o lo borra si no existía).
func (e *Engine) restoreReg(op Op, existed bool, typ, data string) error {
	if existed {
		if typ == "" || strings.EqualFold(typ, "REG_NONE") {
			return nil
		}
		return e.Sys.RegSet(op.Key, op.Name, typ, data)
	}
	return e.deleteReg(op)
}

func (e *Engine) deleteReg(op Op) error {
	cur, err := system.RegGet(e.Sys, op.Key, op.Name)
	if err == nil && cur.Exists {
		if err := e.Sys.RegDeleteValue(op.Key, op.Name); err != nil {
			return err
		}
	}
	if op.RevDelKey != "" && e.Sys.RegKeyExists(op.RevDelKey) {
		// Solo se borra la clave si quedó sin valores: nunca se pierden datos ajenos.
		if left, err := e.Sys.RegEnumValues(op.RevDelKey); err == nil && len(left) == 0 {
			return e.Sys.RegDeleteKey(op.RevDelKey)
		}
	}
	return nil
}

func (e *Engine) runCmdRevert(op Op) error {
	if op.RevArgs == nil {
		return nil
	}
	to := op.Timeout
	if to == 0 {
		to = 2 * time.Minute
	}
	out, code, err := e.Sys.Exec(to, op.Exe, op.RevArgs...)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("%s (revertir) terminó con código %d: %s", op.Exe, code, strings.TrimSpace(out))
	}
	return nil
}

// revertOp deshace una operación usando la copia guardada.
func (e *Engine) revertOp(op Op, bk OpBackup) error {
	switch op.Kind {
	case OpReg:
		return e.restoreReg(op, bk.Existed, bk.Type, bk.Data)
	case OpSvc:
		if !e.Sys.RegKeyExists(svcKey + op.Svc) {
			return nil
		}
		start := op.DefStart
		if bk.Existed {
			if n, err := strconv.Atoi(strings.TrimSpace(bk.Data)); err == nil {
				start = n
			}
		}
		return e.Sys.RegSet(svcKey+op.Svc, "Start", "REG_DWORD", strconv.Itoa(start))
	case OpPS:
		_, err := e.Sys.PowerShell(op.Revert, 10*time.Minute)
		return err
	case OpCmd:
		return e.runCmdRevert(op)
	case OpPlan:
		var firstErr error
		if bk.Data != "" {
			if _, c, err := e.Sys.Exec(30*time.Second, "powercfg.exe", "/setactive", bk.Data); err != nil || c != 0 {
				// Si el plan anterior ya no existe, volver a Equilibrado.
				_, _, _ = e.Sys.Exec(30*time.Second, "powercfg.exe", "/setactive", "381b4222-f694-41f0-9685-ff5bb260df2e")
			}
		}
		if bk.Type != "" {
			if _, _, err := e.Sys.Exec(30*time.Second, "powercfg.exe", "/delete", bk.Type); err != nil {
				firstErr = err
			}
		}
		return firstErr
	}
	return nil
}

// revertDefault restaura el valor de fábrica cuando no hay copia propia.
func (e *Engine) revertDefault(op Op) error {
	switch op.Kind {
	case OpReg:
		if op.Def == Del {
			return e.deleteReg(op)
		}
		return e.Sys.RegSet(op.Key, op.Name, op.Type, op.Def)
	case OpSvc:
		if !e.Sys.RegKeyExists(svcKey + op.Svc) {
			return nil
		}
		return e.Sys.RegSet(svcKey+op.Svc, "Start", "REG_DWORD", strconv.Itoa(op.DefStart))
	case OpPS:
		_, err := e.Sys.PowerShell(op.Revert, 10*time.Minute)
		return err
	case OpCmd:
		return e.runCmdRevert(op)
	}
	return nil
}

// Revert deshace un tweak. Sin registro propio restaura los valores de fábrica.
func (e *Engine) Revert(id string) Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.revertLocked(id)
}

// revertLocked es Revert con el candado ya tomado.
func (e *Engine) revertLocked(id string) Result {
	res := Result{ID: id}
	tw, ok := ByID(id)
	if !ok {
		res.Errors = []string{"tweak desconocido: " + id}
		return res
	}
	rec, has := e.recs[id]
	for i := len(tw.Ops) - 1; i >= 0; i-- {
		op := tw.Ops[i]
		var err error
		if has {
			var bk *OpBackup
			for j := range rec.Backups {
				if rec.Backups[j].Index == i {
					bk = &rec.Backups[j]
				}
			}
			if bk == nil || !bk.Done {
				continue
			}
			err = e.revertOp(op, *bk)
		} else {
			err = e.revertDefault(op)
		}
		if err != nil {
			if op.Optional {
				res.Notes = append(res.Notes, "omitido: "+err.Error())
				continue
			}
			res.Errors = append(res.Errors, err.Error())
		}
	}
	if len(res.Errors) > 0 {
		// Se conserva el registro para poder reintentar la reversión.
		return res
	}
	delete(e.recs, id)
	if err := e.save(); err != nil {
		res.Notes = append(res.Notes, "no se pudo guardar el estado: "+err.Error())
	}
	res.OK = true
	res.Reboot = tw.Reboot
	return res
}

// DetectAll indica, para cada tweak, si los valores de registro/servicio ya
// están aplicados (aunque no los haya puesto este programa). Hace una sola
// lectura de registro para todo el catálogo.
func (e *Engine) DetectAll(list []Tweak) map[string]bool {
	type want struct {
		tweak   int
		typ     string
		data    string
		service bool
	}
	var refs []system.RegRef
	var wants []want
	for ti, tw := range list {
		for _, op := range tw.Ops {
			switch op.Kind {
			case OpReg:
				if op.Aux || op.Optional {
					continue
				}
				refs = append(refs, system.RegRef{Key: op.Key, Name: op.Name})
				wants = append(wants, want{ti, op.Type, op.Data, false})
			case OpSvc:
				refs = append(refs, system.RegRef{Key: svcKey + op.Svc, Name: "Start"})
				wants = append(wants, want{ti, "REG_DWORD", strconv.Itoa(op.Start), true})
			}
		}
	}
	res := make(map[string]bool, len(list))
	vals, err := e.Sys.RegGetMany(refs)
	if err != nil || len(vals) != len(refs) {
		return res
	}
	checked := make([]int, len(list))
	failed := make([]bool, len(list))
	for i, v := range vals {
		w := wants[i]
		if w.service && !v.Exists {
			continue // el servicio no existe en este equipo
		}
		checked[w.tweak]++
		if !v.Matches(w.typ, w.data) {
			failed[w.tweak] = true
		}
	}
	for ti, tw := range list {
		res[tw.ID] = checked[ti] > 0 && !failed[ti]
	}
	return res
}

// Detect es DetectAll para un único tweak.
func (e *Engine) Detect(tw Tweak) bool { return e.DetectAll([]Tweak{tw})[tw.ID] }

// RevertAll revierte todo lo aplicado por el programa, del más nuevo al más
// viejo. Devuelve un resultado por tweak.
func (e *Engine) RevertAll() []Result {
	e.mu.Lock()
	type it struct {
		id string
		at time.Time
	}
	var list []it
	for id, r := range e.recs {
		list = append(list, it{id, r.Applied})
	}
	e.mu.Unlock()
	sort.Slice(list, func(i, j int) bool { return list[i].at.After(list[j].at) })
	out := make([]Result, 0, len(list))
	for _, x := range list {
		out = append(out, e.Revert(x.id))
	}
	return out
}
