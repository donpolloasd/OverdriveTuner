//go:build !windows

package system

// New devuelve un sistema simulado fuera de Windows (solo para desarrollo).
func New() System { return NewFake() }
