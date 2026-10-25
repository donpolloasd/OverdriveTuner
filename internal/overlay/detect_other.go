//go:build !windows

package overlay

func foregroundFullscreenExe() string { return "" }

func foregroundAnyExe() string { return "" }

func launcherGame(extra []string) string { return "" }
