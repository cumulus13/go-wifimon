package wifi

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func getLinux() Info {
	now := time.Now()
	out, err := exec.Command("nmcli", "-t", "-f", "DEVICE,STATE,CONNECTION", "device").CombinedOutput()
	if err != nil {
		return Info{
			Timestamp: now,
			Error:     strings.TrimSpace(string(out)),
		}
	}

	interfaces, _ := exec.Command("nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL,DEVICE", "dev", "wifi").CombinedOutput()
	deviceMap := map[string]Adapter{}
	for _, line := range strings.Split(normalizeNewlines(string(interfaces)), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}
		device := parts[3]
		signal, _ := strconv.Atoi(parts[2])
		deviceMap[device] = Adapter{
			Name:          device,
			SSID:          parts[1],
			SignalText:    parts[2] + "%",
			SignalPercent: signal,
			State:         "disconnected",
		}
		if parts[0] == "yes" {
			deviceMap[device] = Adapter{
				Name:          device,
				SSID:          parts[1],
				SignalText:    parts[2] + "%",
				SignalPercent: signal,
				State:         "connected",
			}
		}
	}

	var adapters []Adapter
	activeIndex := 0
	for _, line := range strings.Split(normalizeNewlines(string(out)), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		device := parts[0]
		state := parts[1]
		connection := parts[2]
		if state == "unavailable" {
			continue
		}

		adapter := deviceMap[device]
		adapter.Name = device
		if adapter.State == "" {
			adapter.State = state
		}
		if adapter.SSID == "" && connection != "--" {
			adapter.SSID = connection
		}
		adapters = append(adapters, adapter)
		if adapter.Connected() {
			activeIndex = len(adapters) - 1
		}
	}

	return Info{
		Timestamp:   now,
		HasWifi:     len(adapters) > 0,
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
	if strings.Contains(text, "1 received") {
		result.Reachable = true
		result.Received = 1
		result.PacketLoss = 0
	}
	if idx := strings.Index(text, "time="); idx >= 0 {
		chunk := text[idx+5:]
		if end := strings.Index(chunk, " ms"); end >= 0 {
			if latency, err := strconv.ParseFloat(chunk[:end], 64); err == nil {
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
