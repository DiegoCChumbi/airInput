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
- **Sticks Analógicos con Detección de Rotación**: Motor propio de joysticks virtuales que proyecta vectores 2D con precisión tanto en modo horizontal como vertical (`+90°` y `-90°`), con ajuste de zona muerta (*deadzone*) e inversión de eje Y.
- **Motor Háptico Multicapa**: Combina la Web Vibration API, soporte para Taptic Engine en iOS Safari y un pulso táctil sub-grave por Web Audio como refuerzo mecánico.
- **Telemetría de Latencia y Logs en Vivo**: Indicador de latencia (`ms`) en tiempo real integrado en la barra superior del mando y visor interactivo de eventos en la terminal.
- **Driver Nativo `uinput` en Linux con Respaldo**: Incluye implementación directa mediante `ioctl` sobre `/dev/uinput` en Python para Linux si la librería `python-uinput` no está instalada.

---

## Galería de Interfaz y Controles

### Pantalla de Conexión y Menú de Calibración

Antes de conectar, cada jugador elige su identificador y el diseño inicial del mando. Durante la partida, la píldora superior muestra la latencia en vivo (`ms`) y permite abrir el menú del mando para cambiar de carcasa, invertir el giro de pantalla o el eje Y, probar la vibración y calibrar la zona muerta del stick.

| Pantalla de Conexión (`PORT 01`) | Menú de Calibración en Partida |
| :---: | :---: |
| ![Pantalla de Conexión](docs/screenshots/join-screen.png) | ![Menú del Mando](docs/screenshots/settings-menu.png) |

---

### Controles Clásicos y Estándar

| Diseño | Vista Previa | Ergonomía y Uso Recomendado |
| :--- | :---: | :--- |
| **16-Bit (`snes`)** | ![Mando 16-Bit](docs/screenshots/skin-snes.png) | Cruceta clásica con zonas diagonales invisibles, botones `SELECT`/`START` inclinados y clúster de 4 botones (`X`, `Y`, `A`, `B`) con balancines diagonales. Ideal para plataformas 2D y RPGs retro. |
| **64-Bit (`n64`)** | ![Mando 64-Bit](docs/screenshots/skin-n64.png) | Stick analógico con guía octogonal, cruceta, gatillo central `Z` (`SELECT`), botón `START` central, botones `A`/`B` desplazados y clúster amarillo de 4 botones `C`. |
| **Simétrico (`xbox`)** | ![Mando Simétrico](docs/screenshots/skin-xbox.png) | Doble stick analógico alineado en la parte inferior, cruceta arriba a la izquierda, botones `XYAB` arriba a la derecha, hombros `L1`/`L2` y `R1`/`R2`, y botones `L3`/`R3`. |
| **Asimétrico (`asymmetric`)** | ![Mando Asimétrico](docs/screenshots/skin-asymmetric.png) | Sticks analógicos desplazados (stick izquierdo en la posición primaria superior-izquierda, cruceta abajo a la izquierda, `XYAB` arriba a la derecha y stick derecho abajo a la derecha). Ideal para acción 3D y *twin-stick shooters*. |

---

### Controles Touch-First Adaptados por Género

Diseñados desde cero para pantallas táctiles móviles evitando posturas incómodas de varios dedos:

| Diseño | Vista Previa | Detalles de Ergonomía Táctil |
| :--- | :---: | :--- |
| **Brawler (`brawler`)** | ![Mando Brawler](docs/screenshots/skin-brawler.png) | **Plataforma de Pelea**: Botón gigante central `A` (Ataque) rodeado por dos botones curvos de salto (`Y` arriba y `X` a la derecha) para deslizar el pulgar de salto a ataque aéreo sin levantar el dedo, botón `B` (Especial) a la izquierda y `C-Stick` amarillo ubicado directamente debajo de `A` para ataques *Smash* instantáneos. |
| **Kart (`kart`)** | ![Mando Kart](docs/screenshots/skin-kart.png) | **Carreras / Arcade Racer**: Volante analógico a la izquierda con un botón grande `OBJETO (L)` justo encima para lanzar o mantener arrastrando ítems sin soltar la dirección, más botón `ESPEJO ATRÁS (X)`. A la derecha, pedalera vertical con `FRENO (B)`, `ACELERAR (A)` y un **pedal combo `DERRAPE (A+R)` de 1 pulgar** que mantiene el acelerador y activa el derrape simultáneamente. |
| **Arcade 6-Botones (`arcade`)** | ![Mando Arcade](docs/screenshots/skin-arcade.png) | **Pelea 2D / Beat 'em up**: Cruceta de 8 direcciones de gran tamaño con diagonales explícitas (`↖`, `↗`, `↙`, `↘`) para *motion inputs* (`236`, `623`) más palanca auxiliar `LEVER` a la izquierda. A la derecha, tablero curvo estilo *Vewlix* de 6 botones en 2 filas de 3 (`X`, `Y`, `R1` arriba; `A`, `B`, `R2` abajo) y macros superiores `AGARRE (A+X)` / `SKILL (B+Y)`. |

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
