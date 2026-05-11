package wifi

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	rxInterfaceName = regexp.MustCompile(`(?m)^Interface name:\s*(.+?)\s*$`)
	rxIPv4          = regexp.MustCompile(`IPv4 Address[\s.]*:\s*([0-9.]+)`)
	rxIPv6          = regexp.MustCompile(`IPv6 Address[\s.]*:\s*([0-9a-fA-F:%]+)`)
	rxGateway       = regexp.MustCompile(`Default Gateway[\s.]*:\s*([0-9.]+)`)
	rxSuffix        = regexp.MustCompile(`Connection-specific DNS Suffix[\s.]*:\s*(.*)`)
	rxLatency       = regexp.MustCompile(`time[=<]\s*(\d+)ms`)
	rxLoss          = regexp.MustCompile(`Lost = \d+ \((\d+)% loss\)`)
)

func getWindows() Info {
	now := time.Now()

	interfaceOut, err := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput()
	if err != nil {
		return Info{
			Timestamp: now,
			Error:     strings.TrimSpace(string(interfaceOut)),
		}
	}

	text := normalizeNewlines(string(interfaceOut))
	if strings.Contains(text, "There is no wireless interface on the system") {
		return Info{
			Timestamp: now,
			HasWifi:   false,
			Error:     "no Wi-Fi adapter detected",
		}
	}

	adapters := parseWindowsInterfaces(text)
	if len(adapters) == 0 {
		return Info{
			Timestamp: now,
			HasWifi:   false,
			Error:     "no Wi-Fi adapter detected",
		}
	}

	driverOut, _ := exec.Command("netsh", "wlan", "show", "drivers").CombinedOutput()
	driverBands := parseWindowsDrivers(normalizeNewlines(string(driverOut)))

	ipconfigOut, _ := exec.Command("ipconfig").CombinedOutput()
	ipBlocks := parseWindowsIPConfig(normalizeNewlines(string(ipconfigOut)))

	activeIndex := 0
	hasInternet := false
	for idx := range adapters {
		adapter := &adapters[idx]
		if bands, ok := driverBands[adapter.Name]; ok {
			adapter.SupportedBands = bands
		}
		if ip, ok := ipBlocks[adapter.Name]; ok {
			adapter.IPv4 = ip.IPv4
			adapter.IPv6 = ip.IPv6
			adapter.Gateway = ip.Gateway
			adapter.DNSSuffix = ip.DNSSuffix
		}
		if adapter.Connected() {
			activeIndex = idx
			if adapter.Gateway != "" {
				hasInternet = true
			}
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

func pingGatewayWindows(gateway string) PingResult {
	now := time.Now()
	if strings.TrimSpace(gateway) == "" {
		return PingResult{
			Error:     "no gateway",
			CheckedAt: now,
		}
	}

	out, err := exec.Command("ping", "-n", "1", "-w", "1200", gateway).CombinedOutput()
	if err != nil && len(out) == 0 {
		return PingResult{
			Error:     err.Error(),
			CheckedAt: now,
		}
	}

	return parseWindowsPing(normalizeNewlines(string(out)), now)
}

func parseWindowsInterfaces(text string) []Adapter {
	blocks := splitBlocks(text)
	adapters := make([]Adapter, 0, len(blocks))

	for _, block := range blocks {
		fields := parseColonBlock(block)
		name := fields["Name"]
		if name == "" {
			continue
		}

		adapter := Adapter{
			Name:             name,
			Description:      fields["Description"],
			GUID:             fields["GUID"],
			PhysicalAddress:  fields["Physical address"],
			State:            strings.ToLower(fields["State"]),
			SSID:             fields["SSID"],
			BSSID:            fields["BSSID"],
			Profile:          fields["Profile"],
			Authentication:   fields["Authentication"],
			Cipher:           fields["Cipher"],
			RadioType:        fields["Radio type"],
			Band:             fields["Band"],
			Channel:          fields["Channel"],
			SignalText:       fields["Signal"],
			ReceiveRateMbps:  firstToken(fields["Receive rate (Mbps)"]),
			TransmitRateMbps: firstToken(fields["Transmit rate (Mbps)"]),
		}
		adapter.SignalPercent = parsePercent(adapter.SignalText)
		if adapter.State == "" {
			adapter.State = "disconnected"
		}
		adapters = append(adapters, adapter)
	}

	return adapters
}

func parseWindowsDrivers(text string) map[string][]string {
	result := map[string][]string{}
	var current string
	var bands []string
	capture := false

	flush := func() {
		if current != "" && len(bands) > 0 {
			result[current] = append([]string(nil), bands...)
		}
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if matches := rxInterfaceName.FindStringSubmatch(trimmed); len(matches) == 2 {
			flush()
			current = strings.TrimSpace(matches[1])
			bands = nil
			capture = false
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "Number of supported bands"):
			capture = true
		case capture && strings.Contains(trimmed, "GHz"):
			if parts := strings.Fields(trimmed); len(parts) >= 2 {
				bands = append(bands, parts[0]+" "+parts[1])
			}
		case capture && trimmed != "" && !strings.Contains(trimmed, "GHz"):
			capture = false
		}
	}
	flush()
	return result
}

type ipBlock struct {
	IPv4      string
	IPv6      string
	Gateway   string
	DNSSuffix string
}

func parseWindowsIPConfig(text string) map[string]ipBlock {
	result := map[string]ipBlock{}
	var current string
	var block []string

	flush := func() {
		if current == "" {
			return
		}
		raw := strings.Join(block, "\n")
		entry := ipBlock{}
		if match := rxIPv4.FindStringSubmatch(raw); len(match) == 2 {
			entry.IPv4 = match[1]
		}
		if match := rxIPv6.FindStringSubmatch(raw); len(match) == 2 {
			entry.IPv6 = match[1]
		}
		if match := rxGateway.FindStringSubmatch(raw); len(match) == 2 {
			entry.Gateway = match[1]
		}
		if match := rxSuffix.FindStringSubmatch(raw); len(match) == 2 {
			entry.DNSSuffix = strings.TrimSpace(match[1])
		}
		result[current] = entry
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if name := extractAdapterName(trimmed); name != "" {
			flush()
			current = name
			block = nil
			continue
		}
		if current != "" {
			block = append(block, line)
		}
	}
	flush()
	return result
}

func parseWindowsPing(text string, now time.Time) PingResult {
	result := PingResult{
		CheckedAt: now,
		Sent:      1,
	}
	if match := rxLatency.FindStringSubmatch(text); len(match) == 2 {
		if latency, err := strconv.Atoi(match[1]); err == nil {
			result.LatencyMS = latency
			result.Reachable = true
		}
	}
	if strings.Contains(text, "time<1ms") {
		result.LatencyMS = 0
		result.Reachable = true
	}
	if result.Reachable {
		result.Received = 1
	}
	if match := rxLoss.FindStringSubmatch(text); len(match) == 2 {
		if loss, err := strconv.Atoi(match[1]); err == nil {
			result.PacketLoss = float64(loss)
		}
	} else if !result.Reachable {
		result.PacketLoss = 100
	}
	if !result.Reachable && result.Error == "" {
		result.Error = "request timed out"
	}
	return result
}

func splitBlocks(text string) []string {
	parts := strings.Split(text, "\n\n")
	blocks := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			blocks = append(blocks, trimmed)
		}
	}
	return blocks
}

// parseColonBlock parses "Key   : Value" lines, using SplitN so values
// containing colons (IPv6 addresses, GUIDs, etc.) are preserved intact.
func parseColonBlock(block string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(block, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		// Only store the first occurrence; netsh repeats some keys for
		// secondary values (e.g. multiple gateways) which we don't need.
		if _, exists := fields[key]; !exists {
			fields[key] = value
		}
	}
	return fields
}

func extractAdapterName(header string) string {
	header = strings.TrimSuffix(header, ":")
	prefixes := []string{
		"Wireless LAN adapter ",
		"Ethernet adapter ",
		"Unknown adapter ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(header, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(header, prefix))
		}
	}
	return ""
}

func normalizeNewlines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func parsePercent(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "%"))
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func firstToken(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}