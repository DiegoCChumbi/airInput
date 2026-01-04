import socket
import vgamepad as vg
import time

MAX_ABS_VAL = 32767

BTN_MAP = {
    "A": vg.XUSB_BUTTON.XUSB_GAMEPAD_A,
    "B": vg.XUSB_BUTTON.XUSB_GAMEPAD_B,
    "X": vg.XUSB_BUTTON.XUSB_GAMEPAD_X,
    "Y": vg.XUSB_BUTTON.XUSB_GAMEPAD_Y,
    "L": vg.XUSB_BUTTON.XUSB_GAMEPAD_LEFT_SHOULDER,
    "R": vg.XUSB_BUTTON.XUSB_GAMEPAD_RIGHT_SHOULDER,
    "L3": vg.XUSB_BUTTON.XUSB_GAMEPAD_LEFT_THUMB,
    "R3": vg.XUSB_BUTTON.XUSB_GAMEPAD_RIGHT_THUMB,
    "START": vg.XUSB_BUTTON.XUSB_GAMEPAD_START,
    "SELECT": vg.XUSB_BUTTON.XUSB_GAMEPAD_BACK,
    "UP": vg.XUSB_BUTTON.XUSB_GAMEPAD_DPAD_UP,
    "DOWN": vg.XUSB_BUTTON.XUSB_GAMEPAD_DPAD_DOWN,
    "LEFT": vg.XUSB_BUTTON.XUSB_GAMEPAD_DPAD_LEFT,
    "RIGHT": vg.XUSB_BUTTON.XUSB_GAMEPAD_DPAD_RIGHT,
}

# Player dictionary { 'client_ip': gamepad_object }
players = {}
players_axes = {}

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(("0.0.0.0", 9999))
sock.settimeout(1.0)

print("Virtual Gamepad Server started.")
print("Waiting for connections from Node.js...")

while True:
    try:
        data, addr = sock.recvfrom(1024)
        msg = data.decode().strip()
        parts = msg.split(":")

        if len(parts) < 2: continue

        client_id = parts[0]
        msg_type = parts[1]

        # Create control
        if client_id not in players:
            print(f"New player logged in: {client_id}")
            gamepad = vg.VX360Gamepad()
            players[client_id] = gamepad
            players_axes[client_id] = {'lx': 0, 'ly': 0, 'rx': 0, 'ry': 0}

        gamepad = players[client_id]
        axes_state = players_axes[client_id]

        if msg_type == "btn":
            if len(parts) < 4: continue
            btn_name = parts[2]
            state = int(float(parts[3]))

            if btn_name == "L2":
                gamepad.left_trigger(255 if state else 0)
            elif btn_name == "R2":
                gamepad.right_trigger(255 if state else 0)
            
            elif btn_name in BTN_MAP:
                if state:
                    gamepad.press_button(BTN_MAP[btn_name])
                else:
                    gamepad.release_button(BTN_MAP[btn_name])
            
            print(
                f"P{list(players.keys()).index(client_id) + 1}: {btn_name} -> {state}"
             )
            gamepad.update()

        elif msg_type == "axis":
            if len(parts) < 4: continue
            axis = parts[2]
            val = float(parts[3])

            axes_state[axis] = val

            lx_val = int(axes_state['lx'] * MAX_ABS_VAL)
            ly_val = int(-axes_state['ly'] * MAX_ABS_VAL)
            
            rx_val = int(axes_state['rx'] * MAX_ABS_VAL)
            ry_val = int(-axes_state['ry'] * MAX_ABS_VAL)

            gamepad.left_joystick(x_value=lx_val, y_value=ly_val)
            
            gamepad.right_joystick(x_value=rx_val, y_value=ry_val)

            print(
                f"P{list(players.keys()).index(client_id) + 1}: {axis} -> {val}"
            )
            gamepad.update()

    except socket.timeout:
        continue
    except Exception as e:
        print(f" Error: {e}")

