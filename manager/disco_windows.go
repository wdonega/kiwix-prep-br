//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var procEspaco = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// espacoEmDisco devolve (livre para o usuário, total) em bytes.
func espacoEmDisco(dir string) (int64, int64, error) {
	p, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return 0, 0, err
	}
	var livre, total, livreTotal uint64
	r, _, e := procEspaco.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&livre)),
		uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&livreTotal)))
	if r == 0 {
		return 0, 0, e
	}
	return int64(livre), int64(total), nil
}
