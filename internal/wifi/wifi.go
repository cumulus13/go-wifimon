package wifi

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"time"
)

type Info struct {
	Timestamp   time.Time
	HasWifi     bool
	HasInternet bool
	ActiveIndex int
	Adapters    []Adapter
	Error       string
}

type Adapter struct {
	Name             string
	Description      string
	GUID             string
	PhysicalAddress  string
	State            string
	SSID             string
	BSSID            string
	Profile          string
	Authentication   string
	Cipher           string
	RadioType        string
	Band             string
	Channel          string
	SignalText       string
	SignalPercent    int
	ReceiveRateMbps  string
	TransmitRateMbps string
	IPv4             string
	IPv6             string
	Gateway          string
	DNSSuffix        string
	SupportedBands   []string
}

type PingResult struct {
	Reachable  bool
	LatencyMS  int
	PacketLoss float64
	Sent       int
	Received   int
	Error      string
	CheckedAt  time.Time
}

// normalizeNewlines replaces \r\n and lone \r with \n.
// Shared by linux.go and darwin.go which parse CLI output.
func normalizeNewlines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func Get() Info {
	switch runtime.GOOS {
	case "windows":
		return getWindows()
	case "linux":
		return getLinux()
	case "darwin":
		return getDarwin()
	default:
		return Info{
			Timestamp: time.Now(),
			Error:     fmt.Sprintf("unsupported operating system: %s", runtime.GOOS),
		}
	}
}

func PingGateway(gateway string) PingResult {
	switch runtime.GOOS {
	case "windows":
		return pingGatewayWindows(gateway)
	case "linux":
		return pingGatewayLinux(gateway)
	case "darwin":
		return pingGatewayDarwin(gateway)
	default:
		return PingResult{
			Error:     fmt.Sprintf("unsupported operating system: %s", runtime.GOOS),
			CheckedAt: time.Now(),
		}
	}
}

func (i Info) ActiveAdapter() (Adapter, bool) {
	if len(i.Adapters) == 0 {
		return Adapter{}, false
	}
	if i.ActiveIndex < 0 || i.ActiveIndex >= len(i.Adapters) {
		return i.Adapters[0], true
	}
	return i.Adapters[i.ActiveIndex], true
}

func (a Adapter) Connected() bool {
	return a.State == "connected"
}

func (a Adapter) DisplayState() string {
	if a.State == "" {
		return "unknown"
	}
	return a.State
}

func (a Adapter) DisplaySSID() string {
	if a.SSID == "" {
		return "-"
	}
	return a.SSID
}

func (a Adapter) DisplayBand() string {
	if a.Band != "" {
		return a.Band
	}
	if len(a.SupportedBands) > 0 {
		return joinOrDash(a.SupportedBands)
	}
	return "-"
}

func (a Adapter) SignalBars() string {
	switch {
	case a.SignalPercent >= 85:
		return "▂▄▆█"
	case a.SignalPercent >= 65:
		return "▂▄▆▇"
	case a.SignalPercent >= 45:
		return "▂▄▅▆"
	case a.SignalPercent >= 20:
		return "▂▃▄▅"
	case a.SignalPercent > 0:
		return "▁▂▃▄"
	default:
		return "----"
	}
}

func (p PingResult) LossPercentText() string {
	if p.Sent == 0 {
		return "-"
	}
	return formatFloat(p.PacketLoss) + "%"
}

func formatFloat(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.05 {
		return intString(int(math.Round(v)))
	}
	return stringsTrimZero(v)
}

func intString(v int) string {
	return fmt.Sprintf("%d", v)
}

func stringsTrimZero(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	if len(s) >= 2 && s[len(s)-2:] == ".0" {
		return s[:len(s)-2]
	}
	return s
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}