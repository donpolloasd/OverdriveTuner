// Package rtss lee los FPS que publica RivaTuner Statistics Server (el motor de MSI Afterburner)
// en su memoria compartida «RTSSSharedMemoryV2». Solo lee: no se inyecta nada en el juego, así que
// no cambia nada para los anti-cheat (la inyección la hace RTSS, no este programa).
package rtss

import (
	"encoding/binary"
	"strings"
)

// Entry es lo que RTSS informa de un programa que está dibujando cuadros.
type Entry struct {
	PID       uint32
	Name      string // ruta del programa
	Time0     uint32 // ms (GetTickCount) del inicio del período de medición
	Time1     uint32 // ms del fin del período (se actualiza con los cuadros)
	Frames    uint32 // cuadros del período
	FrameTime uint32 // duración del último cuadro, en microsegundos
}

const signature = 0x52545353 // 'RTSS'

// Parse interpreta la memoria compartida y devuelve los programas medidos.
func Parse(b []byte) []Entry {
	if len(b) < 32 || binary.LittleEndian.Uint32(b[0:]) != signature {
		return nil
	}
	size := int(binary.LittleEndian.Uint32(b[8:]))
	off := int(binary.LittleEndian.Uint32(b[12:]))
	n := int(binary.LittleEndian.Uint32(b[16:]))
	if size < 284 || off < 0 || n < 0 || n > 4096 {
		return nil
	}
	var out []Entry
	for i := 0; i < n; i++ {
		p := off + i*size
		if p+284 > len(b) {
			break
		}
		pid := binary.LittleEndian.Uint32(b[p:])
		if pid == 0 {
			continue
		}
		name := b[p+4 : p+264]
		if z := indexZero(name); z >= 0 {
			name = name[:z]
		}
		out = append(out, Entry{
			PID: pid, Name: string(name),
			Time0:     binary.LittleEndian.Uint32(b[p+268:]),
			Time1:     binary.LittleEndian.Uint32(b[p+272:]),
			Frames:    binary.LittleEndian.Uint32(b[p+276:]),
			FrameTime: binary.LittleEndian.Uint32(b[p+280:]),
		})
	}
	return out
}

func indexZero(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return -1
}

// Find busca el programa por nombre de archivo (sin distinguir mayúsculas).
func Find(es []Entry, exe string) (Entry, bool) {
	exe = strings.ToLower(exe)
	for _, e := range es {
		n := strings.ToLower(strings.ReplaceAll(e.Name, "/", `\`))
		if i := strings.LastIndex(n, `\`); i >= 0 {
			n = n[i+1:]
		}
		if n == exe {
			return e, true
		}
	}
	return Entry{}, false
}

// FPS devuelve los FPS del período y el último tiempo de cuadro en ms. now es GetTickCount; si el
// programa dejó de dibujar hace más de 2 s, ok=false.
func (e Entry) FPS(now uint32) (fps, frameMs float64, ok bool) {
	if now-e.Time1 > 2000 && now-e.Time1 < 1<<31 {
		return 0, 0, false
	}
	if e.FrameTime > 0 {
		frameMs = float64(e.FrameTime) / 1000
	}
	if e.Time1 > e.Time0 && e.Frames > 0 {
		fps = 1000 * float64(e.Frames) / float64(e.Time1-e.Time0)
	} else if frameMs > 0 {
		fps = 1000 / frameMs
	}
	if fps <= 0 {
		return 0, 0, false
	}
	if frameMs <= 0 {
		frameMs = 1000 / fps
	}
	return fps, frameMs, true
}
