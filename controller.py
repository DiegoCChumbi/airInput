import socket
import uinput

MIN_ABS = -32767
MAX_ABS = 32767

# Definir botones NES
EVENTS = (
    uinput.BTN_A,
    uinput.BTN_B,
    uinput.BTN_X,
    uinput.BTN_Y,
    uinput.BTN_TR,
    uinput.BTN_TL,
    uinput.BTN_TR2,
    uinput.BTN_TL2,
    uinput.BTN_START,
    uinput.BTN_SELECT,
    uinput.BTN_DPAD_UP,
    uinput.BTN_DPAD_DOWN,
    uinput.BTN_DPAD_LEFT,
    uinput.BTN_DPAD_RIGHT,
    uinput.ABS_X + (MIN_ABS, MAX_ABS, 0, 0),
    uinput.ABS_Y + (MIN_ABS, MAX_ABS, 0, 0),
    uinput.ABS_RX + (MIN_ABS, MAX_ABS, 0, 0),
    uinput.ABS_RY + (MIN_ABS, MAX_ABS, 0, 0),
)

mapping = {
    "A": uinput.BTN_A,
    "B": uinput.BTN_B,
    "X": uinput.BTN_X,
    "Y": uinput.BTN_Y,
    "R": uinput.BTN_TR,
    "L": uinput.BTN_TL,
    "R2": uinput.BTN_TR2,
    "L2": uinput.BTN_TL2,
    "START": uinput.BTN_START,
    "SELECT": uinput.BTN_SELECT,
    "UP": uinput.BTN_DPAD_UP,
    "DOWN": uinput.BTN_DPAD_DOWN,
    "LEFT": uinput.BTN_DPAD_LEFT,
    "RIGHT": uinput.BTN_DPAD_RIGHT,
}

axis_mapping = {
    "lx": uinput.ABS_X,
    "ly": uinput.ABS_Y,
    "rx": uinput.ABS_RX,
    "ry": uinput.ABS_RY,
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
        parts = msg.split(":")
        client_id = parts[0]
        msg_type = parts[1]

        # Si es un jugador nuevo, creamos un control nuevo
        if client_id not in players:
            player_num = len(players) + 1
            device_name = f"PhoneGamepad {player_num}"
            new_device = uinput.Device(EVENTS, name=device_name)
            players[client_id] = new_device
            print(f"Nuevo control creado: Jugador {player_num} ({client_id})")

        if msg_type == "btn":
            button = parts[2]
            state = int(parts[3])
            if button in mapping:
                players[client_id].emit(mapping[button], state)
                print(
                    f"P{list(players.keys()).index(client_id) + 1}: {button} -> {state}"
                )

        elif msg_type == "axis":
            axis_name = parts[2]
            value = float(parts[3])
            if axis_name in axis_mapping:
                scaled_value = int(value * MAX_ABS)
                players[client_id].emit(axis_mapping[axis_name], scaled_value)
                print(
                    f"P{list(players.keys()).index(client_id) + 1}: {axis_name} -> {scaled_value}"
                )

    except ValueError:
        print("Error de formato en mensaje")
    except Exception as e:
        print(f"Error: {e}")
