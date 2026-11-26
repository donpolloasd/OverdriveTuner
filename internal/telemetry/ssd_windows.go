package telemetry

import (
	"encoding/binary"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var pDeviceIoControl = syscall.NewLazyDLL("kernel32.dll").NewProc("DeviceIoControl")

// diskQuery hace una consulta de solo lectura (IOCTL_STORAGE_QUERY_PROPERTY) al disco n. Se abre sin
// permisos de lectura ni escritura: no toca los datos.
func diskQuery(n int, prop uint32) []byte {
	p, _ := syscall.UTF16PtrFromString(fmt.Sprintf(`\\.\PhysicalDrive%d`, n))
	h, err := syscall.CreateFile(p, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(h)
	q := [3]uint32{prop, 0, 0}
	out := make([]byte, 1024)
	var got uint32
	r, _, _ := pDeviceIoControl.Call(uintptr(h), 0x2D1400, uintptr(unsafe.Pointer(&q[0])), unsafe.Sizeof(q),
		uintptr(unsafe.Pointer(&out[0])), uintptr(len(out)), uintptr(unsafe.Pointer(&got)), 0)
	if r == 0 || int(got) > len(out) {
		return nil
	}
	return out[:got]
}

// parseDiskTemp interpreta STORAGE_TEMPERATURE_DATA_DESCRIPTOR: la temperatura más alta de sus sensores.
func parseDiskTemp(b []byte) (float64, bool) {
	const head, item = 24, 16
	if len(b) < head+item {
		return 0, false
	}
	n := int(binary.LittleEndian.Uint16(b[12:]))
	best := 0.0
	for i := 0; i < n && head+(i+1)*item <= len(b); i++ {
		t := float64(int16(binary.LittleEndian.Uint16(b[head+i*item+2:])))
		if t > best && t < 120 {
			best = t
		}
	}
	return best, best > 0
}

var (
	ssdMu   sync.Mutex
	ssdAt   time.Time
	ssdTemp = Unknown
)

// SSDTemp devuelve la temperatura del SSD más caliente (Unknown si Windows no la informa; pasa con muchos
// discos SATA). Se consulta como mucho cada 5 segundos.
func SSDTemp() float64 {
	ssdMu.Lock()
	defer ssdMu.Unlock()
	if time.Since(ssdAt) < 5*time.Second {
		return ssdTemp
	}
	ssdAt, ssdTemp = time.Now(), Unknown
	for n := 0; n < 8; n++ {
		// Solo discos sin costo de búsqueda (SSD): así no se despierta un disco mecánico.
		if s := diskQuery(n, 7); len(s) < 9 || s[8] != 0 {
			continue
		}
		for _, prop := range []uint32{52, 51} { // temperatura del dispositivo y, si no, del controlador
			if t, ok := parseDiskTemp(diskQuery(n, prop)); ok {
				if t > ssdTemp {
					ssdTemp = t
				}
				break
			}
		}
	}
	return ssdTemp
}
