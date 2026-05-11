package wifi

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	rxLinuxIPv4    = regexp.MustCompile(`inet\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)/`)
	rxLinuxIPv6    = regexp.MustCompile(`inet6\s+([0-9a-fA-F:%]+)/`)
	rxLinuxGateway = regexp.MustCompile(`default\s+via\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	rxLinuxSignal  = regexp.MustCompile(`signal:\s*(-?\d+)\s*dBm`)
	rxLinuxFreq    = regexp.MustCompile(`freq:\s*(\d+)`)
	rxLinuxBitrate = regexp.MustCompile(`tx bitrate:\s*([\d.]+)\s*MBit/s`)
	rxLinuxRxRate  = regexp.MustCompile(`rx bitrate:\s*([\d.]+)\s*MBit/s`)
	rxLinuxBSSID   = regexp.MustCompile(`Connected to\s+([0-9a-fA-F:]{17})`)
	rxLinuxSSIDiw  = regexp.MustCompile(`SSID:\s*(.+)`)
)

func getLinux() Info {
	now := time.Now()

	// --- 1. Enumerate wireless interfaces via nmcli (or fall back to iw) ---
	nmcliOut, err := exec.Command("nmcli", "-t", "-f", "DEVICE,TYPE,STATE,CONNECTION", "device").CombinedOutput()
	if err != nil {
		// nmcli absent – try iw dev as a minimal fallback
		return getLinuxViaIw(now)
	}

	// Parse nmcli device list; keep only wifi rows
	type nmcliRow struct {
		device     string
		state      string
		connection string
	}
	var wifiRows []nmcliRow
	for _, line := range strings.Split(normalizeNewlines(string(nmcliOut)), "\n") {
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		if parts[1] != "wifi" {
			continue
		}
		state := parts[2]
		if state == "unavailable" || state == "unmanaged" {
			continue
		}
		wifiRows = append(wifiRows, nmcliRow{
			device:     parts[0],
			state:      state,
			connection: parts[3],
		})
	}

	if len(wifiRows) == 0 {
		return Info{
			Timestamp: now,
			HasWifi:   false,
			Error:     "no Wi-Fi adapter detected",
		}
	}

	// --- 2. Build per-device SSID/signal map from nmcli dev wifi ---
	type wifiInfo struct {
		ssid    string
		signal  int
		bssid   string
		channel string
		band    string
	}
	deviceWifi := map[string]wifiInfo{}

	wifiOut, _ := exec.Command("nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL,BSSID,CHAN,FREQ,DEVICE", "dev", "wifi").CombinedOutput()
	for _, line := range strings.Split(normalizeNewlines(string(wifiOut)), "\n") {
		parts := strings.SplitN(line, ":", 7)
		if len(parts) < 7 {
			continue
		}
		active := parts[0]
		ssid := parts[1]
		signal, _ := strconv.Atoi(parts[2])
		bssid := strings.ReplaceAll(parts[3], "\\:", ":")
		channel := parts[4]
		freq := parts[5]
		device := parts[6]

		band := ""
		if strings.HasPrefix(freq, "5") {
			band = "5 GHz"
		} else if strings.HasPrefix(freq, "6") {
			band = "6 GHz"
		} else if strings.HasPrefix(freq, "2") {
			band = "2.4 GHz"
		}

		if active == "yes" {
			deviceWifi[device] = wifiInfo{
				ssid:    ssid,
				signal:  signal,
				bssid:   bssid,
				channel: channel,
				band:    band,
			}
		} else if _, already := deviceWifi[device]; !already {
			deviceWifi[device] = wifiInfo{
				ssid:    ssid,
				signal:  signal,
				bssid:   bssid,
				channel: channel,
				band:    band,
			}
		}
	}

	// --- 3. Collect IP/gateway per device via `ip addr` and `ip route` ---
	type ipInfo struct {
		ipv4    string
		ipv6    string
		gateway string
	}
	deviceIP := map[string]ipInfo{}

	for _, row := range wifiRows {
		dev := row.device

		addrOut, _ := exec.Command("ip", "-o", "addr", "show", dev).CombinedOutput()
		addrText := normalizeNewlines(string(addrOut))

		var ipv4, ipv6 string
		if m := rxLinuxIPv4.FindStringSubmatch(addrText); len(m) == 2 {
			ipv4 = m[1]
		}
		if m := rxLinuxIPv6.FindStringSubmatch(addrText); len(m) == 2 {
			// skip link-local (fe80::)
			if !strings.HasPrefix(strings.ToLower(m[1]), "fe80") {
				ipv6 = m[1]
			}
		}

		routeOut, _ := exec.Command("ip", "route", "show", "dev", dev).CombinedOutput()
		var gateway string
		if m := rxLinuxGateway.FindStringSubmatch(normalizeNewlines(string(routeOut))); len(m) == 2 {
			gateway = m[1]
		}

		deviceIP[dev] = ipInfo{ipv4: ipv4, ipv6: ipv6, gateway: gateway}
	}

	// --- 4. Try `iw dev <device> link` for extra radio detail ---
	type iwDetail struct {
		bssid   string
		ssid    string
		signal  int // dBm
		freq    string
		txRate  string
		rxRate  string
		channel string
	}
	deviceIw := map[string]iwDetail{}

	for _, row := range wifiRows {
		out, err := exec.Command("iw", "dev", row.device, "link").CombinedOutput()
		if err != nil {
			continue
		}
		text := normalizeNewlines(string(out))
		if strings.Contains(text, "Not connected") {
			continue
		}
		detail := iwDetail{}
		if m := rxLinuxBSSID.FindStringSubmatch(text); len(m) == 2 {
			detail.bssid = m[1]
		}
		if m := rxLinuxSSIDiw.FindStringSubmatch(text); len(m) == 2 {
			detail.ssid = strings.TrimSpace(m[1])
		}
		if m := rxLinuxSignal.FindStringSubmatch(text); len(m) == 2 {
			detail.signal, _ = strconv.Atoi(m[1])
		}
		if m := rxLinuxFreq.FindStringSubmatch(text); len(m) == 2 {
			detail.freq = m[1]
		}
		if m := rxLinuxBitrate.FindStringSubmatch(text); len(m) == 2 {
			detail.txRate = m[1]
		}
		if m := rxLinuxRxRate.FindStringSubmatch(text); len(m) == 2 {
			detail.rxRate = m[1]
		}
		deviceIw[row.device] = detail
	}

	// --- 5. Assemble Adapter list ---
	adapters := make([]Adapter, 0, len(wifiRows))
	activeIndex := 0

	for _, row := range wifiRows {
		state := "disconnected"
		if strings.Contains(row.state, "connected") {
			state = "connected"
		}

		wi := deviceWifi[row.device]
		ip := deviceIP[row.device]
		iwd := deviceIw[row.device]

		ssid := wi.ssid
		if ssid == "" && iwd.ssid != "" {
			ssid = iwd.ssid
		}
		if ssid == "" && row.connection != "--" && row.connection != "" {
			ssid = row.connection
		}

		bssid := wi.bssid
		if bssid == "" {
			bssid = iwd.bssid
		}

		band := wi.band
		channel := wi.channel

		// Convert dBm signal from iw to percent if nmcli didn't give us one.
		signalPct := wi.signal
		if signalPct == 0 && iwd.signal != 0 {
			signalPct = dbmToPercent(iwd.signal)
		}

		signalText := ""
		if signalPct > 0 {
			signalText = strconv.Itoa(signalPct) + "%"
		}

		// Derive band from frequency if still missing.
		if band == "" && iwd.freq != "" {
			band = freqToBand(iwd.freq)
		}

		adapter := Adapter{
			Name:             row.device,
			State:            state,
			SSID:             ssid,
			BSSID:            bssid,
			Band:             band,
			Channel:          channel,
			SignalText:       signalText,
			SignalPercent:    signalPct,
			TransmitRateMbps: iwd.txRate,
			ReceiveRateMbps:  iwd.rxRate,
			IPv4:             ip.ipv4,
			IPv6:             ip.ipv6,
			Gateway:          ip.gateway,
		}

		adapters = append(adapters, adapter)
		if adapter.Connected() {
			activeIndex = len(adapters) - 1
		}
	}

	return Info{
		Timestamp:   now,
		HasWifi:     true,
		HasInternet: len(adapters) > 0,
		ActiveIndex: activeIndex,
		Adapters:    adapters,
	}
}

// getLinuxViaIw is a minimal fallback when nmcli is absent.
func getLinuxViaIw(now time.Time) Info {
	out, err := exec.Command("iw", "dev").CombinedOutput()
	if err != nil {
		return Info{
			Timestamp: now,
			Error:     "neither nmcli nor iw found",
		}
	}

	var devices []string
	for _, line := range strings.Split(normalizeNewlines(string(out)), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Interface ") {
			devices = append(devices, strings.TrimPrefix(trimmed, "Interface "))
		}
	}
	if len(devices) == 0 {
		return Info{Timestamp: now, HasWifi: false}
	}

	adapters := make([]Adapter, 0, len(devices))
	activeIndex := 0
	for _, dev := range devices {
		lout, _ := exec.Command("iw", "dev", dev, "link").CombinedOutput()
		text := normalizeNewlines(string(lout))
		state := "disconnected"
		ssid := ""
		bssid := ""
		signal := 0

		if !strings.Contains(text, "Not connected") {
			state = "connected"
			if m := rxLinuxSSIDiw.FindStringSubmatch(text); len(m) == 2 {
				ssid = strings.TrimSpace(m[1])
			}
			if m := rxLinuxBSSID.FindStringSubmatch(text); len(m) == 2 {
				bssid = m[1]
			}
			if m := rxLinuxSignal.FindStringSubmatch(text); len(m) == 2 {
				dbm, _ := strconv.Atoi(m[1])
				signal = dbmToPercent(dbm)
			}
		}
		signalText := ""
		if signal > 0 {
			signalText = strconv.Itoa(signal) + "%"
		}
		adapter := Adapter{
			Name:          dev,
			State:         state,
			SSID:          ssid,
			BSSID:         bssid,
			SignalPercent: signal,
			SignalText:    signalText,
		}
		adapters = append(adapters, adapter)
		if adapter.Connected() {
			activeIndex = len(adapters) - 1
		}
	}

	return Info{
		Timestamp:   now,
		HasWifi:     true,
		HasInternet: len(adapters) > 0,
		ActiveIndex: activeIndex,
		Adapters:    adapters,
	}
}

func pingGatewayLinux(gateway string) PingResult {
	now := time.Now()
	if strings.TrimSpace(gateway) == "" {
		return PingResult{
			Error:     "no gateway",
			CheckedAt: now,
		}
	}

	out, err := exec.Command("ping", "-c", "1", "-W", "1", gateway).CombinedOutput()
	if err != nil && len(out) == 0 {
		return PingResult{
			Error:     err.Error(),
			CheckedAt: now,
		}
	}

	text := normalizeNewlines(string(out))
	result := PingResult{
		CheckedAt: now,
		Sent:      1,
	}
	if strings.Contains(text, "1 received") || strings.Contains(text, "1 packets received") {
		result.Reachable = true
		result.Received = 1
		result.PacketLoss = 0
	}
	if idx := strings.Index(text, "time="); idx >= 0 {
		chunk := text[idx+5:]
		if end := strings.Index(chunk, " ms"); end >= 0 {
			if latency, err := strconv.ParseFloat(strings.TrimSpace(chunk[:end]), 64); err == nil {
				result.LatencyMS = int(latency + 0.5)
			}
		}
	}
	if !result.Reachable {
		result.PacketLoss = 100
		result.Error = "request timed out"
	}
	return result
}

// dbmToPercent converts a dBm RSSI value to a 0–100 signal percentage.
// The mapping uses the common -30 dBm (excellent) → 100% to -90 dBm → 0% range.
func dbmToPercent(dbm int) int {
	if dbm >= -30 {
		return 100
	}
	if dbm <= -90 {
		return 0
	}
	return (dbm + 90) * 100 / 60
}

// freqToBand converts a frequency string (MHz) to a human-readable band label.
func freqToBand(freq string) string {
	mhz, err := strconv.Atoi(freq)
	if err != nil {
		return ""
	}
	switch {
	case mhz >= 5925:
		return "6 GHz"
	case mhz >= 5000:
		return "5 GHz"
	case mhz >= 2400:
		return "2.4 GHz"
	default:
		return ""
	}
}