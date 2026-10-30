package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	// ErrNotInstalled: esta copia no es la instalada por el Setup, así que no se reemplaza sola.
	ErrNotInstalled = errors.New("esta copia no está instalada con el instalador")
	// ErrBusy: otra copia del programa (la ventana o el overlay) ya está actualizando.
	ErrBusy = errors.New("ya hay otra actualización en curso")
)

const lockName = "update.lock"

// Lock evita que la ventana y el overlay (dos procesos) actualicen a la vez. El candado se considera
// abandonado si tiene más de 15 minutos (por ejemplo, si el equipo se apagó en pleno proceso).
func Lock(exeDir string) (unlock func(), ok bool) {
	p := filepath.Join(exeDir, lockName)
	for try := 0; try < 2; try++ {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprint(f, os.Getpid())
			f.Close()
			return func() { _ = os.Remove(p) }, true
		}
		fi, serr := os.Stat(p)
		if serr != nil || time.Since(fi.ModTime()) < 15*time.Minute {
			if serr != nil {
				continue // desapareció justo ahora: se reintenta
			}
			return nil, false
		}
		_ = os.Remove(p)
	}
	return nil, false
}

// Updater junta las piezas: consultar, descargar, verificar y reemplazar.
type Updater struct {
	Client  *Client
	Current string // versión de este programa
	ExePath string // programa instalado que se puede reemplazar ("" = no se actualiza solo)
}

// Installed dice si esta copia puede reemplazarse sola.
func (u *Updater) Installed() bool { return u.ExePath != "" }

// Check consulta GitHub y dice si hay una versión posterior a la actual.
func (u *Updater) Check(ctx context.Context) (Release, bool, error) {
	r, err := u.Client.Latest(ctx)
	if err != nil {
		return Release{}, false, err
	}
	return r, Newer(r.Version, u.Current), nil
}

// Install descarga la versión, comprueba su SHA-256 y reemplaza el programa instalado.
// fresh es false si ese programa ya estaba reemplazado (falta solo reiniciarlo).
func (u *Updater) Install(ctx context.Context, r Release) (fresh bool, err error) {
	if u.ExePath == "" {
		return false, ErrNotInstalled
	}
	dir := filepath.Dir(u.ExePath)
	unlock, ok := Lock(dir)
	if !ok {
		return false, ErrBusy
	}
	defer unlock()
	want, err := u.Client.Sum(ctx, r)
	if err != nil {
		return false, err
	}
	if got, err := fileSHA(u.ExePath); err == nil && got == want {
		return false, nil // el programa del disco ya es el publicado
	}
	if err := u.Client.download(ctx, r, dir, want); err != nil {
		return false, err
	}
	v, staged, ok := Staged(dir)
	if !ok || v != r.Version {
		_ = os.RemoveAll(StageDir(dir))
		return false, errors.New("la descarga no pasó la verificación")
	}
	if err := Apply(u.ExePath, staged); err != nil {
		_ = os.RemoveAll(StageDir(dir))
		return false, err
	}
	return true, nil
}
