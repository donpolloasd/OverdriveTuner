//go:build !windows

package rtss

// Read no hace nada fuera de Windows.
func Read() []Entry { return nil }

// ReadSensors no hace nada fuera de Windows.
func ReadSensors() []Sensor { return nil }

// Tick no se usa fuera de Windows.
func Tick() uint32 { return 0 }

// Fault existe solo para que compile en otros sistemas.
var Fault string

// Running no se usa fuera de Windows.
func Running() bool { return false }

// ExePath no se usa fuera de Windows.
func ExePath() string { return "" }

// EnsureRunning no se usa fuera de Windows.
func EnsureRunning() bool { return false }

// EnsureCoreTemp no se usa fuera de Windows.
func EnsureCoreTemp() bool { return false }
