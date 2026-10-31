package overlay

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/bench"
	"fpsforge/internal/fpslog"
	"fpsforge/internal/procs"
	"fpsforge/internal/rtss"
	"fpsforge/internal/system"
	"fpsforge/internal/telemetry"
)

const (
	maxFrames   = 4000
	maxRecord   = 1_500_000 // ~ 100 minutos a 240 FPS
	graphPoints = 1500
)

// Deps son las piezas externas del motor (se reemplazan en las pruebas).
type Deps struct {
	Dir         string
	Sys         system.System
	GameRunning func(exe string) bool
	// FindGame devuelve el programa del juego abierto («» si no hay); se usa con la detección automática.
	FindGame func() string
	// StartPM empieza a medir cuadros y devuelve la función para detenerlo.
	StartPM func(process string, onFrame func(ms float64), onExit func(error)) (func(), error)
	// Exclusive dice si hay un juego en pantalla completa exclusiva (nil: no se consulta).
	Exclusive func() bool
	Now       func() time.Time
	PID       int
	Quit      func()
}

// Engine reúne mediciones, cuadros, grabación y perfiles.
type Engine struct {
	d  Deps
	mu sync.Mutex

	cfg         Config
	sample      telemetry.Sample
	recent      []telemetry.Sample
	frames      []float64
	lastFrame   time.Time
	rec         bool
	recLabel    string
	recBuf      []float64
	gameRun     bool
	pmStop      func()
	pmGame      string
	pmGen       int       // se incrementa en cada partida: invalida reinicios viejos
	pmStarted   time.Time // cuándo se lanzó la medición
	visible     bool
	msg         string
	lastSession string
	lastSeq     int
	profile     *profileState
	hkErrs      []string
	alertCPU    bool
	alertGPU    bool
	exclusive   bool   // el juego está en pantalla completa exclusiva (el overlay no puede verse)
	exclSince   int64  // desde cuándo (Unix ms): identifica cada vez que pasa, para avisar una sola vez
	lang        string // idioma de la interfaz (se relee cada segundo por si lo cambian en Configuración)
}

// NewEngine crea el motor.
func NewEngine(d Deps) *Engine {
	if d.Now == nil {
		d.Now = time.Now
	}
	e := &Engine{d: d, cfg: LoadConfig(d.Dir), visible: true, lang: UILang(d.Dir)}
	// Ignora una orden vieja que haya quedado en disco.
	if c, ok := loadCmd(d.Dir); ok {
		e.lastSeq = c.Seq
	}
	return e
}

// OnSample recibe una medición de CPU/GPU/RAM.
func (e *Engine) OnSample(s telemetry.Sample) {
	// Si Windows no da la temperatura, se toma la que publica MSI Afterburner (solo lectura).
	if s.CPUTemp == telemetry.Unknown || s.GPUTemp == telemetry.Unknown {
		ss := safeSensors()
		if v, ok := rtss.CPUTemp(ss); ok && s.CPUTemp == telemetry.Unknown {
			s.CPUTemp = v
		}
		if v, ok := rtss.GPUTemp(ss); ok && s.GPUTemp == telemetry.Unknown {
			s.GPUTemp = v
		}
	}
	e.mu.Lock()
	e.alertCPU = heat(e.alertCPU, s.CPUTemp, e.cfg.AlertCPU)
	e.alertGPU = heat(e.alertGPU, s.GPUTemp, e.cfg.AlertGPU)
	e.sample = s
	e.recent = append(e.recent, s)
	if len(e.recent) > 20 {
		e.recent = e.recent[len(e.recent)-20:]
	}
	e.mu.Unlock()
}

// heat dice si hay que avisar de temperatura. Con histéresis de 3 °C: una vez encendido, el aviso
// no se apaga hasta que baja del límite, para que no parpadee cuando la temperatura oscila.
func heat(on bool, t float64, limit int) bool {
	if t == telemetry.Unknown || t <= 0 || limit <= 0 {
		return false
	}
	if on {
		return t >= float64(limit-3)
	}
	return t >= float64(limit)
}

// OnFrame recibe el tiempo de un cuadro en ms.
func (e *Engine) OnFrame(ms float64) {
	e.mu.Lock()
	e.frames = append(e.frames, ms)
	if len(e.frames) > maxFrames {
		e.frames = append(e.frames[:0], e.frames[len(e.frames)-maxFrames/2:]...)
	}
	e.lastFrame = e.d.Now()
	if e.rec && len(e.recBuf) < maxRecord {
		e.recBuf = append(e.recBuf, ms)
	}
	e.mu.Unlock()
}

// SetVisible lo llama la ventana al usar el atajo.
func (e *Engine) SetVisible(v bool) { e.mu.Lock(); e.visible = v; e.mu.Unlock() }

