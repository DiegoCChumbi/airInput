package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	qrStyle    = lipgloss.NewStyle().MarginLeft(4)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9"))

	playerListTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#5C5C5C")).Padding(0, 1)
	playerStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
	cursorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")) // Bright green for cursor
	selectedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")) // Yellow for selected

	hotspotActiveStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF88"))
	hotspotInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9"))
	warningStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF8C00"))

	logBoxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#555555")).Padding(0, 1).MarginTop(1)
	logTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E5FF"))
	logLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
)

// --- Model ---
type Player struct {
	Username     string
	ControllerID int
}

type model struct {
	port        int
	devMode     bool
	qrCode      string
	url         string
	originalURL string // URL before hotspot was activated
	players     []Player
	cursor      int
	selected    int // -1 means nothing is selected
	err         error

	gm        *GamepadManager
	webServer *WebServer
	eventChan chan tea.Msg

	// Hotspot state
	wifi            wifiInfo
	hotspotActive   bool
	hotspotStarting bool
	hotspotWarning  bool // waiting for user confirmation
	hotspotErr      string

	// Logs state
	showLogs bool
	logs     []string
	maxLogs  int
	logFile  *os.File
}

type qrCodeMsg struct{ qr, url string }
type playerConnectMsg Player
type playerDisconnectMsg string
type errorMsg struct{ err error }
type wifiStatusMsg wifiInfo
type hotspotStartedMsg struct{ ip string }
type hotspotStoppedMsg struct{}
type hotspotErrMsg struct{ err error }
type gamepadLogMsg string

func initialModel(showLogs, devMode bool, port int) model {
	f, _ := os.OpenFile("airinput.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	return model{
		port:      port,
		devMode:   devMode,
		players:   []Player{},
		selected:  -1,
		showLogs:  showLogs,
		logs:      []string{},
		maxLogs:   8,
		logFile:   f,
		eventChan: make(chan tea.Msg, 256),
	}
}

// --- Commands & Logic ---
func (m *model) Init() tea.Cmd {
	return tea.Batch(m.startServices(), m.waitForEvents(), m.detectWifi())
}

func (m *model) pushEvent(msg tea.Msg) {
	select {
	case m.eventChan <- msg:
	default:
	}
}

func (m *model) waitForEvents() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.eventChan
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *model) detectWifi() tea.Cmd {
	return func() tea.Msg { return wifiStatusMsg(getWifiInfo()) }
}

func (m *model) doStartHotspot() tea.Cmd {
	return func() tea.Msg {
		if err := startHotspot(m.wifi.iface); err != nil {
			return hotspotErrMsg{err}
		}
		ip, err := waitForHotspotIP(m.wifi.iface)
		if err != nil {
			return hotspotErrMsg{err}
		}
		// Open firewall AFTER IP is assigned — ensures the interface
		// is fully registered with firewalld and zone detection works.
		openFirewallPort(m.wifi.iface)
		return hotspotStartedMsg{ip: ip}
	}
}

func (m *model) doStopHotspot() tea.Cmd {
	return func() tea.Msg {
		stopHotspot()
		return hotspotStoppedMsg{}
	}
}

