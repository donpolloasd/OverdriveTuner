package tools

import (
	"strings"
	"testing"

	"fpsforge/internal/tweaks"
)

func TestParseNetMon(t *testing.T) {
	ps, err := ParseNetMon(`[{"pid":10,"name":"chrome","tcp":5,"udp":1,"kbs":12.5,"remote":["1.2.3.4"]},{"pid":11,"name":"cs2","tcp":1,"udp":3,"kbs":90,"remote":null},{"pid":12,"name":"","tcp":0,"udp":0,"kbs":0,"remote":[]}]`)
	if err != nil || len(ps) != 2 || ps[0].Name != "cs2" || ps[1].Remote == nil {
		t.Fatalf("%+v %v", ps, err)
	}
	one, err := ParseNetMon(`{"pid":1,"name":"x","tcp":1,"udp":0,"kbs":0,"remote":[]}`)
	if err != nil || len(one) != 1 {
		t.Fatal(one, err)
	}
	if e, _ := ParseNetMon(""); e == nil || len(e) != 0 {
		t.Fatal("vacío debe ser lista no nula")
	}
}

func TestDNSTweaksUseTheirServers(t *testing.T) {
	g, ok := tweaks.ByID("net.dns.google")
	if !ok || !strings.Contains(g.Ops[0].Apply, "8.8.8.8") || strings.Contains(g.Ops[0].Apply, "1.1.1.1") {
		t.Fatal("google mal armado")
	}
	for _, id := range []string{"net.dns", "net.dns.google", "net.dns.quad9", "net.dns.opendns"} {
		tw, ok := tweaks.ByID(id)
		if !ok || tw.Group != "dns" || tw.Ops[0].Revert == "" {
			t.Fatalf("%s", id)
		}
	}
}
