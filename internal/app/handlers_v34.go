package app

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/apps"
	"fpsforge/internal/audio"
	"fpsforge/internal/diskmap"
	"fpsforge/internal/extras"
	"fpsforge/internal/overlay"
	"fpsforge/internal/perf"
	"fpsforge/internal/rtss"
	"fpsforge/internal/schedule"
	"fpsforge/internal/telemetry"
	"fpsforge/internal/winopts"
)

// ---------------------------------------------------------------- pruebas de rendimiento

// perfState es la prueba en curso o la última terminada. Una a la vez.
type perfState struct {
	mu      sync.Mutex
	running string
	prog    float64
	cancel  context.CancelFunc
	err     string
	last    *perf.Entry
	stab    *perf.StabResult
	live    []perf.StabPoint
}

var pf perfState

func (s *Server) perfStore() *perf.Store {
	s.perfOnce.Do(func() { s.perf = perf.OpenStore(s.Dir) })
	return s.perf
}

// handlePerf devuelve la prueba en curso, el último resultado y el ranking local.
func (s *Server) handlePerf(w http.ResponseWriter, r *http.Request) {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	live := pf.live
	if live == nil {
		live = []perf.StabPoint{}
	}
	writeJSON(w, map[string]any{"running": pf.running, "progress": pf.prog, "error": pf.err, "last": pf.last, "stab": pf.stab,
		"live": live, "kinds": perf.Kinds, "list": s.perfStore().List(), "volumes": diskmap.Volumes(), "maxTemp": perf.MaxTemp})
}

