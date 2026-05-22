# WiFiMon

`wifimon` is a professional terminal UI for watching your current Wi-Fi connection in real time.

It is built for Windows first, keeps the UI in English, and is structured so Linux support can grow without changing the public project shape.

## Features

- Live TUI with color, emoji, and automatic refresh
- Detects connected, disconnected, and no-device states
- Multiple adapter support with keyboard switching
- Live signal history graph with sparklines
- Gateway latency checks and rolling packet-loss monitor
- Current band, radio type, channel, IP, gateway, auth, and profile info
- Growl notifications for connect, disconnect, and network switch events via [`github.com/cumulus13/go-gntp`](https://github.com/cumulus13/go-gntp)
- Responsive layout that follows terminal width

[![Screenshot](https://raw.githubusercontent.com/cumulus13/go-wifimon/master/screenshot.png)](https://raw.githubusercontent.com/cumulus13/go-wifimon/master/screenshot.png)

## Shortcuts

- `q`: quit
- `r`: refresh immediately
- `Left` / `Right` / `Tab`: switch adapters

## Install

### Go

```bash
go install github.com/cumulus13/go-wifimon/cmd/wifimon@latest
```

### From source

```bash
git clone https://github.com/cumulus13/go-wifimon.git
cd go-wifimon
go build -buildvcs=false -o wifimon.exe ./cmd/wifimon
```

### Windows quick build

```bat
build.bat
```

This creates `wifimon.exe` in the project root.

## Run

```bash
wifimon
```

## Notifications

WiFiMon uses Growl GNTP notifications through `github.com/cumulus13/go-gntp` `v1.0.3`.

This project follows the library README/examples instead of guessing:

- The client is registered once
- The app icon is loaded with `gntp.LoadResource(...)`
- Binary icon mode is used for Windows Growl compatibility
- Notification type registration includes the icon before sending events

If Growl is not running on `localhost:23053`, the TUI still works and notifications fail silently.

## Development

```bash
go test ./...
go run ./cmd/wifimon
```

Useful local build commands:

```bash
go build -buildvcs=false -o wifimon.exe ./cmd/wifimon
make build
make build-windows
```

On Windows release builds with GoReleaser:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

## Releases

The repository includes:

- GitHub Actions release workflow
- GoReleaser configuration
- Homebrew tap generation
- Scoop bucket generation

Default packaging targets assume:

- Homebrew tap: `cumulus13/homebrew-tap`
- Scoop bucket: `cumulus13/scoop-bucket`

Update `.goreleaser.yaml` if you want different repository names.

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)