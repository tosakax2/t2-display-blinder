//go:build windows

package blinder

import (
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modGdi32    = syscall.NewLazyDLL("gdi32.dll")
	modShcore   = syscall.NewLazyDLL("shcore.dll")

	procRegisterClassExW              = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW               = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow                 = modUser32.NewProc("DestroyWindow")
	procShowWindow                    = modUser32.NewProc("ShowWindow")
	procUpdateWindow                  = modUser32.NewProc("UpdateWindow")
	procSetForegroundWindow           = modUser32.NewProc("SetForegroundWindow")
	procSetFocus                      = modUser32.NewProc("SetFocus")
	procShowCursor                    = modUser32.NewProc("ShowCursor")
	procGetMessageW                   = modUser32.NewProc("GetMessageW")
	procTranslateMessage              = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW              = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage               = modUser32.NewProc("PostQuitMessage")
	procPostThreadMessageW            = modUser32.NewProc("PostThreadMessageW")
	procDefWindowProcW                = modUser32.NewProc("DefWindowProcW")
	procGetCursorPos                  = modUser32.NewProc("GetCursorPos")
	procPostMessageW                  = modUser32.NewProc("PostMessageW")
	procEnumDisplayMonitors           = modUser32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW               = modUser32.NewProc("GetMonitorInfoW")
	procSetProcessDpiAwarenessContext = modUser32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = modUser32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwareness        = modShcore.NewProc("SetProcessDpiAwareness")
	procSetWindowsHookExW             = modUser32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx           = modUser32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx                = modUser32.NewProc("CallNextHookEx")
	procGetAsyncKeyState              = modUser32.NewProc("GetAsyncKeyState")

	procGetCurrentThreadId = modKernel32.NewProc("GetCurrentThreadId")
	procGetStockObject     = modGdi32.NewProc("GetStockObject")
)

const (
	blackBrush = 4

	wsPopup   = 0x80000000
	wsVisible = 0x10000000

	wsExTopMost    = 0x00000008
	wsExToolWindow = 0x00000080

	swShow = 5

	wmDestroy     = 0x0002
	wmClose       = 0x0010
	wmQuit        = 0x0012
	wmKeyDown     = 0x0100
	wmSysKeyDown  = 0x0104
	wmLButtonDown = 0x0201
	wmRButtonDown = 0x0204
	wmMButtonDown = 0x0207
	wmMouseMove   = 0x0200

	whKeyboardLL = 13
	whMouseLL    = 14

	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
	dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)
)

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

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

// Blinder manages the fullscreen blackout windows across all displays.
type Blinder struct {
	mu           sync.Mutex
	hwnds        []syscall.Handle
	activeFlag   int32
	threadID     uintptr
	initialPos   point
	startTime    time.Time
	onDismiss    func()
	hHookKey     uintptr
	hHookMouse   uintptr
	stopPollChan chan struct{}
}

var (
	globalBlinder     *Blinder
	globalBlinderLock sync.Mutex
	classRegistered   bool
)

const blinderClassName = "T2DisplayBlinderWindowClass"

// EnableDpiAwareness enables per-monitor DPI awareness to ensure precise multi-monitor coordinates.
func EnableDpiAwareness() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2)
		return
	}
	if procSetProcessDpiAwareness.Find() == nil {
		procSetProcessDpiAwareness.Call(2) // PROCESS_PER_MONITOR_DPI_AWARE = 2
		return
	}
	if procSetProcessDPIAware.Find() == nil {
		procSetProcessDPIAware.Call()
	}
}

