package tools

import (
	"context"
	"math"
	"net"
	"sort"
	"sync"
	"time"
)

// GameTarget es un destino de la prueba de ping a servidores.
type GameTarget struct {
	Group string // "games" o "regions"
	Name  string
	Host  string
}

// GameTargets: servicios de juegos y centros de datos donde suelen estar los servidores. Los servicios de
// juegos usan CDN, así que miden el punto de entrada más cercano; las regiones (endpoints públicos de AWS)
// dan una idea más fiel de la distancia a los servidores de partida.
var GameTargets = []GameTarget{
	{"games", "Steam", "steamcommunity.com"},
	{"games", "FACEIT", "api.faceit.com"},
	{"games", "EA", "www.ea.com"},
	{"games", "Epic Games", "www.epicgames.com"},
	{"games", "Riot Games", "www.riotgames.com"},
	{"games", "Battle.net", "www.battle.net"},
	{"games", "Ubisoft", "www.ubisoft.com"},
	{"games", "Xbox / Microsoft", "www.xboxlive.com"},
	{"regions", "Sudamérica (São Paulo)", "ec2.sa-east-1.amazonaws.com"},
	{"regions", "EE. UU. Este (Virginia)", "ec2.us-east-1.amazonaws.com"},
	{"regions", "EE. UU. Este (Ohio)", "ec2.us-east-2.amazonaws.com"},
	{"regions", "EE. UU. Oeste (Oregón)", "ec2.us-west-2.amazonaws.com"},
	{"regions", "Europa (Fráncfort)", "ec2.eu-central-1.amazonaws.com"},
	{"regions", "Europa (Irlanda)", "ec2.eu-west-1.amazonaws.com"},
	{"regions", "Europa (España)", "ec2.eu-south-2.amazonaws.com"},
	{"regions", "Asia (Singapur)", "ec2.ap-southeast-1.amazonaws.com"},
	{"regions", "Asia (Tokio)", "ec2.ap-northeast-1.amazonaws.com"},
}

// GamePing es el resultado de un destino.
type GamePing struct {
	Group  string  `json:"group"`
	Name   string  `json:"name"`
	Host   string  `json:"host"`
	Sent   int     `json:"sent"`
	LossPc int     `json:"lossPct"`
	Min    int     `json:"min"`
	Avg    int     `json:"avg"`
	Max    int     `json:"max"`
	Jitter float64 `json:"jitter"`
	Status string  `json:"status"` // ok, warn, bad, error
}

// DialFn abre una conexión TCP (se reemplaza en las pruebas).
type DialFn func(ctx context.Context, addr string) error

func realDial(ctx context.Context, addr string) error {
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return c.Close()
}

// measureHost mide el tiempo de conexión TCP varias veces. Un intento que falla cuenta como paquete perdido:
// es una estimación (TCP reintenta por su cuenta), pero muestra bien una conexión inestable.
func measureHost(dial DialFn, t GameTarget, n int, gap, timeout time.Duration) GamePing {
	res := GamePing{Group: t.Group, Name: t.Name, Host: t.Host, Sent: n}
	var ms []float64
	for i := 0; i < n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		t0 := time.Now()
		err := dial(ctx, net.JoinHostPort(t.Host, "443"))
		d := time.Since(t0)
		cancel()
		if err == nil {
			ms = append(ms, float64(d.Microseconds())/1000)
		}
		if i < n-1 {
			time.Sleep(gap)
		}
	}
	lost := n - len(ms)
	res.LossPc = int(math.Round(float64(lost) * 100 / float64(n)))
	if len(ms) == 0 {
		res.Status = "error"
		return res
	}
	// jitter: variación media entre mediciones consecutivas, en el orden en que se hicieron
	if len(ms) > 1 {
		dev := 0.0
		for i := 1; i < len(ms); i++ {
			dev += math.Abs(ms[i] - ms[i-1])
		}
		res.Jitter = math.Round(dev/float64(len(ms)-1)*10) / 10
	}
	sum := 0.0
	for _, v := range ms {
		sum += v
	}
	res.Avg = int(math.Round(sum / float64(len(ms))))
	sort.Float64s(ms)
	res.Min, res.Max = int(math.Round(ms[0])), int(math.Round(ms[len(ms)-1]))
	switch {
	case res.LossPc >= 10:
		res.Status = "bad"
	case res.LossPc > 0 || res.Jitter > 15:
		res.Status = "warn"
	default:
		res.Status = "ok"
	}
	return res
}

// GamePingTest mide todos los destinos a la vez (cada uno con su propia secuencia) y los ordena por grupo y ping.
func GamePingTest(dial DialFn, targets []GameTarget, n int, gap, timeout time.Duration) []GamePing {
	if dial == nil {
		dial = realDial
	}
	if targets == nil {
		targets = GameTargets
	}
	out := make([]GamePing, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t GameTarget) {
			defer wg.Done()
			out[i] = measureHost(dial, t, n, gap, timeout)
		}(i, t)
	}
	wg.Wait()
	rank := func(p GamePing) int {
		if p.Status == "error" {
			return 1 << 30
		}
		return p.Avg
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return rank(out[i]) < rank(out[j])
	})
	return out
}
