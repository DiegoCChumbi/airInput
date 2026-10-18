const express = require("express");
const http = require("http");
const socketio = require("socket.io");
const dgram = require("dgram");
const os = require("os");
const qrcode = require("qrcode");

const app = express();
const server = http.createServer(app);
const io = socketio(server);
const udp = dgram.createSocket("udp4");

app.use(express.static("public"));

const players = new Map(); // Stores { socket.id -> {username, controllerId} }
const activeUsernames = new Set(); 

function getLocalIP() {
    const interfaces = os.networkInterfaces();
    for (const name of Object.keys(interfaces)) {
        for (const iface of interfaces[name]) {
            if (iface.family === 'IPv4' && !iface.internal) {
                return iface.address;
            }
        }
    }
    return 'localhost';
}

function sendToParent(message) {
    process.stdout.write(JSON.stringify(message) + '\n');
}

// Accept commands from the Go parent process via stdin (e.g. generate_qr)
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin, terminal: false });
rl.on('line', (line) => {
    try {
        const cmd = JSON.parse(line.trim());
        if (cmd.cmd === 'generate_qr' && cmd.url) {
            qrcode.toString(cmd.url, { type: 'terminal', small: true }, (err, qrString) => {
                sendToParent({
                    event: 'server_ready',
                    url: cmd.url,
                    qr: qrString || 'Could not generate QR code.'
                });
            });
        }
    } catch (_) { /* ignore malformed lines */ }
});

io.on("connection", socket => {
    socket.on("ping_check", (cb) => {
        if (typeof cb === "function") cb();
    });

    socket.on("register_player", (payload) => {
        const rawUsername = payload && typeof payload.username === "string" ? payload.username : "";
        const username = rawUsername.replace(/:/g, "").trim().slice(0, 24);

        if (!username || username.toLowerCase() === "system") {
            socket.emit('registration_failed', 'Nombre de usuario inválido.');
            return;
        }

        // If this socket is already registered under the same username, succeed idempotently
        if (players.has(socket.id) && players.get(socket.id).username === username) {
            socket.emit('registration_success');
            return;
        }

        if (activeUsernames.has(username)) {
            socket.emit('registration_failed', `El nombre "${username}" ya está en uso.`);
            return;
        }

        let settled = false;
        const timeoutId = setTimeout(() => {
            if (settled) return;
            settled = true;
            udp.removeListener('message', listener);
            socket.emit('registration_failed', 'El controlador virtual no respondió (verifica permisos/uinput o ViGEmBus).');
        }, 3000);

        const listener = (data) => {
            const parts = data.toString().split(":");
            if (parts.length < 3 || parts[0] !== username || parts[1] !== 'controller_id') {
                return;
            }
            if (settled) return;
            settled = true;
            clearTimeout(timeoutId);
            udp.removeListener('message', listener);

            const controllerId = parseInt(parts[2], 10);

            const playerData = { username, controllerId };
            players.set(socket.id, playerData);
            activeUsernames.add(username);

            sendToParent({
                event: "player_connect",
                username: username,
                controllerId: controllerId
            });

            socket.emit('registration_success');
        };

        udp.on('message', listener);

        const msg = Buffer.from(`${username}:get_id`);
        udp.send(msg, 9999, "127.0.0.1", (err) => {
            if (err && !settled) {
                settled = true;
                clearTimeout(timeoutId);
                udp.removeListener('message', listener);
                socket.emit('registration_failed', 'Error interno del servidor.');
            }
        });
    });

    socket.on("input", ({ button, state } = {}) => {
        if (!players.has(socket.id) || typeof button !== "string") return;
        const cleanBtn = button.replace(/:/g, "").trim();
        const numState = state ? 1 : 0;
        const { username } = players.get(socket.id);
        const msg = Buffer.from(`${username}:btn:${cleanBtn}:${numState}`);
        udp.send(msg, 9999, "127.0.0.1");
    });

    socket.on("axis", ({ axis, value } = {}) => {
        if (!players.has(socket.id) || typeof axis !== "string") return;
        const numVal = Number(value);
        if (!Number.isFinite(numVal)) return;
        const clamped = Math.max(-1, Math.min(1, numVal));
        const { username } = players.get(socket.id);
        const msg = Buffer.from(`${username}:axis:${axis}:${clamped.toFixed(4)}`);
        udp.send(msg, 9999, "127.0.0.1");
    });

    socket.on("disconnect", () => {
        if (players.has(socket.id)) {
            const { username } = players.get(socket.id);
            const msg = Buffer.from(`${username}:disconnect`);
            udp.send(msg, 9999, "127.0.0.1");

            players.delete(socket.id);
            activeUsernames.delete(username);

            sendToParent({
                event: "player_disconnect",
                username: username
            });
        }
    });
});

const PORT = 3000;
server.listen(PORT, () => {
    const ip = getLocalIP();
    const url = `http://${ip}:${PORT}`;

    qrcode.toString(url, { type: 'terminal', small: true }, (err, qrString) => {
        sendToParent({
            event: "server_ready",
            url: url,
            qr: qrString || "Could not generate QR code."
        });
    });
});
