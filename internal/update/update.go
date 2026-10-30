// Package update busca versiones nuevas en los Releases de GitHub del proyecto, descarga el programa,
// comprueba su SHA-256 contra el SHA256.txt del mismo Release y lo reemplaza.
//
// Seguridad: solo se habla con GitHub por HTTPS y solo con el repositorio del proyecto; el archivo
// descargado se guarda dentro de la carpeta del programa (no en una carpeta que cualquier usuario pueda
// escribir) y se vuelve a comprobar su hash justo antes de reemplazar. No hay firma digital: la confianza
// descansa en la cuenta de GitHub de quien publica los Releases.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	Repo      = "donpolloasd/OverdriveTuner"
	AssetExe  = "OverdriveTuner-Portable.exe" // el mismo programa que instala el Setup
	AssetSums = "SHA256.txt"
	maxExe    = 64 << 20
	stageDir  = "update"
)

// Release describe la última versión publicada.
type Release struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	Page    string `json:"page"`
	ExeURL  string `json:"-"`
	SumsURL string `json:"-"`
}

// Client habla con GitHub. Los campos se pueden cambiar en las pruebas.
type Client struct {
	HTTP    *http.Client
	APIBase string
	Allow   func(*url.URL) bool
}

// NewClient crea un cliente que solo acepta direcciones de GitHub.
func NewClient() *Client {
	c := &Client{APIBase: "https://api.github.com", Allow: allowedGitHub}
	c.HTTP = &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return errors.New("demasiadas redirecciones")
		}
		if !c.Allow(r.URL) {
			return fmt.Errorf("redirección a un sitio no permitido: %s", r.URL.Host)
		}
		return nil
	}}
	return c
}

// allowedGitHub acepta solo HTTPS hacia GitHub (API, descargas y su CDN de archivos).
func allowedGitHub(u *url.URL) bool {
	if u.Scheme != "https" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	switch {
	case h == "api.github.com":
		return strings.HasPrefix(u.Path, "/repos/"+Repo+"/")
	case h == "github.com":
		return strings.HasPrefix(u.Path, "/"+Repo+"/releases/download/")
	case strings.HasSuffix(h, ".githubusercontent.com"):
		return true // destino de las descargas de Releases; el contenido se verifica con SHA-256
	}
	return false
}

// StatusError es una respuesta de GitHub que no fue 200.
type StatusError struct{ Code int }

func (e *StatusError) Error() string { return fmt.Sprintf("GitHub respondió %d", e.Code) }

// ErrNoRelease: el repositorio todavía no tiene ningún Release estable publicado.
var ErrNoRelease = errors.New("todavía no hay ninguna versión publicada en GitHub")

