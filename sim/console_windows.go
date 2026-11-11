//go:build windows

package sim

import (
	"os"
	"syscall"
	"unsafe"
)

const enableVirtualTerminalProcessing = 0x0004

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

// EnableANSI включает обработку ANSI-escape в текущей консоли (Win10+).
// В Go 1.22 syscall.SetConsoleMode убран — вызываем kernel32.dll напрямую.
func EnableANSI() {
	handle := syscall.Handle(os.Stdout.Fd())

	var mode uint32
	ret, _, _ := procGetConsoleMode.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&mode)),
	)
	if ret == 0 {
		return // не консоль (перенаправлено в файл/pipe) — молча выходим
	}

	ret, _, _ = procSetConsoleMode.Call(
		uintptr(handle),
		uintptr(mode|enableVirtualTerminalProcessing),
	)
	_ = ret
}