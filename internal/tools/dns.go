package tools

import (
	"context"
	"net"
	"sort"
	"time"
)

// DNSResult es la medición de un servidor DNS.
type DNSResult struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Median  int    `json:"median"` // ms
	Fails   int    `json:"fails"`
	Best    bool   `json:"best"`
	TweakID string `json:"tweakId,omitempty"` // ajuste que lo aplica
	Applied bool   `json:"applied"`
}

type dnsServer struct{ name, ip, tweak string }

var dnsServers = []dnsServer{
	{"Cloudflare", "1.1.1.1", "net.dns"},
	{"Google", "8.8.8.8", "net.dns.google"},
	{"Quad9", "9.9.9.9", "net.dns.quad9"},
	{"OpenDNS", "208.67.222.222", "net.dns.opendns"},
}

var dnsHosts = []string{"google.com", "steampowered.com", "cloudflare.com", "microsoft.com", "faceit.com", "ea.com"}

// LookupFn mide una consulta DNS a un servidor concreto.
type LookupFn func(ctx context.Context, server, host string) (time.Duration, error)

func realLookup(ctx context.Context, server, host string) (time.Duration, error) {
	r := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "udp", net.JoinHostPort(server, "53"))
	}}
	t0 := time.Now()
	_, err := r.LookupHost(ctx, host)
	return time.Since(t0), err
}

// DNSBench mide cada servidor con varias consultas y devuelve la mediana.
func DNSBench(lookup LookupFn) []DNSResult {
	if lookup == nil {
		lookup = realLookup
	}
	out := make([]DNSResult, 0, len(dnsServers))
	for _, sv := range dnsServers {
		res := DNSResult{Name: sv.name, IP: sv.ip, TweakID: sv.tweak}
		var ms []int
		consecutive := 0
		for round := 0; round < 3 && consecutive < 3; round++ {
			for _, h := range dnsHosts {
				ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
				d, err := lookup(ctx, sv.ip, h)
				cancel()
				if err != nil {
					res.Fails++
					consecutive++
					if consecutive >= 3 {
						break
					}
					continue
				}
				consecutive = 0
				if round > 0 { // la primera ronda calienta la caché del servidor
					ms = append(ms, int(d.Milliseconds()))
				}
			}
		}
		if len(ms) > 0 {
			sort.Ints(ms)
			res.Median = ms[len(ms)/2]
		} else {
			res.Median = -1
		}
		out = append(out, res)
	}
	best := -1
	for i, r := range out {
		if r.Median >= 0 && (best < 0 || r.Median < out[best].Median) {
			best = i
		}
	}
	if best >= 0 {
		out[best].Best = true
	}
	return out
}
