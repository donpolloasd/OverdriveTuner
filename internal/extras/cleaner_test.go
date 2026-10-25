package extras

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveContentsKeepsRootAndCounts(t *testing.T) {
	root := filepath.Join(t.TempDir(), "carpeta-temporal-larga")
	os.MkdirAll(filepath.Join(root, "sub"), 0o755)
	os.WriteFile(filepath.Join(root, "a.bin"), make([]byte, 1000), 0o644)
	os.WriteFile(filepath.Join(root, "sub", "b.bin"), make([]byte, 500), 0o644)
	freed, removed, skipped := removeContents(root)
	if freed != 1500 || removed != 2 || skipped != 0 {
		t.Fatalf("freed=%d removed=%d skipped=%d", freed, removed, skipped)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("el directorio raíz debe conservarse")
	}
}

func TestSafeRootRefusesDangerousPaths(t *testing.T) {
	for _, p := range []string{"/", `C:\`, `C:\Windows`, `c:\users`, ""} {
		if safeRoot(p) {
			t.Errorf("%q no debería considerarse seguro", p)
		}
	}
}

func TestDeepTargetsAreScopedAndSafe(t *testing.T) {
	ts := deepTargets(`C:\Users\x\AppData\Local`, `C:\Users\x\AppData\Roaming`, `C:\Windows`)
	if len(ts) < 6 {
		t.Fatal(len(ts))
	}
	seen := map[string]bool{}
	for _, x := range ts {
		if !x.Deep || x.Safe || seen[x.ID] {
			t.Fatalf("objetivo mal marcado: %+v", x)
		}
		seen[x.ID] = true
		for _, p := range x.Paths {
			low := strings.ToLower(p)
			for _, bad := range []string{"cookies", "login data", "history", "bookmarks", "preferences", "local storage"} {
				if strings.Contains(low, bad) {
					t.Fatalf("%s apunta a datos personales: %s", x.ID, p)
				}
			}
		}
	}
}
