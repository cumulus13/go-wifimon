package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cumulus13/go-wifimon/internal/wifi"
)

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))
	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)
	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)
)

func (m Model) View() string {
	width := m.Width
	if width == 0 {
		width = 100
	}

	header := titleStyle.Render("📶 WiFiMon") + "  " + dimStyle.Render("live Wi-Fi monitor for Windows and beyond")
	status := dimStyle.Render(m.Status)

	if len(m.Info.Adapters) == 0 {
		body := cardStyle.Width(width - 8).Render(strings.Join([]string{
			statusLine("Adapter", "❌ none"),
			statusLine("State", "No Wi-Fi hardware detected"),
			statusLine("Hint", "Press r to refresh, q to quit"),
			statusLine("Updated", timeString(m.Info.Timestamp)),
		}, "\n"))
		return appStyle.Width(width).Render(header + "\n\n" + body)
	}

	selected, _ := m.currentAdapter()
	key := adapterKey(selected, m.Selected)
	history := m.historyFor(key)

	adapterTabs := renderTabs(m.Info.Adapters, m.Selected, width-8)
	signalGraph := sparkline(history.Signal, "▁▂▃▄▅▆▇█")
	latencyGraph := sparkline(history.Latency, "▁▂▃▄▅▆▇█")
	lossGraph := sparklineFloat(history.Loss, "▁▂▃▄▅▆▇█")

	mainCard := strings.Join([]string{
		statusLine("Adapter", fmt.Sprintf("%s (%d/%d)", selected.Name, m.Selected+1, len(m.Info.Adapters))),
		statusLine("SSID", selected.DisplaySSID()),
		statusLine("State", stateBadge(selected.DisplayState())),
		statusLine("Signal", fmt.Sprintf("%s %s", selected.SignalBars(), valueOrDash(selected.SignalText))),
		statusLine("Band", valueOrDash(selected.DisplayBand())),
		statusLine("Radio", valueOrDash(selected.RadioType)),
		statusLine("Channel", valueOrDash(selected.Channel)),
		statusLine("Speed", valueOrDash(selected.ReceiveRateMbps)+"↓ / "+valueOrDash(selected.TransmitRateMbps)+"↑ Mbps"),
		statusLine("IP / Gateway", joinCompact(selected.IPv4, selected.Gateway)),
		statusLine("Auth", joinCompact(selected.Authentication, selected.Cipher)),
		statusLine("Signal Graph", signalGraph),
	}, "\n")

	latencyText := "offline"
	lossText := "-"
	if history.Last.CheckedAt.After(time.Time{}) {
		if history.Last.Reachable {
			latencyText = fmt.Sprintf("%d ms", history.Last.LatencyMS)
		} else if history.Last.Error != "" {
			latencyText = history.Last.Error
		}
		lossText = history.Last.LossPercentText()
	}

	networkCard := strings.Join([]string{
		statusLine("Gateway", valueOrDash(selected.Gateway)),
		statusLine("Latency", latencyText),
		statusLine("Packet Loss", lossText),
		statusLine("Latency Graph", latencyGraph),
		statusLine("Loss Graph", lossGraph),
		statusLine("IPv6", valueOrDash(selected.IPv6)),
		statusLine("Profile", valueOrDash(selected.Profile)),
		statusLine("BSSID", valueOrDash(selected.BSSID)),
		statusLine("Updated", timeString(m.Info.Timestamp)),
	}, "\n")

	footer := dimStyle.Render("Shortcuts: q quit • r refresh • ←/→ switch adapter")

	contentWidth := width - 4
	if contentWidth < 40 {
		contentWidth = width
	}
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		cardStyle.Width(maxInt(28, (contentWidth/2)-2)).Render(mainCard),
		cardStyle.Width(maxInt(28, (contentWidth/2)-2)).Render(networkCard),
	)
	if width < 92 {
		content = lipgloss.JoinVertical(lipgloss.Left,
			cardStyle.Width(maxInt(24, width-8)).Render(mainCard),
			cardStyle.Width(maxInt(24, width-8)).Render(networkCard),
		)
	}

	return appStyle.Width(width).Render(strings.Join([]string{
		header,
		status,
		adapterTabs,
		content,
		footer,
	}, "\n\n"))
}

func renderTabs(adapters []wifi.Adapter, selected int, width int) string {
	tabs := make([]string, 0, len(adapters))
	for idx, adapter := range adapters {
		label := fmt.Sprintf(" %s %s ", stateEmoji(adapter.DisplayState()), trimLabel(adapter.Name, 20))
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238"))
		if idx == selected {
			style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("31")).Bold(true)
		}
		tabs = append(tabs, style.Render(label))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return lipgloss.NewStyle().MaxWidth(width).Render(row)
}

func statusLine(label, value string) string {
	return fmt.Sprintf("%-13s %s", label+":", value)
}

func stateBadge(state string) string {
	switch state {
	case "connected":
		return okStyle.Render("🟢 connected")
	case "disconnected":
		return errStyle.Render("🔴 disconnected")
	default:
		return warnStyle.Render("🟡 " + state)
	}
}

func stateEmoji(state string) string {
	switch state {
	case "connected":
		return "🟢"
	case "disconnected":
		return "🔴"
	default:
		return "🟡"
	}
}

func sparkline(values []int, charset string) string {
	if len(values) == 0 {
		return dimStyle.Render("no samples yet")
	}
	maxVal := 1
	for _, value := range values {
		if value > maxVal {
			maxVal = value
		}
	}
	runes := []rune(charset)
	var b strings.Builder
	for _, value := range values {
		scaled := int(math.Round(float64(value) / float64(maxVal) * float64(len(runes)-1)))
		if scaled < 0 {
			scaled = 0
		}
		if scaled >= len(runes) {
			scaled = len(runes) - 1
		}
		b.WriteRune(runes[scaled])
	}
	return b.String()
}

func sparklineFloat(values []float64, charset string) string {
	if len(values) == 0 {
		return dimStyle.Render("no samples yet")
	}
	ints := make([]int, 0, len(values))
	for _, value := range values {
		ints = append(ints, int(math.Round(value)))
	}
	return sparkline(ints, charset)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func joinCompact(left, right string) string {
	switch {
	case left != "" && right != "":
		return left + "  →  " + right
	case left != "":
		return left
	case right != "":
		return right
	default:
		return "-"
	}
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05")
}

func trimLabel(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:limit]
	}
	return value[:limit-1] + "…"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