// low-level keyboard hook callback
func lowLevelKeyboardProc(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 {
		globalBlinderLock.Lock()
		b := globalBlinder
		globalBlinderLock.Unlock()

		if b != nil && b.IsActive() {
			go b.Dismiss()
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// low-level mouse hook callback
func lowLevelMouseProc(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 {
		globalBlinderLock.Lock()
		b := globalBlinder
		globalBlinderLock.Unlock()

		if b != nil && b.IsActive() {
			switch wParam {
			case wmLButtonDown, wmRButtonDown, wmMButtonDown, 0x020A: // WM_MOUSEWHEEL
				go b.Dismiss()
			case wmMouseMove:
				if time.Since(b.startTime) > 300*time.Millisecond {
					var pt point
					procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
					dx := float64(pt.x - b.initialPos.x)
					dy := float64(pt.y - b.initialPos.y)
					if math.Hypot(dx, dy) > 15 {
						go b.Dismiss()
					}
				}
			}
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	globalBlinderLock.Lock()
	b := globalBlinder
	globalBlinderLock.Unlock()

	switch msg {
	case wmKeyDown, wmSysKeyDown, wmLButtonDown, wmRButtonDown, wmMButtonDown:
		if b != nil {
			go b.Dismiss()
		}
		return 0

	case wmMouseMove:
		if b != nil && time.Since(b.startTime) > 300*time.Millisecond {
			var pt point
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			dx := float64(pt.x - b.initialPos.x)
			dy := float64(pt.y - b.initialPos.y)
			if math.Hypot(dx, dy) > 15 {
				go b.Dismiss()
			}
		}
		return 0

	case wmClose:
		procDestroyWindow.Call(uintptr(hwnd))
		return 0

	case wmDestroy:
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func registerClass() error {
	if classRegistered {
		return nil
	}

	classNameUTF16, err := syscall.UTF16PtrFromString(blinderClassName)
	if err != nil {
		return err
	}

	hBrush, _, _ := procGetStockObject.Call(uintptr(blackBrush))

	var wc wndClassExW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = syscall.NewCallback(wndProc)
	wc.hbrBackground = syscall.Handle(hBrush)
	wc.lpszClassName = classNameUTF16

	ret, _, errCall := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 && errCall != nil && errCall.(syscall.Errno) != 0 {
		return errCall
	}

	classRegistered = true
	return nil
}

// GetAllMonitors returns the bounding rectangles of all connected display monitors.
func GetAllMonitors() []rect {
	var monitors []rect

	enumCallback := syscall.NewCallback(func(hMonitor syscall.Handle, hdcMonitor syscall.Handle, lprcMonitor *rect, dwData uintptr) uintptr {
		var mi monitorInfo
		mi.cbSize = uint32(unsafe.Sizeof(mi))
		ret, _, _ := procGetMonitorInfoW.Call(uintptr(hMonitor), uintptr(unsafe.Pointer(&mi)))
		if ret != 0 {
			monitors = append(monitors, mi.rcMonitor)
		} else if lprcMonitor != nil {
			monitors = append(monitors, *lprcMonitor)
		}
		return 1
	})

	procEnumDisplayMonitors.Call(0, 0, enumCallback, 0)
	return monitors
}

// New creates a new Blinder instance.
func New(onDismiss func()) *Blinder {
	return &Blinder{
		onDismiss: onDismiss,
	}
}

// Show displays full-screen blackout windows covering all connected monitors individually.
func (b *Blinder) Show() error {
	if !atomic.CompareAndSwapInt32(&b.activeFlag, 0, 1) {
		return nil // Already active
	}

	b.mu.Lock()
	b.hwnds = nil
	b.stopPollChan = make(chan struct{})
	b.mu.Unlock()

	globalBlinderLock.Lock()
	globalBlinder = b
	globalBlinderLock.Unlock()

	done := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		tid, _, _ := procGetCurrentThreadId.Call()
		b.mu.Lock()
		b.threadID = tid
		b.mu.Unlock()

		// Enable Per-Monitor DPI Awareness
		EnableDpiAwareness()

		if err := registerClass(); err != nil {
			atomic.StoreInt32(&b.activeFlag, 0)
			done <- err
			return
		}

		monitors := GetAllMonitors()
		if len(monitors) == 0 {
			monitors = append(monitors, rect{left: 0, top: 0, right: 1920, bottom: 1080})
		}

		classNameUTF16, _ := syscall.UTF16PtrFromString(blinderClassName)
		windowTitleUTF16, _ := syscall.UTF16PtrFromString("T2 Display Blinder Blackout")

		var createdHwnds []syscall.Handle

		for _, m := range monitors {
			w := m.right - m.left
			h := m.bottom - m.top

			hwnd, _, _ := procCreateWindowExW.Call(
				uintptr(wsExTopMost|wsExToolWindow),
				uintptr(unsafe.Pointer(classNameUTF16)),
				uintptr(unsafe.Pointer(windowTitleUTF16)),
				uintptr(wsPopup|wsVisible),
				uintptr(m.left), uintptr(m.top), uintptr(w), uintptr(h),
				0, 0, 0, 0,
			)

			if hwnd != 0 {
				createdHwnds = append(createdHwnds, syscall.Handle(hwnd))
				procShowWindow.Call(hwnd, uintptr(swShow))
				procUpdateWindow.Call(hwnd)
			}
		}

		if len(createdHwnds) == 0 {
			atomic.StoreInt32(&b.activeFlag, 0)
			done <- syscall.GetLastError()
			return
		}

		b.mu.Lock()
		b.hwnds = createdHwnds
		b.startTime = time.Now()
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&b.initialPos)))
		b.mu.Unlock()

		// Install low-level input hooks for guaranteed wake-up
		hKey, _, _ := procSetWindowsHookExW.Call(
			uintptr(whKeyboardLL),
			syscall.NewCallback(lowLevelKeyboardProc),
			0, 0,
		)
		hMouse, _, _ := procSetWindowsHookExW.Call(
			uintptr(whMouseLL),
			syscall.NewCallback(lowLevelMouseProc),
			0, 0,
		)

		b.mu.Lock()
		b.hHookKey = hKey
		b.hHookMouse = hMouse
		b.mu.Unlock()

		// Focus the first window and hide cursor
		procSetForegroundWindow.Call(uintptr(createdHwnds[0]))
		procSetFocus.Call(uintptr(createdHwnds[0]))
		procShowCursor.Call(0)

		// Start background polling safety net (checks for key/mouse activity every 30ms)
		go b.pollSafetyNet()

		done <- nil

		// Windows Message Loop
		var msgObj msg
		for {
			ret, _, _ := procGetMessageW.Call(
				uintptr(unsafe.Pointer(&msgObj)),
				0, 0, 0,
			)
			if int32(ret) <= 0 || msgObj.message == wmQuit {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msgObj)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msgObj)))

			if !b.IsActive() {
				break
			}
		}

		// Cleanup hooks
		b.mu.Lock()
		if b.hHookKey != 0 {
			procUnhookWindowsHookEx.Call(b.hHookKey)
			b.hHookKey = 0
		}
		if b.hHookMouse != 0 {
			procUnhookWindowsHookEx.Call(b.hHookMouse)
			b.hHookMouse = 0
		}
		hwndsToDestroy := b.hwnds
		b.hwnds = nil
		b.mu.Unlock()

		// Destroy all blackout windows
		for _, h := range hwndsToDestroy {
			if h != 0 {
				procDestroyWindow.Call(uintptr(h))
			}
		}

		// Ensure cursor visibility is fully restored
		for {
			cRet, _, _ := procShowCursor.Call(1)
			if int32(cRet) >= 0 {
				break
			}
		}

		atomic.StoreInt32(&b.activeFlag, 0)

		if b.onDismiss != nil {
			b.onDismiss()
		}
	}()

	return <-done
}

