package wifi

import (
	"testing"
	"time"
)

func TestParseWindowsInterfaces(t *testing.T) {
	raw := `
There are 2 interfaces on the system:

    Name                   : Wi-Fi 3
    Description            : Tenda Wireless USB Adapter
    GUID                   : adapter-1
    Physical address       : c8:3a:35:64:b6:98
    State                  : connected
    SSID                   : TUYUL
    BSSID                  : 5e:19:d5:06:35:d5
    Radio type             : 802.11ac
    Authentication         : WPA2-Personal
    Cipher                 : CCMP
    Band                   : 5 GHz
    Channel                : 36
    Receive rate (Mbps)    : 866
    Transmit rate (Mbps)   : 866
    Signal                 : 88%
    Profile                : TUYUL

    Name                   : Wi-Fi 4
    Description            : Intel AX210
    GUID                   : adapter-2
    Physical address       : 00:11:22:33:44:55
    State                  : disconnected
    Signal                 : 0%
`

	adapters := parseWindowsInterfaces(normalizeNewlines(raw))
	if len(adapters) != 2 {
		t.Fatalf("expected 2 adapters, got %d", len(adapters))
	}
	if adapters[0].Name != "Wi-Fi 3" || adapters[0].SSID != "TUYUL" {
		t.Fatalf("unexpected first adapter: %+v", adapters[0])
	}
	if adapters[0].SignalPercent != 88 {
		t.Fatalf("expected signal 88, got %d", adapters[0].SignalPercent)
	}
	if adapters[1].DisplayState() != "disconnected" {
		t.Fatalf("expected disconnected, got %s", adapters[1].DisplayState())
	}
}

func TestParseWindowsDrivers(t *testing.T) {
	raw := `
Interface name: Wi-Fi 3

    Number of supported bands : 3
                                2.4 GHz [ 0 MHz - 0 MHz]
                                5 GHz   [ 0 MHz - 0 MHz]
                                6 GHz   [ 0 MHz - 0 MHz]
`

	bands := parseWindowsDrivers(normalizeNewlines(raw))
	got := bands["Wi-Fi 3"]
	if len(got) != 3 {
		t.Fatalf("expected 3 bands, got %v", got)
	}
}

func TestParseWindowsIPConfig(t *testing.T) {
	raw := `
Wireless LAN adapter Wi-Fi 3:

   Connection-specific DNS Suffix  . :
   Link-local IPv6 Address . . . . . : fe80::1234%15
   IPv4 Address. . . . . . . . . . . : 192.168.10.9
   Default Gateway . . . . . . . . . : 192.168.10.1
`

	blocks := parseWindowsIPConfig(normalizeNewlines(raw))
	got := blocks["Wi-Fi 3"]
	if got.IPv4 != "192.168.10.9" {
		t.Fatalf("unexpected ipv4: %+v", got)
	}
	if got.Gateway != "192.168.10.1" {
		t.Fatalf("unexpected gateway: %+v", got)
	}
}

func TestParseWindowsPing(t *testing.T) {
	raw := `
Pinging 192.168.10.1 with 32 bytes of data:
Reply from 192.168.10.1: bytes=32 time=4ms TTL=64

Ping statistics for 192.168.10.1:
    Packets: Sent = 1, Received = 1, Lost = 0 (0% loss),
`

	result := parseWindowsPing(normalizeNewlines(raw), time.Now())
	if !result.Reachable {
		t.Fatalf("expected reachable result: %+v", result)
	}
	if result.LatencyMS != 4 {
		t.Fatalf("expected latency 4, got %d", result.LatencyMS)
	}
	if result.PacketLoss != 0 {
		t.Fatalf("expected 0 loss, got %f", result.PacketLoss)
	}
}
