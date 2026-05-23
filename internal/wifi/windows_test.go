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

func TestParseWindowsInterfacesGUID(t *testing.T) {
	// GUID contains colons on some platforms - SplitN must preserve them
	raw := `
    Name                   : Wi-Fi
    GUID                   : {12345678-abcd-ef01-2345-67890abcdef0}
    State                  : connected
    SSID                   : TestNet
    Signal                 : 72%
`
	adapters := parseWindowsInterfaces(normalizeNewlines(raw))
	if len(adapters) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(adapters))
	}
	if adapters[0].GUID != "{12345678-abcd-ef01-2345-67890abcdef0}" {
		t.Fatalf("GUID mangled: %q", adapters[0].GUID)
	}
	if adapters[0].SignalPercent != 72 {
		t.Fatalf("expected signal 72, got %d", adapters[0].SignalPercent)
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

// ── ping tests updated for 4-packet output ───────────────────────────────────

func TestParseWindowsPing(t *testing.T) {
	// 4-packet ping, all received → 0% loss
	raw := `
Pinging 192.168.10.1 with 32 bytes of data:
Reply from 192.168.10.1: bytes=32 time=4ms TTL=64
Reply from 192.168.10.1: bytes=32 time=3ms TTL=64
Reply from 192.168.10.1: bytes=32 time=5ms TTL=64
Reply from 192.168.10.1: bytes=32 time=4ms TTL=64

Ping statistics for 192.168.10.1:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss),
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
	if result.Sent != 4 || result.Received != 4 {
		t.Fatalf("expected Sent=4 Received=4, got Sent=%d Received=%d", result.Sent, result.Received)
	}
}

func TestParseWindowsPingPartialLoss(t *testing.T) {
	// 4-packet ping, 1 dropped → 25% loss — the key scenario that was broken
	raw := `
Pinging 192.168.10.1 with 32 bytes of data:
Reply from 192.168.10.1: bytes=32 time=4ms TTL=64
Request timed out.
Reply from 192.168.10.1: bytes=32 time=5ms TTL=64
Reply from 192.168.10.1: bytes=32 time=4ms TTL=64

Ping statistics for 192.168.10.1:
    Packets: Sent = 4, Received = 3, Lost = 1 (25% loss),
`

	result := parseWindowsPing(normalizeNewlines(raw), time.Now())
	if !result.Reachable {
		t.Fatalf("expected reachable (some packets received): %+v", result)
	}
	if result.PacketLoss != 25.0 {
		t.Fatalf("expected 25%% loss, got %f", result.PacketLoss)
	}
	if result.Sent != 4 || result.Received != 3 {
		t.Fatalf("expected Sent=4 Received=3, got Sent=%d Received=%d", result.Sent, result.Received)
	}
}

func TestParseWindowsPingTimeout(t *testing.T) {
	// All 4 packets dropped → 100% loss
	raw := `
Pinging 192.168.10.1 with 32 bytes of data:
Request timed out.
Request timed out.
Request timed out.
Request timed out.

Ping statistics for 192.168.10.1:
    Packets: Sent = 4, Received = 0, Lost = 4 (100% loss),
`
	result := parseWindowsPing(normalizeNewlines(raw), time.Now())
	if result.Reachable {
		t.Fatalf("expected unreachable: %+v", result)
	}
	if result.PacketLoss != 100 {
		t.Fatalf("expected 100%% loss, got %f", result.PacketLoss)
	}
	if result.Error == "" {
		t.Fatal("expected non-empty error string")
	}
}

func TestParseWindowsPingSubMs(t *testing.T) {
	raw := `
Pinging 192.168.1.1 with 32 bytes of data:
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64
Reply from 192.168.1.1: bytes=32 time<1ms TTL=64

Ping statistics for 192.168.1.1:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss),
`
	result := parseWindowsPing(normalizeNewlines(raw), time.Now())
	if !result.Reachable {
		t.Fatalf("expected reachable for sub-ms reply: %+v", result)
	}
	if result.LatencyMS != 0 {
		t.Fatalf("expected latency 0, got %d", result.LatencyMS)
	}
	if result.PacketLoss != 0 {
		t.Fatalf("expected 0%% loss, got %f", result.PacketLoss)
	}
}

func TestDbmToPercent(t *testing.T) {
	cases := []struct{ dbm, want int }{
		{-30, 100},
		{-60, 50},
		{-90, 0},
		{0, 100},
		{-100, 0},
	}
	for _, c := range cases {
		got := dbmToPercent(c.dbm)
		if got != c.want {
			t.Errorf("dbmToPercent(%d) = %d, want %d", c.dbm, got, c.want)
		}
	}
}

func TestFreqToBand(t *testing.T) {
	cases := []struct {
		freq string
		want string
	}{
		{"2412", "2.4 GHz"},
		{"5180", "5 GHz"},
		{"5925", "6 GHz"},
		{"6135", "6 GHz"},
		{"1000", ""},
	}
	for _, c := range cases {
		got := freqToBand(c.freq)
		if got != c.want {
			t.Errorf("freqToBand(%q) = %q, want %q", c.freq, got, c.want)
		}
	}
}
