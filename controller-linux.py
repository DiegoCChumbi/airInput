import socket
import uinput

MIN_ABS = -32767
MAX_ABS = 32767

EVENTS = (
    uinput.BTN_A, uinput.BTN_B, uinput.BTN_X, uinput.BTN_Y,
    uinput.BTN_TR, uinput.BTN_TL, uinput.BTN_TR2, uinput.BTN_TL2,
    uinput.BTN_START, uinput.BTN_SELECT,
    uinput.BTN_DPAD_UP, uinput.BTN_DPAD_DOWN, uinput.BTN_DPAD_LEFT, uinput.BTN_DPAD_RIGHT,
    uinput.ABS_X + (MIN_ABS, MAX_ABS, 0, 0), uinput.ABS_Y + (MIN_ABS, MAX_ABS, 0, 0),
    uinput.ABS_RX + (MIN_ABS, MAX_ABS, 0, 0), uinput.ABS_RY + (MIN_ABS, MAX_ABS, 0, 0),
)

mapping = {
    "A": uinput.BTN_A, "B": uinput.BTN_B, "X": uinput.BTN_X, "Y": uinput.BTN_Y,
    "R": uinput.BTN_TR, "L": uinput.BTN_TL, "R2": uinput.BTN_TR2, "L2": uinput.BTN_TL2,
    "START": uinput.BTN_START, "SELECT": uinput.BTN_SELECT,
    "UP": uinput.BTN_DPAD_UP, "DOWN": uinput.BTN_DPAD_DOWN,
    "LEFT": uinput.BTN_DPAD_LEFT, "RIGHT": uinput.BTN_DPAD_RIGHT,
}

axis_mapping = {
    "lx": uinput.ABS_X, "ly": uinput.ABS_Y,
    "rx": uinput.ABS_RX, "ry": uinput.ABS_RY,
}

# Dictionary to store controls: { 'username': uinput_device }
players = {}
player_numbers = {} # {username: controller_number}
global_controller_counter = 0

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(("127.0.0.1", 9999))

print("Virtual Gamepad Server started (Linux).")
print("Waiting for players to connect...")

while True:
    try:
        data, addr = sock.recvfrom(1024)
        msg = data.decode().strip()
        parts = msg.split(":")
        
        username = parts[0]
        msg_type = parts[1]

        # Handle system commands from Go TUI
        if username == "system":
            command = parts[1]
            if command == "swap":
                user_a, user_b = parts[2], parts[3]
                if user_a in players and user_b in players:
                    # Swap the actual device objects
                    players[user_a], players[user_b] = players[user_b], players[user_a]
                    
                    # Swap numbers for consistent logging
                    player_numbers[user_a], player_numbers[user_b] = player_numbers[user_b], player_numbers[user_a]
                    print(f"[SYSTEM] Swapped controller devices for '{user_a}' and '{user_b}'")
            continue

        # Handle disconnection
        if msg_type == "disconnect":
            if username in players:
                controller_num = player_numbers.get(username, 'N/A')
                del players[username]
                del player_numbers[username]
                print(f"Control for player '{username}' (Controller{controller_num}) removed.")
            continue
        
        # When Node.js asks for a new controller ID
        if msg_type == "get_id":
            if username not in players:
                global_controller_counter += 1
                controller_number = global_controller_counter
                player_numbers[username] = controller_number

                device_name = f"airInput-Controller{controller_number}"
                new_device = uinput.Device(EVENTS, name=device_name)
                players[username] = new_device
                print(f"New control created for player: '{username}' as '{device_name}'")
            
            # Respond to Node.js with the controller ID
            response_msg = f"{username}:controller_id:{player_numbers[username]}".encode()
            sock.sendto(response_msg, addr)
            continue

        # If a player is not registered, ignore their input
        if username not in players:
            continue

        # Handle button presses
        if msg_type == "btn":
            button = parts[2]
            state = int(parts[3])
            if button in mapping:
                players[username].emit(mapping[button], state)
                print(f"[{username} (C{player_numbers[username]})] Button: {button} -> {'Pressed' if state else 'Released'}")

        # Handle axis movement
        elif msg_type == "axis":
            axis_name = parts[2]
            value = float(parts[3])
            if axis_name in axis_mapping:
                scaled_value = int(value * MAX_ABS)
                players[username].emit(axis_mapping[axis_name], scaled_value)
                print(f"[{username} (C{player_numbers[username]})] Axis: {axis_name} -> {scaled_value}")

    except (ValueError, IndexError) as e:
        print(f"Error processing message: '{msg}'. Malformed message. Details: {e}")
    except Exception as e:
        print(f"An unexpected error occurred: {e}")
