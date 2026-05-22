package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/cumulus13/go-wifimon/internal/config"
)

// Styles holds every lipgloss.Style derived from a Theme.
// A single Styles value is embedded in Model and rebuilt whenever the
// theme is reloaded (currently only at startup).
type Styles struct {
	// top-level chrome
	App      lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Footer   lipgloss.Style
	Status   lipgloss.Style

	// card
	Card lipgloss.Style

	// row anatomy
	Label lipgloss.Style
	Value lipgloss.Style

	// state badges
	Ok   lipgloss.Style
	Warn lipgloss.Style
	Err  lipgloss.Style

	// per-quality signal styles (index 0=none … 4=great)
	Signal [5]lipgloss.Style

	// per-quality latency styles (index 0=bad … 3=great)
	Latency [4]lipgloss.Style

	// per-quality loss styles
	Loss [3]lipgloss.Style

	// raw theme kept for helper methods (SignalColor etc.)
	Theme config.Theme
}

// NewStyles builds a Styles value from the given theme.
func NewStyles(t config.Theme) Styles {
	c := func(col string) lipgloss.Color { return lipgloss.Color(col) }

	s := Styles{Theme: t}

	s.App = lipgloss.NewStyle().Padding(1, 2)

	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(c(t.Title))

	s.Subtitle = lipgloss.NewStyle().
		Foreground(c(t.Subtitle))

	s.Footer = lipgloss.NewStyle().
		Foreground(c(t.Footer))

	s.Status = lipgloss.NewStyle().
		Foreground(c(t.Status))

	s.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(t.CardBorder)).
		Padding(0, 1)

	s.Label = lipgloss.NewStyle().
		Foreground(c(t.Label))

	s.Value = lipgloss.NewStyle().
		Foreground(c(t.Value))

	s.Ok = lipgloss.NewStyle().
		Foreground(c(t.Ok)).
		Bold(true)

	s.Warn = lipgloss.NewStyle().
		Foreground(c(t.Warn)).
		Bold(true)

	s.Err = lipgloss.NewStyle().
		Foreground(c(t.Err)).
		Bold(true)

	// Signal quality: index maps to tier (0=none, 1=poor, 2=fair, 3=good, 4=great)
	s.Signal[0] = lipgloss.NewStyle().Foreground(c(t.SignalNone))
	s.Signal[1] = lipgloss.NewStyle().Foreground(c(t.SignalPoor))
	s.Signal[2] = lipgloss.NewStyle().Foreground(c(t.SignalFair))
	s.Signal[3] = lipgloss.NewStyle().Foreground(c(t.SignalGood))
	s.Signal[4] = lipgloss.NewStyle().Foreground(c(t.SignalGreat))

	// Latency quality: index maps to tier (0=bad, 1=fair, 2=good, 3=great)
	s.Latency[0] = lipgloss.NewStyle().Foreground(c(t.LatencyBad))
	s.Latency[1] = lipgloss.NewStyle().Foreground(c(t.LatencyFair))
	s.Latency[2] = lipgloss.NewStyle().Foreground(c(t.LatencyGood))
	s.Latency[3] = lipgloss.NewStyle().Foreground(c(t.LatencyGreat))

	// Loss quality: index maps to tier (0=bad, 1=fair, 2=great)
	s.Loss[0] = lipgloss.NewStyle().Foreground(c(t.LossBad))
	s.Loss[1] = lipgloss.NewStyle().Foreground(c(t.LossFair))
	s.Loss[2] = lipgloss.NewStyle().Foreground(c(t.LossGreat))

	return s
}

// SignalStyle returns the lipgloss style for a given signal percentage.
func (s Styles) SignalStyle(pct int) lipgloss.Style {
	switch {
	case pct >= 85:
		return s.Signal[4]
	case pct >= 65:
		return s.Signal[3]
	case pct >= 45:
		return s.Signal[2]
	case pct >= 20:
		return s.Signal[1]
	default:
		return s.Signal[0]
	}
}

// LatencyStyle returns the lipgloss style for a given latency in ms.
func (s Styles) LatencyStyle(ms int, reachable bool) lipgloss.Style {
	if !reachable {
		return s.Latency[0]
	}
	switch {
	case ms <= 20:
		return s.Latency[3]
	case ms <= 60:
		return s.Latency[2]
	case ms <= 120:
		return s.Latency[1]
	default:
		return s.Latency[0]
	}
}

// LossStyle returns the lipgloss style for a given packet-loss percentage.
func (s Styles) LossStyle(pct float64) lipgloss.Style {
	switch {
	case pct == 0:
		return s.Loss[2]
	case pct <= 5:
		return s.Loss[1]
	default:
		return s.Loss[0]
	}
}
