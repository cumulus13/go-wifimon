package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cumulus13/go-wifimon/internal/wifi"
)

// ── layout constants ────────────────────────────────────────────────────────

const (
	labelColWidth = 13 // "Packet Loss:" padded to this
	labelPrefix   = labelColWidth + 2 // label + ": " + space = 15 chars
	cardChrome    = 4                 // border(1 each side) + padding(1 each side)
)

// ── main View ───────────────────────────────────────────────────────────────

func (m Model) View() string {
	s := m.Styles
	width := m.Width
	if width == 0 {
		width = 100
	}

	header := s.Title.Render("📶 WiFiMon") +
		"  " + s.Subtitle.Render("live Wi-Fi monitor for Windows and beyond")
	status := s.Status.Render(m.Status)

	// ── no adapter ──────────────────────────────────────────────────────────
	if len(m.Info.Adapters) == 0 {
		body := s.Card.Width(width - 8).Render(strings.Join([]string{
			row(s, "Adapter", s.Err.Render("❌ none")),
			row(s, "State",   s.Warn.Render("No Wi-Fi hardware detected")),
			row(s, "Hint",    s.Value.Render("Press r to refresh, q to quit")),
			row(s, "Updated", s.Value.Render(timeString(m.Info.Timestamp))),
		}, "\n"))
		return s.App.Width(width).Render(header + "\n\n" + body)
	}

	// ── layout geometry ─────────────────────────────────────────────────────
	contentWidth := width - 4
	if contentWidth < 40 {
		contentWidth = width
	}
	narrow := width < 92

	cardWidth := maxInt(28, (contentWidth/2)-2)
	if narrow {
		cardWidth = maxInt(24, width-8)
	}
	graphWidth := cardWidth - cardChrome - labelPrefix
	if graphWidth < 4 {
		graphWidth = 4
	}

	// ── selected adapter data ───────────────────────────────────────────────
	selected, _ := m.currentAdapter()
	key := adapterKey(selected, m.Selected)
	history := m.historyFor(key)

	adapterTabs := renderTabs(m, width-8)

	// ── graphs ──────────────────────────────────────────────────────────────
	signalGraph  := sparkline(history.Signal,  "▁▂▃▄▅▆▇█", graphWidth)
	latencyGraph := sparkline(history.Latency, "▁▂▃▄▅▆▇█", graphWidth)
	lossGraph    := sparklineFloat(history.Loss, "▁▂▃▄▅▆▇█", graphWidth)

	// Colour each graph bar individually.
	signalGraph  = colorSignalGraph(history.Signal,  signalGraph,  s, graphWidth)
	latencyGraph = colorLatencyGraph(history.Latency, latencyGraph, s, graphWidth, history.Last.Reachable)
	lossGraph    = colorLossGraph(history.Loss, lossGraph, s, graphWidth)

	// ── left card ───────────────────────────────────────────────────────────
	adapterVal := fmt.Sprintf("%s (%d/%d)",
		selected.Name, m.Selected+1, len(m.Info.Adapters))

	mainCard := strings.Join([]string{
		row(s, "Adapter",      s.Value.Render(adapterVal)),
		row(s, "SSID",         s.Value.Render(selected.DisplaySSID())),
		row(s, "State",        stateBadge(s, selected.DisplayState())),
		row(s, "Signal",       signalValue(s, selected)),
		row(s, "Band",         s.Value.Render(valueOrDash(selected.DisplayBand()))),
		row(s, "Radio",        s.Value.Render(valueOrDash(selected.RadioType))),
		row(s, "Channel",      s.Value.Render(valueOrDash(selected.Channel))),
		row(s, "Speed",        speedValue(s, selected)),
		row(s, "IP / Gateway", ipGatewayValue(s, selected)),
		row(s, "Auth",         s.Value.Render(joinCompact(selected.Authentication, selected.Cipher))),
		row(s, "Signal Graph", signalGraph),
	}, "\n")

	// ── right card ──────────────────────────────────────────────────────────
	latencyText, lossText := pingValues(s, history)

	networkCard := strings.Join([]string{
		row(s, "Gateway",      s.Value.Render(valueOrDash(selected.Gateway))),
		row(s, "Latency",      latencyText),
		row(s, "Packet Loss",  lossText),
		row(s, "Latency Graph",latencyGraph),
		row(s, "Loss Graph",   lossGraph),
		row(s, "IPv6",         s.Value.Render(valueOrDash(selected.IPv6))),
		row(s, "Profile",      s.Value.Render(valueOrDash(selected.Profile))),
		row(s, "BSSID",        s.Value.Render(valueOrDash(selected.BSSID))),
		row(s, "Updated",      s.Value.Render(timeString(m.Info.Timestamp))),
	}, "\n")

	footer := s.Footer.Render("Shortcuts: q quit • r refresh • ←/→ switch adapter")

	// ── assemble ─────────────────────────────────────────────────────────────
	var content string
	if narrow {
		content = lipgloss.JoinVertical(lipgloss.Left,
			s.Card.Width(cardWidth).Render(mainCard),
			s.Card.Width(cardWidth).Render(networkCard),
		)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top,
			s.Card.Width(cardWidth).Render(mainCard),
			s.Card.Width(cardWidth).Render(networkCard),
		)
	}

	return s.App.Width(width).Render(strings.Join([]string{
		header,
		status,
		adapterTabs,
		content,
		footer,
	}, "\n\n"))
}

// ── row builder ─────────────────────────────────────────────────────────────

// row formats a label+value line.
// The label is always coloured with s.Label; the caller is responsible for
// colouring the value string before passing it in.
func row(s Styles, label, value string) string {
	l := s.Label.Render(fmt.Sprintf("%-*s", labelColWidth, label+":"))
	return l + " " + value
}