// Snapshot es lo que dibuja la ventana.
type Snapshot struct {
	Cfg         Config
	Sample      telemetry.Sample
	FPS, Low1   float64
	Spark       []float32
	GameRunning bool
	Recording   bool
	Visible     bool
	AlertCPU    bool      // el procesador pasó el límite de temperatura
	AlertGPU    bool      // la gráfica pasó el límite de temperatura
	Game        string    // juego que se está midiendo (para la fila «Juego»)
	Lang        string    // idioma de los textos del panel: es o en
	Now         time.Time // hora (para la fila «Hora»)
}

// liveFPS promedia los últimos windowMs milisegundos de cuadros.
func liveFPS(fr []float64, windowMs float64) float64 {
	sum, n := 0.0, 0
	for i := len(fr) - 1; i >= 0 && sum < windowMs; i-- {
		sum += fr[i]
		n++
	}
	if sum <= 0 {
		return 0
	}
	return float64(n) * 1000 / sum
}

// liveLow1 es el «1% low» del overlay: los FPS del percentil 99 de los tiempos de cuadro de los últimos
// segundos (el método de CapFrameX y de la mayoría de los análisis). Es más estable que promediar el
// peor 1%, donde un solo pico domina el número. Las corridas grabadas siguen usando el promedio del peor 1%.
func liveLow1(fr []float64) float64 {
	if len(fr) < 50 {
		r, _ := bench.FromFrameTimes(fr)
		return r.Low1
	}
	s := append([]float64(nil), fr...)
	sort.Float64s(s)
	i := int(math.Ceil(0.99*float64(len(s)))) - 1
	if i < 0 {
		i = 0
	}
	if s[i] <= 0 {
		return 0
	}
	return 1000 / s[i]
}

// Snapshot devuelve una copia consistente del estado actual.
func (e *Engine) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked()
}

func (e *Engine) snapshotLocked() Snapshot {
	s := Snapshot{Cfg: e.cfg, Sample: e.sample, GameRunning: e.gameRun, Recording: e.rec, Visible: e.visible, AlertCPU: e.alertCPU, AlertGPU: e.alertGPU,
		Lang: e.lang, Now: e.d.Now()}
	if e.gameRun {
		s.Game = e.pmGame
	}
	if e.d.Now().Sub(e.lastFrame) < 2*time.Second && len(e.frames) > 0 {
		s.FPS = liveFPS(e.frames, fpsWindowMs(e.cfg.UpdateMs))
		tail := e.frames
		if len(tail) > 3000 {
			tail = tail[len(tail)-3000:]
		}
		s.Low1 = liveLow1(tail)
		w := e.frames
		if len(w) > 360 {
			w = w[len(w)-360:]
		}
		s.Spark = bench.Downsample(w, 90)
	}
	return s
}

// Run procesa mediciones y atiende el reloj interno hasta que ctx termina.
func (e *Engine) Run(ctx context.Context, samples <-chan telemetry.Sample) {
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			e.Shutdown()
			return
		case s, ok := <-samples:
			if !ok {
				samples = nil
				continue
			}
			e.OnSample(s)
		case <-tk.C:
			e.Tick()
		}
	}
}

// Tick hace el trabajo de cada segundo: configuración, órdenes, juego y estado.
func (e *Engine) Tick() {
	e.mu.Lock()
	e.cfg = LoadConfig(e.d.Dir)
	e.lang = UILang(e.d.Dir)
	cfg := e.cfg
	e.mu.Unlock()

	if c, ok := loadCmd(e.d.Dir); ok && c.Seq != e.lastSeq {
		e.lastSeq = c.Seq
		switch c.Action {
		case "record_start":
			e.StartRecording(c.Label)
		case "record_stop":
			e.StopRecording()
		case "quit":
			e.Shutdown()
			if e.d.Quit != nil {
				e.d.Quit()
			}
			return
		}
	}
	e.checkGame(cfg) // cada segundo: el juego se detecta apenas se abre
	e.checkExclusive()
	e.watchFrames()
	e.writeState()
}

func (e *Engine) checkGame(cfg Config) {
	exe, running := cfg.GameExe, false
	if cfg.AutoGame && e.d.FindGame != nil {
		exe = e.d.FindGame()
		running = exe != ""
	} else {
		running = e.d.GameRunning != nil && e.d.GameRunning(exe)
	}
	e.mu.Lock()
	was, game := e.gameRun, e.pmGame
	e.mu.Unlock()
	if cfg.AutoGame && e.d.FindGame != nil && (running != was || (running && !strings.EqualFold(game, exe))) {
		if running {
			fpslog.Diag("detección automática: juego %q", exe)
		} else {
			fpslog.Diag("detección automática: ya no hay juego abierto")
		}
	}
	switch {
	case running && (!was || !strings.EqualFold(game, exe)):
		if was {
			e.gameEnded()
		}
		e.gameStarted(cfg, exe)
	case !running && was:
		e.gameEnded()
	}
}

