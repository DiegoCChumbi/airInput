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

# Player dictionaries: { 'username': gamepad_object }
players = {}
players_axes = {}
player_numbers = {} # {username: controller_number}
global_controller_counter = 0

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(("127.0.0.1", 9999))
sock.settimeout(1.0)

print("Virtual Gamepad Server started (Windows).")
print("Waiting for players to connect...")

while True:
    try:
        data, addr = sock.recvfrom(1024)
        msg = data.decode().strip()
        parts = msg.split(":")

        if len(parts) < 2: continue

        username = parts[0]
        msg_type = parts[1]

        # Handle system commands from Go TUI
        if username == "system":
            command = parts[1]
            if command == "swap":
                user_a, user_b = parts[2], parts[3]
                if user_a in players and user_b in players:
                    # Swap the actual device objects and their states
                    players[user_a], players[user_b] = players[user_b], players[user_a]
                    players_axes[user_a], players_axes[user_b] = players_axes[user_b], players_axes[user_a]
                    
                    # Swap numbers for consistent logging
                    player_numbers[user_a], player_numbers[user_b] = player_numbers[user_b], player_numbers[user_a]
                    print(f"[SYSTEM] Swapped controller devices for '{user_a}' and '{user_b}'")
            continue

        # Handle disconnection
        if msg_type == "disconnect":
            if username in players:
                controller_num = player_numbers.get(username, 'N/A')
                del players[username]
                del players_axes[username]
                del player_numbers[username]
                print(f"Control for player '{username}' (Controller{controller_num}) removed.")
            continue

        # When Node.js asks for a new controller ID
        if msg_type == "get_id":
            if username not in players:
                global_controller_counter += 1
                controller_number = global_controller_counter
                player_numbers[username] = controller_number

                print(f"New player registered: '{username}' as 'airInput-Controller{controller_number}'")
                gamepad = vg.VX360Gamepad()
                players[username] = gamepad
                players_axes[username] = {'lx': 0, 'ly': 0, 'rx': 0, 'ry': 0, 'lt': 0, 'rt': 0}

            # Respond to Node.js with the controller ID
            response_msg = f"{username}:controller_id:{player_numbers[username]}".encode()
            sock.sendto(response_msg, addr)
            continue
        
        # If a player is not registered, ignore their input
        if username not in players:
            continue

        gamepad = players[username]
        axes_state = players_axes[username]

        # Handle button presses
        if msg_type == "btn":
            if len(parts) < 4: continue
            btn_name = parts[2]
            state = int(float(parts[3]))

            if btn_name == "L2":
                gamepad.left_trigger(255 if state else 0)
                print(f"[{username} (C{player_numbers[username]})] Button: {btn_name} -> {'Pressed' if state else 'Released'}")
            elif btn_name == "R2":
                gamepad.right_trigger(255 if state else 0)
                print(f"[{username} (C{player_numbers[username]})] Button: {btn_name} -> {'Pressed' if state else 'Released'}")
            elif btn_name in BTN_MAP:
                if state:
                    gamepad.press_button(BTN_MAP[btn_name])
                else:
                    gamepad.release_button(BTN_MAP[btn_name])
                print(f"[{username} (C{player_numbers[username]})] Button: {btn_name} -> {'Pressed' if state else 'Released'}")
            
            gamepad.update()

        # Handle axis movement
        elif msg_type == "axis":
            if len(parts) < 4: continue
            axis = parts[2]
            val = float(parts[3])

            axes_state[axis] = val
            
            lx_val = int(axes_state['lx'] * MAX_ABS_VAL)
            ly_val = int(axes_state['ly'] * MAX_ABS_VAL) # Y-axis is often inverted
            rx_val = int(axes_state['rx'] * MAX_ABS_VAL)
            ry_val = int(axes_state['ry'] * MAX_ABS_VAL)

            gamepad.left_joystick(x_value=lx_val, y_value=ly_val)
            gamepad.right_joystick(x_value=rx_val, y_value=ry_val)
            print(f"[{username} (C{player_numbers[username]})] Axis: {axis} -> {val:.2f}")
            gamepad.update()

    except socket.timeout:
        continue
    except (ValueError, IndexError) as e:
        print(f"Error processing message: '{msg}'. Malformed message. Details: {e}")
    except Exception as e:
        print(f"An unexpected error occurred: {e}")

