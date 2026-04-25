package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cumulus13/go-wifimon/internal/notify"
	"github.com/cumulus13/go-wifimon/internal/wifi"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(), Tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case TickMsg:
		return m, tea.Batch(refreshCmd(), Tick())

	case RefreshMsg:
		m.Info = msg.Info
		if len(m.Info.Adapters) == 0 {
			m.Selected = 0
			if !m.Info.HasWifi {
				m.Status = "No Wi-Fi adapter detected."
			} else {
				m.Status = "No active Wi-Fi data."
			}
			return m, nil
		}

		if !m.ManualSelect || m.Selected >= len(m.Info.Adapters) {
			m.Selected = m.Info.ActiveIndex
		}
		if m.Selected < 0 {
			m.Selected = 0
		}

		commands := make([]tea.Cmd, 0, len(m.Info.Adapters))
		for idx, adapter := range m.Info.Adapters {
			key := adapterKey(adapter, idx)
			history := m.historyFor(key)
			history.Signal = appendInt(history.Signal, adapter.SignalPercent)

			if previous, ok := m.LastNotified[key]; ok && changeForNotification(previous, adapter) {
				notify.Send(adapter, previous)
			}
			m.LastNotified[key] = adapter

			if adapter.Connected() && adapter.Gateway != "" {
				commands = append(commands, pingCmd(key, adapter.Gateway))
			}
		}

		if selected, ok := m.currentAdapter(); ok {
			m.Status = fmt.Sprintf("Watching %s on %s", selected.DisplaySSID(), selected.Name)
			if !selected.Connected() {
				m.Status = fmt.Sprintf("%s is %s", selected.Name, selected.DisplayState())
			}
		}

		return m, tea.Batch(commands...)

	case PingMsg:
		history := m.historyFor(msg.Key)
		history.Last = msg.Result
		history.Latency = appendInt(history.Latency, msg.Result.LatencyMS)
		history.Loss = appendFloat(history.Loss, msg.Result.PacketLoss)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.Status = "Refreshing Wi-Fi status..."
			return m, refreshCmd()
		case "left", "h":
			if len(m.Info.Adapters) > 0 {
				m.ManualSelect = true
				m.Selected--
				if m.Selected < 0 {
					m.Selected = len(m.Info.Adapters) - 1
				}
			}
			return m, nil
		case "right", "l", "tab":
			if len(m.Info.Adapters) > 0 {
				m.ManualSelect = true
				m.Selected = (m.Selected + 1) % len(m.Info.Adapters)
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) historyFor(key string) *adapterHistory {
	history, ok := m.Histories[key]
	if ok {
		return history
	}
	history = &adapterHistory{}
	m.Histories[key] = history
	return history
}

func (m Model) currentAdapter() (wifi.Adapter, bool) {
	if len(m.Info.Adapters) == 0 {
		return wifi.Adapter{}, false
	}
	if m.Selected < 0 || m.Selected >= len(m.Info.Adapters) {
		return m.Info.Adapters[m.Info.ActiveIndex], true
	}
	return m.Info.Adapters[m.Selected], true
}

func adapterKey(adapter wifi.Adapter, idx int) string {
	if adapter.GUID != "" {
		return adapter.GUID
	}
	if adapter.Name != "" {
		return adapter.Name
	}
	return fmt.Sprintf("adapter-%d", idx)
}

func changeForNotification(previous, current wifi.Adapter) bool {
	return previous.State != current.State || previous.SSID != current.SSID
}

func appendInt(values []int, next int) []int {
	values = append(values, next)
	if len(values) > historyLimit {
		values = values[len(values)-historyLimit:]
	}
	return values
}

func appendFloat(values []float64, next float64) []float64 {
	values = append(values, next)
	if len(values) > historyLimit {
		values = values[len(values)-historyLimit:]
	}
	return values
}
