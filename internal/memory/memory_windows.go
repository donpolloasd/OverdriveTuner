//go:build windows

package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	k32         = syscall.NewLazyDLL("kernel32.dll")
	adv         = syscall.NewLazyDLL("advapi32.dll")
	nt          = syscall.NewLazyDLL("ntdll.dll")
	u32         = syscall.NewLazyDLL("user32.dll")
	pMemStatus  = k32.NewProc("GlobalMemoryStatusEx")
	pSnapshot   = k32.NewProc("CreateToolhelp32Snapshot")
	pFirst      = k32.NewProc("Process32FirstW")
	pNext       = k32.NewProc("Process32NextW")
	pOpenProc   = k32.NewProc("OpenProcess")
	pClose      = k32.NewProc("CloseHandle")
	pEmptyWS    = k32.NewProc("K32EmptyWorkingSet")
	pCurProc    = k32.NewProc("GetCurrentProcess")
	pOpenToken  = adv.NewProc("OpenProcessToken")
	pLookupPriv = adv.NewProc("LookupPrivilegeValueW")
	pAdjustPriv = adv.NewProc("AdjustTokenPrivileges")
	pSetSysInfo = nt.NewProc("NtSetSystemInformation")
	pForeground = u32.NewProc("GetForegroundWindow")
	pWindowPID  = u32.NewProc("GetWindowThreadProcessId")
)

type memStatus struct {
	Length                                           uint32
	Load                                             uint32
	TotalPhys, AvailPhys, TotalPage, AvailPage       uint64
	TotalVirtual, AvailVirtual, AvailExtendedVirtual uint64
}

type procEntry struct {
	Size     uint32
	Usage    uint32
	PID      uint32
	HeapID   uintptr
	ModuleID uint32
	Threads  uint32
	Parent   uint32
	PriClass int32
	Flags    uint32
	Exe      [260]uint16
}

func status() (total, avail int) {
	var m memStatus
	m.Length = uint32(unsafe.Sizeof(m))
	if r, _, _ := pMemStatus.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return 0, 0
	}
	return int(m.TotalPhys >> 20), int(m.AvailPhys >> 20)
}

// enablePrivilege activa un privilegio del proceso (hace falta ser administrador).
func enablePrivilege(name string) bool {
	var tok syscall.Handle
	cur, _, _ := pCurProc.Call()
	if r, _, _ := pOpenToken.Call(cur, 0x28, uintptr(unsafe.Pointer(&tok))); r == 0 { // ADJUST_PRIVILEGES | QUERY
		return false
	}
	defer syscall.CloseHandle(tok)
	var luid struct {
		Low  uint32
		High int32
	}
	n, _ := syscall.UTF16PtrFromString(name)
	if r, _, _ := pLookupPriv.Call(0, uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(&luid))); r == 0 {
		return false
	}
	tp := struct {
		Count uint32
		Luid  struct {
			Low  uint32
			High int32
		}
		Attr uint32
	}{Count: 1, Attr: 2} // SE_PRIVILEGE_ENABLED
	tp.Luid = luid
	r, _, e := pAdjustPriv.Call(uintptr(tok), 0, uintptr(unsafe.Pointer(&tp)), 0, 0, 0)
	return r != 0 && e != syscall.Errno(1300) // 1300 = el privilegio no está asignado a este usuario
}

// Free libera memoria y devuelve cuánta había libre antes y después.
func Free(o Options) (res Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("falló la liberación de memoria: %v", r)
		}
	}()
	res.TotalMB, res.BeforeMB = status()
	if res.TotalMB == 0 {
		return res, fmt.Errorf("no se pudo leer el estado de la memoria")
	}
	var fg uint32
	if hwnd, _, _ := pForeground.Call(); hwnd != 0 {
		pWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&fg)))
	}
	snap, _, _ := pSnapshot.Call(2, 0)
	if snap != 0 && snap != ^uintptr(0) {
		defer pClose.Call(snap)
		var pe procEntry
		pe.Size = uint32(unsafe.Sizeof(pe))
		ok, _, _ := pFirst.Call(snap, uintptr(unsafe.Pointer(&pe)))
		for ok != 0 {
			name := strings.ToLower(syscall.UTF16ToString(pe.Exe[:]))
			skip := pe.PID == 0 || pe.PID == 4 || int(pe.PID) == os.Getpid() || pe.PID == fg || neverTouch[name]
			if o.Game != "" && strings.EqualFold(filepath.Base(name), o.Game) {
				skip = true
				res.SkippedGame = o.Game
			}
			if !skip {
				// PROCESS_SET_QUOTA (0x100) | PROCESS_QUERY_LIMITED_INFORMATION (0x1000)
				if h, _, _ := pOpenProc.Call(0x1100, 0, uintptr(pe.PID)); h != 0 {
					if r, _, _ := pEmptyWS.Call(h); r != 0 {
						res.Trimmed++
					}
					pClose.Call(h)
				}
			}
			ok, _, _ = pNext.Call(snap, uintptr(unsafe.Pointer(&pe)))
		}
	}
	if o.Game == "" && enablePrivilege("SeProfileSingleProcessPrivilege") {
		cmd := int32(4) // MemoryPurgeStandbyList
		if st, _, _ := pSetSysInfo.Call(80, uintptr(unsafe.Pointer(&cmd)), 4); st == 0 {
			res.Standby = true
		}
	}
	time.Sleep(1500 * time.Millisecond)
	_, res.AfterMB = status()
	return res, nil
}
