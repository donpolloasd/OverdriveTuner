//go:build !windows

package winhost

import "errors"

// AppOptions describe la ventana propia (solo existe en Windows).
type AppOptions struct {
	Title         string
	Maximized     bool
	Width, Height int
	HideOnClose   bool
	OnClose       func()
}

// AppWindow es la ventana propia del programa (solo existe en Windows).
type AppWindow struct{ done chan struct{} }

// WebViewAvailable: fuera de Windows no hay ventana propia.
func WebViewAvailable() bool { return false }

// OpenApp: fuera de Windows no hay ventana propia; se usa el navegador.
func OpenApp(url, dataDir string, o AppOptions) (*AppWindow, error) {
	return nil, errors.New("la ventana propia solo existe en Windows")
}

func (w *AppWindow) Done() <-chan struct{} { return w.done }
func (w *AppWindow) Show()                 {}
func (w *AppWindow) Close()                {}