func (m *model) startServices() tea.Cmd {
	m.gm = NewGamepadManager(func(line string) {
		m.pushEvent(gamepadLogMsg(line))
	})

	m.webServer = NewWebServer(
		m.port,
		m.devMode,
		m.gm,
		func(username string, controllerID int) {
			m.pushEvent(playerConnectMsg{Username: username, ControllerID: controllerID})
		},
		func(username string) {
			m.pushEvent(playerDisconnectMsg(username))
		},
	)

	if err := m.webServer.Start(); err != nil {
		return func() tea.Msg {
			return errorMsg{fmt.Errorf("no se pudo iniciar el servidor web en el puerto %d: %w", m.port, err)}
		}
	}

	// Ensure port 3000 is open in UFW/firewalld/iptables for LAN WiFi connections
	go openFirewallPort("")

	ip := GetLocalIP()
	url := fmt.Sprintf("http://%s:%d", ip, m.port)
	qr := GenerateTerminalQR(url)

	return func() tea.Msg {
		return qrCodeMsg{qr: qr, url: url}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cleanup()
			return m, tea.Quit

		case "l":
			m.showLogs = !m.showLogs

		case "h":
			m.hotspotErr = ""
			if m.hotspotActive {
				// Stop the hotspot
				return m, m.doStopHotspot()
			}
			if !m.wifi.available || m.hotspotStarting {
				break
			}
			if m.wifi.connected {
				// Warn user before cutting their WiFi
				m.hotspotWarning = true
			} else {
				m.hotspotStarting = true
				return m, m.doStartHotspot()
			}

		case "y":
			if m.hotspotWarning {
				m.hotspotWarning = false
				m.hotspotStarting = true
				return m, m.doStartHotspot()
			}

		case "n", "esc":
			if m.hotspotWarning {
				m.hotspotWarning = false
				break
			}
			m.selected = -1

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.players)-1 {
				m.cursor++
			}
		case "enter", " ":
			if m.cursor >= len(m.players) {
				break
			}
			if m.selected == -1 {
				m.selected = m.cursor
			} else {
				if m.selected == m.cursor {
					m.selected = -1
				} else {
					playerA := m.players[m.selected]
					playerB := m.players[m.cursor]
					if m.gm != nil {
						m.gm.SwapPlayers(playerA.Username, playerB.Username)
					}
					m.players[m.selected].ControllerID, m.players[m.cursor].ControllerID = m.players[m.cursor].ControllerID, m.players[m.selected].ControllerID
					m.selected = -1
				}
			}
		}

	case qrCodeMsg:
		m.qrCode = msg.qr
		m.url = msg.url
		if m.originalURL == "" {
			m.originalURL = msg.url
		}

	case wifiStatusMsg:
		m.wifi = wifiInfo(msg)

	case hotspotStartedMsg:
		m.hotspotStarting = false
		m.hotspotActive = true
		newURL := fmt.Sprintf("http://%s:%d", msg.ip, m.port)
		m.url = newURL
		m.qrCode = GenerateTerminalQR(newURL)

	case hotspotStoppedMsg:
		m.hotspotActive = false
		m.hotspotStarting = false
		// Restore original QR/URL
		if m.originalURL != "" {
			m.url = m.originalURL
			m.qrCode = GenerateTerminalQR(m.originalURL)
		}

	case hotspotErrMsg:
		m.hotspotStarting = false
		m.hotspotActive = false
		m.hotspotErr = msg.err.Error()

	case playerConnectMsg:
		found := false
		for i, p := range m.players {
			if p.Username == msg.Username {
				m.players[i].ControllerID = msg.ControllerID
				found = true
				break
			}
		}
		if !found {
			m.players = append(m.players, Player(msg))
		}
		return m, m.waitForEvents()

	case playerDisconnectMsg:
		var updatedPlayers []Player
		usernameToDisconnect := string(msg)
		for _, p := range m.players {
			if p.Username != usernameToDisconnect {
				updatedPlayers = append(updatedPlayers, p)
			}
		}
		m.players = updatedPlayers
		if m.cursor >= len(m.players) && len(m.players) > 0 {
			m.cursor = len(m.players) - 1
		} else if len(m.players) == 0 {
			m.cursor = 0
		}
		m.selected = -1
		return m, m.waitForEvents()

	case gamepadLogMsg:
		line := string(msg)
		if line != "" {
			if m.logFile != nil {
				_, _ = m.logFile.WriteString(line + "\n")
			}
			m.logs = append(m.logs, line)
			if len(m.logs) > m.maxLogs {
				m.logs = m.logs[len(m.logs)-m.maxLogs:]
			}
		}
		return m, m.waitForEvents()

	case errorMsg:
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\n", m.err)
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("🚀 AirInput Server is Running") + "\n")
	if m.url != "" {
		b.WriteString(fmt.Sprintf("URL: %s\n", lipgloss.NewStyle().Bold(true).Render(m.url)))
	} else {
		b.WriteString("Waiting for server to start...\n")
	}

	if m.qrCode != "" {
		b.WriteString(qrStyle.Render(m.qrCode) + "\n\n")
	} else if m.url != "" {
		b.WriteString("Generating QR code...\n\n")
	}

	// --- Hotspot section ---
	if m.hotspotWarning {
		b.WriteString(warningStyle.Render("⚠️  Activar el hotspot desconectará tu WiFi actual.") + "\n")
		b.WriteString(helpStyle.Render("   Presiona 'y' para continuar o 'n' para cancelar.") + "\n\n")
	} else if m.hotspotStarting {
		b.WriteString(hotspotInactiveStyle.Render("📡 Iniciando hotspot...") + "\n\n")
	} else if m.hotspotActive {
		b.WriteString(hotspotActiveStyle.Render(
			fmt.Sprintf("📡 [ACTIVO]  Red: %s   Contraseña: %s", hotspotSSID, hotspotPass),
		) + "\n")
		b.WriteString(helpStyle.Render("   💡 Si no carga en el móvil: apaga sus Datos Móviles o acepta 'Mantener Wi-Fi'") + "\n\n")
	} else if m.hotspotErr != "" {
		b.WriteString(warningStyle.Render("❌ Error hotspot: "+m.hotspotErr) + "\n\n")
	} else if m.wifi.available {
		b.WriteString(hotspotInactiveStyle.Render("📡 Hotspot inactivo") + "\n\n")
	}

	b.WriteString(playerListTitleStyle.Render("🎮 Connected Players") + "\n")
	if len(m.players) == 0 {
		b.WriteString("No players connected yet.\n")
	} else {
		for i, p := range m.players {
			cursor := " "
			if m.cursor == i {
				cursor = cursorStyle.Render(">")
			}
			line := fmt.Sprintf(" %s (Controller %d)", p.Username, p.ControllerID)
			if m.cursor == i {
				line = cursorStyle.Render(line)
			}
			if m.selected == i {
				line = selectedStyle.Render(line)
			}
			b.WriteString(cursor + line + "\n")
		}
	}

	if m.showLogs {
		var logContent strings.Builder
		logTitle := logTitleStyle.Render("📋 Logs en Vivo (presiona 'l' para ocultar)")
		logContent.WriteString(logTitle + "\n")
		if len(m.logs) == 0 {
			logContent.WriteString(helpStyle.Render("  Esperando eventos de mandos / jugadores...") + "\n")
		} else {
			for _, line := range m.logs {
				logContent.WriteString(logLineStyle.Render("  "+line) + "\n")
			}
		}
		b.WriteString("\n" + logBoxStyle.Render(logContent.String()) + "\n")
	}

	logsHelp := "'l' logs [OFF]"
	if m.showLogs {
		logsHelp = "'l' logs [ON]"
	}

	var help string
	switch {
	case m.hotspotWarning:
		help = "'y' confirmar  'n' cancelar"
	case m.selected != -1:
		help = fmt.Sprintf("Swapping '%s'. Select another to swap or Esc to cancel.", m.players[m.selected].Username)
	case m.hotspotActive:
		help = fmt.Sprintf("↑/↓ navegar  Enter seleccionar  'h' apagar hotspot  %s  'q' salir", logsHelp)
	default:
		help = fmt.Sprintf("↑/↓ navegar  Enter seleccionar  'h' hotspot  %s  'q' salir", logsHelp)
	}
	b.WriteString("\n" + helpStyle.Render(help) + "\n")

	return b.String()
}

func (m *model) cleanup() {
	if m.hotspotActive {
		stopHotspot()
		m.hotspotActive = false
	}
	if m.webServer != nil {
		m.webServer.Stop()
		m.webServer = nil
	}
	if m.gm != nil {
		m.gm.CloseAll()
		m.gm = nil
	}
	if m.logFile != nil {
		_ = m.logFile.Close()
		m.logFile = nil
	}
}

func main() {
	logsFlag := flag.Bool("logs", false, "Habilitar visualización de logs de entrada")
	flag.BoolVar(logsFlag, "l", false, "Habilitar visualización de logs de entrada (shorthand)")
	devFlag := flag.Bool("dev", false, "Servir archivos de public/ desde disco en lugar de los embebidos")
	portFlag := flag.Int("port", 3000, "Puerto HTTP/WebSocket del servidor")
	flag.Parse()

	m := initialModel(*logsFlag, *devFlag, *portFlag)
	defer m.cleanup()

	p := tea.NewProgram(&m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		m.cleanup()
		os.Exit(1)
	}

	if fm, ok := finalModel.(*model); ok && fm.err != nil {
		m.cleanup()
		os.Exit(1)
	}

	fmt.Println("Bye!")
}
