package app

import (
	"testing"

	"fpsforge/internal/scan"
	"fpsforge/internal/system"
)

type winFake struct{ *system.Fake }

func (winFake) Name() string { return "windows" }

func TestHardwareCache(t *testing.T) {
	s := &Server{Sys: winFake{system.NewFake()}, Dir: t.TempDir()}
	if _, ok := s.loadHardwareCache(); ok {
		t.Fatal("sin archivo no debe haber caché")
	}
	s.saveHardwareCache(scan.Hardware{CPU: "Ryzen", Computer: "PC"})
	h, ok := s.loadHardwareCache()
	if !ok || h.CPU != "Ryzen" {
		t.Fatalf("la caché debía devolver lo guardado: %+v %v", h, ok)
	}
	s2 := &Server{Sys: winFake{system.NewFake()}, Dir: t.TempDir()}
	s2.saveHardwareCache(scan.Hardware{CPU: "No detectado"})
	s2.saveHardwareCache(scan.Hardware{CPU: "x", Demo: true})
	if _, ok := s2.loadHardwareCache(); ok {
		t.Fatal("no se deben guardar escaneos de respaldo ni de demostración")
	}
}
