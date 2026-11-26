//go:build !windows

package audio

import "time"

func MicMuted() (bool, error)     { return false, ErrNoMic }
func SetMicMuted(mute bool) error { return ErrNoMic }

func Record(d time.Duration, stop <-chan struct{}, level func(float64)) ([]int16, error) {
	return nil, ErrNoMic
}
