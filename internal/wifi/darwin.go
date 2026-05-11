package wifi

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// airport binary path (present on macOS; requires SIP-exempt access on newer OS).
const airportPath = "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"

var (
	rxDarwinIPv4    = regexp.MustCompile(`inet\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\s`)
	rxDarwinGateway = regexp.MustCompile(`gateway:\s*([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	rxDarwinIPv6    = regexp.MustCompile(`inet6\s+([0-9a-fA-F:]+%?\S*)`)
	rxDarwinLatency = regexp.MustCompile(`time=([0-9.]+)\s*ms`)
	rxDarwinLoss    = regexp.MustCompile(`(\d+)\.?\d*%\s+packet loss`)
)

func getDarwin() Info {
	now := time.Now()

	// --- 1. Find Wi-Fi interface name via networksetup ---
	nsOut, err := exec.Command("networksetup", "-listallhardwareports").CombinedOutput()
	if err != nil {
		return Info{Timestamp: now, Error: "networksetup not available"}
	}

	wifiInterfaces := parseNetworksetupWifi(normalizeNewlines(string(nsOut)))
	if len(wifiInterfaces) == 0 {
		return Info{Timestamp: now, HasWifi: false, Error: "no Wi-Fi adapter detected"}
	}

	// --- 2. Query airport for connection details on each interface ---
	adapters := make([]Adapter, 0, len(wifiInterfaces))
	activeIndex := 0

	for _, iface := range wifiInterfaces {
		adapter := Adapter{Name: iface}

		airOut, err := exec.Command(airportPath, "-I").CombinedOutput()
		if err == nil {
			fields := parseColonLines(normalizeNewlines(string(airOut)))
			ssid := fields["SSID"]
			bssid := fields["BSSID"]
			if ssid == "" {
				ssid = fields[" SSID"] // airport sometimes indents
			}

			agrScore := fields["agrScore"] // 0..1000 → percent
			if agrScore == "" {
				agrScore = fields["lastTxRate"] // fallback
			}
			signalDBM, _ := strconv.Atoi(fields["agrCtlRSSI"])
			noisDBM, _ := strconv.Atoi(fields["agrCtlNoise"])
			_ = noisDBM

			signalPct := 0
			if signalDBM != 0 {
				signalPct = dbmToPercentDarwin(signalDBM)
			}

			channel := fields["channel"]
			radioType := fields["op mode"]
			state := "disconnected"
			if ssid != "" {
				state = "connected"
			}

			// Determine band from channel number.
			band := channelToBandDarwin(channel)

			txRate := fields["lastTxRate"]
			rxRate := fields["maxRate"]

			adapter.State = state
			adapter.SSID = ssid
			adapter.BSSID = bssid
			adapter.RadioType = radioType
			adapter.Channel = channel
			adapter.Band = band
			adapter.SignalPercent = signalPct
			if signalPct > 0 {
				adapter.SignalText = strconv.Itoa(signalPct) + "%"
			}
			adapter.TransmitRateMbps = txRate
			adapter.ReceiveRateMbps = rxRate
		}

		// --- 3. IP / gateway via ifconfig + netstat ---
		ifcOut, _ := exec.Command("ifconfig", iface).CombinedOutput()
		ifcText := normalizeNewlines(string(ifcOut))
		if m := rxDarwinIPv4.FindStringSubmatch(ifcText); len(m) == 2 {
			adapter.IPv4 = m[1]
		}
		if m := rxDarwinIPv6.FindStringSubmatch(ifcText); len(m) == 2 {
			v6 := strings.TrimSuffix(m[1], "%"+iface)
			if !strings.HasPrefix(strings.ToLower(v6), "fe80") {
				adapter.IPv6 = v6
			}
		}

		routeOut, _ := exec.Command("netstat", "-rn", "-f", "inet").CombinedOutput()
		if m := rxDarwinGateway.FindStringSubmatch(normalizeNewlines(string(routeOut))); len(m) == 2 {
			adapter.Gateway = m[1]
		}

		adapters = append(adapters, adapter)
		if adapter.Connected() {
			activeIndex = len(adapters) - 1
		}
	}

	hasInternet := false
	for _, a := range adapters {
		if a.Connected() && a.Gateway != "" {
			hasInternet = true
			break
		}
	}

	return Info{
		Timestamp:   now,
		HasWifi:     true,
		HasInternet: hasInternet,
		ActiveIndex: activeIndex,
		Adapters:    adapters,
	}
}

func pingGatewayDarwin(gateway string) PingResult {
	now := time.Now()
	if strings.TrimSpace(gateway) == "" {
		return PingResult{Error: "no gateway", CheckedAt: now}
	}

	out, err := exec.Command("ping", "-c", "1", "-W", "1000", gateway).CombinedOutput()
	if err != nil && len(out) == 0 {
		return PingResult{Error: err.Error(), CheckedAt: now}
	}

	text := normalizeNewlines(string(out))
	result := PingResult{CheckedAt: now, Sent: 1}

	if m := rxDarwinLatency.FindStringSubmatch(text); len(m) == 2 {
		if latency, err := strconv.ParseFloat(m[1], 64); err == nil {
			result.LatencyMS = int(latency + 0.5)
			result.Reachable = true
			result.Received = 1
		}
	}
	if m := rxDarwinLoss.FindStringSubmatch(text); len(m) == 2 {
		if loss, err := strconv.ParseFloat(m[1], 64); err == nil {
			result.PacketLoss = loss
		}
	} else if !result.Reachable {
		result.PacketLoss = 100
	}
	if !result.Reachable && result.Error == "" {
		result.Error = "request timed out"
	}
	return result
}

// parseNetworksetupWifi returns interface names (e.g. "en0") for Wi-Fi ports.
func parseNetworksetupWifi(text string) []string {
	var ifaces []string
	lines := strings.Split(text, "\n")
	isWifi := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Hardware Port") {
			port := strings.ToLower(trimmed)
			isWifi = strings.Contains(port, "wi-fi") || strings.Contains(port, "airport") || strings.Contains(port, "wireless")
			continue
		}
		if isWifi && strings.HasPrefix(trimmed, "Device:") {
			iface := strings.TrimSpace(strings.TrimPrefix(trimmed, "Device:"))
			if iface != "" {
				ifaces = append(ifaces, iface)
			}
			isWifi = false
		}
	}
	return ifaces
}

// parseColonLines parses "  Key: Value" output (airport -I style).
func parseColonLines(text string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			fields[key] = value
		}
	}
	return fields
}

func dbmToPercentDarwin(dbm int) int {
	if dbm >= -30 {
		return 100
	}
	if dbm <= -90 {
		return 0
	}
	return (dbm + 90) * 100 / 60
}

// channelToBandDarwin infers the band from the Wi-Fi channel number.
// Channels 1–13 are 2.4 GHz; 36+ are 5 GHz; 1–233 mapped from 6 GHz start at 1.
func channelToBandDarwin(ch string) string {
	// airport sometimes reports "6,+1" or "36,80" – take just the number part.
	num, err := strconv.Atoi(strings.SplitN(ch, ",", 2)[0])
	if err != nil {
		return ""
	}
	switch {
	case num <= 14:
		return "2.4 GHz"
	case num >= 182:
		return "6 GHz"
	default:
		return "5 GHz"
	}
}
