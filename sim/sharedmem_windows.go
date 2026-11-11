//go:build windows

package sim

import (
	"errors"
	"syscall"
	"unsafe"
)

const (
	shmemPageReadWrite = 0x04 // PAGE_READWRITE
	shmemFileMapWrite  = 0x02 // FILE_MAP_WRITE
	shmemFileMapRead   = 0x04 // FILE_MAP_READ
)

func openSharedMemWriter() (*SharedMem, error) {
	name, _ := syscall.UTF16PtrFromString(shmemName)
	h, err := syscall.CreateFileMapping(
		syscall.InvalidHandle,
		nil,
		shmemPageReadWrite,
		0, uint32(shmemSize),
		name,
	)
	if err != nil {
		return nil, err
	}
	addr, err := syscall.MapViewOfFile(h, shmemFileMapWrite, 0, 0, uintptr(shmemSize))
	if addr == 0 {
		syscall.CloseHandle(h)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("MapViewOfFile (write) failed")
	}
	return &SharedMem{handle: uintptr(h), addr: addr, size: shmemSize}, nil
}

var (
	kernel32Shm          = syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMappingW = kernel32Shm.NewProc("OpenFileMappingW")
)

func openSharedMemReader() (*SharedMem, error) {
	name, _ := syscall.UTF16PtrFromString(shmemName)

	// OpenFileMappingW(dwDesiredAccess, bInheritHandle, lpName)
	h, _, err := procOpenFileMappingW.Call(
		uintptr(shmemFileMapRead),
		0, // FALSE — не наследовать
		uintptr(unsafe.Pointer(name)),
	)
	if h == 0 {
		return nil, err
	}

	addr, err := syscall.MapViewOfFile(syscall.Handle(h), shmemFileMapRead, 0, 0, uintptr(shmemSize))
	if addr == 0 {
		syscall.CloseHandle(syscall.Handle(h))
		return nil, err
	}
	return &SharedMem{handle: h, addr: addr, size: shmemSize}, nil
}

func closeSharedMem(s *SharedMem) {
	if s.addr != 0 {
		_ = syscall.UnmapViewOfFile(s.addr)
		s.addr = 0
	}
	if s.handle != 0 {
		_ = syscall.CloseHandle(syscall.Handle(s.handle))
		s.handle = 0
	}
}

// Не даём "unused import" на unsafe — используется через uintptr выше.
var _ = unsafe.Pointer(nil)