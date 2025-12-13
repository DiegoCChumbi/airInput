const express = require("express");
const http = require("http");
const socketio = require("socket.io");
const dgram = require("dgram");
const os = require("os"); // Para buscar tu IP automáticamente

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

  socket.on("input", ({ button, state }) => {
    // FORMATO DEL MENSAJE: "ID_SOCKET:BOTON:ESTADO"
    // Ejemplo: "xHy7_zm9:A:1"
    const msg = Buffer.from(`${socket.id}:${button}:${state}`);
    udp.send(msg, 9999, "127.0.0.1");
  });

  socket.on("disconnect", () => {
    console.log(`Jugador desconectado: ${socket.id}`);
    // Opcional: Podrías enviar un evento 'destroy' a Python si quisieras limpiar
  });
});

const PORT = 3000;
server.listen(PORT, () => {
  console.log(`Server web listo. Conecta los celulares a: http://${getLocalIP()}:${PORT}`);
});
