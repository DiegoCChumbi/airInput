import fcntl
import os
import signal
import socket
import struct
import sys
import time

try:
    import uinput
except ImportError:
    class _NativeUInput:
        EV_SYN = 0x00
        EV_KEY = 0x01
        EV_ABS = 0x03
        SYN_REPORT = 0

        UI_SET_EVBIT = 0x40045564
        UI_SET_KEYBIT = 0x40045565
        UI_SET_ABSBIT = 0x40045567
        UI_DEV_CREATE = 0x5501
        UI_DEV_DESTROY = 0x5502

        BTN_A = (EV_KEY, 0x130)
        BTN_B = (EV_KEY, 0x131)
        BTN_X = (EV_KEY, 0x133)
        BTN_Y = (EV_KEY, 0x134)
        BTN_TL = (EV_KEY, 0x136)
        BTN_TR = (EV_KEY, 0x137)
        BTN_TL2 = (EV_KEY, 0x138)
        BTN_TR2 = (EV_KEY, 0x139)
        BTN_SELECT = (EV_KEY, 0x13a)
        BTN_START = (EV_KEY, 0x13b)
        BTN_THUMBL = (EV_KEY, 0x13d)
        BTN_THUMBR = (EV_KEY, 0x13e)
        BTN_DPAD_UP = (EV_KEY, 0x220)
        BTN_DPAD_DOWN = (EV_KEY, 0x221)
        BTN_DPAD_LEFT = (EV_KEY, 0x222)
        BTN_DPAD_RIGHT = (EV_KEY, 0x223)

        ABS_X = (EV_ABS, 0x00)
        ABS_Y = (EV_ABS, 0x01)
        ABS_RX = (EV_ABS, 0x03)
        ABS_RY = (EV_ABS, 0x04)

        class Device:
            _USER_DEV_FMT = "<80sHHHHI64i64i64i64i"
            _EVENT_FMT = "@llHHi"

            def __init__(self, events, name="airInput-Controller"):
                uinput_paths = ["/dev/uinput", "/dev/input/uinput"]
                self._fd = None
                for path in uinput_paths:
                    if os.path.exists(path):
                        self._fd = os.open(path, os.O_WRONLY | os.O_NONBLOCK)
                        break
                if self._fd is None:
                    raise RuntimeError("Could not find /dev/uinput (ensure uinput kernel module is loaded: sudo modprobe uinput)")

                fcntl.ioctl(self._fd, _NativeUInput.UI_SET_EVBIT, _NativeUInput.EV_SYN)
                fcntl.ioctl(self._fd, _NativeUInput.UI_SET_EVBIT, _NativeUInput.EV_KEY)
                fcntl.ioctl(self._fd, _NativeUInput.UI_SET_EVBIT, _NativeUInput.EV_ABS)

                absmax = [0] * 64
                absmin = [0] * 64
                absfuzz = [0] * 64
                absflat = [0] * 64

                for ev in events:
                    ev_type, ev_code = ev[0], ev[1]
                    if ev_type == _NativeUInput.EV_KEY:
                        fcntl.ioctl(self._fd, _NativeUInput.UI_SET_KEYBIT, ev_code)
                    elif ev_type == _NativeUInput.EV_ABS:
                        fcntl.ioctl(self._fd, _NativeUInput.UI_SET_ABSBIT, ev_code)
                        if len(ev) >= 6:
                            absmin[ev_code] = ev[2]
                            absmax[ev_code] = ev[3]
                            absfuzz[ev_code] = ev[4]
                            absflat[ev_code] = ev[5]

                # Use standard USB Xbox 360 Controller vendor/product IDs for broad game compatibility
                dev_struct = struct.pack(
                    self._USER_DEV_FMT,
                    name.encode("utf-8")[:79],
                    0x03,   # BUS_USB
                    0x045e, # Vendor: Microsoft
                    0x028e, # Product: Xbox 360 Controller
                    0x0110, # Version
                    0,      # ff_effects_max
                    *absmax,
                    *absmin,
                    *absfuzz,
                    *absflat,
                )
                os.write(self._fd, dev_struct)
                fcntl.ioctl(self._fd, _NativeUInput.UI_DEV_CREATE)

            def emit(self, event, value, syn=True):
                if self._fd is None:
                    return
                ev_type, ev_code = event[0], event[1]
                now = time.time()
                sec = int(now)
                usec = int((now - sec) * 1_000_000)
                pkt = struct.pack(self._EVENT_FMT, sec, usec, ev_type, ev_code, int(value))
                if syn:
                    pkt += struct.pack(self._EVENT_FMT, sec, usec, _NativeUInput.EV_SYN, _NativeUInput.SYN_REPORT, 0)
                os.write(self._fd, pkt)

            def close(self):
                if self._fd is not None:
                    try:
                        fcntl.ioctl(self._fd, _NativeUInput.UI_DEV_DESTROY)
                    except OSError:
                        pass
                    try:
                        os.close(self._fd)
                    except OSError:
                        pass
                    self._fd = None

            def __del__(self):
                self.close()

    uinput = _NativeUInput

