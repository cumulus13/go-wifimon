package notify

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	gntp "github.com/cumulus13/go-gntp"
	"github.com/cumulus13/go-wifimon/internal/wifi"
)

const notificationType = "wifi-status"

var (
	client     *gntp.Client
	clientErr  error
	clientOnce sync.Once
)

func Send(current wifi.Adapter, previous wifi.Adapter) {
	c, err := getClient()
	if err != nil {
		return
	}

	title, body, priority, sticky := buildMessage(current, previous)
	options := gntp.NewNotifyOptions().
		WithPriority(priority).
		WithSticky(sticky)

	_ = c.NotifyWithOptions(notificationType, title, body, options)
}

func getClient() (*gntp.Client, error) {
	clientOnce.Do(func() {
		icon, err := loadIcon()
		if err != nil {
			clientErr = err
			return
		}

		client = gntp.NewClient("WiFiMon").
			WithHost("localhost").
			WithPort(23053).
			WithIconMode(gntp.IconModeBinary).
			WithIcon(icon)

		notification := gntp.NewNotificationType(notificationType).
			WithDisplayName("Wi-Fi Status").
			WithEnabled(true).
			WithIcon(icon)

		if err := client.Register([]*gntp.NotificationType{notification}); err != nil {
			clientErr = err
			client = nil
		}
	})

	return client, clientErr
}

func loadIcon() (*gntp.Resource, error) {
	candidates := []string{
		filepath.Join("assets", "wifimon.png"),
		filepath.Join(filepath.Dir(os.Args[0]), "assets", "wifimon.png"),
		filepath.Join(filepath.Dir(os.Args[0]), "wifimon.png"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return gntp.LoadResource(candidate)
		}
	}

	return nil, errors.New("notification icon not found")
}

func buildMessage(current wifi.Adapter, previous wifi.Adapter) (title, body string, priority int, sticky bool) {
	switch {
	case current.State == "connected" && previous.State != "connected":
		title = fmt.Sprintf("Connected to %s", current.DisplaySSID())
		body = fmt.Sprintf("Adapter: %s\nSignal: %s (%d%%)\nBand: %s\nGateway: %s", current.Name, current.SignalBars(), current.SignalPercent, valueOrDash(current.Band), valueOrDash(current.Gateway))
		priority = 1
	case current.State != "connected" && previous.State == "connected":
		title = fmt.Sprintf("Disconnected from %s", previous.DisplaySSID())
		body = fmt.Sprintf("Adapter: %s\nState: %s", current.Name, current.DisplayState())
		priority = 2
		sticky = true
	case current.DisplaySSID() != previous.DisplaySSID():
		title = fmt.Sprintf("Switched to %s", current.DisplaySSID())
		body = fmt.Sprintf("Adapter: %s\nSignal: %s (%d%%)\nBand: %s", current.Name, current.SignalBars(), current.SignalPercent, valueOrDash(current.Band))
	default:
		title = fmt.Sprintf("%s status changed", current.Name)
		body = fmt.Sprintf("State: %s\nSSID: %s\nSignal: %s", current.DisplayState(), current.DisplaySSID(), current.SignalText)
	}
	return title, body, priority, sticky
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
