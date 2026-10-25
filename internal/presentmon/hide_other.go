//go:build !windows

package presentmon

import "os/exec"

func hide(*exec.Cmd) {}
