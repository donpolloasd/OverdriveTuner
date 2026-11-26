package audio

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	ole32           = syscall.NewLazyDLL("ole32.dll")
	pCoInit         = ole32.NewProc("CoInitializeEx")
	pCoUninit       = ole32.NewProc("CoUninitialize")
	pCoCreate       = ole32.NewProc("CoCreateInstance")
	winmm           = syscall.NewLazyDLL("winmm.dll")
	pWaveInNumDevs  = winmm.NewProc("waveInGetNumDevs")
	pWaveInOpen     = winmm.NewProc("waveInOpen")
	pWaveInPrepare  = winmm.NewProc("waveInPrepareHeader")
	pWaveInUnprep   = winmm.NewProc("waveInUnprepareHeader")
	pWaveInAdd      = winmm.NewProc("waveInAddBuffer")
	pWaveInStart    = winmm.NewProc("waveInStart")
	pWaveInReset    = winmm.NewProc("waveInReset")
	pWaveInClose    = winmm.NewProc("waveInClose")
	clsidEnumerator = syscall.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidEnumerator   = syscall.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	iidEndpointVol  = syscall.GUID{Data1: 0x5CDF2C82, Data2: 0x841E, Data3: 0x4546, Data4: [8]byte{0x97, 0x22, 0x0C, 0xF7, 0x40, 0x78, 0x22, 0x9A}}
)

// comMethod llama al método n de una interfaz COM.
func comMethod(obj unsafe.Pointer, n int, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, uintptr(n)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{uintptr(obj)}, args...)...)
	return r
}

// withMic llama a f con el control de volumen del micrófono predeterminado (el normal y el de
// comunicaciones, si son distintos).
func withMic(f func(vol unsafe.Pointer) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if r, _, _ := pCoInit.Call(0, 0); int32(r) >= 0 {
		defer pCoUninit.Call()
	}
	var enum unsafe.Pointer
	if r, _, _ := pCoCreate.Call(uintptr(unsafe.Pointer(&clsidEnumerator)), 0, 23, uintptr(unsafe.Pointer(&iidEnumerator)), uintptr(unsafe.Pointer(&enum))); int32(r) < 0 || enum == nil {
		return fmt.Errorf("Windows no dejó usar el audio (0x%08X)", uint32(r))
	}
	defer comMethod(enum, 2)
	done := false
	var first error
	for _, role := range []uintptr{0, 2} { // eConsole y eCommunications
		var dev unsafe.Pointer
		if r := comMethod(enum, 4, 1, role, uintptr(unsafe.Pointer(&dev))); int32(r) < 0 || dev == nil { // GetDefaultAudioEndpoint(eCapture)
			continue
		}
		var vol unsafe.Pointer
		r := comMethod(dev, 3, uintptr(unsafe.Pointer(&iidEndpointVol)), 23, 0, uintptr(unsafe.Pointer(&vol))) // Activate
		comMethod(dev, 2)
		if int32(r) < 0 || vol == nil {
			continue
		}
		err := f(vol)
		comMethod(vol, 2)
		if err == nil {
			done = true
		} else if first == nil {
			first = err
		}
	}
	if done {
		return nil
	}
	if first != nil {
		return first
	}
	return ErrNoMic
}

// MicMuted dice si el micrófono predeterminado está silenciado en Windows.
func MicMuted() (muted bool, err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("no se pudo leer el micrófono")
		}
	}()
	first := true
	err = withMic(func(vol unsafe.Pointer) error {
		var m int32
		if r := comMethod(vol, 15, uintptr(unsafe.Pointer(&m))); int32(r) < 0 { // GetMute
			return fmt.Errorf("no se pudo leer el micrófono (0x%08X)", uint32(r))
		}
		if first { // el normal manda
			muted, first = m != 0, false
		}
		return nil
	})
	return
}

