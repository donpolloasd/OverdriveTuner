package telemetry

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	gdi32          = syscall.NewLazyDLL("gdi32.dll")
	pEnumAdapters2 = gdi32.NewProc("D3DKMTEnumAdapters2")
	pQueryAdapter  = gdi32.NewProc("D3DKMTQueryAdapterInfo")
	pCloseAdapter  = gdi32.NewProc("D3DKMTCloseAdapter")
	kmtAdapterPerf = uint32(62) // KMTQAITYPE_ADAPTERPERFDATA
	kmtMaxAdapters = 16
)

type kmtAdapterInfo struct {
	Handle     uint32
	LuidLow    uint32
	LuidHigh   int32
	NumSources uint32
	Precise    int32
}

type kmtEnum2 struct {
	Num      uint32
	_        uint32
	Adapters uintptr
}

type kmtQuery struct {
	Handle uint32
	Type   uint32
	Data   uintptr
	Size   uint32
	_      uint32
}

type kmtPerf struct {
	PhysicalAdapterIndex uint32
	MemoryFrequency      uint64
	MaxMemoryFrequency   uint64
	MaxMemoryFrequencyOC uint64
	MemoryBandwidth      uint64
	PCIEBandwidth        uint64
	FanRPM               uint32
	Power                uint32
	Temperature          uint32
	PowerStateOverride   uint8
}

type kmtClose struct{ Handle uint32 }

// GPUTemps devuelve la temperatura (°C) de cada gráfica por luid, usando la
// misma consulta de Windows (WDDM) que el Administrador de tareas. No usa
// drivers de terceros. Si el driver no la expone, el mapa queda vacío.
func GPUTemps() (res map[string]float64) {
	res = map[string]float64{}
	defer func() { _ = recover() }()
	if pEnumAdapters2.Find() != nil {
		return
	}
	list := make([]kmtAdapterInfo, kmtMaxAdapters)
	e := kmtEnum2{Num: uint32(kmtMaxAdapters), Adapters: uintptr(unsafe.Pointer(&list[0]))}
	if st, _, _ := pEnumAdapters2.Call(uintptr(unsafe.Pointer(&e))); st != 0 {
		return
	}
	for i := 0; i < int(e.Num) && i < kmtMaxAdapters; i++ {
		a := list[i]
		var perf kmtPerf
		q := kmtQuery{Handle: a.Handle, Type: kmtAdapterPerf, Data: uintptr(unsafe.Pointer(&perf)), Size: uint32(unsafe.Sizeof(perf))}
		if st, _, _ := pQueryAdapter.Call(uintptr(unsafe.Pointer(&q))); st == 0 && perf.Temperature > 0 {
			key := NormLuid(fmt.Sprintf("0x%08X_0x%08X", uint32(a.LuidHigh), a.LuidLow))
			res[key] = float64(perf.Temperature) / 10
		}
		c := kmtClose{Handle: a.Handle}
		pCloseAdapter.Call(uintptr(unsafe.Pointer(&c)))
	}
	return res
}
