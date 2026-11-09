package agent

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

func processIdentity(pid int) (string, int) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", 0
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if entry.ProcessID == uint32(pid) {
			return windows.UTF16ToString(entry.ExeFile[:]), int(entry.ParentProcessID)
		}
	}
	return "", 0
}
