# AirInput

Transforma tu smartphone en un control inalámbrico para tu PC. Juega tus juegos  con diseños de mando personalizables directamente desde tu navegador, no requiere instalación de aplicaciones.

## Características

- **🌐 Basado en Navegador**: No necesitas app móvil - funciona completamente en el navegador de tu teléfono
- **🎨 Múltiples Diseños**: Elige entre el estilo clásico o moderno
- **🔌 Mando Virtual**: Crea un control virtual reconocido por los juegos
- **📱 Soporte Multi-Jugador**: Conecta múltiples dispositivos como controles separados
- **⚡ Baja Latencia**: Transmisión de entrada en tiempo real vía WebSockets y UDP
- **🔧 Multiplataforma**: Soporta Windows y Linux
- **📶 Red Local**: Toda la comunicación permanece en tu red local

## Inicio Rápido

### Requisitos Previos

- **Node.js** (v14 o superior)
- **Python** (3.7 o superior)
- **Go** (1.19 o superior)

### Instalación

1. **Clona el repositorio**

   ```bash
   git clone https://github.com/DiegoCChumbi/airInput.git
   cd airInput
   ```

2. **Instala las dependencias de Node.js**

   ```bash
   npm install
   ```

3. **Instala las dependencias de Python**

   Windows:

   ```powershell
   pip install vgamepad
   ```

   Linux:

   ```bash
   sudo dnf install python3-devel libudev-devel
   pip install python-uinput
   ```

4. **Compila el ejecutable de Go**

**Windows:**

   ```powershell
   go build -o airInput.exe airInput.go
   ```

**Linux:**

   ```bash
   go build -o airInput airInput.go
   ```

### Ejecutar airInput

Simplemente ejecuta el ejecutable compilado:

**Windows:**

```powershell
airInput.exe
```

**Linux:**

```bash
sudo ./airInput
```

*En Linux se requiere de sudo para poder crear los gamepads virtuales.*

La aplicación:

1. Iniciará el servidor web en el puerto 3000
2. Mostrará la URL de conexión y código QR en la terminal
3. Lanzará el controlador de mando virtual

### Conectar tu Teléfono

1. Asegúrate de que tu teléfono esté en la misma red WiFi que tu PC
2. Abre el navegador de tu teléfono y navega a la URL mostrada en la terminal (O escanéa el QR que aparecerá)
3. Toca "TOCAR PARA INICIAR" para activar el control
4. Elige tu diseño preferido en la configuración (⚙️)

## Cómo Funciona

```
┌─────────────┐         WebSocket           ┌──────────────┐
│  Navegador  │ ◄─────────────────────────► │   Node.js    │
│ (Teléfono)  │   (Datos Botones/Ejes)      │   Server     │
└─────────────┘                             └──────┬───────┘
                                                   │
                                                UDP│ (Puerto 9999)
                                                   │
                                            ┌──────▼───────┐
                                            │   Python     │
                                            │ Controlador  │
                                            └──────┬───────┘
                                                   │
                                            Virtual│Gamepad API
                                                   │
                                            ┌──────▼───────┐
                                            │   Control    │
                                            │  Xbox 360    │
                                            │   Virtual    │
                                            └──────────────┘
```

1. **Interfaz Web**: Control táctil basado en HTML5 con NippleJS para joysticks
2. **Servidor WebSocket**: Node.js con Socket.IO para comunicación en tiempo real
3. **Puente UDP**: Reenvía los datos de entrada al controlador Python
4. **Mando Virtual**: Python crea controles virtuales Xbox 360 usando librerías específicas de cada plataforma
5. **Gestor de Procesos**: Go orquesta los procesos de Node.js y Python

## Estructura del Proyecto

```
airInput/
├── airInput.go              # Gestor de procesos Go
├── server.js                # Servidor WebSocket Node.js
├── controller-win.py        # Mando virtual Windows
├── controller-linux.py      # Mando virtual Linux
├── package.json             # Dependencias Node.js
├── go.mod                   # Definición módulo Go
└── public/                  # Cliente web
    ├── index.html           # HTML principal
    ├── script.js            # Lógica del cliente
    ├── global.css           # Estilos base
    └── skins/               # Diseños de control
        ├── snes/            # Diseño clásico
        └── xbox/            # Diseño moderno
```

## Configuración

El servidor web se ejecuta en el puerto 3000 por defecto. Para cambiarlo, edita [server.js](server.js):

```javascript
const PORT = 3000; // Cambia este valor
```

La comunicación UDP usa el puerto 9999. Asegúrate de que este puerto esté disponible en tu sistema.

### Problemas de Conexión

- Verifica que tu teléfono y PC estén en la misma red
- Revisa que la configuración del firewall permita conexiones en el puerto 3000
- Desactiva la VPN si está activa

### Retraso en la Entrada

- Reduce la distancia entre el teléfono y el router WiFi
- Cierra otras aplicaciones que usen ancho de banda de red
- Usa la banda WiFi de 5GHz si está disponible

## Contribuir

¡Las contribuciones son bienvenidas! Siéntete libre de:

- Reportar errores abriendo un issue
- Sugerir nuevas funcionalidades o mejoras
- Enviar pull requests con mejoras
- Aportar con nuevos layout personalizados

## Licencia

Este proyecto está licenciado bajo la Licencia MIT.

## Agradecimientos

- [NippleJS](https://github.com/yoannmoinet/nipplejs) - Librería de joystick virtual
- [Socket.IO](https://socket.io/) - Comunicación en tiempo real
- [vgamepad](https://github.com/yagnateos/vgamepad) - Mando virtual para Windows
- [ViGEmBus](https://github.com/ViGEm/ViGEmBus) - Driver de mando virtual para Windows

---

**Nota**: Esta es una solución de red local. Tu teléfono y PC deben estar conectados a la misma red WiFi.
