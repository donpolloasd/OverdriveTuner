package main

import "testing"

func TestReleasesSince(t *testing.T) {
	if len(releasesSince("2.4.1")) != 10 || releasesSince("2.4.1")[0].Version != "2.6.9" {
		t.Fatalf("desde 2.4.1 debe mostrar 2.6.9, 2.6.8, 2.6.7, 2.6.6, 2.6.5, 2.6.4, 2.6.3, 2.6.2, 2.6.0 y 2.5.0: %+v", releasesSince("2.4.1"))
	}
	if len(releasesSince("2.2.2")) != 12 {
		t.Fatalf("desde 2.2.2 debe mostrar 2.3.0, 2.4.1, 2.5.0 y 2.6.0")
	}
	if len(releasesSince("")) != len(releases) || len(releasesSince("raro")) != len(releases) {
		t.Fatal("sin versión conocida se muestran todas")
	}
	if len(releasesSince("2.6.9")) != 0 || len(releasesSince("3.0")) != 0 {
		t.Fatal("la misma versión o una más nueva no tiene cambios")
	}
	if !newer("2.10.0", "2.9.5") || newer("2.4", "2.4.0") {
		t.Fatal("comparación de versiones incorrecta")
	}
}