func (e *Engine) gameStarted(cfg Config, exe string) {
	e.mu.Lock()
	e.pmGen++
	gen := e.pmGen
	e.mu.Unlock()
	stop, err := e.startPM(exe, gen, 0)
	e.mu.Lock()
	e.gameRun, e.pmStop, e.pmGame = true, stop, exe
	e.pmStarted = e.d.Now()
	if err != nil {
		e.msg = "No se pudo medir los FPS: " + err.Error()
	} else {
		e.msg = ""
	}
	e.mu.Unlock()
	g, _ := cfg.GameFor(exe)
	g.Exe = exe
	g.Close = mergeClose(cfg.CloseList, g.Close, exe)
	if _, ok := cfg.GameFor(exe); ok || len(g.Close) > 0 {
		e.applyProfile(g)
	}
}

// startPM lanza la medición de cuadros. Si la medición termina sola mientras el juego sigue abierto,
// se vuelve a lanzar.
func (e *Engine) startPM(exe string, gen, attempt int) (func(), error) {
	if e.d.StartPM == nil {
		return nil, nil
	}
	return e.d.StartPM(exe, e.OnFrame, func(error) {
		if attempt >= 5 {
			return
		}
		time.Sleep(2 * time.Second)
		e.mu.Lock()
		ok := e.pmGen == gen && e.gameRun
		e.mu.Unlock()
		if !ok {
			return
		}
		stop, err := e.startPM(exe, gen, attempt+1)
		e.mu.Lock()
		if e.pmGen == gen && e.gameRun {
			e.pmStop = stop
			if err != nil {
				e.msg = "No se pudo medir los FPS: " + err.Error()
			}
		} else if stop != nil {
			defer stop()
		}
		e.mu.Unlock()
	})
}

// checkExclusive anota si el juego está en pantalla completa exclusiva. En ese modo Windows no deja que
// ninguna ventana externa se dibuje encima, así que el overlay no se ve aunque esté midiendo bien.
func (e *Engine) checkExclusive() {
	e.mu.Lock()
	run, game := e.gameRun, e.pmGame
	e.mu.Unlock()
	on := run && e.d.Exclusive != nil && e.d.Exclusive()
	e.mu.Lock()
	changed := on != e.exclusive
	e.exclusive = on
	if on && changed {
		e.exclSince = e.d.Now().UnixMilli()
	}
	e.mu.Unlock()
	if changed && on {
		fpslog.Diag("%s está en pantalla completa exclusiva: el overlay no puede dibujarse encima", game)
	} else if changed {
		fpslog.Diag("ya no hay un juego en pantalla completa exclusiva")
	}
}

// watchFrames avisa en el estado cuando el juego lleva varios segundos abierto y no llegan cuadros.
func (e *Engine) watchFrames() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.gameRun || e.d.Now().Sub(e.pmStarted) < 10*time.Second {
		return
	}
	if e.lastFrame.IsZero() || e.d.Now().Sub(e.lastFrame) > 5*time.Second {
		e.msg = "RTSS no está midiendo " + e.pmGame + ". RTSS tiene que estar abierto antes de abrir el juego: reiniciá el juego o instalá RTSS desde «Instalar programas»."
	} else if strings.HasPrefix(e.msg, "RTSS no está midiendo") {
		e.msg = ""
	}
}

func (e *Engine) gameEnded() {
	if e.IsRecording() {
		e.StopRecording()
	}
	e.mu.Lock()
	stop := e.pmStop
	e.gameRun, e.pmStop, e.pmGame = false, nil, ""
	e.frames, e.lastFrame = nil, time.Time{}
	e.pmGen++
	e.mu.Unlock()
	if stop != nil {
		stop()
	}
	e.restoreProfile()
}

// Shutdown detiene la medición y devuelve el sistema a como estaba.
func (e *Engine) Shutdown() {
	if e.IsRecording() {
		e.StopRecording()
	}
	e.mu.Lock()
	stop := e.pmStop
	e.pmStop = nil
	e.mu.Unlock()
	if stop != nil {
		stop()
	}
	e.restoreProfile()
}

// IsRecording indica si hay una grabación en curso.
func (e *Engine) IsRecording() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.rec }

// StartRecording empieza a guardar cada cuadro.
func (e *Engine) StartRecording(label string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(label) > 60 {
		label = label[:60]
	}
	e.rec, e.recLabel, e.recBuf = true, label, nil
	e.msg = ""
	if !e.gameRun {
		e.msg = "Grabando: abrí un juego para que lleguen cuadros."
	}
}

