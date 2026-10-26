package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
)

//go:embed public
var embeddedPublic embed.FS

type wsClientMsg struct {
	Event    string  `json:"event"`
	Username string  `json:"username,omitempty"`
	Button   string  `json:"button,omitempty"`
	State    int     `json:"state,omitempty"`
	Axis     string  `json:"axis,omitempty"`
	Value    float64 `json:"value,omitempty"`
}

type wsServerMsg struct {
	Event   string `json:"event"`
	Message string `json:"message,omitempty"`
}

type wsConnState struct {
	conn         *websocket.Conn
	writeMu      sync.Mutex
	username     string
	controllerID int
	registered   bool
}

func (c *wsConnState) sendJSON(msg wsServerMsg) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(msg)
}

type WebServer struct {
	port               int
	devMode            bool
	gm                 *GamepadManager
	onPlayerConnect    func(username string, controllerID int)
	onPlayerDisconnect func(username string)
	httpServer         *http.Server

	mu              sync.Mutex
	activeUsernames map[string]*wsConnState
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWebServer(
	port int,
	devMode bool,
	gm *GamepadManager,
	onPlayerConnect func(username string, controllerID int),
	onPlayerDisconnect func(username string),
) *WebServer {
	return &WebServer{
		port:               port,
		devMode:            devMode,
		gm:                 gm,
		onPlayerConnect:    onPlayerConnect,
		onPlayerDisconnect: onPlayerDisconnect,
		activeUsernames:    make(map[string]*wsConnState),
	}
}

func (ws *WebServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)

	var staticFS http.FileSystem
	if ws.devMode {
		staticFS = http.Dir("public")
	} else {
		sub, err := fs.Sub(embeddedPublic, "public")
		if err != nil {
			return err
		}
		staticFS = http.FS(sub)
	}

	fileServer := http.FileServer(staticFS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", ws.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	ws.httpServer = &http.Server{
		Handler: mux,
	}

	go func() {
		_ = ws.httpServer.Serve(ln)
	}()

	return nil
}

func (ws *WebServer) Stop() {
	if ws.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = ws.httpServer.Shutdown(ctx)
	}
}

func (ws *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &wsConnState{
		conn: conn,
	}

	defer func() {
		ws.cleanupClient(client)
		_ = conn.Close()
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg wsClientMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Event {
		case "ping_check":
			_ = client.sendJSON(wsServerMsg{Event: "pong_check"})

		case "register_player":
			ws.handleRegister(client, msg.Username)

		case "input":
			if !client.registered || msg.Button == "" {
				continue
			}
			cleanBtn := strings.TrimSpace(strings.ReplaceAll(msg.Button, ":", ""))
			state := 0
			if msg.State != 0 {
				state = 1
			}
			ws.gm.HandleButton(client.username, cleanBtn, state)

		case "axis":
			if !client.registered || msg.Axis == "" {
				continue
			}
			ws.gm.HandleAxis(client.username, msg.Axis, msg.Value)
		}
	}
}

func (ws *WebServer) handleRegister(client *wsConnState, rawUsername string) {
	username := strings.TrimSpace(strings.ReplaceAll(rawUsername, ":", ""))
	if len(username) > 24 {
		username = username[:24]
	}

	if username == "" || strings.EqualFold(username, "system") {
		_ = client.sendJSON(wsServerMsg{
			Event:   "registration_failed",
			Message: "Nombre de usuario inválido.",
		})
		return
	}

	ws.mu.Lock()
	if client.registered && client.username == username {
		ws.mu.Unlock()
		_ = client.sendJSON(wsServerMsg{Event: "registration_success"})
		return
	}

	if existing, exists := ws.activeUsernames[username]; exists && existing != client {
		ws.mu.Unlock()
		_ = client.sendJSON(wsServerMsg{
			Event:   "registration_failed",
			Message: fmt.Sprintf("El nombre \"%s\" ya está en uso.", username),
		})
		return
	}
	ws.mu.Unlock()

	controllerID, err := ws.gm.RegisterPlayer(username)
	if err != nil {
		_ = client.sendJSON(wsServerMsg{
			Event:   "registration_failed",
			Message: fmt.Sprintf("Error al crear el mando virtual: %v", err),
		})
		return
	}

	ws.mu.Lock()
	client.username = username
	client.controllerID = controllerID
	client.registered = true
	ws.activeUsernames[username] = client
	ws.mu.Unlock()

	if ws.onPlayerConnect != nil {
		ws.onPlayerConnect(username, controllerID)
	}

	_ = client.sendJSON(wsServerMsg{Event: "registration_success"})
}

func (ws *WebServer) cleanupClient(client *wsConnState) {
	ws.mu.Lock()
	if !client.registered || client.username == "" {
		ws.mu.Unlock()
		return
	}
	username := client.username
	if current, ok := ws.activeUsernames[username]; ok && current == client {
		delete(ws.activeUsernames, username)
	}
	client.registered = false
	ws.mu.Unlock()

	ws.gm.DisconnectPlayer(username)
	if ws.onPlayerDisconnect != nil {
		ws.onPlayerDisconnect(username)
	}
}

func GetLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "localhost"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "localhost"
}

func GenerateTerminalQR(text string) string {
	qr, err := qrcode.New(text, qrcode.Low)
	if err != nil {
		return "Could not generate QR code."
	}
	return strings.TrimRight(qr.ToSmallString(false), "\n")
}
