package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cumulus13/go-wifimon/internal/config"
	"github.com/cumulus13/go-wifimon/internal/wifi"
)

const (
	refreshInterval = 2 * time.Second
	historyLimit    = 48
)

type adapterHistory struct {
	Signal  []int
	Latency []int
	Loss    []float64
	Last    wifi.PingResult
}

type Model struct {
	Info         wifi.Info
	Width        int
	Height       int
	Selected     int
	ManualSelect bool
	LastNotified map[string]wifi.Adapter
	Histories    map[string]*adapterHistory
	Status       string
	Styles       Styles
}

type TickMsg struct{}

type RefreshMsg struct {
	Info wifi.Info
}

type PingMsg struct {
	Key    string
	Result wifi.PingResult
}

// NewModel loads the theme from the first config file found (or uses
// built-in defaults) and returns an initialised Model.
func NewModel() Model {
	return NewModelWithConfig("")
}

// NewModelWithConfig creates a Model using an explicit config file path.
// Pass "" to use the default search order.
func NewModelWithConfig(configPath string) Model {
	theme := config.Load(configPath)
	return Model{
		LastNotified: map[string]wifi.Adapter{},
		Histories:    map[string]*adapterHistory{},
		Status:       "Loading Wi-Fi status...",
		Styles:       NewStyles(theme),
	}
}

func Tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}

func refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return RefreshMsg{Info: wifi.Get()}
	}
}

func pingCmd(key, gateway string) tea.Cmd {
	return func() tea.Msg {
		return PingMsg{
			Key:    key,
			Result: wifi.PingGateway(gateway),
		}
	}
}