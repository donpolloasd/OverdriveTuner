//go:build !windows

package overlay

func registryGameDirs() []string { return nil }

func steamRoot() string { return "" }
