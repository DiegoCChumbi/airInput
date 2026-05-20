**Español**: Puedes leer este documento en español [aquí](README.es.md).

# AirInput

Turn your smartphone into a wireless controller for your PC. Play your games with customizable gamepad layouts directly from your browser, no app installation required.

## Features

- **🌐 Browser-Based**: No mobile app needed - works entirely in your phone's browser
- **🎨 Multiple Layouts**: Choose between classic or modern style
- **🔌 Virtual Gamepad**: Create a virtual controller recognized by games
- **📱 Multi-Player Support**: Connect multiple devices as separate controllers
- **⚡ Low Latency**: Real-time input streaming via WebSockets and UDP
- **🔧 Cross-Platform**: Supports Windows and Linux
- **📶 Local Network**: All communication stays on your local network

## Quick Start

### Prerequisites

- **Node.js** (v14 or higher)
- **Python** (3.7 or higher)
- **Go** (1.19 or higher)

### Installation

**Clone the repository**

   ```bash
   git clone https://github.com/DiegoCChumbi/airInput.git
   cd airInput
   ```

**Install Node.js dependencies**

   ```bash
   npm install
   ```

**Install Python dependencies**

   **Windows:**

   ```powershell
   pip install vgamepad
   ```

   **Linux:**

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

**Build the Go executable**

**Windows:**

   ```powershell
   go build -o airInput.exe airInput.go
   ```

**Linux:**

   ```bash
   go build -o airInput airInput.go
   ```

### Running airInput

Simply run the compiled executable:

**Windows:**

```powershell
airInput.exe
```

**Linux:**

```bash
sudo ./airInput
```

*On Linux, sudo is required to create virtual gamepads.*

The application will:

1. Start the web server on port 3000
2. Display the connection URL and QR code in the terminal
3. Launch the virtual gamepad controller

### Connect Your Phone

1. Make sure your phone is on the same WiFi network as your PC
2. Open your phone's browser and navigate to the URL shown in the terminal (Or scan the QR that will appear)
3. Tap "TAP TO START" to activate the controller
4. Choose your preferred layout in settings (⚙️)

## How It Works

```
┌─────────────┐         WebSocket           ┌──────────────┐
│   Browser   │ ◄─────────────────────────► │   Node.js    │
│   (Phone)   │  (Button/Axis Data)         │   Server     │
└─────────────┘                             └──────┬───────┘
                                                   │
                                                UDP│ (Port 9999)
                                                   │
                                            ┌──────▼───────┐
                                            │   Python     │
                                            │  Controller  │
                                            └──────┬───────┘
                                                   │
                                            Virtual│Gamepad API
                                                   │
                                            ┌──────▼───────┐
                                            │  Xbox 360    │
                                            │  Controller  │
                                            │   Virtual    │
                                            └──────────────┘
```

1. **Web Interface**: HTML5-based touch control with NippleJS for joysticks
2. **WebSocket Server**: Node.js with Socket.IO for real-time communication
3. **UDP Bridge**: Forwards input data to Python controller
4. **Virtual Gamepad**: Python creates virtual Xbox 360 controllers using platform-specific libraries
5. **Process Manager**: Go orchestrates Node.js and Python processes

## Project Structure

```
airInput/
├── airInput.go              # Go process manager
├── server.js                # Node.js WebSocket server
├── controller-win.py        # Windows virtual gamepad
├── controller-linux.py      # Linux virtual gamepad
├── package.json             # Node.js dependencies
├── go.mod                   # Go module definition
└── public/                  # Web client
    ├── index.html           # Main HTML
    ├── script.js            # Client logic
    ├── global.css           # Base styles
    └── skins/               # Controller layouts
        ├── snes/            # Classic layout
        └── xbox/            # Modern layout
```

## Configuration

The web server runs on port 3000 by default. To change it, edit [server.js](server.js):

```javascript
const PORT = 3000; // Change this value
```

UDP communication uses port 9999. Make sure this port is available on your system.

### Connection Issues

- Verify that your phone and PC are on the same network
- Check that firewall settings allow connections on port 3000
- Disable VPN if active

### Input Lag

- Reduce the distance between phone and WiFi router
- Close other applications using network bandwidth
- Use the 5GHz WiFi band if available

## Contributing

Contributions are welcome! Feel free to:

- Report bugs by opening an issue
- Suggest new features or improvements
- Submit pull requests with enhancements
- Contribute with new custom layouts

## License

This project is licensed under the MIT License.

## Acknowledgements

- [NippleJS](https://github.com/yoannmoinet/nipplejs) - Virtual joystick library
- [Socket.IO](https://socket.io/) - Real-time communication
- [vgamepad](https://github.com/yagnateos/vgamepad) - Virtual gamepad for Windows
- [ViGEmBus](https://github.com/ViGEm/ViGEmBus) - Virtual gamepad driver for Windows

## TODO

- [x] Allow phone user to input a name to identify each controller
- [x] Allow to link phones with controller numbers
- [ ] Create your own network to work without wifi.

---

**Note**: This is a local network solution. Your phone and PC must be connected to the same WiFi network.
