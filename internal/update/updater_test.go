package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	dir := t.TempDir()
	un, ok := Lock(dir)
	if !ok {
		t.Fatal("el primer candado debía tomarse")
	}
	if _, ok2 := Lock(dir); ok2 {
		t.Fatal("con el candado tomado, el segundo debe esperar")
	}
	un()
	un2, ok := Lock(dir)
	if !ok {
		t.Fatal("liberado el candado, se puede volver a tomar")
	}
	un2()
	// un candado abandonado hace rato se recupera
	p := filepath.Join(dir, lockName)
	_ = os.WriteFile(p, []byte("1"), 0o644)
	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes(p, old, old)
	un3, ok := Lock(dir)
	if !ok {
		t.Fatal("un candado abandonado debía recuperarse")
	}
	un3()
}

func TestUpdaterInstall(t *testing.T) {
	exe := append([]byte("MZ"), bytes.Repeat([]byte{9}, 2<<20)...)
	sum := sha256.Sum256(exe)
	c, rel := fakeServer(t, exe, hex.EncodeToString(sum[:])+"  "+AssetExe+"\n")
	dir := t.TempDir()
	target := filepath.Join(dir, "OverdriveTuner.exe")
	_ = os.WriteFile(target, []byte("viejo"), 0o755)
	u := &Updater{Client: c, Current: "2.7.5", ExePath: target}

	fresh, err := u.Install(context.Background(), rel)
	if err != nil || !fresh {
		t.Fatalf("debía instalarse: fresh=%v err=%v", fresh, err)
	}
	if b, _ := os.ReadFile(target); !bytes.Equal(b, exe) {
		t.Fatal("el programa no quedó reemplazado")
	}
	if _, err := os.Stat(filepath.Join(dir, lockName)); err == nil {
		t.Error("el candado debía liberarse")
	}
	if _, err := os.Stat(StageDir(dir)); err == nil {
		t.Error("la carpeta de descarga debía borrarse")
	}
	// segunda vez: el disco ya tiene la versión publicada, no se descarga ni se reemplaza otra vez
	fresh, err = u.Install(context.Background(), rel)
	if err != nil || fresh {
		t.Fatalf("ya estaba instalada: fresh=%v err=%v", fresh, err)
	}
	// una copia que no es la instalada no se reemplaza
	if _, err := (&Updater{Client: c, Current: "2.7.5"}).Install(context.Background(), rel); err != ErrNotInstalled {
		t.Errorf("sin ExePath debía dar ErrNotInstalled, dio %v", err)
	}
	// con el candado tomado por otro proceso, no se pisa
	un, _ := Lock(dir)
	_ = os.WriteFile(target, []byte("viejo2"), 0o755)
	if _, err := u.Install(context.Background(), rel); err != ErrBusy {
		t.Errorf("con el candado tomado debía dar ErrBusy, dio %v", err)
	}
	un()
	if b, _ := os.ReadFile(target); string(b) != "viejo2" {
		t.Error("con el candado tomado no se debía tocar el programa")
	}
}

func TestUpdaterInstallBadHashKeepsOld(t *testing.T) {
	exe := append([]byte("MZ"), bytes.Repeat([]byte{4}, 2<<20)...)
	c, rel := fakeServer(t, exe, "0000000000000000000000000000000000000000000000000000000000000000  "+AssetExe)
	dir := t.TempDir()
	target := filepath.Join(dir, "OverdriveTuner.exe")
	_ = os.WriteFile(target, []byte("viejo"), 0o755)
	u := &Updater{Client: c, Current: "2.7.5", ExePath: target}
	if _, err := u.Install(context.Background(), rel); err == nil {
		t.Fatal("con un hash que no coincide no debe instalarse")
	}
	if b, _ := os.ReadFile(target); string(b) != "viejo" {
		t.Error("el programa actual debe quedar intacto")
	}
	if _, err := os.Stat(filepath.Join(dir, lockName)); err == nil {
		t.Error("el candado debía liberarse aun con error")
	}
}

func TestLatestAndCheck(t *testing.T) {
	var code int
	body := `{"tag_name":"v2.8.0","body":"Notas","html_url":"https://github.com/donpolloasd/OverdriveTuner/releases/tag/v2.8.0","assets":[{"name":"OverdriveTuner-Portable.exe","browser_download_url":"https://github.com/donpolloasd/OverdriveTuner/releases/download/v2.8.0/OverdriveTuner-Portable.exe"},{"name":"SHA256.txt","browser_download_url":"https://github.com/donpolloasd/OverdriveTuner/releases/download/v2.8.0/SHA256.txt"}]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if code != 0 {
			w.WriteHeader(code)
			return
		}
		w.Write([]byte(body))
	}))
	defer ts.Close()
	c := NewClient()
	c.APIBase = ts.URL
	c.Allow = func(*url.URL) bool { return true }
	u := &Updater{Client: c, Current: "2.7.5"}

	r, newer, err := u.Check(context.Background())
	if err != nil || !newer || r.Version != "2.8.0" || r.ExeURL == "" || r.SumsURL == "" {
		t.Fatalf("Check: %+v newer=%v err=%v", r, newer, err)
	}
	u.Current = "2.8.0"
	if _, newer, _ := u.Check(context.Background()); newer {
		t.Error("con la misma versión no hay nada nuevo")
	}
	code = 404
	if _, _, err := u.Check(context.Background()); err != ErrNoRelease {
		t.Errorf("404 debía ser ErrNoRelease, fue %v", err)
	}
	code = 403
	if _, _, err := u.Check(context.Background()); err == nil || err == ErrNoRelease {
		t.Errorf("403 debía ser un error común, fue %v", err)
	}
}
