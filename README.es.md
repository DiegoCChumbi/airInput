**English**: Read this document in English [here](README.md).

# AirInput

Transforma tu smartphone en un mando inalámbrico de baja latencia para tu PC. AirInput transmite entradas táctiles y analógicas desde el navegador de tu teléfono directamente hacia mandos virtuales en Linux (`uinput`) y Windows (`vgamepad` / `ViGEmBus`), sin necesidad de instalar aplicaciones en el móvil.

![AirInput Terminal TUI](docs/screenshots/terminal-tui.png)

---

## Características Principales

- **Sin Instalación en el Móvil**: Funciona íntegramente desde el navegador web del teléfono a través de tu red local o mediante punto de acceso directo.
- **7 Diseños de Mando (Clásicos y Touch-First)**: Cambia en tiempo real entre mandos clásicos, controles modernos de doble stick (simétrico y asimétrico) y controles *Touch-First* diseñados específicamente para la ergonomía de pantallas táctiles según el género del juego.
- **Modo Hotspot Wi-Fi Sin Internet**: Crea una red Wi-Fi propia (`AirInput-Net`) desde cero directamente desde el PC cuando no hay router ni conexión a internet disponible. *(Probado en Linux con NetworkManager; soporte en Windows incluido como experimental/no testeado).*
- **Intercambio de Mandos en Caliente (Hot-Swap)**: Reasigna qué teléfono controla al Jugador 1, Jugador 2, etc. directamente desde la interfaz de terminal (TUI) sin reiniciar el juego ni desconectar a los jugadores.

---

## Galería de Interfaz y Controles

### Pantalla de Conexión y Menú de Calibración

Antes de conectar, cada jugador elige su identificador y el diseño inicial del mando. Durante la partida, la píldora superior muestra la latencia en vivo (`ms`) y permite abrir el menú del mando para cambiar de carcasa, invertir el giro de pantalla o el eje Y, probar la vibración y calibrar la zona muerta del stick.

| Pantalla de Conexión | Menú de Calibración en Partida |
| :---: | :---: |
| ![Pantalla de Conexión](docs/screenshots/join-screen.png) | ![Menú del Mando](docs/screenshots/settings-menu.png) |

---

### Controles Clásicos y Estándar

| Diseño | Vista Previa |
| :--- | :---: |
| **16-Bit** | ![Mando 16-Bit](docs/screenshots/skin-snes.png) |
| **64-Bit** | ![Mando 64-Bit](docs/screenshots/skin-n64.png) |
| **Simétrico** | ![Mando Simétrico](docs/screenshots/skin-xbox.png) |
| **Asimétrico** | ![Mando Asimétrico](docs/screenshots/skin-asymmetric.png) |

---

### Controles Touch-First Adaptados por Género

Diseñados desde cero para pantallas táctiles móviles evitando posturas incómodas de varios dedos:

| Diseño | Vista Previa |
| :--- | :---: |
| **Brawler (`brawler`)** | ![Mando Brawler](docs/screenshots/skin-brawler.png) |
| **Kart / Racing (`kart`)** | ![Mando Kart](docs/screenshots/skin-kart.png) |
| **Arcade 6-Botones (`arcade`)** | ![Mando Arcade](docs/screenshots/skin-arcade.png) |

---

## Controles de la Terminal (TUI)

Mientras `airInput` está en ejecución, puedes gestionar la red y los mandos directamente desde la terminal:

| Tecla | Acción |
| :--- | :--- |
| `Arriba` / `Abajo` (o `k` / `j`) | Navegar por la lista de jugadores conectados. |
| `Enter` / `Espacio` | Seleccionar un jugador y luego presionar `Enter` sobre otro para **intercambiar en caliente (hot-swap)** sus números de mando virtual. |
| `Esc` | Cancelar la selección de intercambio o el aviso de confirmación del hotspot. |
| `h` | Activar o desactivar el **Hotspot Wi-Fi sin internet** (`AirInput-Net`). Regenera el código QR con la IP del punto de acceso (`10.42.0.1`) y abre el puerto `3000` en el firewall. |
| `l` | Mostrar u ocultar el panel de **Logs en Vivo** en la terminal. |
| `q` / `Ctrl+C` | Apagar el hotspot (si estaba activo), eliminar los mandos virtuales y cerrar el programa limpiamente. |

También puedes iniciar AirInput con el panel de logs activado desde el arranque usando el parámetro `-l` o `--logs`:

```bash
sudo ./airInput -l
```

---

## Inicio Rápido

### Requisitos Previos

- **Node.js** (v14 o superior)
- **Python** (3.7 o superior)
- **Go** (1.19 o superior)

### 1. Clonar el repositorio e instalar dependencias de Node.js

```bash
git clone https://github.com/DiegoCChumbi/airInput.git
cd airInput
npm install
```

### 2. Instalar dependencias de Python

**Linux:**
AirInput incluye un controlador nativo `/dev/uinput` en `controller-linux.py`, por lo que funciona directamente con Python 3 estándar. Opcionalmente puedes instalar los paquetes de desarrollo para `python-uinput`:

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

### 3. Compilar el ejecutable de Go

**Linux:**
```bash
go build -o airInput .
```

**Windows:**
```powershell
go build -o airInput.exe .
```

### 4. Ejecutar AirInput