// pollSafetyNet provides a secondary redundant check for keyboard/mouse wake events.
func (b *Blinder) pollSafetyNet() {
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	// Keys to check: Esc(0x1B), Space(0x20), Enter(0x0D), Any Key, Left Click(0x01), Right Click(0x02)
	checkKeys := []uintptr{0x1B, 0x20, 0x0D, 0x01, 0x02, 0x04}

	for {
		select {
		case <-b.stopPollChan:
			return
		case <-ticker.C:
			if !b.IsActive() {
				return
			}

			// Check key states
			for _, vk := range checkKeys {
				ret, _, _ := procGetAsyncKeyState.Call(vk)
				if int16(ret) < 0 { // Key is pressed
					b.Dismiss()
					return
				}
			}

			// Check mouse movement after initial grace period
			if time.Since(b.startTime) > 300*time.Millisecond {
				var pt point
				procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
				dx := float64(pt.x - b.initialPos.x)
				dy := float64(pt.y - b.initialPos.y)
				if math.Hypot(dx, dy) > 20 {
					b.Dismiss()
					return
				}
			}
		}
	}
}

// Dismiss closes all blackout windows across all monitors and restores the screen.
func (b *Blinder) Dismiss() {
	if !atomic.CompareAndSwapInt32(&b.activeFlag, 1, 0) {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Stop poll safety net
	select {
	case <-b.stopPollChan:
	default:
		close(b.stopPollChan)
	}

	// Unhook immediately
	if b.hHookKey != 0 {
		procUnhookWindowsHookEx.Call(b.hHookKey)
		b.hHookKey = 0
	}
	if b.hHookMouse != 0 {
		procUnhookWindowsHookEx.Call(b.hHookMouse)
		b.hHookMouse = 0
	}

	// Signal the message loop thread to terminate
	if b.threadID != 0 {
		procPostThreadMessageW.Call(b.threadID, uintptr(wmQuit), 0, 0)
	}

	for _, hwnd := range b.hwnds {
		if hwnd != 0 {
			procPostMessageW.Call(uintptr(hwnd), uintptr(wmClose), 0, 0)
		}
	}
}

// IsActive returns whether the blinder is currently covering the screens.
func (b *Blinder) IsActive() bool {
	return atomic.LoadInt32(&b.activeFlag) == 1
}
