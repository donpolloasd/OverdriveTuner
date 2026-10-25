// Package presentmon mide los tiempos de cuadro de un juego con PresentMon
// (Intel, licencia MIT). PresentMon solo escucha eventos de Windows (ETW):
// no se mete dentro del juego, por eso es compatible con los anti-cheat.
package presentmon

import (
	"bufio"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed PresentMon-2.3.0-x64.exe
var exeBytes []byte

// Hash esperado del ejecutable incluido (PresentMon 2.3.0, release oficial).
const Hash = "690533cf5c2591ef571a3037aa4c205c8e75860ef3f56927bf3a98acf7198485"

// Verify comprueba que el ejecutable incluido es el esperado.
func Verify() error {
	h := sha256.Sum256(exeBytes)
	if hex.EncodeToString(h[:]) != Hash {
		return fmt.Errorf("PresentMon incluido no coincide con el hash esperado")
	}
	return nil
}

// Extract escribe PresentMon en dir (si falta o está alterado) y devuelve su ruta.
func Extract(dir string) (string, error) {
	if err := Verify(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, "PresentMon.exe")
	if b, err := os.ReadFile(p); err == nil {
		if h := sha256.Sum256(b); hex.EncodeToString(h[:]) == Hash {
			return p, nil
		}
	}
	if err := os.WriteFile(p+".tmp", exeBytes, 0o755); err != nil {
		return "", err
	}
	return p, os.Rename(p+".tmp", p)
}

// Args devuelve los argumentos para medir un proceso.
func Args(process string) []string { return argsFor(process, "OverdriveTuner") }

func argsFor(process, session string) []string {
	return []string{"--process_name", process, "--output_stdout", "--no_console_stats",
		"--stop_existing_session", "--session_name", session, "--no_track_gpu",
		"--no_track_input", "--no_track_display"}
}

var (
	failMu     sync.Mutex
	failStreak int
	lastErr    string
)

// LastError devuelve el último mensaje de error que escribió PresentMon (vacío si no hubo).
func LastError() string { failMu.Lock(); defer failMu.Unlock(); return lastErr }

// Diag anota una línea en el archivo de diagnóstico.
func Diag(format string, a ...any) { diagf(format, a...) }

// Parser interpreta las líneas CSV de PresentMon (versiones 1 y 2). Descarta lo que no es un
// cuadro real del juego, para no mostrar FPS inflados:
//   - cuadros generados (FrameType distinto de «Application»: AFMF, DLSS/FSR frame generation…),
//   - presentaciones de cadenas de intercambio secundarias (overlays, vistas auxiliares): solo se
//     cuenta la cadena dominante (la que más presenta), y
//   - tiempos absurdamente chicos (filas duplicadas).
type Parser struct {
	ftCol, chainCol, typeCol, startCol int
	seen, emitted                      int
	lenient                            bool    // los filtros no dejaron pasar nada: se usan los tiempos tal cual
	lastStart                          float64 // inicio (CPUStartTime, en ms) del último cuadro contado
	haveStart                          bool
	ready                              bool
	Now                                func() time.Time // para pruebas

	winStart time.Time
	counts   map[string]int
	dominant string
}

// minFrameMs descarta duplicados: 0,25 ms serían 4000 FPS en una misma cadena.
const minFrameMs = 0.25

func (p *Parser) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// Line devuelve el tiempo de cuadro en ms de una línea de datos. La primera
// línea (encabezado) se consume y devuelve ok=false.
func (p *Parser) Line(line string) (ms float64, ok bool) {
	ms, ok = p.filtered(line)
	if ok {
		p.emitted++
	}
	return ms, ok
}

func (p *Parser) filtered(line string) (ms float64, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, false
	}
	f := strings.Split(line, ",")
	if !p.ready {
		for i, c := range f {
			switch strings.ToLower(strings.TrimSpace(c)) {
			case "msbetweenpresents", "frametime":
				if p.ftCol == 0 || strings.EqualFold(strings.TrimSpace(c), "msBetweenPresents") {
					p.ftCol = i + 1 // 0 = no encontrado
				}
			case "swapchainaddress":
				p.chainCol = i + 1
			case "frametype":
				p.typeCol = i + 1
			case "cpustarttime":
				p.startCol = i + 1
			}
		}
		if p.ftCol > 0 {
			p.ready = true
		}
		return 0, false
	}
	i := p.ftCol - 1
	if i >= len(f) {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(f[i]), 64)
	if err != nil || v <= 0 || v > 5000 {
		return 0, false
	}
	p.seen++
	if !p.lenient && p.seen > 300 && p.emitted == 0 {
		p.lenient = true // red de seguridad: mejor FPS algo imprecisos que ninguno
	}
	if p.lenient {
		return v, v >= minFrameMs
	}
	if p.typeCol > 0 && p.typeCol <= len(f) {
		if t := strings.TrimSpace(f[p.typeCol-1]); t != "" && !strings.EqualFold(t, "Application") && !strings.EqualFold(t, "Unspecified") {
			return 0, false // cuadro generado por el driver o el juego, no uno que renderizó la GPU
		}
	}
	if p.chainCol > 0 && p.chainCol <= len(f) {
		chain := strings.TrimSpace(f[p.chainCol-1])
		now := p.now()
		if p.counts == nil {
			p.counts = map[string]int{}
			p.winStart = now
		}
		p.counts[chain]++
		if p.dominant == "" {
			p.dominant = chain
		}
		if now.Sub(p.winStart) >= time.Second {
			best, bn := p.dominant, p.counts[p.dominant]
			for c, n := range p.counts {
				if n > bn {
					best, bn = c, n
				}
			}
			p.dominant = best
			p.counts = map[string]int{}
			p.winStart = now
		}
		if chain != p.dominant {
			return 0, false
		}
	}
	// PresentMon 2.x emite, antes de cada cuadro real, una fila «de espera» de ~0,2 ms que comparte
	// el instante de inicio (CPUStartTime) con el cuadro siguiente. Contarla duplicaba los FPS.
	// En vez de fiarse de cómo viene cada fila, el tiempo de cuadro se calcula como la distancia entre
	// inicios distintos: filas con el mismo inicio (o a menos de minFrameMs) son el mismo cuadro.
	if p.startCol == 0 || p.startCol > len(f) {
		return v, v >= minFrameMs
	}
	st, err := strconv.ParseFloat(strings.TrimSpace(f[p.startCol-1]), 64)
	if err != nil {
		return v, v >= minFrameMs
	}
	if !p.haveStart {
		p.lastStart, p.haveStart = st, true
		return 0, false
	}
	d := st - p.lastStart
	if d < 0 || d > 5000 { // el reloj saltó: se vuelve a empezar
		p.lastStart = st
		return 0, false
	}
	if d < minFrameMs { // misma fila de cuadro o fila de espera: no mueve el inicio
		return 0, false
	}
	p.lastStart = st
	return d, true
}

