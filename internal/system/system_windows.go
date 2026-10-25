//go:build windows

package system

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const createNoWindow = 0x08000000

type winSys struct{}

// New devuelve la implementación real para Windows.
func New() System { return &winSys{} }

func (winSys) Name() string { return "windows" }

// IsAdmin comprueba si el proceso corre con token elevado.
func (winSys) IsAdmin() bool {
	p, err := syscall.GetCurrentProcess()
	if err != nil {
		return false
	}
	var token syscall.Token
	if err := syscall.OpenProcessToken(p, syscall.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()
	var elevated uint32
	var ret uint32
	const tokenElevation = 20
	err = syscall.GetTokenInformation(token, tokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &ret)
	return err == nil && elevated != 0
}

func decodeOutput(b []byte) string {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	return string(b)
}

func (winSys) Exec(timeout time.Duration, name string, args ...string) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
			err = nil
		} else {
			code = -1
		}
	}
	if ctx.Err() == context.DeadlineExceeded {
		return decodeOutput(buf.Bytes()), -1, fmt.Errorf("tiempo agotado ejecutando %s", filepath.Base(name))
	}
	return decodeOutput(buf.Bytes()), code, err
}

func powershellPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
}

const psPrelude = "$ErrorActionPreference='Continue'\r\n$ProgressPreference='SilentlyContinue'\r\n" +
	"try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false } catch {}\r\n"

func (w winSys) PowerShell(script string, timeout time.Duration) (string, error) {
	dir := filepath.Join(os.TempDir(), "overdrivetuner")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "run-*.ps1")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)
	// BOM UTF-8 para que PowerShell 5.1 lea bien los acentos.
	if _, err := f.Write(append([]byte{0xEF, 0xBB, 0xBF}, []byte(psPrelude+script+"\r\nexit 0\r\n")...)); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	out, code, err := w.Exec(timeout, powershellPath(), "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-File", path)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, fmt.Errorf("PowerShell terminó con código %d: %s", code, strings.TrimSpace(out))
	}
	return out, nil
}
