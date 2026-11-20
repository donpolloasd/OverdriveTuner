//go:build !windows

package extras

func RecycleBin() (bytes, items int64, err error) { return 0, 0, nil }

func EmptyRecycleBin() error { return nil }