// handlePerfRun empieza una prueba (POST {kind, root, seconds}); {cancel:true} la detiene.
func (s *Server) handlePerfRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind    string `json:"kind"`
		Root    string `json:"root"`
		Seconds int    `json:"seconds"`
		Cancel  bool   `json:"cancel"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	if req.Cancel {
		pf.mu.Lock()
		if pf.cancel != nil {
			pf.cancel()
		}
		pf.mu.Unlock()
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	if !perf.ValidKind(req.Kind) {
		fail(w, 400, "prueba desconocida")
		return
	}
	dir := ""
	if req.Kind == perf.DiskRead || req.Kind == perf.DiskWrite {
		for _, v := range diskmap.Volumes() {
			if strings.EqualFold(v.Root, req.Root) {
				req.Root, dir = v.Root, v.Root
			}
		}
		if dir == "" {
			fail(w, 400, "Elegí una unidad que exista.")
			return
		}
		// En la unidad de los temporales se usa esa carpeta: la raíz de Windows no siempre deja escribir.
		if tmp := os.TempDir(); strings.EqualFold(filepath.VolumeName(tmp), filepath.VolumeName(dir)) {
			dir = tmp
		}
	}
	if req.Kind == perf.Stability && (req.Seconds < 30 || req.Seconds > 600) {
		req.Seconds = 60
	}
	pf.mu.Lock()
	defer pf.mu.Unlock()
	if pf.running != "" {
		fail(w, 409, "Ya hay una prueba en curso.")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	pf.running, pf.prog, pf.cancel, pf.err, pf.last, pf.stab, pf.live = req.Kind, 0, cancel, "", nil, nil, nil
	go s.perfRun(ctx, req.Kind, dir, req.Root, time.Duration(req.Seconds)*time.Second)
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) perfRun(ctx context.Context, kind, dir, root string, secs time.Duration) {
	prog := func(p float64) { pf.mu.Lock(); pf.prog = p; pf.mu.Unlock() }
	e := perf.Entry{Kind: kind}
	var err error
	var stab *perf.StabResult
	switch kind {
	case perf.CPU1:
		e.Value, err = perf.CPUSingle(ctx, 6*time.Second, prog)
	case perf.CPUAll:
		e.Value, err = perf.CPUMulti(ctx, 8*time.Second, prog)
	case perf.RAMSpeed:
		var wr, cp float64
		e.Value, wr, cp, err = perf.RAM(ctx, 6*time.Second, 512, prog)
		e.Extra = []float64{wr, cp}
	case perf.RAMLat:
		e.Value, err = perf.RAMLatency(ctx, 4*time.Second, 256, prog)
	case perf.DiskRead, perf.DiskWrite:
		var iops float64
		e.Value, iops, err = perf.Disk(ctx, dir, kind == perf.DiskWrite, prog)
		e.Extra, e.Target = []float64{iops}, root
	case perf.Stability:
		res := s.perfStress(ctx, secs, prog)
		stab = &res
		e.Value, e.Extra = res.Score, []float64{float64(res.Errors), res.DropPct, res.MaxTemp, res.Seconds}
	}
	var saved *perf.Entry
	// Las pruebas cortadas o fallidas no entran en el ranking.
	if err == nil && (stab == nil || (stab.Stopped == "" && stab.Stable())) {
		x, serr := s.perfStore().Add(e)
		saved, err = &x, serr
	}
	pf.mu.Lock()
	defer pf.mu.Unlock()
	if pf.cancel != nil {
		pf.cancel()
	}
	pf.running, pf.cancel, pf.last, pf.stab = "", nil, saved, stab
	if ctx.Err() != nil && err != nil {
		pf.err = "Prueba cancelada."
	} else if err != nil {
		pf.err = err.Error()
	}
}

// perfStress corre la prueba de estabilidad leyendo los sensores mientras tanto.
func (s *Server) perfStress(ctx context.Context, d time.Duration, prog func(float64)) perf.StabResult {
	var mu sync.Mutex
	var cur telemetry.Sample
	have := false
	stop := make(chan struct{})
	defer close(stop)
	if !s.isDemo() {
		rtss.EnsureCoreTemp() // si está instalado, de ahí sale la temperatura del procesador
	}
	go func() {
		for x := range telemetry.Stream(stop) {
			if x.CPUTemp == telemetry.Unknown {
				if v, ok := rtss.CPUTemp(rtss.ReadSensors()); ok {
					x.CPUTemp = v
				}
			}
			mu.Lock()
			cur, have = x, true
			mu.Unlock()
		}
	}()
	return perf.Stress(ctx, d, func() (telemetry.Sample, bool) {
		mu.Lock()
		defer mu.Unlock()
		return cur, have
	}, func(p perf.StabPoint) {
		prog(p.T / d.Seconds())
		pf.mu.Lock()
		pf.live = append(pf.live, p)
		pf.mu.Unlock()
	})
}

// handlePerfDelete borra una prueba del ranking ({id}) o todas ({all:true}).
func (s *Server) handlePerfDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID  int64 `json:"id"`
		All bool  `json:"all"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || (req.ID == 0) != req.All {
		fail(w, 400, "petición no válida")
		return
	}
	if err := s.perfStore().Delete(req.ID); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// ---------------------------------------------------------------- micrófono

type micState struct {
	mu      sync.Mutex
	running bool
	level   float64
	stop    chan struct{}
	wav     []byte
	rep     *audio.Report
	err     string
	demoOff bool
}

var mic micState

// handleMic devuelve el estado del micrófono; POST {mute} lo silencia o activa, {record} graba unos segundos
// en memoria, {cancel} corta la grabación y {forget} la descarta.
func (s *Server) handleMic(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Mute   *bool `json:"mute"`
			Record bool  `json:"record"`
			Cancel bool  `json:"cancel"`
			Forget bool  `json:"forget"`
		}
		if readJSON(r, &req) != nil {
			fail(w, 400, "petición no válida")
			return
		}
		switch {
		case req.Mute != nil:
			if s.isDemo() {
				mic.mu.Lock()
				mic.demoOff = *req.Mute
				mic.mu.Unlock()
			} else if err := audio.SetMicMuted(*req.Mute); err != nil {
				fail(w, 500, err.Error())
				return
			}
		case req.Record:
			if !s.micRecord() {
				fail(w, 409, "Ya se está grabando.")
				return
			}
		case req.Cancel:
			mic.mu.Lock()
			if mic.running && mic.stop != nil {
				close(mic.stop)
				mic.stop = nil
			}
			mic.mu.Unlock()
		case req.Forget:
			mic.mu.Lock()
			mic.wav, mic.rep = nil, nil
			mic.mu.Unlock()
		}
	}
	out := map[string]any{}
	mic.mu.Lock()
	out["recording"], out["level"], out["report"], out["error"] = mic.running, mic.level, mic.rep, mic.err
	muted := mic.demoOff
	mic.mu.Unlock()
	if !s.isDemo() && r.URL.Query().Get("level") == "" {
		m, err := audio.MicMuted()
		muted = m
		if err != nil {
			out["micError"] = err.Error()
		}
	}
	out["muted"] = muted
	writeJSON(w, out)
}

