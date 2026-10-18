package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	hotspotSSID      = "AirInput"
	hotspotPass      = "airinput1"
	hotspotConName   = "AirInput-Hotspot"
	hotspotDefaultIP = "10.42.0.1"
)

// wifiInfo holds the current state of the WiFi adapter.
type wifiInfo struct {
	iface     string
	available bool
	connected bool // true if currently connected to another network
}

// wifiQRPayload returns the standard WIFI QR code string for automatic mobile connection.
func wifiQRPayload() string {
	return fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", hotspotSSID, hotspotPass)
}

// getWifiInfo detects the WiFi adapter and its connection state.
func getWifiInfo() wifiInfo {
	if runtime.GOOS == "windows" {
		return getWifiInfoWindows()
	}
	return getWifiInfoLinux()
}

func getWifiInfoLinux() wifiInfo {
	cmd := exec.Command("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "device")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return wifiInfo{}
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, ":", 3)
		// parts[1] must be "wifi" (not "wifi-p2p")
		if len(parts) == 3 && parts[1] == "wifi" {
			state := strings.ToLower(strings.TrimSpace(parts[2]))
			isConnected := (strings.HasPrefix(state, "connected") || strings.HasPrefix(state, "conectado")) &&
				!strings.HasPrefix(state, "disconnected") && !strings.HasPrefix(state, "desconectado")
			return wifiInfo{
				iface:     parts[0],
				available: true,
				connected: isConnected,
			}
		}
	}
	return wifiInfo{}
}

// getDeviceStateCodeLinux returns the numeric NetworkManager state code for iface:
// 10 = unmanaged, 20 = unavailable, 30 = disconnected, 40..90 = preparing/configuring, 100 = connected.
func getDeviceStateCodeLinux(iface string) int {
	if iface == "" {
		return 0
	}
	out, err := exec.Command("nmcli", "-g", "GENERAL.STATE", "device", "show", iface).Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) > 0 {
		if code, err := strconv.Atoi(fields[0]); err == nil {
			return code
		}
	}
	return 0
}

// waitUntilWifiDeviceReady ensures the WiFi adapter is unblocked, enabled, and in state >= 30
// (disconnected or connected) before attempting to start a hotspot.
func waitUntilWifiDeviceReady(iface string) error {
	// Fast path: if the device is already in state >= 30 (disconnected or connected), do not touch
	// wpa_supplicant or interface link state so we don't disrupt a working driver/backend (e.g. iwd).
	if getDeviceStateCodeLinux(iface) >= 30 {
		return nil
	}

	// Device is currently unavailable (20) or unmanaged (10): unblock and enable WiFi radio
	exec.Command("rfkill", "unblock", "wifi").Run()
	exec.Command("rfkill", "unblock", "all").Run()
	exec.Command("/usr/sbin/rfkill", "unblock", "all").Run()
	exec.Command("nmcli", "radio", "wifi", "on").Run()

	if iface != "" {
		exec.Command("nmcli", "device", "set", iface, "managed", "yes").Run()
		exec.Command("ip", "link", "set", iface, "up").Run()
	}

	// Poll up to 10 seconds (25 * 400ms) until device state >= 30 (disconnected or connected)
	for i := 0; i < 25; i++ {
		if getDeviceStateCodeLinux(iface) >= 30 {
			// Give the WiFi backend a moment to finish initializing after leaving unavailable state
			time.Sleep(600 * time.Millisecond)
			return nil
		}
		if i == 8 {
			exec.Command("nmcli", "radio", "wifi", "on").Run()
			if iface != "" {
				exec.Command("nmcli", "device", "set", iface, "managed", "yes").Run()
				exec.Command("ip", "link", "set", iface, "up").Run()
			}
		}
		if i == 15 {
			// Last resort if NetworkManager is stuck holding the device in unavailable state
			exec.Command("systemctl", "restart", "NetworkManager").Run()
		}
		time.Sleep(400 * time.Millisecond)
	}

	return nil
}

// getWifiDiagLinux returns a short diagnostic summary of the device state and radio flags.
func getWifiDiagLinux(iface string) string {
	stateOut, _ := exec.Command("nmcli", "-g", "GENERAL.STATE", "device", "show", iface).Output()
	radioOut, _ := exec.Command("nmcli", "-t", "-f", "WIFI-HW,WIFI", "radio", "all").Output()
	return fmt.Sprintf("[state=%s radio=%s]",
		strings.TrimSpace(string(stateOut)),
		strings.TrimSpace(string(radioOut)))
}

