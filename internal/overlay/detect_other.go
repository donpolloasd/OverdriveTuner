//go:build !windows

package overlay

func foregroundFullscreenExe() string { return "" }

func foregroundAnyExe() string { return "" }

func launcherGame(extra []string) string { return "" }

func exclusiveFullscreen() bool { return false }

func runningExes() (map[string]string, bool) { return nil, false }
