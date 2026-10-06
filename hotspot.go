package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	hotspotSSID    = "AirInput"
	hotspotPass    = "airinput1"
	hotspotConName = "AirInput-Hotspot"
)

// wifiInfo holds the current state of the WiFi adapter.
type wifiInfo struct {
	iface     string
	available bool
	connected bool // true if currently connected to another network
}

// getWifiInfo detects the WiFi adapter and its connection state.
func getWifiInfo() wifiInfo {
	if runtime.GOOS == "windows" {
		return getWifiInfoWindows()
	}
	return getWifiInfoLinux()
}

func getWifiInfoLinux() wifiInfo {
	out, err := exec.Command("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "device").Output()
	if err != nil {
		return wifiInfo{}
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, ":", 3)
		// parts[1] must be exactly "wifi" (not "wifi-p2p")
		if len(parts) == 3 && parts[1] == "wifi" {
			return wifiInfo{
				iface:     parts[0],
				available: true,
				connected: strings.HasPrefix(parts[2], "connected"),
			}
		}
	}
	return wifiInfo{}
}

func getWifiInfoWindows() wifiInfo {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return wifiInfo{}
	}
	var iface string
	connected := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name") {
			if p := strings.SplitN(line, ":", 2); len(p) == 2 {
				iface = strings.TrimSpace(p[1])
			}
		}
		if strings.Contains(line, "State") && strings.Contains(strings.ToLower(line), "connected") {
			connected = true
		}
	}
	return wifiInfo{iface: iface, available: iface != "", connected: connected}
}

// startHotspot launches the OS hotspot. Blocking — meant to run inside a tea.Cmd.
func startHotspot(iface string) error {
	if runtime.GOOS == "windows" {
		return startHotspotWindows()
	}
	return startHotspotLinux(iface)
}

func startHotspotLinux(iface string) error {
	// Remove any leftover connection from a previous session
	exec.Command("nmcli", "connection", "delete", hotspotConName).Run()

	out, err := exec.Command("nmcli", "device", "wifi", "hotspot",
		"con-name", hotspotConName,
		"ifname", iface,
		"ssid", hotspotSSID,
		"password", hotspotPass,
		"band", "bg",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}

	// Tell NetworkManager to treat this connection as trusted so firewalld allows all traffic on it
	exec.Command("nmcli", "connection", "modify", hotspotConName, "connection.zone", "trusted").Run()
	exec.Command("nmcli", "device", "reapply", iface).Run()

	return nil
}

// openFirewallPort allows port 3000 (TCP) through firewalld, ufw, or iptables.
func openFirewallPort(iface string) {
	if runtime.GOOS == "windows" {
		exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name=AirInput", "dir=in", "action=allow", "protocol=TCP", "localport=3000").Run()
		return
	}

	// 1. firewalld: allow port 3000 on nm-shared and default zones
	exec.Command("firewall-cmd", "--zone=nm-shared", "--add-port=3000/tcp").Run()
	exec.Command("firewall-cmd", "--zone=trusted", "--add-port=3000/tcp").Run()
	exec.Command("firewall-cmd", "--add-port=3000/tcp").Run()

	// 2. ufw: if installed and active
	if _, err := exec.LookPath("ufw"); err == nil {
		exec.Command("ufw", "allow", "3000/tcp").Run()
	}

	// 3. iptables: ensure port 3000 is accepted
	if _, err := exec.LookPath("iptables"); err == nil {
		if exec.Command("iptables", "-C", "INPUT", "-p", "tcp", "--dport", "3000", "-j", "ACCEPT").Run() != nil {
			exec.Command("iptables", "-I", "INPUT", "-p", "tcp", "--dport", "3000", "-j", "ACCEPT").Run()
		}
	}
}

// closeFirewallPort removes the firewall rules added by openFirewallPort.
func closeFirewallPort(iface string) {
	if runtime.GOOS == "windows" {
		exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=AirInput").Run()
		return
	}

	exec.Command("firewall-cmd", "--zone=nm-shared", "--remove-port=3000/tcp").Run()
	exec.Command("firewall-cmd", "--zone=trusted", "--remove-port=3000/tcp").Run()
	exec.Command("firewall-cmd", "--remove-port=3000/tcp").Run()

	if _, err := exec.LookPath("ufw"); err == nil {
		exec.Command("ufw", "delete", "allow", "3000/tcp").Run()
	}

	if _, err := exec.LookPath("iptables"); err == nil {
		exec.Command("iptables", "-D", "INPUT", "-p", "tcp", "--dport", "3000", "-j", "ACCEPT").Run()
	}
}

func startHotspotWindows() error {
	steps := [][]string{
		{"netsh", "wlan", "set", "hostednetwork", "mode=allow",
			"ssid=" + hotspotSSID, "key=" + hotspotPass},
		{"netsh", "wlan", "start", "hostednetwork"},
	}
	for _, args := range steps {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// waitForHotspotIP polls until the hotspot interface has an IP or times out (~20s).
func waitForHotspotIP(iface string) (string, error) {
	for i := 0; i < 20; i++ {
		ip, err := getHotspotIP(iface)
		if err == nil && ip != "" {
			return ip, nil
		}
		time.Sleep(1 * time.Second)
	}
	return "", fmt.Errorf("timed out waiting for hotspot IP address")
}

func getHotspotIP(iface string) (string, error) {
	if runtime.GOOS == "windows" {
		return getHotspotIPWindows()
	}
	return getHotspotIPLinux(iface)
}

func getHotspotIPLinux(iface string) (string, error) {
	// First check via net.InterfaceByName
	if iface != "" {
		if ifi, err := net.InterfaceByName(iface); err == nil {
			if addrs, err := ifi.Addrs(); err == nil {
				for _, addr := range addrs {
					if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
						if ip4 := ipNet.IP.To4(); ip4 != nil {
							return ip4.String(), nil
						}
					}
				}
			}
		}
	}

	// Fallback to nmcli connection show
	out, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "connection", "show", hotspotConName).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if raw != "" {
			return strings.Split(raw, "/")[0], nil
		}
	}

	// Fallback to nmcli device show
	if iface != "" {
		out, err = exec.Command("nmcli", "-g", "IP4.ADDRESS", "device", "show", iface).Output()
		if err == nil {
			raw := strings.TrimSpace(string(out))
			if raw != "" {
				return strings.Split(raw, "/")[0], nil
			}
		}
	}

	return "", fmt.Errorf("no IP assigned yet")
}

func getHotspotIPWindows() (string, error) {
	// The Windows hosted network creates a "Local Area Connection*" virtual adapter
	out, err := exec.Command("ipconfig").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	inHotspot := false
	for _, line := range lines {
		if strings.Contains(line, "Local Area Connection*") {
			inHotspot = true
		}
		if inHotspot && strings.Contains(line, "IPv4 Address") {
			if p := strings.SplitN(line, ":", 2); len(p) == 2 {
				return strings.TrimSpace(p[1]), nil
			}
		}
		if inHotspot && strings.TrimSpace(line) == "" {
			inHotspot = false
		}
	}
	return "", fmt.Errorf("hotspot IP not found in ipconfig output")
}

// stopHotspot shuts down the hotspot cleanly.
func stopHotspot() {
	if runtime.GOOS == "windows" {
		closeFirewallPort("")
		exec.Command("netsh", "wlan", "stop", "hostednetwork").Run()
		return
	}
	closeFirewallPort("")
	exec.Command("nmcli", "connection", "down", hotspotConName).Run()
	exec.Command("nmcli", "connection", "delete", hotspotConName).Run()
}