func getWifiInfoWindows() wifiInfo {
	// Language-independent detection via PowerShell Get-NetAdapter
	psCmd := `Get-NetAdapter | Where-Object { $_.InterfaceDescription -match 'Wi-Fi|Wireless|802\.11|WLAN' -or $_.Name -match 'Wi-Fi|WLAN|Inalámbrica' } | Select-Object -First 1 -Property Name,Status | ForEach-Object { "$($_.Name):$($_.Status)" }`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err == nil {
		line := strings.TrimSpace(string(out))
		if line != "" {
			parts := strings.SplitN(line, ":", 2)
			iface := strings.TrimSpace(parts[0])
			status := ""
			if len(parts) == 2 {
				status = strings.ToLower(strings.TrimSpace(parts[1]))
			}
			if iface != "" {
				return wifiInfo{
					iface:     iface,
					available: true,
					connected: status == "up" || status == "connected" || status == "conectado",
				}
			}
		}
	}

	// Fallback to netsh wlan show interfaces (supports English and Spanish output)
	out, err = exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return wifiInfo{}
	}
	var iface string
	connected := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "name") || strings.HasPrefix(lower, "nombre") {
			if p := strings.SplitN(line, ":", 2); len(p) == 2 && iface == "" {
				iface = strings.TrimSpace(p[1])
			}
		}
		if (strings.HasPrefix(lower, "state") || strings.HasPrefix(lower, "estado")) &&
			(strings.Contains(lower, "connected") || strings.Contains(lower, "conectado")) &&
			!strings.Contains(lower, "disconnected") && !strings.Contains(lower, "desconectado") {
			connected = true
		}
	}
	return wifiInfo{iface: iface, available: iface != "", connected: connected}
}

// startHotspot launches the OS hotspot from scratch (works without internet).
func startHotspot(iface string) error {
	if runtime.GOOS == "windows" {
		return startHotspotWindows()
	}
	return startHotspotLinux(iface)
}

func startHotspotLinux(iface string) error {
	// 1. Ensure WiFi radio is unblocked, managed, and wait until device leaves "unavailable" state
	if err := waitUntilWifiDeviceReady(iface); err != nil {
		return err
	}

	// 2. Remove any leftover hotspot profile from a previous session
	exec.Command("nmcli", "connection", "delete", hotspotConName).Run()

	// 3. Create the standalone hotspot with NetworkManager (same proven method as main)
	out, err := exec.Command("nmcli", "device", "wifi", "hotspot",
		"con-name", hotspotConName,
		"ifname", iface,
		"ssid", hotspotSSID,
		"password", hotspotPass,
		"band", "bg",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s %s", err, strings.TrimSpace(string(out)), getWifiDiagLinux(iface))
	}

	// 4. Ensure explicit static IPv4 (10.42.0.1/24), shared DHCP method, and trusted firewall zone
	exec.Command("nmcli", "connection", "modify", hotspotConName,
		"ipv4.method", "shared",
		"ipv4.addresses", hotspotDefaultIP+"/24",
		"connection.zone", "trusted",
	).Run()
	if iface != "" {
		exec.Command("nmcli", "device", "reapply", iface).Run()
	}

	return nil
}

