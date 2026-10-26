package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestGenerateTerminalQR(t *testing.T) {
	qr := GenerateTerminalQR("http://192.168.1.10:3000")
	if qr == "" || qr == "Could not generate QR code." {
		t.Fatalf("expected valid QR string, got: %q", qr)
	}
	if !strings.ContainsAny(qr, "█▀▄") {
		t.Fatalf("expected block characters in QR output")
	}
}

func TestLinuxStructSizes(t *testing.T) {
	var dev uinputUserDev
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, &dev); err != nil {
		t.Fatalf("binary.Write uinputUserDev failed: %v", err)
	}
	if buf.Len() != 1116 {
		t.Fatalf("expected uinputUserDev size 1116 bytes, got %d", buf.Len())
	}

	buf.Reset()
	var ev inputEvent
	if err := binary.Write(&buf, binary.LittleEndian, &ev); err != nil {
		t.Fatalf("binary.Write inputEvent failed: %v", err)
	}
	if buf.Len() != 24 {
		t.Fatalf("expected inputEvent size 24 bytes, got %d", buf.Len())
	}
}

func TestEmbeddedWebServerAndWebSocket(t *testing.T) {
	gm := NewGamepadManager(nil)
	defer gm.CloseAll()

	port := 13847
	ws := NewWebServer(port, false, gm, nil, nil)
	if err := ws.Start(); err != nil {
		t.Fatalf("failed to start WebServer: %v", err)
	}
	defer ws.Stop()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	skins := []string{"snes", "n64", "xbox", "asymmetric", "brawler", "kart", "arcade"}
	paths := []string{"/", "/global.css", "/script.js"}
	for _, s := range skins {
		paths = append(paths, fmt.Sprintf("/skins/%s/layout.html", s), fmt.Sprintf("/skins/%s/style.css", s))
	}

	for _, p := range paths {
		resp, err := http.Get(baseURL + p)
		if err != nil {
			t.Fatalf("GET %s failed: %v", p, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s returned status %d", p, resp.StatusCode)
		}
	}

	// Test WebSocket ping_check -> pong_check
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.WriteJSON(wsClientMsg{Event: "ping_check"}); err != nil {
		t.Fatalf("WriteJSON ping_check failed: %v", err)
	}

	var reply wsServerMsg
	if err := conn.ReadJSON(&reply); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
	if reply.Event != "pong_check" {
		t.Fatalf("expected pong_check, got %q", reply.Event)
	}
}
