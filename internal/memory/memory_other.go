//go:build !windows

package memory

// Free simula la liberación fuera de Windows (modo demostración).
func Free(o Options) (Result, error) {
	r := Result{TotalMB: 32768, BeforeMB: 14200, AfterMB: 17900, Trimmed: 41, Standby: o.Game == "", SkippedGame: o.Game}
	r.Note = "Modo demostración: no se tocó la memoria de este equipo."
	return r, nil
}
