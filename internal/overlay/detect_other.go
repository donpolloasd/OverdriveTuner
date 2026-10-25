//go:build !windows

package overlay

func foregroundFullscreenExe() string { return "" }

func launcherGame(extra []string) string { return "" }
