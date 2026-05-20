const express = require("express");
const http = require("http");
const socketio = require("socket.io");
const dgram = require("dgram");
const os = require("os");
const qrcode = require("qrcode-terminal");

const app = express();
const server = http.createServer(app);
const io = socketio(server);

const udp = dgram.createSocket("udp4");

app.use(express.static("public"));

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

const players = new Map(); // Almacenará { socket.id -> username }
const activeUsernames = new Set(); // Para evitar nombres de usuario duplicados

io.on("connection", socket => {
  console.log(`Client connected: ${socket.id}`);

  socket.on("register_player", ({ username }) => {
    if (activeUsernames.has(username)) {
      socket.emit('registration_failed', `El nombre "${username}" ya está en uso.`);
      return;
    }

    console.log(`Player registered: ${username} (${socket.id})`);
    players.set(socket.id, username);
    activeUsernames.add(username);
    socket.emit('registration_success');
  });

  socket.on("input", ({ button, state }) => {
    if (!players.has(socket.id)) return; // Ignorar si no está registrado
    const username = players.get(socket.id);
    const msg = Buffer.from(`${username}:btn:${button}:${state}`);
    udp.send(msg, 9999, "127.0.0.1");
  });

  socket.on("axis", ({ axis, value }) => {
    if (!players.has(socket.id)) return;
    const username = players.get(socket.id);
    const msg = Buffer.from(`${username}:axis:${axis}:${value.toFixed(4)}`);
    udp.send(msg, 9999, "127.0.0.1");
  });

  socket.on("disconnect", () => {
    if (players.has(socket.id)) {
      const username = players.get(socket.id);
      console.log(`Player disconnected: ${username}`);

      // Notificar a Python para que elimine el mando virtual
      const msg = Buffer.from(`${username}:disconnect`);
      udp.send(msg, 9999, "127.0.0.1");

      players.delete(socket.id);
      activeUsernames.delete(username);
    } else {
      console.log(`Client disconnected: ${socket.id}`);
    }
  });
});


const PORT = 3000;
server.listen(PORT, () => {
  const ip = getLocalIP();
  const url = `http://${ip}:${PORT}`;

  const c = {
    cyan: "\x1b[36m",
    green: "\x1b[32m",
    yellow: "\x1b[33m",
    blue: "\x1b[34m",
    white: "\x1b[37m",
    dim: "\x1b[2m",
    bold: "\x1b[1m",
    reset: "\x1b[0m"
  };

  const boxWidth = 50;
  const pad = (textLength) => " ".repeat(Math.max(0, boxWidth - textLength));

  const statusText = "🟢 Online - Ready to connect!";
  const urlLabel = "  URL:    ";

  console.clear();

  console.log(c.cyan + "╔" + "═".repeat(boxWidth) + "╗" + c.reset);
  console.log(c.cyan + "║" + c.reset + c.bold + "                AIR-INPUT SERVER                  " + c.reset + c.cyan + "║" + c.reset);
  console.log(c.cyan + "╠" + "═".repeat(boxWidth) + "╣" + c.reset);
  console.log(c.cyan + "║" + " ".repeat(boxWidth) + "║" + c.reset);

  const line1Base = "  Status: " + statusText;
  console.log(c.cyan + "║" + c.reset + "  Status: " + c.green + statusText + c.reset + pad(line1Base.length) + c.cyan + "║" + c.reset);

  const line2Base = urlLabel + url;
  console.log(c.cyan + "║" + c.reset + c.yellow + urlLabel + c.reset + c.bold + url + c.reset + pad(line2Base.length) + c.cyan + "║" + c.reset);

  console.log(c.cyan + "║" + " ".repeat(boxWidth) + "║" + c.reset);
  console.log(c.cyan + "╠" + "═".repeat(boxWidth) + "╣" + c.reset);

  const footerText = " Connect via URL or scan the QR code ";
  const footerPad = Math.floor((boxWidth - footerText.length) / 2);
  const footerPadStr = " ".repeat(footerPad);
  const footerPadRight = " ".repeat(boxWidth - footerText.length - footerPad);

  console.log(c.cyan + "║" + c.reset + c.dim + footerPadStr + footerText + footerPadRight + c.reset + c.cyan + "║" + c.reset);
  console.log(c.cyan + "╚" + "═".repeat(boxWidth) + "╝" + c.reset);
  console.log("");

  qrcode.generate(url, { small: true }, function (qrcode) {
    const lines = qrcode.split('\n');
    lines.forEach(line => {
      console.log("          " + line);
    });
  });

  console.log("\n" + c.dim + "  [ CTRL+C to stop ]" + c.reset + "\n");
});
