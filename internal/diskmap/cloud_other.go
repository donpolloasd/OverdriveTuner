//go:build !windows

package diskmap

import "io/fs"

func cloudOnly(fs.FileInfo) bool { return false }
