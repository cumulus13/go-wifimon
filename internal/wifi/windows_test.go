//go:build windows

package wifi

import (
	"testing"
)

// ── GUID formatting ───────────────────────────────────────────────────────────

func TestGUIDString(t *testing.T) {
	g := GUID{
		Data1: 0x12345678,
		Data2: 0xABCD,
		Data3: 0xEF01,
		Data4: [8]byte{0x23, 0x45, 0x67, 0x89, 0x0A, 0xBC, 0xDE, 0xF0},
	}
	want := "{12345678-ABCD-EF01-2345-67890ABCDEF0}"
	if got := g.String(); got != want {
		t.Errorf("GUID.String() = %q, want %q", got, want)
	}
}

// ── DOT11_SSID ────────────────────────────────────────────────────────────────

func TestDot11SSIDString(t *testing.T) {
	cases := []struct {
		ssid   dot11SSID
		want   string
	}{
		{dot11SSID{uSSIDLength: 0}, ""},
		{
			dot11SSID{uSSIDLength: 5, ucSSID: [32]byte{'T', 'U', 'Y', 'U', 'L'}},
			"TUYUL",
		},
		{
			dot11SSID{uSSIDLength: 7, ucSSID: [32]byte{'M', 'y', ' ', 'W', 'i', 'F', 'i'}},
			"My WiFi",
		},
	}
	for _, c := range cases {
		if got := c.ssid.String(); got != c.want {
			t.Errorf("dot11SSID.String() = %q, want %q", got, c.want)
		}
	}
}

// ── MAC address ───────────────────────────────────────────────────────────────

func TestDot11MacAddressString(t *testing.T) {
	mac := dot11MacAddress{0x5E, 0x19, 0xD5, 0x06, 0x35, 0xD5}
	want := "5E:19:D5:06:35:D5"
	if got := mac.String(); got != want {
		t.Errorf("dot11MacAddress.String() = %q, want %q", got, want)
	}
}

// ── rssiToPercent ─────────────────────────────────────────────────────────────

func TestRssiToPercent(t *testing.T) {
	cases := []struct {
		rssi int32
		want int
	}{
		{-20, 100}, // above ceiling → clamped to 100
		{-30, 100},
		{-60, 50},
		{-90, 0},
		{-100, 0}, // below floor → clamped to 0
	}
	for _, c := range cases {
		got := rssiToPercent(c.rssi)
		if got != c.want {
			t.Errorf("rssiToPercent(%d) = %d, want %d", c.rssi, got, c.want)
		}
	}
}

// ── authLabel ─────────────────────────────────────────────────────────────────

func TestAuthLabel(t *testing.T) {
	cases := []struct {
		algo uint32
		want string
	}{
		{dot11AuthAlgorithmOpen, "Open"},
		{dot11AuthAlgorithmWPA_PSK, "WPA-Personal"},
		{dot11AuthAlgorithmRSNA_PSK, "WPA2-Personal"},
		{dot11AuthAlgorithmWPA3_SAE, "WPA3-Personal"},
		{99, "Auth(99)"},
	}
	for _, c := range cases {
		got := authLabel(c.algo)
		if got != c.want {
			t.Errorf("authLabel(%d) = %q, want %q", c.algo, got, c.want)
		}
	}
}

// ── cipherLabel ───────────────────────────────────────────────────────────────

func TestCipherLabel(t *testing.T) {
	cases := []struct {
		algo uint32
		want string
	}{
		{dot11CipherAlgorithmNone, "None"},
		{dot11CipherAlgorithmTKIP, "TKIP"},
		{dot11CipherAlgorithmCCMP, "CCMP"},
		{dot11CipherAlgorithmGCMP, "GCMP"},
		{dot11CipherAlgorithmGCMP256, "GCMP-256"},
		{dot11CipherAlgorithmWEP40, "WEP"},
		{88, "Cipher(88)"},
	}
	for _, c := range cases {
		got := cipherLabel(c.algo)
		if got != c.want {
			t.Errorf("cipherLabel(%d) = %q, want %q", c.algo, got, c.want)
		}
	}
}

// ── phyLabel ──────────────────────────────────────────────────────────────────

func TestPhyLabel(t *testing.T) {
	cases := []struct {
		phy  uint32
		want string
	}{
		{dot11PhyTypeHT, "802.11n (Wi-Fi 4)"},
		{dot11PhyTypeVHT, "802.11ac (Wi-Fi 5)"},
		{dot11PhyTypeHE, "802.11ax (Wi-Fi 6)"},
		{dot11PhyTypeEHT, "802.11be (Wi-Fi 7)"},
	}
	for _, c := range cases {
		got := phyLabel(c.phy)
		if got != c.want {
			t.Errorf("phyLabel(%d) = %q, want %q", c.phy, got, c.want)
		}
	}
}

// ── freqKHzToBand ─────────────────────────────────────────────────────────────

func TestFreqKHzToBand(t *testing.T) {
	cases := []struct {
		khz  uint32
		want string
	}{
		{2412000, "2.4 GHz"},
		{5180000, "5 GHz"},
		{5955000, "6 GHz"},
		{6135000, "6 GHz"},
		{0, ""},
	}
	for _, c := range cases {
		got := freqKHzToBand(c.khz)
		if got != c.want {
			t.Errorf("freqKHzToBand(%d) = %q, want %q", c.khz, got, c.want)
		}
	}
}

// ── freqKHzToChannel ──────────────────────────────────────────────────────────

func TestFreqKHzToChannel(t *testing.T) {
	cases := []struct {
		khz  uint32
		want string
	}{
		{2412000, "1"},
		{2437000, "6"},
		{2462000, "11"},
		{2484000, "14"},
		{5180000, "36"},
		{5745000, "149"},
		{5955000, "1"},   // 6 GHz channel 1
		{0, ""},
	}
	for _, c := range cases {
		got := freqKHzToChannel(c.khz)
		if got != c.want {
			t.Errorf("freqKHzToChannel(%d kHz) = %q, want %q", c.khz, got, c.want)
		}
	}
}
