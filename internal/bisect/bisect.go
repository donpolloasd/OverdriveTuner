// Package bisect ayuda a encontrar qué ajuste empeoró el rendimiento:
// revierte la mitad de los sospechosos, el usuario mide, y se repite con la
// mitad que corresponda hasta quedar uno.
package bisect

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

// State es el estado de la búsqueda (se guarda en bisect.json).
type State struct {
	Active     bool     `json:"active"`
	Candidates []string `json:"candidates"` // todos los ajustes al empezar
	Suspects   []string `json:"suspects"`   // donde puede estar el culpable
	Tested     []string `json:"tested"`     // la mitad que se está probando revertida
	Reverted   []string `json:"reverted"`   // lo que está revertido ahora mismo
	Round      int      `json:"round"`
	Culprit    string   `json:"culprit"`
	Done       bool     `json:"done"`
	Message    string   `json:"message"`
}

// Actions son los cambios que hay que hacer en el sistema.
type Actions struct {
	Revert []string `json:"revert"`
	Apply  []string `json:"apply"`
}

func diff(cur, target []string) Actions {
	in := func(l []string, s string) bool {
		for _, x := range l {
			if x == s {
				return true
			}
		}
		return false
	}
	var a Actions
	for _, c := range cur {
		if !in(target, c) {
			a.Apply = append(a.Apply, c)
		}
	}
	for _, t := range target {
		if !in(cur, t) {
			a.Revert = append(a.Revert, t)
		}
	}
	return a
}

func half(s []string) []string {
	n := (len(s) + 1) / 2
	return append([]string{}, s[:n]...)
}

func minus(a, b []string) []string {
	var out []string
	for _, x := range a {
		keep := true
		for _, y := range b {
			if x == y {
				keep = false
			}
		}
		if keep {
			out = append(out, x)
		}
	}
	return out
}

// Start inicia la búsqueda con los ajustes aplicados.
func Start(applied []string) (State, Actions, error) {
	c := append([]string{}, applied...)
	sort.Strings(c)
	if len(c) < 2 {
		return State{}, Actions{}, errors.New("hacen falta al menos 2 ajustes aplicados para buscar cuál es el culpable")
	}
	s := State{Active: true, Candidates: c, Suspects: c, Round: 1}
	return s.plan()
}

func (s State) plan() (State, Actions, error) {
	s.Tested = half(s.Suspects)
	a := diff(s.Reverted, s.Tested)
	s.Reverted = append([]string{}, s.Tested...)
	s.Message = "Medí ahora (mismas apps abiertas que antes) y decime si el problema desapareció."
	return s, a, nil
}

// Answer registra el resultado de la medición. good = el rendimiento volvió a la normalidad.
func (s State) Answer(good bool) (State, Actions, error) {
	if !s.Active || s.Done {
		return s, Actions{}, errors.New("no hay una búsqueda en curso")
	}
	if good {
		s.Suspects = append([]string{}, s.Tested...)
		if len(s.Suspects) == 1 {
			s.Culprit, s.Done, s.Active = s.Suspects[0], true, false
			s.Message = "Encontrado: este ajuste es el que empeora el rendimiento. Queda revertido; el resto se volvió a aplicar."
			a := diff(s.Reverted, []string{s.Culprit})
			s.Reverted = []string{s.Culprit}
			return s, a, nil
		}
	} else {
		s.Suspects = minus(s.Suspects, s.Tested)
		if len(s.Suspects) == 0 {
			// Revertir la mitad no cambió nada y no quedan sospechosos.
			s.Done, s.Active = true, false
			s.Message = "Ningún ajuste individual explica el problema (puede ser ruido, otra causa del sistema o varios ajustes a la vez). Se volvió a aplicar todo."
			a := diff(s.Reverted, nil)
			s.Reverted = nil
			return s, a, nil
		}
	}
	s.Round++
	return s.plan()
}

// Cancel devuelve todo a como estaba antes de empezar.
func (s State) Cancel() (State, Actions) {
	a := diff(s.Reverted, nil)
	s.Reverted, s.Active, s.Done = nil, false, false
	s.Message = "Búsqueda cancelada; se volvieron a aplicar los ajustes."
	return s, a
}

// Load / Save persisten el estado.
func Load(dir string) State {
	var s State
	if b, err := os.ReadFile(filepath.Join(dir, "bisect.json")); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func Save(dir string, s State) error {
	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "bisect.json"), b, 0o644)
}
