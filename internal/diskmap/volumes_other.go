//go:build !windows

package diskmap

// Volumes fuera de Windows: solo la raíz.
func Volumes() []Volume { return []Volume{{Root: "/"}} }
