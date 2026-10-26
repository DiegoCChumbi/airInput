**Español**: Puedes leer este documento en español [aquí](README.es.md).

# AirInput

Turn any smartphone into a low-latency wireless controller for your PC. AirInput compiles into a **single self-contained Go binary** (with the web frontend and all controller skins embedded via `//go:embed`) and streams touch and analog inputs from your mobile browser directly to virtual gamepads on Linux (`uinput`) and Windows (`ViGEmBus`), requiring no mobile app installation and no external Node.js or Python runtimes.

![AirInput Terminal TUI](docs/screenshots/terminal-tui.png)

---

## Key Features

- **Zero Mobile Installation**: Runs entirely in the phone's web browser over a local network or direct Wi-Fi hotspot.
- **8 Controller Layouts (Classic & Touch-First)**: Switch on the fly between classic console layouts, modern dual-stick pads, and genre-adapted touch-first controllers designed specifically for mobile ergonomics.
- **Dynamic Analog Stick Zones**: Every analog stick pad features a dynamic touch well where the stick origin automatically centers at your thumb's initial touch point (can be toggled on or off at any time from the in-game menu), plus a dedicated **Dual Dynamic Stick (`twin-stick`)** layout with oversized left and right touchpads.
- **Offline Wi-Fi Hotspot Mode**: Create a standalone local Wi-Fi network (`AirInput-Net`) directly from the PC when no router or internet connection is available. *(Tested on Linux via NetworkManager; Windows implementation included as experimental/untested).*
- **Controller Slot Hot-Swapping**: Reassign which phone controls Player 1, Player 2, etc. directly from the terminal TUI without restarting the game or disconnecting players.

---

## Interface & Controller Gallery

### Connection Deck & Hardware Calibration Menu

Before joining, players choose an identifier and initial shell. During gameplay, the top HUD bar displays live latency (`ms`) and opens the calibration modal to switch layouts, invert screen rotation or Y-axis, test haptics, and adjust stick deadzones.

| Connection Screen | In-Game Calibration Menu |
| :---: | :---: |
| ![Join Screen](docs/screenshots/join-screen.png) | ![Settings Menu](docs/screenshots/settings-menu.png) |

---

### Classic & Standard Layouts

| Layout | Preview |
| :--- | :---: |
| **16-Bit** | ![16-Bit Layout](docs/screenshots/skin-snes.png) |
| **64-Bit** | ![64-Bit Layout](docs/screenshots/skin-n64.png) |
| **Symmetric** | ![Symmetric Layout](docs/screenshots/skin-xbox.png) |
| **Asymmetric** | ![Asymmetric Layout](docs/screenshots/skin-asymmetric.png) |

---

### Touch-First Genre-Adapted Layouts

Designed from the ground up for glass touchscreens so you never have to perform awkward multi-finger claw grips on a phone:

| Layout | Preview |
| :--- | :---: |
| **Brawler (`brawler`)** | ![Brawler Layout](docs/screenshots/skin-brawler.png) |
| **Kart / Racing (`kart`)** | ![Kart Layout](docs/screenshots/skin-kart.png) |
| **Arcade 6-Button (`arcade`)** | ![Arcade Layout](docs/screenshots/skin-arcade.png) |
| **Dual Dynamic Stick (`twin-stick`)** | ![Dual Dynamic Stick Layout](docs/screenshots/skin-twin-stick.png) |

---

## Terminal Controls (TUI)

When `airInput` is running, you can manage the server and connected controllers directly from the terminal:

| Key | Action |
| :--- | :--- |
| `Up` / `Down` (or `k` / `j`) | Navigate the list of connected players. |
| `Enter` / `Space` | Select a player, then press `Enter` on a second player to **hot-swap** their virtual controller slots. |
| `Esc` | Cancel an active player swap or hotspot confirmation prompt. |
| `h` | Toggle **Offline Wi-Fi Hotspot** (`AirInput-Net`). Regenerates the QR code with the hotspot gateway IP (`10.42.0.1`) and opens port `3000` in the firewall. |
| `l` | Show or hide the **Live Input Logs** panel in the terminal. |
| `q` / `Ctrl+C` | Gracefully stop the hotspot (if active), destroy virtual gamepads, and exit. |

You can also pass command-line flags when starting AirInput:

```bash
sudo ./airInput -l            # Start with live input logs enabled
sudo ./airInput -dev          # Serve ./public from disk instead of embedded assets (for UI development)
sudo ./airInput -port 3000    # Custom HTTP/WebSocket port (default: 3000)
```

---

## Quick Start

### Prerequisites