// Stats devuelve cuántas filas válidas llegaron y cuántos cuadros salieron de ellas.
func (p *Parser) Stats() (rows, frames int, lenient bool) { return p.seen, p.emitted, p.lenient }

// Runner ejecuta PresentMon y entrega los tiempos de cuadro.
type Runner struct {
	cmd *exec.Cmd
}

// DiagPath, si no está vacío, es el archivo donde se anota qué hace PresentMon: argumentos, errores,
// líneas crudas (al empezar y en pleno juego) y estadísticas cada pocos segundos. Sirve para
// diagnosticar si los FPS no coinciden con los del juego.
var DiagPath string

var (
	diagMu    sync.Mutex
	diagReset bool
)

func diagf(format string, a ...any) {
	if DiagPath == "" {
		return
	}
	diagMu.Lock()
	defer diagMu.Unlock()
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if !diagReset { // la primera vez de cada ejecución del overlay empieza un archivo nuevo
		flags |= os.O_TRUNC
		diagReset = true
	}
	f, err := os.OpenFile(DiagPath, flags, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, a...))
}

// cleanupSessions detiene las sesiones de seguimiento (ETW) «OverdriveTuner*» que hayan quedado
// abiertas por un PresentMon cerrado a la fuerza: una sesión huérfana puede impedir abrir otra.
func cleanupSessions() {
	c := exec.Command("logman.exe", "query", "-ets")
	hide(c)
	out, err := c.Output()
	if err != nil {
		return
	}
	for _, l := range strings.Split(string(out), "\n") {
		f := strings.Fields(l)
		if len(f) > 0 && strings.HasPrefix(f[0], "OverdriveTuner") {
			st := exec.Command("logman.exe", "stop", f[0], "-ets")
			hide(st)
			res, serr := st.CombinedOutput()
			diagf("sesión vieja %q detenida: %v %s", f[0], serr, strings.TrimSpace(string(res)))
		}
	}
}

