import socket
import uinput

# Definir botones NES
EVENTS = (
    uinput.BTN_A,
    uinput.BTN_B,
    uinput.BTN_X,
    uinput.BTN_Y,
    uinput.BTN_TR,
    uinput.BTN_TL,
    uinput.BTN_START,
    uinput.BTN_SELECT,
    uinput.BTN_DPAD_UP,
    uinput.BTN_DPAD_DOWN,
    uinput.BTN_DPAD_LEFT,
    uinput.BTN_DPAD_RIGHT,
)

mapping = {
    "A": uinput.BTN_A,
    "B": uinput.BTN_B,
    "X": uinput.BTN_X,
    "Y": uinput.BTN_Y,
    "R": uinput.BTN_TR,
    "L": uinput.BTN_TL,
    "START": uinput.BTN_START,
    "SELECT": uinput.BTN_SELECT,
    "UP": uinput.BTN_DPAD_UP,
    "DOWN": uinput.BTN_DPAD_DOWN,
    "LEFT": uinput.BTN_DPAD_LEFT,
    "RIGHT": uinput.BTN_DPAD_RIGHT,
}

# Diccionario para guardar los controles: { 'socket_id': uinput_device }
players = {}

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(("127.0.0.1", 9999)) 

print("Servidor de Gamepads Virtuales iniciado.")
print("Esperando conexiones desde Node.js...")

while True:
    data, addr = sock.recvfrom(1024)
    msg = data.decode().strip()
    
    try:
        # Desempaquetamos: ID, Botón, Valor
        client_id, button, value = msg.split(":")
        value = int(value)

        # Si es un jugador nuevo, creamos un control nuevo
        if client_id not in players:
            player_num = len(players) + 1
            new_device = uinput.Device(EVENTS, name=f"PhoneGamepad {player_num}")
            players[client_id] = new_device
            print(f"Nuevo control creado: Jugador {player_num} ({client_id})")

        # Enviamos el input al dispositivo correspondiente a ese ID
        if button in mapping:
            players[client_id].emit(mapping[button], value)
            print(f"P{list(players.keys()).index(client_id)+1}: {button} -> {value}")

    except ValueError:
        print("Error de formato en mensaje")
    except Exception as e:
        print(f"Error: {e}")