const micSeconds = 6

func (s *Server) micRecord() bool {
	mic.mu.Lock()
	defer mic.mu.Unlock()
	if mic.running {
		return false
	}
	stop := make(chan struct{})
	mic.running, mic.stop, mic.wav, mic.rep, mic.err, mic.level = true, stop, nil, nil, "", 0
	demo := s.isDemo()
	go func() {
		var pcm []int16
		var err error
		level := func(v float64) { mic.mu.Lock(); mic.level = v; mic.mu.Unlock() }
		if demo { // en demostración no se usa el micrófono real
			pcm = audio.Demo(micSeconds * time.Second)
			for i := 0; i < 10; i++ {
				level(float64(i%5) / 5)
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			pcm, err = audio.Record(micSeconds*time.Second, stop, level)
		}
		mic.mu.Lock()
		defer mic.mu.Unlock()
		mic.running, mic.level, mic.stop = false, 0, nil
		if err != nil {
			mic.err = err.Error()
			return
		}
		rep := audio.Analyze(pcm)
		mic.rep, mic.wav = &rep, audio.WAV(pcm)
	}()
	return true
}

// handleMicWAV entrega la última grabación para escucharla (está solo en memoria).
func (s *Server) handleMicWAV(w http.ResponseWriter, r *http.Request) {
	mic.mu.Lock()
	b := mic.wav
	mic.mu.Unlock()
	if b == nil {
		fail(w, 404, "no hay una grabación")
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

// ---------------------------------------------------------------- todas las opciones de Windows

var (
	optsMu    sync.Mutex
	optsCache []winopts.Option
)

func (s *Server) winOptions() ([]winopts.Option, error) {
	if s.isDemo() {
		return winopts.Demo(), nil
	}
	optsMu.Lock()
	defer optsMu.Unlock()
	if optsCache == nil {
		l, err := winopts.List(s.Sys)
		if err != nil {
			return nil, err
		}
		optsCache = l
	}
	return optsCache, nil
}

// handleWinOpts lista las opciones de Windows; POST {id} abre una.
func (s *Server) handleWinOpts(w http.ResponseWriter, r *http.Request) {
	l, err := s.winOptions()
	if err != nil {
		fail(w, 500, "No se pudo leer la lista de opciones de Windows: "+err.Error())
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]any{"options": l, "favs": s.loadSettings().WinFavs})
		return
	}
	var req struct {
		ID  string `json:"id"`
		Fav *bool  `json:"fav"`
	}
	if readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	known := false
	for _, o := range l {
		known = known || o.ID == req.ID
	}
	if !known {
		fail(w, 404, "Windows ya no tiene esa opción")
		return
	}
	if req.Fav != nil {
		_ = updateSettings(s.Dir, func(st *settings) {
			out := []string{}
			for _, x := range st.WinFavs {
				if x != req.ID {
					out = append(out, x)
				}
			}
			if *req.Fav && len(out) < 60 {
				out = append(out, req.ID)
			}
			st.WinFavs = out
		})
		writeJSON(w, map[string]any{"favs": s.loadSettings().WinFavs})
		return
	}
	if !s.isDemo() {
		if err := winopts.Open(s.Sys, req.ID); err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleWinOptsFolder abre en el Explorador la carpeta de Windows con todas las opciones.
func (s *Server) handleWinOptsFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 400, "petición no válida")
		return
	}
	if !s.isDemo() { // explorer.exe devuelve un código distinto de cero aunque abra bien
		if _, _, err := s.Sys.Exec(15*time.Second, "explorer.exe", winopts.Folder); err != nil {
			fail(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true})
}

// ---------------------------------------------------------------- actualizar programas de a uno

// handleAppsUpgrades lista los programas con versión nueva; POST {id} actualiza uno.
func (s *Server) handleAppsUpgrades(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req idReq
		if readJSON(r, &req) != nil || !apps.ValidPackage(req.ID) {
			fail(w, 400, "petición no válida")
			return
		}
		res := apps.UpgradeOne(s.Sys, req.ID)
		appsMu.Lock()
		appsCache = nil
		appsMu.Unlock()
		writeJSON(w, res)
		return
	}
	if s.isDemo() {
		writeJSON(w, []apps.Upgrade{{Name: "7-Zip", ID: "7zip.7zip", From: "26.05", To: "26.06"}, {Name: "OBS Studio", ID: "OBSProject.OBSStudio", From: "32.2.2", To: "33.0.3"}})
		return
	}
	l, err := apps.Upgrades(s.Sys)
	if err != nil {
		fail(w, 500, "No se pudo consultar a winget: "+err.Error())
		return
	}
	writeJSON(w, l)
}