// ── value renderers ─────────────────────────────────────────────────────────

func stateBadge(s Styles, state string) string {
	switch state {
	case "connected":
		return s.Ok.Render("🟢 connected")
	case "disconnected":
		return s.Err.Render("🔴 disconnected")
	default:
		return s.Warn.Render("🟡 " + state)
	}
}

func signalValue(s Styles, a wifi.Adapter) string {
	st := s.SignalStyle(a.SignalPercent)
	bars := st.Render(a.SignalBars())
	txt  := st.Render(valueOrDash(a.SignalText))
	return bars + " " + txt
}

func speedValue(s Styles, a wifi.Adapter) string {
	rx := valueOrDash(a.ReceiveRateMbps)
	tx := valueOrDash(a.TransmitRateMbps)
	return s.Value.Render(rx+"↓") + s.Subtitle.Render(" / ") + s.Value.Render(tx+"↑ Mbps")
}

func ipGatewayValue(s Styles, a wifi.Adapter) string {
	if a.IPv4 != "" && a.Gateway != "" {
		return s.Value.Render(a.IPv4) +
			s.Subtitle.Render("  →  ") +
			s.Value.Render(a.Gateway)
	}
	return s.Value.Render(joinCompact(a.IPv4, a.Gateway))
}

func pingValues(s Styles, history *adapterHistory) (latencyText, lossText string) {
	if !history.Last.CheckedAt.After(time.Time{}) {
		return s.Subtitle.Render("offline"), s.Subtitle.Render("-")
	}

	lst := s.LatencyStyle(history.Last.LatencyMS, history.Last.Reachable)
	if history.Last.Reachable {
		latencyText = lst.Render(fmt.Sprintf("%d ms", history.Last.LatencyMS))
	} else {
		msg := history.Last.Error
		if msg == "" {
			msg = "unreachable"
		}
		latencyText = lst.Render(msg)
	}

	lossText = s.LossStyle(history.Last.PacketLoss).
		Render(history.Last.LossPercentText())

	return latencyText, lossText
}

// ── tabs ─────────────────────────────────────────────────────────────────────

func renderTabs(m Model, width int) string {
	s := m.Styles
	tabs := make([]string, 0, len(m.Info.Adapters))
	for idx, adapter := range m.Info.Adapters {
		label := fmt.Sprintf(" %s %s ", stateEmoji(adapter.DisplayState()), trimLabel(adapter.Name, 20))
		var style lipgloss.Style
		if idx == m.Selected {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color(s.Theme.TabActiveFG)).
				Background(lipgloss.Color(s.Theme.TabActiveBG)).
				Bold(true)
		} else {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color(s.Theme.TabFG)).
				Background(lipgloss.Color(s.Theme.TabBG))
		}
		tabs = append(tabs, style.Render(label))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return lipgloss.NewStyle().MaxWidth(width).Render(row)
}

// ── sparkline helpers ────────────────────────────────────────────────────────

// sparkline renders a bar graph using block characters.
// maxWidth caps the number of bars shown (keeping the most recent samples).
func sparkline(values []int, charset string, maxWidth int) string {
	if len(values) == 0 {
		return "no data"
	}
	if maxWidth > 0 && len(values) > maxWidth {
		values = values[len(values)-maxWidth:]
	}
	maxVal := 1
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	runes := []rune(charset)
	var b strings.Builder
	for _, v := range values {
		scaled := int(math.Round(float64(v) / float64(maxVal) * float64(len(runes)-1)))
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

func sparklineFloat(values []float64, charset string, maxWidth int) string {
	if len(values) == 0 {
		return "no data"
	}
	ints := make([]int, 0, len(values))
	for _, v := range values {
		ints = append(ints, int(math.Round(v)))
	}
	return sparkline(ints, charset, maxWidth)
}

// colorSignalGraph recolours each bar rune individually by its signal value.
func colorSignalGraph(values []int, raw string, s Styles, maxWidth int) string {
	if raw == "no data" {
		return s.Subtitle.Render(raw)
	}
	if maxWidth > 0 && len(values) > maxWidth {
		values = values[len(values)-maxWidth:]
	}
	runes := []rune(raw)
	if len(runes) != len(values) {
		return raw // safety: lengths mismatch, return uncoloured
	}
	var b strings.Builder
	for i, r := range runes {
		b.WriteString(s.SignalStyle(values[i]).Render(string(r)))
	}
	return b.String()
}

// colorLatencyGraph recolours each bar by its latency value.
func colorLatencyGraph(values []int, raw string, s Styles, maxWidth int, reachable bool) string {
	if raw == "no data" {
		return s.Subtitle.Render(raw)
	}
	if maxWidth > 0 && len(values) > maxWidth {
		values = values[len(values)-maxWidth:]
	}
	runes := []rune(raw)
	if len(runes) != len(values) {
		return raw
	}
	var b strings.Builder
	for i, r := range runes {
		// A latency of 0 means not-yet-measured, treat as unreachable.
		reach := reachable && values[i] > 0
		b.WriteString(s.LatencyStyle(values[i], reach).Render(string(r)))
	}
	return b.String()
}

// colorLossGraph recolours each bar by its loss value.
func colorLossGraph(values []float64, raw string, s Styles, maxWidth int) string {
	if raw == "no data" {
		return s.Subtitle.Render(raw)
	}
	if maxWidth > 0 && len(values) > maxWidth {
		values = values[len(values)-maxWidth:]
	}
	runes := []rune(raw)
	if len(runes) != len(values) {
		return raw
	}
	var b strings.Builder
	for i, r := range runes {
		b.WriteString(s.LossStyle(values[i]).Render(string(r)))
	}
	return b.String()
}

// ── misc helpers ─────────────────────────────────────────────────────────────

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