MIN_ABS = -32767
MAX_ABS = 32767

EVENTS = (
    uinput.BTN_A, uinput.BTN_B, uinput.BTN_X, uinput.BTN_Y,
    uinput.BTN_TR, uinput.BTN_TL, uinput.BTN_TR2, uinput.BTN_TL2,
    uinput.BTN_THUMBL, uinput.BTN_THUMBR,
    uinput.BTN_START, uinput.BTN_SELECT,
    uinput.BTN_DPAD_UP, uinput.BTN_DPAD_DOWN, uinput.BTN_DPAD_LEFT, uinput.BTN_DPAD_RIGHT,
    uinput.ABS_X + (MIN_ABS, MAX_ABS, 0, 0), uinput.ABS_Y + (MIN_ABS, MAX_ABS, 0, 0),
    uinput.ABS_RX + (MIN_ABS, MAX_ABS, 0, 0), uinput.ABS_RY + (MIN_ABS, MAX_ABS, 0, 0),
)

mapping = {
    "A": uinput.BTN_A, "B": uinput.BTN_B, "X": uinput.BTN_X, "Y": uinput.BTN_Y,
    "R": uinput.BTN_TR, "L": uinput.BTN_TL, "R2": uinput.BTN_TR2, "L2": uinput.BTN_TL2,
    "L3": uinput.BTN_THUMBL, "R3": uinput.BTN_THUMBR,
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
last_axis_log = {}  # {(username, axis_name): timestamp}
global_controller_counter = 0

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(("127.0.0.1", 9999))

def cleanup_and_exit(signum, frame):
    players.clear()
    sock.close()
    sys.exit(0)

signal.signal(signal.SIGTERM, cleanup_and_exit)
signal.signal(signal.SIGINT, cleanup_and_exit)

print("Virtual Gamepad Server started (Linux).", flush=True)
print("Waiting for players to connect...", flush=True)

while True:
    try:
        data, addr = sock.recvfrom(1024)
        msg = data.decode().strip()
        parts = msg.split(":")
        if len(parts) < 2:
            continue
        
        username = parts[0]
        msg_type = parts[1]

        # Handle system commands from Go TUI
        if username == "system":
            command = parts[1]
            if command == "swap" and len(parts) >= 4:
                user_a, user_b = parts[2], parts[3]
                if user_a in players and user_b in players:
                    # Swap the actual device objects
                    players[user_a], players[user_b] = players[user_b], players[user_a]
                    
                    # Swap numbers for consistent logging
                    player_numbers[user_a], player_numbers[user_b] = player_numbers[user_b], player_numbers[user_a]
                    print(f"[SYSTEM] Swapped controller devices for '{user_a}' and '{user_b}'", flush=True)
            continue

        # Handle disconnection
        if msg_type == "disconnect":
            if username in players:
                controller_num = player_numbers.get(username, 'N/A')
                del players[username]
                del player_numbers[username]
                print(f"Control for player '{username}' (Controller{controller_num}) removed.", flush=True)
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
                print(f"New control created for player: '{username}' as '{device_name}'", flush=True)
            
            # Respond to Node.js with the controller ID
            response_msg = f"{username}:controller_id:{player_numbers[username]}".encode()
            sock.sendto(response_msg, addr)
            continue

        # If a player is not registered, ignore their input
        if username not in players:
            continue

        # Handle button presses
        if msg_type == "btn" and len(parts) >= 4:
            button = parts[2]
            state = int(parts[3])
            if button in mapping:
                players[username].emit(mapping[button], state)
                print(f"[{username} (C{player_numbers[username]})] Button: {button} -> {'Pressed' if state else 'Released'}", flush=True)

        # Handle axis movement
        elif msg_type == "axis" and len(parts) >= 4:
            axis_name = parts[2]
            value = float(parts[3])
            if axis_name in axis_mapping:
                clamped_val = max(-1.0, min(1.0, value))
                scaled_value = int(clamped_val * MAX_ABS)
                players[username].emit(axis_mapping[axis_name], scaled_value)

                now = time.monotonic()
                log_key = (username, axis_name)
                if scaled_value == 0 or (now - last_axis_log.get(log_key, 0.0) >= 0.25):
                    last_axis_log[log_key] = now
                    print(f"[{username} (C{player_numbers[username]})] Axis: {axis_name} -> {scaled_value}", flush=True)

    except (ValueError, IndexError) as e:
        print(f"Error processing message: '{msg}'. Malformed message. Details: {e}", flush=True)
    except Exception as e:
        print(f"An unexpected error occurred: {e}", flush=True)

