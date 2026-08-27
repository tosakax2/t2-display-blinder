//go:build windows

package power

import (
	"syscall"
	"time"
)

var (
	modUser32 = syscall.NewLazyDLL("user32.dll")

	procSendMessageW     = modUser32.NewProc("SendMessageW")
	procPostMessageW    = modUser32.NewProc("PostMessageW")
	procGetDesktopWindow = modUser32.NewProc("GetDesktopWindow")
	procDefWindowProcW   = modUser32.NewProc("DefWindowProcW")
)

const (
	hwndBroadcast   = 0xFFFF
	wmSysCommand    = 0x0112
	scMonitorPower  = 0xF170
	monitorPowerOff = 2
	monitorPowerOn  = -1
)

// TurnOff turns off all connected displays immediately by triggering the Windows monitor standby power state.
func TurnOff() error {
	// Send SC_MONITORPOWER with parameter 2 (Power Off)
	// Using SendMessageW with HWND_BROADCAST is the standard Windows API method for monitor standby.
	_, _, err := procSendMessageW.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSysCommand),
		uintptr(scMonitorPower),
		uintptr(monitorPowerOff),
	)
	if err != nil && err.(syscall.Errno) != 0 {
		return err
	}
	return nil
}

// TurnOffWithDelay waits for the specified duration and then turns off the displays.
// This gives the user time to release the mouse or keyboard to prevent accidental immediate wake-up.
func TurnOffWithDelay(delay time.Duration) error {
	if delay > 0 {
		time.Sleep(delay)
	}
	return TurnOff()
}
