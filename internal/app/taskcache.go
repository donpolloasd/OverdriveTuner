package app

import (
	"sync"
	"time"

	"fpsforge/internal/autostart"
	"fpsforge/internal/schedule"
)

// taskTTL es cuánto se reutiliza el estado de las tareas programadas de Windows (inicio con Windows y
// mantenimiento). Consultarlo lanza PowerShell, unos 2 segundos por tarea: sin esto Configuración tardaba
// unos 6 segundos en aparecer.
const taskTTL = time.Minute

type taskCache struct {
	mu    sync.Mutex
	at    time.Time
	auto  autostart.Info
	sched map[string]schedule.Info
}

// taskStatus devuelve el estado de todas las tareas: las consulta a la vez y lo guarda un minuto. fresh
// vuelve a consultar (después de activar, desactivar o probar una tarea). Si ya se está consultando, por
// ejemplo el precálculo que arranca al abrir el programa, espera ese resultado en vez de lanzar otro.
func (s *Server) taskStatus(fresh bool) (autostart.Info, map[string]schedule.Info) {
	c := &s.tasks
	c.mu.Lock()
	defer c.mu.Unlock()
	if !fresh && !c.at.IsZero() && time.Since(c.at) < taskTTL {
		return c.auto, c.sched
	}
	var wg sync.WaitGroup
	var smu sync.Mutex
	sched := map[string]schedule.Info{}
	var auto autostart.Info
	wg.Add(1)
	go func() { defer wg.Done(); auto = autostart.Status(s.Sys) }()
	for _, j := range schedule.Jobs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			i := schedule.Status(s.Sys, id)
			smu.Lock()
			sched[id] = i
			smu.Unlock()
		}(j.ID)
	}
	wg.Wait()
	c.auto, c.sched, c.at = auto, sched, time.Now()
	return c.auto, c.sched
}
