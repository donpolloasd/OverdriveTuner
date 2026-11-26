package tools

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/system"
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
	Jitter  int    `json:"jitter"`            // diferencia entre las consultas rápidas y las lentas (ms)
	Current bool   `json:"current,omitempty"` // es el DNS que usa Windows ahora
	Note    string `json:"note,omitempty"`
}

type dnsServer struct{ name, ip, tweak, note string }

var dnsServers = []dnsServer{
	{"Cloudflare", "1.1.1.1", "net.dns", ""},
	{"Google", "8.8.8.8", "net.dns.google", ""},
	{"Quad9", "9.9.9.9", "net.dns.quad9", "Bloquea sitios maliciosos."},
	{"OpenDNS", "208.67.222.222", "net.dns.opendns", ""},
	{"AdGuard", "94.140.14.14", "", "Bloquea publicidad. Solo se compara."},
	{"Cloudflare sin malware", "1.1.1.2", "", "Bloquea sitios maliciosos. Solo se compara."},
	{"Level3", "4.2.2.1", "", "Solo se compara."},
}

var dnsHosts = []string{"google.com", "steampowered.com", "cloudflare.com", "microsoft.com", "faceit.com", "ea.com"}

// CurrentDNS devuelve los DNS (IPv4) que usa ahora la conexión a internet.
func CurrentDNS(sys system.System) []string {
	out, err := sys.PowerShell(`$i = Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Sort-Object { $_.RouteMetric + $_.InterfaceMetric } | Select-Object -First 1
if ($i) { (Get-DnsClientServerAddress -InterfaceIndex $i.ifIndex -AddressFamily IPv4).ServerAddresses | ForEach-Object { 'DNS=' + $_ } }`, 20*time.Second)
	if err != nil {
		return nil
	}
	var l []string
	for _, ln := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(ln), "DNS="); ok && net.ParseIP(v) != nil && len(l) < 3 {
			l = append(l, v)
		}
	}
	return l
}

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

// DNSBench mide cada servidor con varias consultas y devuelve la mediana. current son los DNS que usa
// Windows ahora y custom, los que escribió el usuario: se comparan junto con los conocidos, todos a la vez.
func DNSBench(lookup LookupFn, current, custom []string) []DNSResult {
	if lookup == nil {
		lookup = realLookup
	}
	list := append([]dnsServer(nil), dnsServers...)
	add := func(ip, name string) {
		for _, sv := range list {
			if sv.ip == ip {
				return
			}
		}
		list = append(list, dnsServer{name: name, ip: ip})
	}
	for _, ip := range current {
		add(ip, "Tu DNS actual")
	}
	for _, ip := range custom {
		add(ip, "Personalizado")
	}
	out := make([]DNSResult, len(list))
	var wg sync.WaitGroup
	for i, sv := range list {
		wg.Add(1)
		go func() { defer wg.Done(); out[i] = dnsMeasure(lookup, sv) }()
	}
	wg.Wait()
	best := -1
	for i := range out {
		for _, ip := range current {
			if out[i].IP == ip {
				out[i].Current = true
			}
		}
		if r := out[i]; r.Median >= 0 && (best < 0 || r.Median < out[best].Median) {
			best = i
		}
	}
	if best >= 0 {
		out[best].Best = true
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Median, out[j].Median
		if (a < 0) != (b < 0) {
			return b < 0
		}
		return a < b
	})
	return out
}

func dnsMeasure(lookup LookupFn, sv dnsServer) DNSResult {
	res := DNSResult{Name: sv.name, IP: sv.ip, TweakID: sv.tweak, Note: sv.note}
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
		res.Jitter = ms[len(ms)*9/10] - ms[len(ms)/10]
	} else {
		res.Median = -1
	}
	return res
}
