// Package perf mide el rendimiento del equipo sin drivers ni programas externos: procesador (un núcleo y
// todos), memoria RAM (velocidad y latencia), disco (lectura y escritura) y una prueba corta de estabilidad
// que además detecta si el procesador baja su velocidad por temperatura o energía (throttling).
package perf

import (
	"context"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Clases de prueba.
const (
	CPU1      = "cpu1"
	CPUAll    = "cpuall"
	RAMSpeed  = "ram"
	RAMLat    = "ramlat"
	DiskRead  = "diskr"
	DiskWrite = "diskw"
	Stability = "stab"
)

// Kinds son las pruebas disponibles, en el orden en que se muestran.
var Kinds = []string{CPU1, CPUAll, RAMSpeed, RAMLat, DiskRead, DiskWrite, Stability}

// LowerBetter dice si en esa prueba un número más chico es mejor.
func LowerBetter(kind string) bool { return kind == RAMLat }

// ValidKind dice si la prueba existe.
func ValidKind(k string) bool {
	for _, x := range Kinds {
		if x == k {
			return true
		}
	}
	return false
}

// refRate es la velocidad de un núcleo (bloques por segundo) que equivale a 1000 puntos: la de un
// Ryzen 5 5600G. Los puntos solo sirven para comparar entre equipos con este mismo programa.
const refRate = 2800.0

// sink evita que el compilador descarte las cuentas.
var sink atomic.Uint64

// block hace un bloque fijo de trabajo, mezcla de enteros, punto flotante y saltos, y devuelve un número
// que depende de todas las cuentas: siempre tiene que dar lo mismo con la misma semilla.
func block(seed uint64) uint64 {
	x := seed | 1
	f := 1.0 + float64(seed%97)/97
	var tab [256]uint64
	for i := range tab {
		tab[i] = uint64(i) * 0x9E3779B97F4A7C15
	}
	for i := 0; i < 60000; i++ {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		j := x & 255
		tab[j] += x
		if tab[j]&4 == 0 {
			x += tab[(j+31)&255]
		}
		f = f*1.0000001 + float64(j)*1e-6
		if i&63 == 0 {
			f = math.Sqrt(f*f + 1)
		}
	}
	for _, v := range tab {
		x ^= v
	}
	return x ^ math.Float64bits(f)
}

// work repite bloques durante d y devuelve cuántos hizo.
func work(ctx context.Context, d time.Duration) float64 {
	t0 := time.Now()
	n := 0
	var acc uint64
	for time.Since(t0) < d && ctx.Err() == nil {
		acc ^= block(uint64(n))
		n++
	}
	sink.Add(acc)
	return float64(n) / time.Since(t0).Seconds()
}

// CPUSingle mide un núcleo: el mejor de tres tramos, para descartar interrupciones de otros programas.
func CPUSingle(ctx context.Context, d time.Duration, prog func(float64)) (float64, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	work(ctx, d/10) // el procesador sube de velocidad antes de medir
	best := 0.0
	for i := 0; i < 3; i++ {
		best = math.Max(best, work(ctx, d*3/10))
		prog(float64(i+1) / 3)
	}
	return best / refRate * 1000, ctx.Err()
}

// CPUMulti mide todos los hilos a la vez.
func CPUMulti(ctx context.Context, d time.Duration, prog func(float64)) (float64, error) {
	n := runtime.NumCPU()
	rates := make([]float64, n)
	var wg sync.WaitGroup
	for i := range rates {
		wg.Add(1)
		go func(i int) { defer wg.Done(); rates[i] = work(ctx, d) }(i)
	}
	tick(ctx, d, prog)
	wg.Wait()
	sum := 0.0
	for _, r := range rates {
		sum += r
	}
	return sum / refRate * 1000, ctx.Err()
}

// tick informa el avance de algo que dura d.
func tick(ctx context.Context, d time.Duration, prog func(float64)) {
	t0 := time.Now()
	tk := time.NewTicker(100 * time.Millisecond)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			p := time.Since(t0).Seconds() / d.Seconds()
			prog(math.Min(1, p))
			if p >= 1 {
				return
			}
		}
	}
}

// RAM devuelve la velocidad de lectura, escritura y copia en GB/s (con varios hilos, sobre bloques mucho
// más grandes que la caché).
func RAM(ctx context.Context, d time.Duration, mb int, prog func(float64)) (read, write, cp float64, err error) {
	th := runtime.NumCPU() / 2
	if th < 1 {
		th = 1
	}
	if th > 8 {
		th = 8
	}
	per := mb << 20 / 8 / th
	src, dst := make([][]uint64, th), make([][]uint64, th)
	for i := range src {
		src[i], dst[i] = make([]uint64, per), make([]uint64, per)
		for j := range src[i] {
			src[i][j] = uint64(j)
			dst[i][j] = 1
		}
	}
	phase := func(base float64, f func(i int)) float64 {
		best, t0 := 0.0, time.Now()
		for time.Since(t0) < d/3 && ctx.Err() == nil {
			s, reps := time.Now(), 0
			for reps == 0 || time.Since(s) < 40*time.Millisecond { // pasadas muy cortas no se pueden cronometrar bien
				var wg sync.WaitGroup
				for i := 0; i < th; i++ {
					wg.Add(1)
					go func(i int) { defer wg.Done(); f(i) }(i)
				}
				wg.Wait()
				reps++
			}
			best = math.Max(best, float64(reps)*float64(per*8*th)/time.Since(s).Seconds()/1e9)
			prog(base + math.Min(1, time.Since(t0).Seconds()/(d/3).Seconds())/3)
		}
		return best
	}
	read = phase(0, func(i int) {
		var a, b, c, e uint64
		s := src[i]
		for j := 0; j+16 <= len(s); j += 16 {
			v := s[j : j+16 : j+16]
			a += v[0] + v[1] + v[2] + v[3]
			b += v[4] + v[5] + v[6] + v[7]
			c += v[8] + v[9] + v[10] + v[11]
			e += v[12] + v[13] + v[14] + v[15]
		}
		sink.Add(a + b + c + e)
	})
	write = phase(1.0/3, func(i int) {
		s := dst[i]
		for j := 0; j+16 <= len(s); j += 16 {
			v := s[j : j+16 : j+16]
			for k := range v {
				v[k] = uint64(j)
			}
		}
	})
	cp = phase(2.0/3, func(i int) { copy(dst[i], src[i]) })
	return read, write, cp, ctx.Err()
}

// RAMLatency mide cuánto tarda un acceso al azar a la RAM, en nanosegundos: sigue una cadena desordenada
// donde cada dirección depende de la anterior, así el procesador no puede adelantarse.
func RAMLatency(ctx context.Context, d time.Duration, mb int, prog func(float64)) (float64, error) {
	const line = 16 // uint32 por línea de caché (64 bytes)
	n := mb << 20 / 64
	mem := make([]uint32, n*line)
	order := rand.New(rand.NewSource(42)).Perm(n)
	for i := range order {
		mem[order[i]*line] = uint32(order[(i+1)%n] * line)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	best, p, t0 := math.Inf(1), uint32(0), time.Now()
	for time.Since(t0) < d && ctx.Err() == nil {
		const hops = 1 << 20
		s := time.Now()
		for i := 0; i < hops; i++ {
			p = mem[p]
		}
		best = math.Min(best, float64(time.Since(s).Nanoseconds())/hops)
		prog(math.Min(1, time.Since(t0).Seconds()/d.Seconds()))
	}
	sink.Add(uint64(p))
	if math.IsInf(best, 1) {
		best = 0
	}
	return best, ctx.Err()
}
