//go:build windows

package rtss

import (
	"encoding/binary"
	"fmt"
	"runtime/debug"
	"syscall"
	"unsafe"
)

var (
	k32         = syscall.NewLazyDLL("kernel32.dll")
	pOpenMap    = k32.NewProc("OpenFileMappingW")
	pMapView    = k32.NewProc("MapViewOfFile")
	pUnmapView  = k32.NewProc("UnmapViewOfFile")
	pCloseH     = k32.NewProc("CloseHandle")
	pTickCount  = k32.NewProc("GetTickCount")
	pVirtualQry = k32.NewProc("VirtualQuery")
)

// Fault guarda el último error de lectura (vacío si no hubo), para dejarlo en el diagnóstico.
var Fault string

// Read devuelve los programas que RTSS está midiendo (nil si RTSS no está abierto).
func Read() []Entry {
	return Parse(readRegion("RTSSSharedMemoryV2", func(h []byte) uintptr {
		size := uintptr(binary.LittleEndian.Uint32(h[8:]))
		off := uintptr(binary.LittleEndian.Uint32(h[12:]))
		n := uintptr(binary.LittleEndian.Uint32(h[16:]))
		if size < 284 || n > 4096 {
			return 0
		}
		return off + n*size
	}))
}

// ReadSensors devuelve los sensores de MSI Afterburner (MAHM) y la temperatura que publica Core Temp.
func ReadSensors() []Sensor {
	ct := ParseCoreTemp(readRegion("CoreTempMappingObject", func([]byte) uintptr { return CoreTempBytes }))
	return append(readMAHM(), ct...)
}

func readMAHM() []Sensor {
	return ParseSensors(readRegion("MAHMSharedMemory", func(h []byte) uintptr {
		off := uintptr(binary.LittleEndian.Uint32(h[8:]))
		n := uintptr(binary.LittleEndian.Uint32(h[12:]))
		size := uintptr(binary.LittleEndian.Uint32(h[16:]))
		if size < 1324 || n > 512 {
			return 0
		}
		return off + n*size
	}))
}

// readRegion abre una memoria compartida de solo lectura y copia lo que indique extent (calculado
// a partir de los primeros 32 bytes). Nunca debe tumbar el programa: un acceso inválido se
// convierte en un error guardado en Fault.
func readRegion(name string, extent func(head []byte) uintptr) (out []byte) {
	prev := debug.SetPanicOnFault(true)
	defer func() {
		debug.SetPanicOnFault(prev)
		if r := recover(); r != nil {
			Fault = fmt.Sprint("lectura de ", name, " falló: ", r)
			out = nil
		}
	}()
	nm, _ := syscall.UTF16PtrFromString(name)
	h, _, _ := pOpenMap.Call(0x4, 0, uintptr(unsafe.Pointer(nm))) // FILE_MAP_READ
	if h == 0 {
		return nil
	}
	defer pCloseH.Call(h)
	v, _, _ := pMapView.Call(h, 0x4, 0, 0, 0)
	if v == 0 {
		return nil
	}
	defer pUnmapView.Call(v)
	var mbi struct {
		Base, AllocBase uintptr
		AllocProt       uint32
		_               uint32
		Size            uintptr
		State, Prot, Ty uint32
	}
	r, _, _ := pVirtualQry.Call(v, uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
	if r == 0 || mbi.Size < 32 || mbi.Size > 64<<20 {
		return nil
	}
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&v)) // v es una dirección del sistema, no memoria de Go
	head := make([]byte, 32)
	copy(head, unsafe.Slice((*byte)(ptr), 32))
	need := extent(head)
	if need == 0 {
		return nil
	}
	if need > mbi.Size {
		need = mbi.Size
	}
	buf := make([]byte, need)
	copy(buf, unsafe.Slice((*byte)(ptr), need))
	return buf
}

// Tick es GetTickCount (la misma hora que usa RTSS).
func Tick() uint32 { r, _, _ := pTickCount.Call(); return uint32(r) }
