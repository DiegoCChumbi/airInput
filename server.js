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

io.on("connection", socket => {
  console.log(`Jugador conectado: ${socket.id}`);

  // socket.on("input", ({ button, state }) => {
  //   // FORMATO DEL MENSAJE: "ID_SOCKET:BOTON:ESTADO"
  //   // Ejemplo: "xHy7_zm9:A:1"
  //   const msg = Buffer.from(`${socket.id}:${button}:${state}`);
  //   udp.send(msg, 9999, "127.0.0.1");
  // });

  socket.on("input", ({ button, state }) => {
    // Formato: "ID:btn:NOMBRE:ESTADO"
    const msg = Buffer.from(`${socket.id}:btn:${button}:${state}`);
    udp.send(msg, 9999, "127.0.0.1");
  });

  socket.on("axis", ({ axis, value }) => {
    // Formato: "ID:axis:NOMBRE_EJE:VALOR_FLOAT"
    // Ejemplo: "xHy7:axis:lx:-0.54"
    const msg = Buffer.from(`${socket.id}:axis:${axis}:${value.toFixed(4)}`);
    udp.send(msg, 9999, "127.0.0.1");
  });

  socket.on("disconnect", () => {
    console.log(`Jugador desconectado: ${socket.id}`);
  });
});

const PORT = 3000;
server.listen(PORT, () => {
  const ip = getLocalIP();
  const url = `http://${ip}:${PORT}`;
  console.log(`Escanea el qr y conecta los controlles.`);
  qrcode.generate(url, { small: true });
  console.log(`O de conectate a este dirección: ${url}`);
});