// StopRecording termina la grabación y guarda la sesión. Devuelve su id.
func (e *Engine) StopRecording() string {
	e.mu.Lock()
	buf, label, game := e.recBuf, e.recLabel, e.pmGame
	e.rec, e.recBuf = false, nil
	e.mu.Unlock()
	if len(buf) < 300 {
		e.mu.Lock()
		e.msg = fmt.Sprintf("La grabación fue muy corta (%d cuadros); no se guardó.", len(buf))
		e.mu.Unlock()
		return ""
	}
	run, spikes := bench.FromFrameTimes(buf)
	now := e.d.Now()
	run.Label, run.Date, run.App = label, now.Format("02-01-2006"), game
	id := now.Format("20060102-150405")
	ses := bench.Session{ID: id, At: now.Format(time.RFC3339), Label: label, Game: game, Run: run, Spikes: spikes,
		Frames: bench.Downsample(buf, graphPoints)}
	if err := writeJSON(filepath.Join(e.d.Dir, "sessions", id+".json"), ses); err != nil {
		e.mu.Lock()
		e.msg = "No se pudo guardar la sesión: " + err.Error()
		e.mu.Unlock()
		return ""
	}
	e.mu.Lock()
	e.lastSession = id
	e.msg = fmt.Sprintf("Sesión guardada: %.0f FPS promedio, %d picos.", run.Avg, spikes)
	e.mu.Unlock()
	return id
}

func (e *Engine) writeState() {
	e.mu.Lock()
	sn := e.snapshotLocked()
	st := State{PID: e.d.PID, Updated: e.d.Now(), Visible: e.visible, Game: e.gameName(), GameRunning: e.gameRun,
		FPS: sn.FPS, Low1: sn.Low1, Recording: e.rec, RecLabel: e.recLabel, RecFrames: len(e.recBuf),
		Sample: e.sample, Message: e.msg, LastSession: e.lastSession, HotkeyErrs: e.hkErrs}
	if e.exclusive {
		st.Exclusive, st.ExclusiveSince = true, e.exclSince
	}
	if e.rec && len(e.recBuf) > 0 {
		for _, v := range e.recBuf {
			st.RecSeconds += v
		}
		st.RecSeconds /= 1000
	}
	st.Verdict = telemetry.Bottleneck(e.recent)
	e.mu.Unlock()
	_ = saveState(e.d.Dir, st)
}

// ClearState borra el archivo de estado (al salir).
func ClearState(dir string) { _ = os.Remove(filepath.Join(dir, "overlay_state.json")) }

// SetHotkeyErrors anota los atajos que Windows no pudo registrar (por ejemplo, porque otro programa los usa).
func (e *Engine) SetHotkeyErrors(errs []string) { e.mu.Lock(); e.hkErrs = errs; e.mu.Unlock() }

// ToggleRecording empieza o detiene la grabación (para el atajo global).
func (e *Engine) ToggleRecording() {
	if e.IsRecording() {
		e.StopRecording()
		return
	}
	e.StartRecording("Atajo " + e.d.Now().Format("15:04"))
}

// CycleCorner pasa a la esquina siguiente y la guarda (para el atajo global).
func (e *Engine) CycleCorner() {
	cfg := LoadConfig(e.d.Dir)
	order := []string{"tl", "tr", "br", "bl"}
	for i, c := range order {
		if c == cfg.Corner {
			cfg.Corner = order[(i+1)%len(order)]
			break
		}
	}
	_ = SaveConfig(e.d.Dir, cfg)
}

// gameName es el juego que se muestra en el estado: el que está abierto o, si no hay, el configurado
// (con la detección automática no hay uno fijo).
func (e *Engine) gameName() string {
	if e.gameRun && e.pmGame != "" {
		return e.pmGame
	}
	if e.cfg.AutoGame && e.d.FindGame != nil {
		return ""
	}
	return e.cfg.GameExe
}

// safeSensors lee los sensores de Afterburner sin dejar que un error cierre el overlay.
func safeSensors() (ss []rtss.Sensor) {
	defer func() { _ = recover() }()
	return rtss.ReadSensors()
}

// fpsWindowMs es el tramo que se promedia para el número de FPS: más corto cuanto más rápido se actualiza
// el panel, para que el número siga a los cambios (entre 250 ms y 1 s).
func fpsWindowMs(updateMs int) float64 {
	w := float64(updateMs) * 4
	if w < 250 {
		w = 250
	}
	if w > 1000 {
		w = 1000
	}
	return w
}

// mergeClose une la lista del Modo juego con la del perfil del juego, sin repetidos y sin cerrar
// nunca el propio juego ni programas protegidos de Windows.
func mergeClose(global, perGame []string, game string) []string {
	var out []string
	seen := map[string]bool{strings.ToLower(game): true}
	for _, n := range append(append([]string{}, global...), perGame...) {
		k := strings.ToLower(n)
		if seen[k] || procs.Protected(n) {
			continue
		}
		seen[k] = true
		out = append(out, n)
	}
	return out
}
