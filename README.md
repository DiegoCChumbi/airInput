**Español**: Puedes leer este documento en español [aquí](README.es.md).

# AirInput

Turn any smartphone into a low-latency wireless controller for your PC. AirInput streams touch and analog inputs from your mobile browser directly to virtual gamepads on Linux (`uinput`) and Windows (`vgamepad` / `ViGEmBus`), requiring no mobile app installation.

![AirInput Terminal TUI](docs/screenshots/terminal-tui.png)

---

## Key Features

- **Zero Mobile Installation**: Runs entirely in the phone's web browser over a local network or direct Wi-Fi hotspot.
- **7 Controller Layouts (Classic & Touch-First)**: Switch on the fly between classic console layouts, modern dual-stick pads, and genre-adapted touch-first controllers designed specifically for mobile ergonomics.
- **Offline Wi-Fi Hotspot Mode**: Create a standalone local Wi-Fi network (`AirInput-Net`) directly from the PC when no router or internet connection is available. *(Tested on Linux via NetworkManager; Windows implementation included as experimental/untested).*
- **Controller Slot Hot-Swapping**: Reassign which phone controls Player 1, Player 2, etc. directly from the terminal TUI without restarting the game or disconnecting players.
- **Rotation-Aware Analog Sticks**: Custom virtual joystick engine that accurately projects 2D vectors across landscape, portrait (`+90°`), and inverted portrait (`-90°`) orientations, with configurable radial deadzones and Y-axis inversion.
- **Multi-Layer Haptic Feedback**: Combines the Web Vibration API, iOS Safari Taptic Engine switch triggers, and a low-frequency Web Audio tactile sub-bass pulse.
- **Live Input Telemetry & Logs**: Real-time round-trip latency indicator (`ms`) on the controller HUD and an interactive live log viewer in the terminal.
- **Zero-Dependency Linux `uinput` Fallback**: Includes a native `ioctl` `/dev/uinput` implementation in Python on Linux if `python-uinput` is not installed.

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

You can also start AirInput with the live log panel enabled by default using the `-l` or `--logs` flag:

```bash
sudo ./airInput -l
```

---

## Quick Start

### Prerequisites

- **Node.js** (v14 or higher)
- **Python** (3.7 or higher)
- **Go** (1.19 or higher)

### 1. Clone the repository & install Node.js dependencies

```bash
git clone https://github.com/DiegoCChumbi/airInput.git
cd airInput
npm install
```

### 2. Install Python dependencies

**Linux:**
AirInput includes a built-in native `/dev/uinput` driver in `controller-linux.py`, so it works out of the box with standard Python 3. Optionally, you can install `python-uinput`:

- **Debian, Ubuntu, Linux Mint, Pop!_OS (`apt`):**
  ```bash
  sudo apt update
  sudo apt install python3-dev libudev-dev
  ```
- **Fedora, RHEL, CentOS (`dnf`):**
  ```bash
  sudo dnf install python3-devel libudev-devel
  ```
- **Arch Linux, Manjaro (`pacman`):**
  ```bash
  sudo pacman -S base-devel
  ```

**Windows:**
```powershell
pip install vgamepad
```

### 3. Build the Go executable

**Linux:**
```bash
go build -o airInput .
```

**Windows:**
```powershell
go build -o airInput.exe .
```

### 4. Run AirInput

**Linux:**
```bash
sudo ./airInput
```
*(Root privileges are required on Linux to access `/dev/uinput` and manage NetworkManager hotspots).*

**Windows:**
```powershell
airInput.exe
```

---

## How It Works

```text
┌─────────────────┐        WebSocket (TCP 3000)       ┌──────────────────┐
│  Mobile Browser │ ◄───────────────────────────────► │  Node.js Server  │
│  (Touch Client) │     Buttons, Axes & Ping RTT      │   (server.js)    │
└─────────────────┘                                   └────────┬─────────┘
                                                               │
                                                JSON (stdio)   │   UDP (127.0.0.1:9999)
                                              ◄────────────────┼────────────────►
                                              │                                 │
                                     ┌────────▼─────────┐              ┌────────▼─────────┐
                                     │   Go TUI Host    │  UDP System  │  Python Driver   │
                                     │  (airInput.go)   │ ───────────► │ (uinput/vgamepad)│
                                     └──────────────────┘  Hot-Swap    └────────┬─────────┘
                                                                                │
                                                                       ┌────────▼─────────┐
                                                                       │ OS Virtual Pads  │
                                                                       │ (Controller 1..N)│
                                                                       └──────────────────┘
```

1. **Go Process & TUI Manager (`airInput.go`, `hotspot.go`)**: Launches and supervises the Node.js and Python subprocesses, renders the terminal interface (Bubble Tea), manages offline Wi-Fi hotspot creation and firewall rules, and dispatches hot-swap commands.
2. **WebSocket Server (`server.js`)**: Serves the mobile web interface on port `3000` (with cache-control headers disabled for instant updates), handles player registration, measures RTT ping, and forwards button/axis packets over UDP port `9999`.
3. **Virtual Gamepad Driver (`controller-linux.py`, `controller-win.py`)**: Listens on UDP port `9999`, dynamically spawns an independent virtual gamepad for each registered player, and translates incoming button/axis events into kernel `uinput` (Linux) or `XInput` (`vgamepad` on Windows) events.

---

## Project Structure

```text
airInput/
├── airInput.go              # Go TUI & subprocess orchestrator
├── hotspot.go               # Offline Wi-Fi hotspot & firewall management (Linux/Windows)
├── server.js                # Node.js Express + Socket.IO server & UDP bridge
├── controller-linux.py      # Linux virtual gamepad driver (uinput + native ioctl fallback)
├── controller-win.py        # Windows virtual gamepad driver (vgamepad / ViGEmBus)
├── docs/
│   └── screenshots/         # UI & controller layout screenshots
└── public/                  # Mobile web client
    ├── index.html           # Connection deck, HUD bar & calibration modal
    ├── global.css           # Hardware design system & responsive orientation styles
    ├── script.js            # Multi-touch scanner, rotation-aware joysticks & haptics
    └── skins/               # Modular controller layouts
        ├── snes/            # 16-Bit classic pad
        ├── n64/             # 64-Bit 3-prong analog pad
        ├── xbox/            # Symmetric dual-stick pad
        ├── asymmetric/      # Asymmetric offset dual-stick pad
        ├── brawler/         # Touch-first platform fighter pad
        ├── kart/            # Touch-first racing wheel + pedalboard
        └── arcade/          # Touch-first 6-button fightpad
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
- [x] Add touch-first genre-adapted controller layouts (`Brawler`, `Kart`, `Arcade 6-Button`, `Asymmetric`)

---

## License

This project is licensed under the MIT License.
