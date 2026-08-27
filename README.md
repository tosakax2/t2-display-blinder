# T2 Display Blinder

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.20%2B-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6.svg)](https://microsoft.com/windows)

A formal, lightweight, and modern display utility for Windows that provides screen standby power-off and fullscreen blackout covering all connected monitors through a standalone native GUI window and global shortcuts.

[日本語版ドキュメント (README_ja.md)](README_ja.md)

---

## Features

1. **Screen Off (Display Standby)**
   - Triggers native Windows monitor power-off standby via `WM_SYSCOMMAND` + `SC_MONITORPOWER: 2`.
   - Includes a 1-second grace period interval to prevent accidental immediate wake-up from subtle mouse movements upon execution.

2. **Blackout Screen (Multi-Monitor Blanking)**
   - Covers all connected displays across any resolution, orientation, and scaling factor (Per-Monitor DPI Aware V2).
   - Generates individual topmost black overlay windows per physical monitor geometry (`EnumDisplayMonitors`).
   - Hides the mouse cursor for complete darkness.
   - Dual-layer safety wake-up system: Low-Level Windows Hooks (`WH_KEYBOARD_LL`, `WH_MOUSE_LL`) and background safety polling ensure instant dismissal on any key press (Esc, Space, Enter) or mouse movement/click across any monitor.

3. **Timer Control**
   - Preset automated screen-off timers: 1 min, 5 min, 15 min, 30 min, 60 min.
   - Real-time countdown timer with progress indicator and instant cancellation.

4. **Global Hotkeys**
   - Execute actions instantly even when the window is in the background:
     - `Ctrl + Alt + S` : Screen Off
     - `Ctrl + Alt + B` : Blackout Screen

5. **Standalone Native GUI**
   - Runs in an independent, single desktop window without spawning external browsers.
   - Uses native Windows DWM dark title bar styling (`DwmSetWindowAttribute`) matching the formal dark theme palette with no emojis.

---

## Project Structure

```text
t2-display-blinder/
├── cmd/
│   └── t2-display-blinder/
│       └── main.go              # Application entrypoint
├── internal/
│   ├── app/
│   │   └── app.go               # GUI window management & Go/JS bindings
│   ├── blinder/
│   │   └── blinder_windows.go   # Multi-monitor blackout overlay & wake listener
│   ├── power/
│   │   └── power_windows.go     # Display power control (Win32 API)
│   ├── hotkey/
│   │   └── hotkey_windows.go    # Global hotkey manager (Win32 API)
│   └── config/
│       ├── config.go            # Configuration structure & defaults
│       └── config_test.go       # Config unit tests
├── web/
│   ├── assets.go                # Embedded frontend assets (embed.FS)
│   ├── assets_test.go           # Assets test
│   └── app/
│       ├── index.html           # UI structure
│       ├── style.css            # Dark theme stylesheet
│       └── app.js               # Frontend controller & Go bindings
├── .gitignore
├── build.bat                    # One-click build script
├── go.mod
├── go.sum
├── LICENSE                      # MIT License
├── README.md                    # English documentation
└── README_ja.md                 # Japanese documentation
```

---

## Prerequisites

- Windows 10 (1803+) or Windows 11 (64-bit)
- Go 1.20 or newer
- Microsoft Edge WebView2 Runtime (Pre-installed on Windows 10/11)
- No CGO / C compiler required (Pure Go build)

---

## Build

### One-Click Build (Recommended)
```cmd
build.bat
```
*Automatically runs tests and compiles the GUI binary to `bin\t2-display-blinder.exe`.*

### Manual Build
```powershell
go build -ldflags="-H windowsgui -s -w" -o bin/t2-display-blinder.exe ./cmd/t2-display-blinder
```

---

## Usage

```powershell
./bin/t2-display-blinder.exe
```
Or double-click `bin\t2-display-blinder.exe` in Windows Explorer.

---

## License

This project is licensed under the [MIT License](LICENSE).
