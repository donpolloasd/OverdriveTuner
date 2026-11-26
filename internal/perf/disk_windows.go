package perf

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const (
	flagNoBuffering  = 0x20000000
	flagWriteThrough = 0x80000000
	flagDeleteClose  = 0x04000000
	diskChunk        = 1 << 20
	diskMB           = 1024 // tamaño del archivo de prueba
)

// alignedBuf devuelve un bloque alineado a 4 KB (lo exige el acceso sin caché).
func alignedBuf(n int) []byte {
	b := make([]byte, n+4096)
	off := int(4096 - uintptr(unsafe.Pointer(&b[0]))%4096)
	return b[off : off+n : off+n]
}

func openRaw(path string, create bool) (syscall.Handle, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	mode := uint32(syscall.OPEN_EXISTING)
	if create {
		mode = syscall.CREATE_ALWAYS
	}
	// Se borra solo al cerrarlo, aunque el programa se cierre de golpe.
	return syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, mode,
		flagNoBuffering|flagWriteThrough|flagDeleteClose, 0)
}

func freeBytes(dir string) int64 {
	p, _ := syscall.UTF16PtrFromString(dir)
	var free int64
	r, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW").Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&free)), 0, 0)
	if r == 0 {
		return -1
	}
	return free
}

// Disk mide un disco con un archivo temporal en dir, sin la caché de Windows. Devuelve MB/s en secuencial y
// operaciones de 4 KB por segundo al azar (de a una, como la mayoría de los juegos y Windows).
func Disk(ctx context.Context, dir string, write bool, prog func(float64)) (mbs, iops float64, err error) {
	return disk(ctx, dir, write, diskMB, 8*time.Second, 3*time.Second, prog)
}

func disk(ctx context.Context, dir string, write bool, mb int, seqMax, rndTime time.Duration, prog func(float64)) (mbs, iops float64, err error) {
	if free := freeBytes(dir); free >= 0 && free < int64(mb)<<20+2<<30 {
		return 0, 0, errors.New("No hay espacio suficiente en ese disco para la prueba (hacen falta unos 3 GB libres).")
	}
	path := filepath.Join(dir, fmt.Sprintf("OverdriveTuner-prueba-%d.tmp", os.Getpid()))
	h, err := openRaw(path, true)
	if err != nil {
		return 0, 0, fmt.Errorf("no se pudo crear el archivo de prueba: %w", err)
	}
	defer syscall.CloseHandle(h)
	buf := alignedBuf(diskChunk)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, _ = rnd.Read(buf) // datos que no se pueden comprimir
	// Escritura secuencial (también prepara el archivo para leer). Se corta por tiempo en discos lentos.
	t0, done := time.Now(), 0
	share := 0.6
	if !write {
		share = 0.4
	}
	for done < mb && ctx.Err() == nil && (time.Since(t0) < seqMax || done < 64 && done < mb) {
		var n uint32
		if err := syscall.WriteFile(h, buf, &n, nil); err != nil || n != diskChunk {
			return 0, 0, fmt.Errorf("no se pudo escribir el archivo de prueba: %v", err)
		}
		done++
		prog(share * math.Max(float64(done)/float64(mb), math.Min(1, time.Since(t0).Seconds()/seqMax.Seconds())))
	}
	mbs = float64(done) * diskChunk / 1e6 / time.Since(t0).Seconds()
	if ctx.Err() != nil {
		return 0, 0, ctx.Err()
	}
	if !write {
		if _, err := syscall.Seek(h, 0, 0); err != nil {
			return 0, 0, err
		}
		t0 = time.Now()
		i := 0
		for ; i < done && ctx.Err() == nil && time.Since(t0) < seqMax; i++ {
			var n uint32
			if err := syscall.ReadFile(h, buf, &n, nil); err != nil || n != diskChunk {
				return 0, 0, fmt.Errorf("no se pudo leer el archivo de prueba: %v", err)
			}
			prog(share + 0.3*float64(i)/float64(done))
		}
		mbs = float64(i) * diskChunk / 1e6 / time.Since(t0).Seconds()
	}
	small := alignedBuf(4096)
	_, _ = rnd.Read(small)
	t0 = time.Now()
	ops := 0
	for time.Since(t0) < rndTime && ctx.Err() == nil {
		off := int64(rnd.Intn(done<<20/4096)) * 4096
		if _, err := syscall.Seek(h, off, 0); err != nil {
			return 0, 0, err
		}
		var n uint32
		if write {
			err = syscall.WriteFile(h, small, &n, nil)
		} else {
			err = syscall.ReadFile(h, small, &n, nil)
		}
		if err != nil {
			return 0, 0, err
		}
		ops++
		prog(0.7 + 0.3*time.Since(t0).Seconds()/rndTime.Seconds())
	}
	return mbs, float64(ops) / time.Since(t0).Seconds(), ctx.Err()
}