// Start lanza PresentMon para process y llama a onFrame por cada cuadro.
// Termina cuando se llama a Stop o PresentMon se cierra.
func Start(exe, process string, onFrame func(ms float64), onExit func(error)) (*Runner, error) {
	failMu.Lock()
	session := "OverdriveTuner"
	if failStreak > 0 { // tras un fallo se prueba con otro nombre de sesión, por si la anterior quedó trabada
		session = fmt.Sprintf("OverdriveTuner-%d", failStreak)
	}
	failMu.Unlock()
	cleanupSessions()
	args := argsFor(process, session)
	cmd := exec.Command(exe, args...)
	hide(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		diagf("no se pudo abrir la salida de PresentMon: %v", err)
		return nil, err
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		errPipe = nil
	}
	if err := cmd.Start(); err != nil {
		diagf("PresentMon no arrancó: %v", err)
		return nil, err
	}
	diagf("PresentMon arrancado (pid %d): %s %s", cmd.Process.Pid, exe, strings.Join(args, " "))
	if errPipe != nil {
		go func() {
			sc := bufio.NewScanner(errPipe)
			for n := 0; sc.Scan() && n < 60; n++ {
				diagf("[stderr] %s", sc.Text())
				failMu.Lock()
				lastErr = sc.Text()
				failMu.Unlock()
			}
			_, _ = io.Copy(io.Discard, errPipe)
		}()
	}
	go func() {
		var p Parser
		sc := bufio.NewScanner(out)
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		lines, midLeft := 0, 0
		var recv atomic.Int64
		done := make(chan struct{})
		go func() { // avisa si PresentMon no entrega nada (las estadísticas solo salen cuando llegan filas)
			tk := time.NewTicker(10 * time.Second)
			defer tk.Stop()
			for {
				select {
				case <-done:
					return
				case <-tk.C:
					if recv.Load() == 0 {
						diagf("[estado] PresentMon sigue sin entregar ninguna línea")
					}
				}
			}
		}()
		began := time.Now()
		lastStat := began
		var lastSeen, lastEmitted int
		var sumMs float64
		var nMs int
		for sc.Scan() {
			t := sc.Text()
			lines++
			recv.Store(int64(lines))
			if lines <= 40 {
				diagf("[crudo %d] %s", lines, t)
			}
			if lines == 3000 { // ya en pleno juego: se guarda otra tanda de líneas crudas
				midLeft = 60
			}
			if midLeft > 0 {
				diagf("[crudo %d] %s", lines, t)
				midLeft--
			}
			if ms, ok := p.Line(t); ok {
				onFrame(ms)
				sumMs += ms
				nMs++
			}
			if now := time.Now(); now.Sub(lastStat) >= 5*time.Second {
				rows, frames, len_ := p.Stats()
				avg := 0.0
				if nMs > 0 {
					avg = sumMs / float64(nMs)
				}
				secs := now.Sub(lastStat).Seconds()
				fps := 0.0
				if avg > 0 {
					fps = 1000 / avg
				}
				diagf("[stats] filas/s=%.0f cuadros/s=%.0f FPS(promedio)=%.0f total filas=%d cuadros=%d tolerante=%v",
					float64(rows-lastSeen)/secs, float64(frames-lastEmitted)/secs, fps, rows, frames, len_)
				lastStat, lastSeen, lastEmitted, sumMs, nMs = now, rows, frames, 0, 0
			}
		}
		close(done)
		rows, frames, len_ := p.Stats()
		werr := cmd.Wait()
		failMu.Lock()
		if lines == 0 {
			failStreak++
		} else {
			failStreak, lastErr = 0, ""
		}
		failMu.Unlock()
		diagf("PresentMon terminó tras %.0f s: %v (líneas=%d filas=%d cuadros=%d tolerante=%v)",
			time.Since(began).Seconds(), werr, lines, rows, frames, len_)
		onExit(werr)
	}()
	return &Runner{cmd: cmd}, nil
}

// Stop cierra PresentMon.
func (r *Runner) Stop() {
	if r != nil && r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
}