func (c *Client) get(ctx context.Context, raw string, limit int64) (*http.Response, error) {
	u, err := url.Parse(raw)
	if err != nil || !c.Allow(u) {
		return nil, fmt.Errorf("dirección no permitida: %s", raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OverdriveTuner-updater")
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, &StatusError{Code: resp.StatusCode}
	}
	if limit > 0 && resp.ContentLength > limit {
		resp.Body.Close()
		return nil, errors.New("el archivo es demasiado grande")
	}
	return resp, nil
}

// Latest consulta el último Release publicado.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	resp, err := c.get(ctx, c.APIBase+"/repos/"+Repo+"/releases/latest", 2<<20)
	if err != nil {
		var se *StatusError
		if errors.As(err, &se) && se.Code == http.StatusNotFound {
			return Release{}, ErrNoRelease
		}
		return Release{}, err
	}
	defer resp.Body.Close()
	var j struct {
		Tag     string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		Pre     bool   `json:"prerelease"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&j); err != nil {
		return Release{}, err
	}
	if j.Draft || j.Pre {
		return Release{}, errors.New("el último Release no es estable")
	}
	r := Release{Version: strings.TrimPrefix(strings.TrimSpace(j.Tag), "v"), Notes: j.Body, Page: j.HTMLURL}
	if len(r.Notes) > 4000 {
		r.Notes = r.Notes[:4000]
	}
	if _, ok := parseVersion(r.Version); !ok {
		return Release{}, fmt.Errorf("versión no válida: %q", j.Tag)
	}
	for _, a := range j.Assets {
		switch a.Name {
		case AssetExe:
			r.ExeURL = a.URL
		case AssetSums:
			r.SumsURL = a.URL
		}
	}
	return r, nil
}

func parseVersion(v string) ([]int, bool) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) < 2 || len(parts) > 4 {
		return nil, false
	}
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 100000 {
			return nil, false
		}
		out[i] = n
	}
	return out, true
}

// Newer dice si latest es una versión posterior a current.
func Newer(latest, current string) bool {
	a, ok1 := parseVersion(latest)
	b, ok2 := parseVersion(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 4; i++ {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// ParseSums busca el hash de un archivo en un SHA256.txt (formato de sha256sum).
func ParseSums(text, name string) (string, bool) {
	for _, ln := range strings.Split(text, "\n") {
		f := strings.Fields(strings.TrimSpace(ln))
		if len(f) != 2 {
			continue
		}
		if strings.TrimPrefix(f[1], "*") == name && len(f[0]) == 64 {
			if _, err := hex.DecodeString(f[0]); err == nil {
				return strings.ToLower(f[0]), true
			}
		}
	}
	return "", false
}

type meta struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

// StageDir es la carpeta donde se deja la descarga: junto al programa (no en una ruta de uso libre).
func StageDir(exeDir string) string { return filepath.Join(exeDir, stageDir) }

// Sum pide el SHA-256 publicado del programa en el SHA256.txt del Release.
func (c *Client) Sum(ctx context.Context, r Release) (string, error) {
	if r.ExeURL == "" || r.SumsURL == "" {
		return "", errors.New("el Release no incluye " + AssetExe + " y " + AssetSums)
	}
	sr, err := c.get(ctx, r.SumsURL, 1<<20)
	if err != nil {
		return "", err
	}
	sb, err := io.ReadAll(io.LimitReader(sr.Body, 1<<20))
	sr.Body.Close()
	if err != nil {
		return "", err
	}
	want, ok := ParseSums(string(sb), AssetExe)
	if !ok {
		return "", errors.New("SHA256.txt no tiene el hash de " + AssetExe)
	}
	return want, nil
}

// Download baja el programa del Release, comprueba su SHA-256 y lo deja listo para reemplazar.
func (c *Client) Download(ctx context.Context, r Release, exeDir string) error {
	want, err := c.Sum(ctx, r)
	if err != nil {
		return err
	}
	return c.download(ctx, r, exeDir, want)
}

func (c *Client) download(ctx context.Context, r Release, exeDir, want string) error {
	dir := StageDir(exeDir)
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	part := filepath.Join(dir, "OverdriveTuner.part")
	if err := c.fetchTo(ctx, r.ExeURL, part, want); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	if err := os.Rename(part, filepath.Join(dir, "OverdriveTuner.exe")); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	b, _ := json.Marshal(meta{Version: r.Version, SHA256: want})
	return os.WriteFile(filepath.Join(dir, "meta.json"), b, 0o644)
}

func (c *Client) fetchTo(ctx context.Context, raw, path, want string) error {
	resp, err := c.get(ctx, raw, maxExe)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxExe+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > maxExe || n < 1<<20 {
		return errors.New("el tamaño de la descarga no es el esperado")
	}
	if hex.EncodeToString(h.Sum(nil)) != want {
		return errors.New("la descarga no coincide con el SHA-256 publicado: se descarta")
	}
	return nil
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Staged devuelve la versión descargada y verificada que está lista para instalar.
func Staged(exeDir string) (version, path string, ok bool) {
	dir := StageDir(exeDir)
	b, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return "", "", false
	}
	var m meta
	if json.Unmarshal(b, &m) != nil || m.SHA256 == "" {
		return "", "", false
	}
	p := filepath.Join(dir, "OverdriveTuner.exe")
	got, err := fileSHA(p) // se vuelve a comprobar: nada de lo guardado se ejecuta sin verificar
	if err != nil || got != m.SHA256 {
		return "", "", false
	}
	if _, ok := parseVersion(m.Version); !ok {
		return "", "", false
	}
	return m.Version, p, true
}

// Apply reemplaza el programa en ejecución por el descargado. Windows permite renombrar un programa que
// está abierto: el viejo queda como «.old» (se borra en el próximo arranque) y el nuevo ocupa su lugar.
func Apply(exe, staged string) error {
	old := exe + ".old"
	_ = os.Remove(old)
	if _, err := os.Stat(old); err == nil {
		old = fmt.Sprintf("%s.old%d", exe, time.Now().Unix())
	}
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("no se pudo apartar el programa actual: %w", err)
	}
	if err := copyFile(staged, exe); err != nil {
		_ = os.Remove(exe)
		if rerr := os.Rename(old, exe); rerr != nil {
			return fmt.Errorf("no se pudo copiar la versión nueva (%v) ni restaurar la anterior (%v)", err, rerr)
		}
		return fmt.Errorf("no se pudo copiar la versión nueva: %w", err)
	}
	_ = os.RemoveAll(filepath.Dir(staged))
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// CleanOld borra los restos de actualizaciones anteriores («.old»).
func CleanOld(exe string) {
	l, _ := filepath.Glob(exe + ".old*")
	for _, p := range l {
		_ = os.Remove(p)
	}
}
