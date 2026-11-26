package tools

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDNSBenchPicksFastest(t *testing.T) {
	lat := map[string]time.Duration{"1.1.1.1": 20 * time.Millisecond, "8.8.8.8": 9 * time.Millisecond, "9.9.9.9": 30 * time.Millisecond}
	res := DNSBench(func(_ context.Context, s, _ string) (time.Duration, error) {
		if d, ok := lat[s]; ok {
			return d, nil
		}
		return 0, errors.New("timeout")
	}, []string{"8.8.8.8", "192.168.0.1"}, []string{"9.9.9.11"})
	if len(res) != len(dnsServers)+2 || res[0].IP != "8.8.8.8" || !res[0].Current {
		t.Fatal(len(res))
	}
	for _, r := range res {
		if r.Best && r.IP != "8.8.8.8" {
			t.Fatalf("mejor incorrecto: %+v", r)
		}
		if r.IP == "208.67.222.222" && (r.Median != -1 || r.Fails == 0) {
			t.Fatalf("caído mal marcado: %+v", r)
		}
	}
}

func TestDNSBenchAllDown(t *testing.T) {
	res := DNSBench(func(context.Context, string, string) (time.Duration, error) { return 0, errors.New("x") }, nil, nil)
	for _, r := range res {
		if r.Best || r.Median != -1 {
			t.Fatalf("%+v", r)
		}
	}
}
