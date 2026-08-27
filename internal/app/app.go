//go:build windows

package app

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"

	"t2-display-blinder/internal/blinder"
	"t2-display-blinder/internal/config"
	"t2-display-blinder/internal/hotkey"
	"t2-display-blinder/internal/power"
	"t2-display-blinder/web"
)

var (
	modUser32                 = syscall.NewLazyDLL("user32.dll")
	modDwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	procMessageBoxW           = modUser32.NewProc("MessageBoxW")
	procDwmSetWindowAttribute = modDwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	mbIconError = 0x00000010
	mbOK        = 0x00000000

	dwmwaUseImmersiveDarkMode           = 20
	dwmwaUseImmersiveDarkModeBefore20H1 = 19
)

func enableDarkModeForWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	darkMode := int32(1)
	// Try Windows 11 & Windows 10 (20H1+) attribute (20)
	ret, _, _ := procDwmSetWindowAttribute.Call(
		hwnd,
		uintptr(dwmwaUseImmersiveDarkMode),
		uintptr(unsafe.Pointer(&darkMode)),
		uintptr(unsafe.Sizeof(darkMode)),
	)
	// Fallback for older Windows 10 builds (1809-1909) (19)
	if ret != 0 {
		procDwmSetWindowAttribute.Call(
			hwnd,
			uintptr(dwmwaUseImmersiveDarkModeBefore20H1),
			uintptr(unsafe.Pointer(&darkMode)),
			uintptr(unsafe.Sizeof(darkMode)),
		)
	}
}

func showErrorMessage(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(msgPtr)), uintptr(unsafe.Pointer(titlePtr)), uintptr(mbIconError|mbOK))
}
type Application struct {
	cfg           *config.Config
	w             webview2.WebView
	blinder       *blinder.Blinder
	hotkeyManager *hotkey.Manager
	mu            sync.Mutex
}

// New creates a new Application instance.
func New(cfg *config.Config) *Application {
	if cfg == nil {
		cfg = config.Default()
	}

	app := &Application{
		cfg: cfg,
	}

	// Initialize blinder with dismiss callback
	app.blinder = blinder.New(func() {
		if app.w != nil {
			app.w.Dispatch(func() {
				app.w.Eval("if (window.onBlinderDismissed) window.onBlinderDismissed();")
			})
		}
	})

	// Initialize global hotkey manager
	app.hotkeyManager = hotkey.NewManager(hotkey.Handler{
		OnBlackout: func() {
			_ = app.blinder.Show()
		},
		OnScreenOff: func() {
			_ = power.TurnOffWithDelay(500 * time.Millisecond)
		},
	})

	return app
}

// buildHTML bundles index.html, style.css, and app.js into a single standalone HTML document.
func (a *Application) buildHTML() (string, error) {
	htmlBytes, err := web.ReadAppFile("index.html")
	if err != nil {
		return "", fmt.Errorf("failed to read index.html: %w", err)
	}

	cssBytes, err := web.ReadAppFile("style.css")
	if err != nil {
		return "", fmt.Errorf("failed to read style.css: %w", err)
	}

	jsBytes, err := web.ReadAppFile("app.js")
	if err != nil {
		return "", fmt.Errorf("failed to read app.js: %w", err)
	}

	html := string(htmlBytes)
	css := fmt.Sprintf("<style>\n%s\n</style>", string(cssBytes))
	js := fmt.Sprintf("<script>\n%s\n</script>", string(jsBytes))

	// Replace external stylesheet link with inline CSS
	html = strings.Replace(html, `<link rel="stylesheet" href="style.css">`, css, 1)

	// Replace external script tag with inline JS
	html = strings.Replace(html, `<script src="app.js"></script>`, js, 1)

	return html, nil
}

// Run starts the standalone GUI application window.
func (a *Application) Run() error {
	// Enable Per-Monitor DPI Awareness for accurate multi-monitor geometry
	blinder.EnableDpiAwareness()

	// Start global hotkey listener
	a.hotkeyManager.Start()
	defer a.hotkeyManager.Stop()

	// Create WebView2 instance (debug: false)
	w := webview2.New(false)
	if w == nil {
		err := fmt.Errorf("failed to create WebView2 instance (WebView2 Runtime may not be installed)")
		showErrorMessage("T2 Display Blinder Error", err.Error())
		return err
	}
	defer w.Destroy()

	a.w = w

	w.SetTitle("T2 Display Blinder")
	w.SetSize(a.cfg.WindowWidth, a.cfg.WindowHeight, webview2.HintFixed)

	// Apply dark mode to native Windows window header / title bar
	enableDarkModeForWindow(uintptr(w.Window()))

	// Bind Go functions to JavaScript
	if err := w.Bind("goTurnOffScreen", func(delayMs int) error {
		delay := time.Duration(delayMs) * time.Millisecond
		if delay <= 0 {
			delay = a.cfg.ScreenOffDelay
		}
		go func() {
			_ = power.TurnOffWithDelay(delay)
		}()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to bind goTurnOffScreen: %w", err)
	}

	if err := w.Bind("goShowBlinder", func() error {
		go func() {
			_ = a.blinder.Show()
		}()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to bind goShowBlinder: %w", err)
	}

	if err := w.Bind("goExitApp", func() error {
		w.Dispatch(func() {
			w.Destroy()
			os.Exit(0)
		})
		return nil
	}); err != nil {
		return fmt.Errorf("failed to bind goExitApp: %w", err)
	}

	// Prepare and set HTML content
	content, err := a.buildHTML()
	if err != nil {
		return fmt.Errorf("failed to build HTML bundle: %w", err)
	}

	w.SetHtml(content)

	// Run message loop
	w.Run()

	return nil
}