// ---------------------------------------------------------------- limpieza programada y atajos

// handleCleanPlan dice qué puede limpiar la limpieza programada y qué está elegido.
func (s *Server) handleCleanPlan(w http.ResponseWriter, r *http.Request) {
	type item struct {
		extras.Target
		On bool `json:"on"`
	}
	pick, out := schedule.LoadClean(s.Dir), []item{}
	for _, t := range extras.Targets() {
		if schedule.Allowed(t) {
			out = append(out, item{t, pick[t.ID] || (pick == nil && t.Safe)})
		}
	}
	writeJSON(w, out)
}

// handleHotkeys devuelve los atajos del overlay (para la sección de atajos).
func (s *Server) handleHotkeys(w http.ResponseWriter, r *http.Request) {
	st, alive := s.overlayRunning()
	writeJSON(w, map[string]any{"config": overlay.LoadConfig(s.Dir), "running": alive, "errors": st.HotkeyErrs})
}

// ---------------------------------------------------------------- arranque optimizado de un juego

// handleGameFind busca cómo abrir un juego por el nombre de su programa ({exe}).
func (s *Server) handleGameFind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Exe string `json:"exe"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil || !overlay.ValidExe(req.Exe) {
		fail(w, 400, "petición no válida")
		return
	}
	found := ""
	if s.isDemo() {
		found = "steam://rungameid/730"
	} else {
		found = overlay.FindLaunch(req.Exe, overlay.LoadConfig(s.Dir).GameDirs)
	}
	writeJSON(w, map[string]any{"launch": found})
}

// handleGameLaunch abre el overlay (si hace falta, para que aplique el perfil) y después el juego ({exe}).
func (s *Server) handleGameLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Exe string `json:"exe"`
	}
	if r.Method != http.MethodPost || readJSON(r, &req) != nil {
		fail(w, 400, "petición no válida")
		return
	}
	g, ok := overlay.LoadConfig(s.Dir).GameFor(req.Exe)
	if !ok || g.Launch == "" {
		fail(w, 400, "Primero elegí cómo se abre este juego.")
		return
	}
	overlayOn := false
	if _, alive := s.overlayRunning(); alive {
		overlayOn = true
	} else if s.Sys.IsAdmin() && s.SpawnOverlay() == nil {
		overlayOn = true
		time.Sleep(1500 * time.Millisecond) // que el overlay esté listo antes de que arranque el juego
	}
	if err := overlay.Launch(s.Sys, g.Launch); err != nil {
		fail(w, 400, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "overlay": overlayOn})
}
