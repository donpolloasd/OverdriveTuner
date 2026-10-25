package telemetry

import (
	"bufio"
	"encoding/base64"
	"os/exec"
	"syscall"
	"unicode/utf16"
)

func encode(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Stream lanza el muestreador y entrega una medición por vez hasta que stop se cierra.
func Stream(stop <-chan struct{}) <-chan Sample {
	out := make(chan Sample, 4)
	go func() {
		defer close(out)
		cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
			"-EncodedCommand", encode(Script))
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return
		}
		if err := cmd.Start(); err != nil {
			return
		}
		go func() { <-stop; _ = cmd.Process.Kill() }()
		sc := bufio.NewScanner(pipe)
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		for sc.Scan() {
			s, err := ParseLine(sc.Text(), GPUTemps())
			if err != nil {
				continue
			}
			select {
			case out <- s:
			case <-stop:
				return
			}
		}
		_ = cmd.Wait()
	}()
	return out
}