// openFirewallPort allows port 3000 (TCP) and, when a hotspot interface is provided,
// also opens DHCP (67,68/udp) and DNS (53/udp,tcp) on that interface so phones can obtain an IP offline.
func openFirewallPort(iface string) {
	if runtime.GOOS == "windows" {
		exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name=AirInput", "dir=in", "action=allow", "protocol=TCP", "localport=3000").Run()
		return
	}

	// 1. firewalld: allow port 3000 and dhcp/dns on nm-shared, trusted, and default zones
	exec.Command("firewall-cmd", "--zone=nm-shared", "--add-port=3000/tcp").Run()
	exec.Command("firewall-cmd", "--zone=nm-shared", "--add-service=dhcp").Run()
	exec.Command("firewall-cmd", "--zone=nm-shared", "--add-service=dns").Run()
	exec.Command("firewall-cmd", "--zone=trusted", "--add-port=3000/tcp").Run()
	exec.Command("firewall-cmd", "--add-port=3000/tcp").Run()

	// 2. ufw: allow port 3000/tcp + full traffic on the hotspot interface (for DHCP/DNS)
	if _, err := exec.LookPath("ufw"); err == nil {
		exec.Command("ufw", "allow", "3000/tcp").Run()
		if iface != "" {
			exec.Command("ufw", "allow", "in", "on", iface).Run()
		}
	}

	// 3. iptables: ensure port 3000 and hotspot interface DHCP/DNS traffic are accepted
	if _, err := exec.LookPath("iptables"); err == nil {
		if exec.Command("iptables", "-C", "INPUT", "-p", "tcp", "--dport", "3000", "-j", "ACCEPT").Run() != nil {
			exec.Command("iptables", "-I", "INPUT", "-p", "tcp", "--dport", "3000", "-j", "ACCEPT").Run()
		}
		if iface != "" {
			if exec.Command("iptables", "-C", "INPUT", "-i", iface, "-j", "ACCEPT").Run() != nil {
				exec.Command("iptables", "-I", "INPUT", "-i", iface, "-j", "ACCEPT").Run()
			}
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
		if iface != "" {
			exec.Command("ufw", "delete", "allow", "in", "on", iface).Run()
		}
	}

	if _, err := exec.LookPath("iptables"); err == nil {
		if iface != "" {
			exec.Command("iptables", "-D", "INPUT", "-i", iface, "-j", "ACCEPT").Run()
		}
	}
}

// startHotspotWindows starts a standalone hotspot on Windows 10/11 (NOTE: Untested on physical Windows hardware).
// It first attempts WinRT NetworkOperatorTetheringManager, then offline Wi-Fi Direct Legacy AP, then netsh hostednetwork.
func startHotspotWindows() error {
	// Method 1: PowerShell WinRT NetworkOperatorTetheringManager (Windows 10/11 Mobile Hotspot)
	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`+"`"+`1' })[0]
Function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    $netTask.Result
}
$connectionProfile = [Windows.Networking.Connectivity.NetworkInformation,Windows.Networking.Connectivity,ContentType=WindowsRuntime]::GetInternetConnectionProfile()
if ($null -eq $connectionProfile) {
    $profiles = [Windows.Networking.Connectivity.NetworkInformation,Windows.Networking.Connectivity,ContentType=WindowsRuntime]::GetConnectionProfiles()
    if ($profiles.Count -gt 0) { $connectionProfile = $profiles[0] }
}
if ($null -ne $connectionProfile) {
    $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows.Networking.NetworkOperators,ContentType=WindowsRuntime]::CreateFromConnectionProfile($connectionProfile)
    $config = New-Object Windows.Networking.NetworkOperators.NetworkOperatorTetheringAccessPointConfiguration
    $config.Ssid = '%s'
    $config.Passphrase = '%s'
    $configureTask = $tetheringManager.ConfigureAccessPointAsync($config)
    [System.WindowsRuntimeSystemExtensions]::AsTask($configureTask).Wait(-1) | Out-Null
    $result = Await ($tetheringManager.StartTetheringAsync()) ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
    if ($result.Status -eq 'Success') { exit 0 }
}
exit 1
`, hotspotSSID, hotspotPass)

	if err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript).Run(); err == nil {
		return nil
	}

	// Method 2: Fallback to netsh wlan hostednetwork
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

// waitForHotspotIP polls until the hotspot interface has its new IP or times out (~20s).
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
	// 1. Primary source: query the AirInput-Hotspot connection profile directly via nmcli
	//    so we never accidentally read an old router IP from before the hotspot started.
	out, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "connection", "show", hotspotConName).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if raw != "" {
			ip := strings.TrimSpace(strings.Split(raw, "/")[0])
			if ip != "" {
				return ip, nil
			}
		}
	}

	// 2. Second source: check active connection on iface via nmcli device show
	if iface != "" {
		connOut, err := exec.Command("nmcli", "-g", "GENERAL.CONNECTION", "device", "show", iface).Output()
		if err == nil && strings.TrimSpace(string(connOut)) == hotspotConName {
			out, err = exec.Command("nmcli", "-g", "IP4.ADDRESS", "device", "show", iface).Output()
			if err == nil {
				raw := strings.TrimSpace(string(out))
				if raw != "" {
					ip := strings.TrimSpace(strings.Split(raw, "/")[0])
					if ip != "" {
						return ip, nil
					}
				}
			}
		}
	}

	// 3. Fallback: inspect interface addresses, preferring 10.42.x.x (NetworkManager shared subnet)
	if iface != "" {
		if ifi, err := net.InterfaceByName(iface); err == nil {
			if addrs, err := ifi.Addrs(); err == nil {
				for _, addr := range addrs {
					if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
						if ip4 := ipNet.IP.To4(); ip4 != nil {
							ipStr := ip4.String()
							if strings.HasPrefix(ipStr, "10.42.") {
								return ipStr, nil
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no hotspot IP assigned yet")
}

func getHotspotIPWindows() (string, error) {
	// Language-independent IP lookup via PowerShell (checks 192.168.137.x ICS/Hotspot subnet or Wi-Fi Direct adapter)
	psCmd := `(Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -like '192.168.137.*' -or $_.InterfaceAlias -match 'Local Area Connection\*|Conexión de área local\*|Wi-Fi Direct' } | Select-Object -First 1).IPAddress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" {
			return ip, nil
		}
	}

	// Fallback to ipconfig (supports both English and Spanish)
	out, err = exec.Command("ipconfig").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	inHotspot := false
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(line, "Local Area Connection*") || strings.Contains(lower, "conexión de área local*") {
			inHotspot = true
		}
		if inHotspot && (strings.Contains(line, "IPv4") || strings.Contains(lower, "dirección ipv4")) {
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
func stopHotspot(iface string) {
	if runtime.GOOS == "windows" {
		closeFirewallPort("")
		exec.Command("netsh", "wlan", "stop", "hostednetwork").Run()
		return
	}
	closeFirewallPort(iface)
	exec.Command("nmcli", "connection", "down", hotspotConName).Run()
	exec.Command("nmcli", "connection", "delete", hotspotConName).Run()
}

