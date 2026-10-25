//go:build !windows

package winhost

import (
	"fmt"
	"os"
	"os/exec"
)

func Fatal(msg string) { fmt.Fprintln(os.Stderr, msg); os.Exit(1) }

func EnsureElevated() bool            { return false }
func SingleInstance(name string) bool { return true }

func OpenWindowMax(url, dir string) *exec.Cmd { return OpenWindow(url, dir) }

func OpenWindow(url, dir string) *exec.Cmd {
	fmt.Println("Abre en tu navegador:", url)
	return nil
}

func AskYesNo(msg string) bool { fmt.Println(msg); return true }

func Info(msg string) { fmt.Println(msg) }

func Cleanup(mode, dataDir string) {}

func StartTray(tip string, onOpen, onQuit func()) {}
func TrayNotify(title, text string)               {}
func SetTrayLabels(open, quit string)             {}
func TrayRemove()                                 {}
func ListenActivate(onShow func())                {}
func ActivateExisting() bool                      { return false }
func FocusWindow() bool                           { return false }

func KillByName(name string) {}
