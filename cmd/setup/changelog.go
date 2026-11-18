package main

import "fpsforge/internal/changelog"

// Release agrupa los cambios de una versión (vienen del paquete changelog, compartido con el programa).
type Release = changelog.Release

var releases = changelog.Releases

func newer(a, b string) bool { return changelog.Newer(a, b) }

func releasesSince(installed string) []Release { return changelog.Since(installed) }
