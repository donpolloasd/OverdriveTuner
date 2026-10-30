package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"2.7.5", "2.7.4", true}, {"2.7.4", "2.7.4", false}, {"2.7.3", "2.7.4", false}, {"2.10.0", "2.9.9", true}, {"v3.0", "2.9.9", true}, {"x", "2.7.4", false}, {"2.7.4", "", false}} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q,%q)=%v", c.a, c.b, got)
		}
	}
}

func TestParseSums(t *testing.T) {
	h := strings.Repeat("ab", 32)
	txt := h + "  OverdriveTuner-Portable.exe\n" + strings.Repeat("cd", 32) + " *OverdriveTuner-Setup.exe\n"
	if g, ok := ParseSums(txt, "OverdriveTuner-Portable.exe"); !ok || g != h {
		t.Errorf("hash mal leído: %q %v", g, ok)
	}
	if g, ok := ParseSums(txt, "OverdriveTuner-Setup.exe"); !ok || g != strings.Repeat("cd", 32) {
		t.Errorf("hash con * mal leído: %q %v", g, ok)
	}
	if _, ok := ParseSums("zz  OverdriveTuner-Portable.exe", "OverdriveTuner-Portable.exe"); ok {
		t.Error("un hash inválido no debe aceptarse")
	}
}

func TestAllowedGitHub(t *testing.T) {
	ok := []string{"https://api.github.com/repos/" + Repo + "/releases/latest", "https://github.com/" + Repo + "/releases/download/v1/x.exe", "https://objects.githubusercontent.com/abc"}
	bad := []string{"http://api.github.com/repos/" + Repo + "/releases/latest", "https://github.com/otro/repo/releases/download/v1/x.exe", "https://evil.com/x", "https://github.com.evil.com/" + Repo + "/releases/download/x", "https://api.github.com/repos/otro/x/releases/latest"}
	for _, s := range ok {
		u, _ := url.Parse(s)
		if !allowedGitHub(u) {
			t.Errorf("debería permitirse %s", s)
		}
	}
	for _, s := range bad {
		u, _ := url.Parse(s)
		if allowedGitHub(u) {
			t.Errorf("no debería permitirse %s", s)
		}
	}
}

func fakeServer(t *testing.T, exe []byte, sums string) (*Client, Release) {
	mux := http.NewServeMux()
	mux.HandleFunc("/exe", func(w http.ResponseWriter, r *http.Request) { w.Write(exe) })
	mux.HandleFunc("/sums", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, sums) })
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	c := NewClient()
	c.Allow = func(*url.URL) bool { return true }
	return c, Release{Version: "9.9.9", ExeURL: ts.URL + "/exe", SumsURL: ts.URL + "/sums"}
}

func TestDownloadVerifiesHash(t *testing.T) {
	exe := append([]byte("MZ"), bytes.Repeat([]byte{7}, 2<<20)...)
	sum := sha256.Sum256(exe)
	good := hex.EncodeToString(sum[:]) + "  " + AssetExe + "\n"
	c, rel := fakeServer(t, exe, good)
	dir := t.TempDir()
	if err := c.Download(context.Background(), rel, dir); err != nil {
		t.Fatal(err)
	}
	v, p, ok := Staged(dir)
	if !ok || v != "9.9.9" {
		t.Fatalf("debía quedar lista la versión descargada: %q %v", v, ok)
	}
	// Apply reemplaza el programa y deja el viejo aparte
	target := filepath.Join(t.TempDir(), "OverdriveTuner.exe")
	_ = os.WriteFile(target, []byte("viejo"), 0o755)
	if err := Apply(target, p); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); !bytes.Equal(b, exe) {
		t.Error("el programa no se reemplazó")
	}
	if b, _ := os.ReadFile(target + ".old"); string(b) != "viejo" {
		t.Error("falta el programa anterior")
	}
	CleanOld(target)
	if _, err := os.Stat(target + ".old"); err == nil {
		t.Error("CleanOld debía borrar el .old")
	}
	// hash que no coincide: se descarta
	bad := strings.Repeat("0", 64) + "  " + AssetExe + "\n"
	c2, rel2 := fakeServer(t, exe, bad)
	dir2 := t.TempDir()
	if err := c2.Download(context.Background(), rel2, dir2); err == nil {
		t.Fatal("una descarga con otro hash debe rechazarse")
	}
	if _, _, ok := Staged(dir2); ok {
		t.Error("no debe quedar nada listo tras un hash incorrecto")
	}
}

func TestStagedDetectsTampering(t *testing.T) {
	exe := append([]byte("MZ"), bytes.Repeat([]byte{1}, 2<<20)...)
	sum := sha256.Sum256(exe)
	c, rel := fakeServer(t, exe, hex.EncodeToString(sum[:])+"  "+AssetExe)
	dir := t.TempDir()
	if err := c.Download(context.Background(), rel, dir); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(StageDir(dir), "OverdriveTuner.exe"), []byte("otro programa"), 0o755)
	if _, _, ok := Staged(dir); ok {
		t.Error("un archivo modificado después de descargarlo no debe aceptarse")
	}
}
