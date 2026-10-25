package history

import (
	"strconv"
	"testing"
)

func TestLogPersistsOrdersAndCaps(t *testing.T) {
	dir := t.TempDir()
	l := Open(dir)
	l.Max = 3
	for i := 0; i < 5; i++ {
		l.Add(Entry{ID: strconv.Itoa(i), Action: "apply", OK: true})
	}
	got := Open(dir).List()
	if len(got) != 3 || got[0].ID != "4" || got[2].ID != "2" || got[0].At == "" {
		t.Fatalf("%+v", got)
	}
	l.Clear()
	if len(Open(dir).List()) != 0 {
		t.Fatal("no se vació")
	}
	if Open(t.TempDir()).List() == nil {
		t.Fatal("la lista debe ser no nula")
	}
}
