//go:build windows

package hotkey

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterHotKey     = modUser32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = modUser32.NewProc("UnregisterHotKey")
	procGetMessageW        = modUser32.NewProc("GetMessageW")
	procPostQuitMessage    = modUser32.NewProc("PostQuitMessage")
	procPostThreadMessageW = modUser32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId = modKernel32.NewProc("GetCurrentThreadId")
)

const (
	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004
	modWin     = 0x0008
	modNoRepeat = 0x4000

	wmHotkey = 0x0312
	wmQuit   = 0x0012

	// Key codes
	vkB = 0x42 // 'B' for Blackout
	vkS = 0x53 // 'S' for Screen Off

	idBlackout  = 1001
	idScreenOff = 1002
)

type point struct {
	x int32
	y int32
}

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// Handler defines the callback functions for registered hotkeys.
type Handler struct {
	OnBlackout  func()
	OnScreenOff func()
}

// Manager manages global hotkey registration and event dispatching.
type Manager struct {
	mu       sync.Mutex
	threadID uintptr
	stopChan chan struct{}
	handler  Handler
	running  bool
}

// NewManager creates a new Hotkey Manager.
func NewManager(handler Handler) *Manager {
	return &Manager{
		handler:  handler,
		stopChan: make(chan struct{}),
	}
}

// Start begins listening for global hotkeys in a dedicated OS thread.
func (m *Manager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	ready := make(chan struct{})

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		tid, _, _ := procGetCurrentThreadId.Call()
		m.mu.Lock()
		m.threadID = tid
		m.mu.Unlock()

		// Register Ctrl+Alt+B for Blackout
		procRegisterHotKey.Call(0, uintptr(idBlackout), uintptr(modControl|modAlt|modNoRepeat), uintptr(vkB))

		// Register Ctrl+Alt+S for Screen Off
		procRegisterHotKey.Call(0, uintptr(idScreenOff), uintptr(modControl|modAlt|modNoRepeat), uintptr(vkS))

		close(ready)

		var mMsg msg
		for {
			ret, _, _ := procGetMessageW.Call(
				uintptr(unsafe.Pointer(&mMsg)),
				0, 0, 0,
			)
			if int32(ret) <= 0 || mMsg.message == wmQuit {
				break
			}

			if mMsg.message == wmHotkey {
				switch mMsg.wParam {
				case uintptr(idBlackout):
					if m.handler.OnBlackout != nil {
						go m.handler.OnBlackout()
					}
				case uintptr(idScreenOff):
					if m.handler.OnScreenOff != nil {
						go m.handler.OnScreenOff()
					}
				}
			}
		}

		// Cleanup hotkeys
		procUnregisterHotKey.Call(0, uintptr(idBlackout))
		procUnregisterHotKey.Call(0, uintptr(idScreenOff))
	}()

	<-ready
}

// Stop unregisters hotkeys and terminates the hotkey listener thread.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.threadID == 0 {
		return
	}

	procPostThreadMessageW.Call(m.threadID, uintptr(wmQuit), 0, 0)
	m.running = false
}
