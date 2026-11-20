package diskmap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, p string, n int, b byte) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(strings.Repeat(string(b), n)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanViewTypesLargestAndDuplicates(t *testing.T) {
	root := t.TempDir()
	mb := 1 << 20
	write(t, filepath.Join(root, "juegos", "a", "datos.pak"), 3*mb, 'x')
	write(t, filepath.Join(root, "juegos", "b", "copia.pak"), 3*mb, 'x') // mismo contenido
	write(t, filepath.Join(root, "juegos", "b", "otro.pak"), 3*mb, 'y')  // mismo tamaño, otro contenido
	write(t, filepath.Join(root, "docs", "nota.txt"), 1000, 'n')
	write(t, filepath.Join(root, "suelto.bin"), 2*mb, 'z')
	r, err := Scan(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Files != 5 || r.Bytes != int64(11*mb+1000) || r.Dirs != 4 {
		t.Fatalf("totales: %d archivos, %d bytes, %d carpetas", r.Files, r.Bytes, r.Dirs)
	}
	v, ok := r.ViewOf(root, 10)
	if !ok || len(v.Kids) != 2 || v.Kids[0].Name != "juegos" || v.Loose != int64(2*mb) {
		t.Fatalf("vista: %+v", v)
	}
	if v, ok := r.ViewOf(filepath.Join(root, "JUEGOS", "b"), 10); !ok || v.Size != int64(6*mb) {
		t.Fatalf("sin distinguir mayúsculas: %+v %v", v, ok)
	}
	if l := r.Largest(2); len(l) != 2 || l[0].Size != int64(3*mb) {
		t.Fatalf("más grandes: %+v", l)
	}
	if ty := r.Types(1); ty[0].Ext != ".pak" || ty[0].Count != 3 {
		t.Fatalf("tipos: %+v", ty)
	}
	d, complete := r.Duplicates(context.Background(), time.Minute, 10)
	if !complete || len(d) != 1 || len(d[0].Files) != 2 || d[0].Wasted != int64(3*mb) {
		t.Fatalf("duplicados: %+v", d)
	}
	// Mandar una carpeta a la Papelera se descuenta del análisis.
	r.Forget(filepath.Join(root, "juegos", "b"))
	if v, _ := r.ViewOf(root, 10); v.Size != int64(5*mb+1000) || len(v.Kids) != 2 || v.Kids[0].Size != int64(3*mb) {
		t.Fatalf("después de borrar: %+v", v)
	}
	if d, _ := r.Duplicates(context.Background(), time.Minute, 10); len(d) != 0 {
		t.Fatal("ya no hay duplicados")
	}
}

func TestScanCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, t.TempDir(), nil); err == nil {
		t.Fatal("cancelado debe fallar")
	}
}

func TestProtected(t *testing.T) {
	t.Setenv("WINDIR", `C:\Windows`)
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramData", `C:\ProgramData`)
	t.Setenv("USERPROFILE", `C:\Users\Ana`)
	for _, p := range []string{`C:\`, `D:\`, `C:\Windows\System32`, `c:\program files\Steam`, `C:\ProgramData\OverdriveTuner`, `C:\Users\Ana`, `C:\Users\Ana\Documents`, `C:\Users\Otro`, `C:\pagefile.sys`, `C:\$Recycle.Bin\x`, `relativa`} {
		if !Protected(p) {
			t.Errorf("%s debe estar protegida", p)
		}
	}
	for _, p := range []string{`C:\Users\Ana\Downloads\setup.exe`, `C:\Users\Ana\Videos\viejo`, `D:\SteamLibrary\steamapps\common\Juego`, `C:\Juegos\x`} {
		if Protected(p) {
			t.Errorf("%s se puede borrar", p)
		}
	}
}
