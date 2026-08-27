# t2-display-blinder

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.20%2B-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6.svg)](https://microsoft.com/windows)

Windows環境向けの軽量かつフォーマルなディスプレイ制御GUIアプリケーション（T2 Display Blinder）です。  
接続されているディスプレイの省電力スタンバイ（スクリーンオフ）および全画面ブランク（真っ暗な画面での覆工）を、単一の独立したネイティブGUIウィンドウおよびショートカットから実行できます。

[English Documentation (README.md)](README.md)

---

## 主な機能

1. **Screen Off (スクリーンオフ)**
   - Windows API (`WM_SYSCOMMAND` + `SC_MONITORPOWER: 2`) を呼び出し、ディスプレイを即座に省電力・スタンバイ状態へ移行させます。
   - 誤操作による意図しない即時復帰を防ぐため、実行前の短いインターバル（1秒）を設けています。

2. **Blackout Screen (画面ブランク・マルチモニター対応)**
   - Per-Monitor DPI Awareness に対応し、異解像度マルチモニター環境でもすべての画面を最前面の黒色ウィンドウで完全に覆います。
   - マウスカーソルを非表示化し、完全な暗闇を提供します。
   - 任意のキー（Esc / スペース / Enter 等）またはマウス操作で即座に一括解除されます。

3. **Timer Control (タイマー自動実行)**
   - 1分 / 5分 / 15分 / 30分 / 60分後の自動スクリーンオフ実行に対応。
   - カウントダウンおよびプログレスバー表示、ワンクリックでのキャンセルが可能。

4. **Global Hotkeys (グローバルショートカット)**
   - ウィンドウがバックグラウンドにある状態でも即座に実行可能です。
   - `Ctrl + Alt + S` : Screen Off
   - `Ctrl + Alt + B` : Blackout Screen

5. **Standalone Native GUI (単一固有ウィンドウ)**
   - 外部ブラウザを起動せず、独立した単一のデスクトップウィンドウとして動作します。
   - OSネイティブのダークタイトルバーに対応し、洗練されたタイポグラフィと深みのあるダークテーマパレットを採用。

---

## プロジェクト構成

```text
t2-display-blinder/
├── cmd/
│   └── t2-display-blinder/
│       └── main.go              # アプリケーションのエントリポイント
├── internal/
│   ├── app/
│   │   └── app.go               # GUIウィンドウ生成・バインディング・ライフサイクル管理
│   ├── blinder/
│   │   └── blinder_windows.go   # 全画面黒ウィンドウ制御（マルチモニター対応・復帰検知）
│   ├── power/
│   │   └── power_windows.go     # ディスプレイ電源制御 (Win32 API)
│   ├── hotkey/
│   │   └── hotkey_windows.go    # グローバルホットキー登録・監視 (Win32 API)
│   └── config/
│       ├── config.go            # 設定管理構造体およびデフォルト値
│       └── config_test.go       # 設定ユニットテスト
├── web/
│   ├── assets.go                # embed.FS による app/ 配下のアセット埋め込み
│   ├── assets_test.go           # 埋め込みアセットテスト
│   └── app/
│       ├── index.html           # フォーマル・モダンなUI構造
│       ├── style.css            # 洗練されたダークテーマスタイル
│       └── app.js               # フロントエンドロジック・バインディング呼び出し
├── .gitignore
├── build.bat                    # ワンクリックビルドスクリプト
├── go.mod
├── go.sum
├── LICENSE                      # MITライセンス
├── README.md                    # 英語ドキュメント
└── README_ja.md                 # 日本語ドキュメント
```

---

## ビルド手順

### 必要要件
- Go 1.20 以上 (Windows 64bit)
- WebView2 Runtime (Windows 10 1803以降 / Windows 11 は標準搭載)
- CGO不要（外部Cコンパイラ不要でビルド可能）

### ビルドコマンド

**バッチファイルでビルド（推奨）:**
```cmd
build.bat
```
※ 自動的にテスト実行後、GUIバイナリ（`bin\t2-display-blinder.exe`）が生成されます。

**手動ビルド:**
```powershell
go build -ldflags="-H windowsgui -s -w" -o bin\t2-display-blinder.exe .\cmd\t2-display-blinder
```

---

## 起動方法

```powershell
.\bin\t2-display-blinder.exe
```
または、エクスプローラーから `bin\t2-display-blinder.exe` をダブルクリックして起動します。

---

## ライセンス

本プロジェクトは [MIT License](LICENSE) の下で公開されています。
