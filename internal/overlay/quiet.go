package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"

	"fpsforge/internal/system"
)

// «Silenciar notificaciones mientras jugás»: al abrir un juego se apagan los avisos emergentes de Windows
// (el mismo interruptor que «Notificaciones» en Configuración de Windows) y al cerrarlo se dejan como
// estaban. El valor anterior se guarda en un archivo, así se restaura aunque el overlay se cierre de golpe.

const pushKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\PushNotifications`

type quietBackup struct {
	Existed bool   `json:"existed"`
	Data    string `json:"data"`
}

func quietPath(dir string) string { return filepath.Join(dir, "overlay_quiet.json") }

// quietOn apaga las notificaciones si estaban encendidas.
func quietOn(sys system.System, dir string) {
	if sys == nil {
		return
	}
	if _, err := os.Stat(quietPath(dir)); err == nil {
		return // ya están silenciadas por el programa
	}
	prev, err := system.RegGet(sys, pushKey, "ToastEnabled")
	if err != nil || (prev.Exists && prev.Matches("REG_DWORD", "0")) {
		return // ya estaban apagadas por el usuario: no hay nada que restaurar después
	}
	b, _ := json.Marshal(quietBackup{Existed: prev.Exists, Data: prev.Data})
	if os.WriteFile(quietPath(dir), b, 0o644) != nil {
		return
	}
	if sys.RegSet(pushKey, "ToastEnabled", "REG_DWORD", "0") != nil {
		_ = os.Remove(quietPath(dir))
	}
}

// quietOff devuelve las notificaciones a como estaban antes del juego.
func quietOff(sys system.System, dir string) {
	if sys == nil {
		return
	}
	b, err := os.ReadFile(quietPath(dir))
	if err != nil {
		return
	}
	var bk quietBackup
	if json.Unmarshal(b, &bk) == nil {
		if bk.Existed {
			err = sys.RegSet(pushKey, "ToastEnabled", "REG_DWORD", bk.Data)
		} else {
			err = sys.RegDeleteValue(pushKey, "ToastEnabled")
		}
		if err != nil {
			return // se reintenta la próxima vez
		}
	}
	_ = os.Remove(quietPath(dir))
}