**Linux:**
```bash
sudo ./airInput
```
*(Se requiere `sudo` en Linux para crear dispositivos en `/dev/uinput` y gestionar el punto de acceso Wi-Fi con NetworkManager).*

**Windows:**
```powershell
airInput.exe
```

---

## Cómo Funciona

```text
┌─────────────────┐        WebSocket (TCP 3000)       ┌──────────────────┐
│ Navegador Móvil │ ◄───────────────────────────────► │ Servidor Node.js │
│ (Cliente Táctil)│     Botones, Ejes y Ping RTT      │   (server.js)    │
└─────────────────┘                                   └────────┬─────────┘
                                                               │
                                                JSON (stdio)   │   UDP (127.0.0.1:9999)
                                              ◄────────────────┼────────────────►
                                              │                                 │
                                     ┌────────▼─────────┐              ┌────────▼─────────┐
                                     │   Host TUI Go    │  Comando UDP │  Driver Python   │
                                     │  (airInput.go)   │ ───────────► │ (uinput/vgamepad)│
                                     └──────────────────┘   Hot-Swap   └────────┬─────────┘
                                                                                │
                                                                       ┌────────▼─────────┐
                                                                       │ Mandos Virtuales │
                                                                       │ (Controller 1..N)│
                                                                       └──────────────────┘
```

1. **Orquestador y TUI en Go (`airInput.go`, `hotspot.go`)**: Inicia y supervisa los subprocesos de Node.js y Python, dibuja la interfaz en terminal (Bubble Tea), gestiona la creación del Hotspot Wi-Fi offline y las reglas del firewall, y envía comandos de intercambio de mandos (*hot-swap*).
2. **Servidor WebSocket (`server.js`)**: Sirve la interfaz web móvil en el puerto `3000` (con cabeceras *no-cache* para actualizaciones instantáneas), gestiona el registro de jugadores, mide el ping RTT y reenvía los eventos de botones y ejes por UDP al puerto `9999`.
3. **Controlador de Mandos Virtuales (`controller-linux.py`, `controller-win.py`)**: Escucha en el puerto UDP `9999`, crea un mando virtual independiente por cada jugador registrado y traduce las entradas a eventos de kernel `uinput` (Linux) o `XInput` (`vgamepad` en Windows).

---

## Estructura del Proyecto

```text
airInput/
├── airInput.go              # TUI en Go y gestor de subprocesos
├── hotspot.go               # Gestión de Hotspot Wi-Fi offline y firewall (Linux/Windows)
├── server.js                # Servidor Node.js Express + Socket.IO y puente UDP
├── controller-linux.py      # Driver de mando virtual en Linux (uinput + fallback nativo ioctl)
├── controller-win.py        # Driver de mando virtual en Windows (vgamepad / ViGEmBus)
├── docs/
│   └── screenshots/         # Capturas de pantalla de la TUI y de los 7 mandos
└── public/                  # Cliente web móvil
    ├── index.html           # Pantalla de conexión, barra HUD y menú de calibración
    ├── global.css           # Sistema de diseño de hardware y rotación responsiva
    ├── script.js            # Escáner multitáctil, joysticks con rotación y motor háptico
    └── skins/               # Diseños modulares de controles
        ├── snes/            # Mando clásico de 16-Bit
        ├── n64/             # Mando analógico de 64-Bit
        ├── xbox/            # Mando moderno simétrico
        ├── asymmetric/      # Mando moderno asimétrico
        ├── brawler/         # Mando Touch-First para plataformas de pelea
        ├── kart/            # Mando Touch-First con volante y pedalera de carreras
        └── arcade/          # Tablero Arcade Fightpad de 6 botones
```

---

## Solución de Problemas

### Problemas de Conexión y Hotspot Offline
- **El teléfono no carga la página al conectarse a `AirInput-Net`**: Como el punto de acceso offline no tiene salida a internet, algunos teléfonos redirigen el tráfico automáticamente por los datos móviles (`4G`/`5G`). Apaga los **Datos Móviles** en el celular o acepta el aviso del sistema **"Mantener conexión Wi-Fi"** al conectarte a `AirInput-Net`.
- **Firewall bloqueando el puerto 3000**: AirInput abre automáticamente el puerto TCP `3000` mediante `ufw`, `firewall-cmd` o `iptables`. Si utilizas otro firewall personalizado, permite conexiones entrantes TCP en el puerto `3000`.

### La Respuesta Háptica No Vibra
- **Android**: Asegúrate de desactivar el **Ahorro de batería**, **No Molestar** y el **Modo Silencio**, y verifica que la **Vibración táctil / Vibración de notificaciones** esté activa en los ajustes del sistema (Android bloquea `navigator.vibrate` en todos los navegadores cuando el ahorro de batería o el modo silencio están activos).
- **Botón de Prueba**: Abre el **Menú** dentro del mando y toca **Probar** junto a *Respuesta háptica* para verificar la vibración del dispositivo.

---

## Estado de Tareas (Roadmap)

- [x] Permitir que cada jugador ingrese un identificador para su mando
- [x] Vincular teléfonos con números de mando e intercambiarlos en caliente desde la terminal
- [x] Crear una red Wi-Fi propia (Hotspot) para funcionar sin router ni internet previo
- [x] Añadir diseños de mandos *Touch-First* adaptados por género (`Brawler`, `Kart`, `Arcade 6-Botones`, `Asimétrico`)

---

## Licencia

Este proyecto está licenciado bajo la Licencia MIT.
