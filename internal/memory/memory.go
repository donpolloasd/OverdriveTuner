// Package memory libera memoria RAM como la opción «Acelerar» de Microsoft PC Manager: vacía el
// conjunto de trabajo de los programas abiertos (Windows les devuelve solo lo que usan) y vacía la
// lista de memoria en espera. Usa funciones de Windows; no instala drivers ni toca el juego.
package memory

// Result es lo que se informa después de liberar memoria.
type Result struct {
	TotalMB     int    `json:"totalMB"`
	BeforeMB    int    `json:"beforeMB"` // memoria libre antes
	AfterMB     int    `json:"afterMB"`  // memoria libre después
	Trimmed     int    `json:"trimmed"`  // programas a los que se les vació el conjunto de trabajo
	Standby     bool   `json:"standby"`  // se vació la lista en espera
	SkippedGame string `json:"skippedGame,omitempty"`
	Note        string `json:"note,omitempty"`
}

// Options controla qué se protege.
type Options struct {
	// Game es el juego abierto ahora («» si no hay): no se toca y no se vacía la lista en espera,
	// porque eso provocaría tirones mientras se juega.
	Game string
}

// neverTouch son procesos del sistema que no se recortan.
var neverTouch = map[string]bool{
	"system": true, "registry": true, "smss.exe": true, "csrss.exe": true, "wininit.exe": true, "services.exe": true,
	"lsass.exe": true, "winlogon.exe": true, "memcompression": true, "memory compression": true, "dwm.exe": true,
	"fontdrvhost.exe": true, "audiodg.exe": true,
}
