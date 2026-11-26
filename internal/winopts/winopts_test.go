package winopts

import (
	"strings"
	"testing"

	"fpsforge/internal/system"
)

func TestListAndOpen(t *testing.T) {
	const id = "{C3915CF9-A3D9-4EFD-B209-62C05793EE0F}"
	f := system.NewFake()
	f.PSHook = func(s string) (string, error) {
		if strings.Contains(s, "InvokeVerb") {
			return "ABIERTO\r\n", nil
		}
		return `aviso` + "\r\n" + `JSON[{"id":"` + id + `","name":"Zoom","group":"B"},{"id":"` + id + `","name":"Repetida","group":"B"},{"id":"'; calc","name":"mala","group":"A"},{"id":"{7B086E4E-366C-454A-85CC-B8B533482B10}","name":"Teclado","group":"A"}]` + "\r\n", nil
	}
	l, err := List(f)
	if err != nil || len(l) != 2 || l[0].Name != "Teclado" {
		t.Fatal(l, err)
	}
	if Open(f, id) != nil || Open(f, "'; calc") == nil {
		t.Fatal("solo se abren identificadores válidos")
	}
	f.PSHook = func(string) (string, error) { return "", nil }
	if _, err := List(f); err == nil {
		t.Fatal("sin datos es un error")
	}
	if Open(f, id) == nil {
		t.Fatal("si no existe, avisa")
	}
}