- **Go** (1.20 or higher) to build from source.
- **Linux**: Kernel `uinput` module (included in all standard distributions; load with `sudo modprobe uinput` if needed).
- **Windows *(Experimental / Untested)***: [ViGEmBus Driver](https://github.com/nefarius/ViGEmBus/releases) installed and `ViGEmClient.dll` placed alongside `airInput.exe` (or in your system `PATH`).

### 1. Clone the repository & build the single binary

**Linux:**
```bash
git clone https://github.com/DiegoCChumbi/airInput.git
cd airInput
go build -o airInput .
```

**Windows:**
```powershell
git clone https://github.com/DiegoCChumbi/airInput.git
cd airInput
go build -o airInput.exe .
```

### 2. Run AirInput

**Linux:**
```bash
sudo ./airInput
```
*(Root privileges are required on Linux to access `/dev/uinput` and manage NetworkManager hotspots).*

**Windows *(Experimental / Untested)*:**
```powershell
airInput.exe
```

---

## How It Works

```text
┌─────────────────┐       WebSocket (/ws :3000)       ┌────────────────────────────────────┐
│  Mobile Browser │ ◄───────────────────────────────► │      AirInput Single Binary (Go)   │
│  (Touch Client) │     Buttons, Axes & Ping RTT      │                                    │
└─────────────────┘                                   │  ┌──────────────┐  ┌────────────┐  │
                                                      │  │  WebServer   │  │ Bubble Tea │  │
                                                      │  │ (go:embed +  │◄─┤  TUI Host  │  │
                                                      │  │  WebSocket)  │  │ & Hotspot  │  │
                                                      │  └──────┬───────┘  └─────┬──────┘  │
                                                      │         │   In-Memory    │         │
                                                      │         ▼    Direct      ▼         │
                                                      │  ┌──────────────────────────────┐  │
                                                      │  │        GamepadManager        │  │
                                                      │  │  (Linux uinput / Win ViGEm)  │  │
                                                      │  └──────────────┬───────────────┘  │
                                                      └─────────────────┼──────────────────┘
                                                                        │
                                                               ┌────────▼─────────┐
                                                               │ OS Virtual Pads  │
                                                               │ (Controller 1..N)│
                                                               └──────────────────┘
```

1. **TUI & Hotspot Host (`airInput.go`, `hotspot.go`)**: Renders the interactive terminal UI (Bubble Tea), generates the terminal QR code in pure Go, manages offline Wi-Fi hotspot creation and firewall rules, and triggers instant controller slot swaps.
2. **Embedded HTTP & WebSocket Server (`web_server.go`)**: Serves the mobile web client directly from memory (`//go:embed public`, or from disk with `-dev`), handles player registration, measures RTT ping, and routes button/axis frames directly in-memory with zero IPC overhead.
3. **Native Virtual Gamepad Driver (`gamepad.go`, `gamepad_linux.go`, `gamepad_windows.go`)**: Dynamically spawns an independent virtual gamepad for each registered player using native Linux `/dev/uinput` `ioctl` syscalls or Windows `ViGEmClient.dll` calls.

---

## Project Structure

```text
airInput/
├── airInput.go              # Go TUI (Bubble Tea) & application entrypoint
├── web_server.go            # Embedded HTTP server (//go:embed public), WebSocket handler & QR generator
├── gamepad.go               # Cross-platform GamepadManager & VirtualDevice interface
├── gamepad_linux.go         # Native Linux /dev/uinput driver via syscall/ioctl
├── gamepad_windows.go       # Windows ViGEmBus driver via ViGEmClient.dll (experimental)
├── hotspot.go               # Offline Wi-Fi hotspot & firewall management (Linux/Windows)
├── docs/
│   └── screenshots/         # UI & controller layout screenshots
└── public/                  # Mobile web client (embedded into the Go binary at build time)
    ├── index.html           # Connection deck, HUD bar & calibration modal
    ├── global.css           # Hardware design system & responsive orientation styles
    ├── script.js            # Multi-touch scanner, rotation-aware joysticks, WebSocket & haptics
    └── skins/               # Modular controller layouts
        ├── snes/            # 16-Bit classic pad
        ├── n64/             # 64-Bit 3-prong analog pad
        ├── xbox/            # Symmetric dual-stick pad
        ├── asymmetric/      # Asymmetric offset dual-stick pad
        ├── brawler/         # Touch-first platform fighter pad
        ├── kart/            # Touch-first racing wheel + pedalboard
        ├── arcade/          # Touch-first 6-button fightpad
        └── twin-stick/      # Dual dynamic floating-origin stick pad
```

---

## Troubleshooting

### Connection & Offline Hotspot Issues
- **Phone won't load the page when connected to `AirInput-Net` hotspot**: Since the offline hotspot has no internet access, some phones automatically route traffic through mobile data (`4G`/`5G`). Turn off **Mobile Data** on your phone or accept the system prompt to **"Keep Wi-Fi connection"** when joining `AirInput-Net`.
- **Firewall blocking port 3000**: AirInput automatically attempts to open TCP port `3000` via `ufw`, `firewall-cmd`, or `iptables`. If you use a custom firewall, allow incoming TCP connections on port `3000`.

### Haptic Feedback Not Vibrating
- **Android**: Make sure **Battery Saver**, **Do Not Disturb**, and **Silent Mode** are turned off, and verify that **Touch Vibration / Notification Vibration** is enabled in Android system settings (Android blocks `navigator.vibrate` across all browsers when power saving or silent modes are active).
- **Use the Test Button**: Open the in-game **Menú** and tap **Probar** next to *Respuesta háptica* to test the device actuator.

---

## Roadmap

- [x] Allow phone user to input a name to identify each controller
- [x] Allow linking and hot-swapping phones with controller numbers from the terminal
- [x] Create a standalone Wi-Fi hotspot network to work without existing Wi-Fi or internet
- [x] Add touch-first genre-adapted controller layouts (`Brawler`, `Kart`, `Arcade 6-Button`, `Asymmetric`, `Dual Dynamic Stick`)
- [x] Support dynamic touch-origin positioning across all analog stick zones

---

## License

This project is licensed under the MIT License.
