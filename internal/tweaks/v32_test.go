package tweaks

import (
	"strings"
	"testing"

	"fpsforge/internal/system"
)

func TestUnneededServicesSkipLockedTrkWks(t *testing.T) {
	f := system.NewFake()
	for _, s := range []string{"Fax", "TrkWks", "MapsBroker"} {
		f.Seed(svcKey+s, "Start", "REG_DWORD", "3")
	}
	f.FailSet = []string{`\TrkWks`} // Windows no deja modificarlo, ni siquiera a un administrador
	e, err := NewEngine(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := e.Apply("svc.unneeded")
	if !r.OK || len(r.Notes) == 0 {
		t.Fatalf("se aplica igual, salteando TrkWks: %+v", r)
	}
	tw, _ := ByID("svc.unneeded")
	if !e.Detect(tw) {
		t.Fatal("TrkWks no cuenta para saber si quedó aplicado")
	}
}

func TestAllBlockedFailsClearly(t *testing.T) {
	f := system.NewFake()
	f.FailSet = []string{`\Dsh`, `\Explorer\Advanced`} // UCPD bloquea las dos claves
	e, err := NewEngine(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := e.Apply("db.widgets")
	if r.OK || len(r.Errors) == 0 || !strings.Contains(r.Errors[0], "Windows no permite") || e.Applied("db.widgets") {
		t.Fatalf("si Windows bloquea todo, falla con una explicación: %+v", r)
	}
	f.FailSet = []string{`\Dsh`}
	if r := e.Apply("db.widgets"); !r.OK {
		t.Fatalf("con una de las dos alcanza: %+v", r)
	}
}

func TestMemCompressionStartsSysMain(t *testing.T) {
	tw, _ := ByID("mem.memcomp")
	for _, s := range []string{tw.Ops[0].Apply, tw.Ops[0].Revert} {
		if !strings.Contains(s, "Start-Service SysMain") || !strings.Contains(s, "Set-Service SysMain -StartupType Disabled") {
			t.Fatalf("enciende SysMain un momento y lo deja como estaba:\n%s", s)
		}
	}
}