// SetMicMuted silencia o activa el micrófono predeterminado de Windows.
func SetMicMuted(mute bool) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("no se pudo cambiar el micrófono")
		}
	}()
	var v uintptr
	if mute {
		v = 1
	}
	return withMic(func(vol unsafe.Pointer) error {
		if r := comMethod(vol, 14, v, 0); int32(r) < 0 { // SetMute
			return fmt.Errorf("no se pudo cambiar el micrófono (0x%08X)", uint32(r))
		}
		return nil
	})
}

type waveFormat struct {
	Tag, Channels uint16
	Rate, Bytes   uint32
	Align, Bits   uint16
	Extra         uint16
}

type waveHdr struct {
	Data             uintptr
	Length, Recorded uint32
	User             uintptr
	Flags, Loops     uint32
	Next, Reserved   uintptr
}

// Record graba d del micrófono predeterminado (mono, 16 bits, Rate Hz). level recibe el nivel (0 a 1) varias
// veces por segundo. Nada se guarda en disco.
func Record(d time.Duration, stop <-chan struct{}, level func(float64)) (pcm []int16, err error) {
	if n, _, _ := pWaveInNumDevs.Call(); n == 0 {
		return nil, ErrNoMic
	}
	wf := waveFormat{Tag: 1, Channels: 1, Rate: Rate, Bytes: Rate * 2, Align: 2, Bits: 16}
	var h uintptr
	if r, _, _ := pWaveInOpen.Call(uintptr(unsafe.Pointer(&h)), 0xFFFFFFFF, uintptr(unsafe.Pointer(&wf)), 0, 0, 0); r != 0 {
		return nil, fmt.Errorf("no se pudo abrir el micrófono (código %d). Revisá que Windows les permita el micrófono a las aplicaciones de escritorio.", r)
	}
	const nbuf, per = 8, Rate / 20
	// Los bloques tienen que quedar quietos en memoria mientras el controlador escribe en ellos.
	var pin runtime.Pinner
	defer pin.Unpin()
	bufs, hdrs := make([][]int16, nbuf), make([]*waveHdr, nbuf)
	hs := unsafe.Sizeof(waveHdr{})
	for i := range bufs {
		bufs[i], hdrs[i] = make([]int16, per), &waveHdr{}
		pin.Pin(&bufs[i][0])
		pin.Pin(hdrs[i])
		hdrs[i].Data, hdrs[i].Length = uintptr(unsafe.Pointer(&bufs[i][0])), per*2
		pWaveInPrepare.Call(h, uintptr(unsafe.Pointer(hdrs[i])), hs)
		pWaveInAdd.Call(h, uintptr(unsafe.Pointer(hdrs[i])), hs)
	}
	defer func() {
		pWaveInReset.Call(h)
		for _, x := range hdrs {
			pWaveInUnprep.Call(h, uintptr(unsafe.Pointer(x)), hs)
		}
		pWaveInClose.Call(h)
	}()
	if r, _, _ := pWaveInStart.Call(h); r != 0 {
		return nil, fmt.Errorf("el micrófono no empezó a grabar (código %d)", r)
	}
	t0, i := time.Now(), 0
	for time.Since(t0) < d {
		select {
		case <-stop:
			return pcm, nil
		case <-time.After(10 * time.Millisecond):
		}
		for hdrs[i].Flags&1 != 0 { // WHDR_DONE
			n := int(hdrs[i].Recorded / 2)
			pcm = append(pcm, bufs[i][:n]...)
			if level != nil {
				level(peak(bufs[i][:n]))
			}
			hdrs[i].Flags &^= 1
			hdrs[i].Recorded = 0
			pWaveInAdd.Call(h, uintptr(unsafe.Pointer(hdrs[i])), hs)
			i = (i + 1) % nbuf
		}
	}
	if len(pcm) == 0 {
		return nil, errors.New("El micrófono no mandó ningún sonido. Revisá que Windows les permita el micrófono a las aplicaciones de escritorio.")
	}
	return pcm, nil
}
