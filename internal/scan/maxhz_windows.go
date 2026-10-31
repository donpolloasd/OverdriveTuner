//go:build windows

package scan

import (
	"syscall"
	"unsafe"
)

// devMode es DEVMODEW (la parte de pantallas).
type devMode struct {
	DeviceName                                    [32]uint16
	SpecVersion, DriverVersion, Size, DriverExtra uint16
	Fields                                        uint32
	PositionX, PositionY                          int32
	Orientation, FixedOutput                      uint32
	Color, Duplex, YResolution, TTOption, Collate int16
	FormName                                      [32]uint16
	LogPixels                                     uint16
	BitsPerPel, PelsWidth, PelsHeight             uint32
	DisplayFlags, DisplayFrequency                uint32
	ICMMethod, ICMIntent, MediaType, DitherType   uint32
	Reserved1, Reserved2                          uint32
	PanningWidth, PanningHeight                   uint32
}

var pEnumDisplaySettings = syscall.NewLazyDLL("user32.dll").NewProc("EnumDisplaySettingsW")

// maxRefresh devuelve la frecuencia más alta que Windows ofrece al monitor principal con esa resolución
// (con w = 0, con cualquier resolución). 0 si no se pudo saber.
func maxRefresh(w, h int) (mx int) {
	defer func() {
		if recover() != nil {
			mx = 0
		}
	}()
	var dm devMode
	dm.Size = uint16(unsafe.Sizeof(dm))
	for i := 0; i < 4000; i++ {
		if r, _, _ := pEnumDisplaySettings.Call(0, uintptr(i), uintptr(unsafe.Pointer(&dm))); r == 0 {
			break
		}
		if (w <= 0 || (int(dm.PelsWidth) == w && int(dm.PelsHeight) == h)) && int(dm.DisplayFrequency) > mx {
			mx = int(dm.DisplayFrequency)
		}
	}
	return mx
}
