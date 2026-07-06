//go:build windows

package wifi

import (
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ════════════════════════════════════════════════════════════════════════════
//  DLL handles
// ════════════════════════════════════════════════════════════════════════════

var (
	modWlanapi  = syscall.NewLazyDLL("wlanapi.dll")
	modIphlpapi = syscall.NewLazyDLL("iphlpapi.dll")
	modIcmp     = syscall.NewLazyDLL("icmp.dll")

	procWlanOpenHandle        = modWlanapi.NewProc("WlanOpenHandle")
	procWlanCloseHandle       = modWlanapi.NewProc("WlanCloseHandle")
	procWlanEnumInterfaces    = modWlanapi.NewProc("WlanEnumInterfaces")
	procWlanQueryInterface    = modWlanapi.NewProc("WlanQueryInterface")
	procWlanGetNetworkBssList = modWlanapi.NewProc("WlanGetNetworkBssList")
	procWlanFreeMemory        = modWlanapi.NewProc("WlanFreeMemory")
	procGetAdaptersAddresses  = modIphlpapi.NewProc("GetAdaptersAddresses")
	procIcmpCreateFile        = modIcmp.NewProc("IcmpCreateFile")
	procIcmpCloseHandle       = modIcmp.NewProc("IcmpCloseHandle")
	procIcmpSendEcho          = modIcmp.NewProc("IcmpSendEcho")
)

// ════════════════════════════════════════════════════════════════════════════
//  Constants
// ════════════════════════════════════════════════════════════════════════════

const (
	wlanAPIVersion2 = 2

	wlanIntfOpcodeConnectionAttributes = 7

	wlanNotConnected = 0
	wlanConnected    = 1
	wlanAdHocNetwork = 2

	dot11AuthAlgorithmOpen     = 1
	dot11AuthAlgorithmWPA      = 4
	dot11AuthAlgorithmWPA_PSK  = 5
	dot11AuthAlgorithmRSNA     = 6
	dot11AuthAlgorithmRSNA_PSK = 7
	dot11AuthAlgorithmWPA3     = 8
	dot11AuthAlgorithmWPA3_SAE = 9

	dot11CipherAlgorithmNone    = 0x00
	dot11CipherAlgorithmWEP40   = 0x01
	dot11CipherAlgorithmTKIP    = 0x02
	dot11CipherAlgorithmCCMP    = 0x04
	dot11CipherAlgorithmWEP104  = 0x05
	dot11CipherAlgorithmGCMP    = 0x06
	dot11CipherAlgorithmGCMP256 = 0x07
	dot11CipherAlgorithmWEP     = 0x101

	dot11PhyTypeHRDSSS  = 5
	dot11PhyTypeOFDM    = 6
	dot11PhyTypeERPOFDM = 7
	dot11PhyTypeHT      = 8
	dot11PhyTypeVHT     = 9
	dot11PhyTypeHE      = 10
	dot11PhyTypeEHT     = 11

	gaaMFlagIncludeGateways = 0x0080
	afUnspec                = 0
	afInet                  = 2
	afInet6                 = 23
	ifTypeIeee80211         = 71

	maxAdapterAddressLength = 8

	errorBufferOverflow = 111
	noError             = 0
)

// ════════════════════════════════════════════════════════════════════════════
//  Structs
// ════════════════════════════════════════════════════════════════════════════

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g GUID) String() string {
	return fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1],
		g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

type wlanInterfaceInfo struct {
	InterfaceGuid           GUID
	strInterfaceDescription [256]uint16
	isState                 uint32
}

type wlanInterfaceInfoList struct {
	dwNumberOfItems uint32
	dwIndex         uint32
}

type dot11SSID struct {
	uSSIDLength uint32
	ucSSID      [32]byte
}

func (s dot11SSID) String() string {
	if s.uSSIDLength == 0 {
		return ""
	}
	return string(s.ucSSID[:s.uSSIDLength])
}

type dot11MacAddress [6]byte

func (m dot11MacAddress) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		m[0], m[1], m[2], m[3], m[4], m[5])
}

type wlanAssociationAttributes struct {
	dot11Ssid         dot11SSID
	dot11BssType      uint32
	dot11Bssid        dot11MacAddress
	dot11PhyType      uint32
	uDot11PhyIndex    uint32
	wlanSignalQuality uint32
	ulRxRate          uint32
	ulTxRate          uint32
}

type wlanSecurityAttributes struct {
	bSecurityEnabled     int32
	bOneXEnabled         int32
	dot11AuthAlgorithm   uint32
	dot11CipherAlgorithm uint32
}

type wlanConnectionAttributes struct {
	isState                   uint32
	wlanConnectionMode        uint32
	strProfileName            [256]uint16
	wlanAssociationAttributes wlanAssociationAttributes
	wlanSecurityAttributes    wlanSecurityAttributes
}

// wlanBssEntry mirrors WLAN_BSS_ENTRY from wlanapi.h
type wlanBssEntry struct {
	dot11Ssid               dot11SSID      // 36 bytes
	uPhyId                  uint32         // 4
	dot11Bssid              dot11MacAddress // 6
	dot11BssType            uint32         // NOTE: 6-byte MAC + natural alignment pads to 8, then uint32
	dot11BssPhyType         uint32
	lRssi                   int32
	uLinkQuality            uint32
	bInRegDomain            int32
	usBeaconPeriod          uint16
	_                       [2]byte
	ullTimestamp            uint64
	ullHostTimestamp        uint64
	usCapabilityInformation uint16
	_                       [2]byte
	ulChCenterFrequency     uint32 // kHz
	// WLAN_RATE_SET + IE blob follow — not accessed
}

type wlanBssList struct {
	dwTotalSize     uint32
	dwNumberOfItems uint32
}

type socketAddress struct {
	lpSockaddr      uintptr
	iSockaddrLength int32
}

type ipAdapterUnicastAddress struct {
	Length             uint32
	Flags              uint32
	Next               *ipAdapterUnicastAddress
	Address            socketAddress
	PrefixOrigin       uint32
	SuffixOrigin       uint32
	DadState           uint32
	ValidLifetime      uint32
	PreferredLifetime  uint32
	LeaseLifetime      uint32
	OnLinkPrefixLength uint8
	_                  [3]byte
}

type ipAdapterGatewayAddress struct {
	Length  uint32
	_       uint32
	Next    *ipAdapterGatewayAddress
	Address socketAddress
}

type ipAdaptersAddresses struct {
	Length                 uint32
	IfIndex                uint32
	Next                   *ipAdaptersAddresses
	AdapterName            *byte
	FirstUnicastAddress    *ipAdapterUnicastAddress
	FirstAnycastAddress    uintptr
	FirstMulticastAddress  uintptr
	FirstDnsServerAddress  uintptr
	DnsSuffix              *uint16
	Description            *uint16
	FriendlyName           *uint16
	PhysicalAddress        [maxAdapterAddressLength]byte
	PhysicalAddressLength  uint32
	Flags                  uint32
	Mtu                    uint32
	IfType                 uint32
	OperStatus             uint32
	Ipv6IfIndex            uint32
	ZoneIndices            [16]uint32
	FirstPrefix            uintptr
	TransmitLinkSpeed      uint64
	ReceiveLinkSpeed       uint64
	FirstWinsServerAddress uintptr
	FirstGatewayAddress    *ipAdapterGatewayAddress
}

type ipOptionInformation struct {
	Ttl         uint8
	Tos         uint8
	Flags       uint8
	OptionsSize uint8
	OptionsData uintptr
}

type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

// ════════════════════════════════════════════════════════════════════════════
//  String helpers
// ════════════════════════════════════════════════════════════════════════════

func utf16ArrayToString(arr []uint16) string {
	for i, c := range arr {
		if c == 0 {
			return syscall.UTF16ToString(arr[:i])
		}
	}
	return syscall.UTF16ToString(arr)
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	n := 0
	for ptr := uintptr(unsafe.Pointer(p)); *(*uint16)(unsafe.Pointer(ptr)) != 0; ptr += 2 {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n))
}

func bytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	var buf []byte
	for ptr := uintptr(unsafe.Pointer(p)); *(*byte)(unsafe.Pointer(ptr)) != 0; ptr++ {
		buf = append(buf, *(*byte)(unsafe.Pointer(ptr)))
	}
	return string(buf)
}

// ════════════════════════════════════════════════════════════════════════════
//  Label helpers
// ════════════════════════════════════════════════════════════════════════════

func authLabel(algo uint32) string {
	switch algo {
	case dot11AuthAlgorithmOpen:
		return "Open"
	case dot11AuthAlgorithmWPA:
		return "WPA"
	case dot11AuthAlgorithmWPA_PSK:
		return "WPA-Personal"
	case dot11AuthAlgorithmRSNA:
		return "WPA2-Enterprise"
	case dot11AuthAlgorithmRSNA_PSK:
		return "WPA2-Personal"
	case dot11AuthAlgorithmWPA3:
		return "WPA3-Enterprise"
	case dot11AuthAlgorithmWPA3_SAE:
		return "WPA3-Personal"
	default:
		return fmt.Sprintf("Auth(%d)", algo)
	}
}

func cipherLabel(algo uint32) string {
	switch algo {
	case dot11CipherAlgorithmNone:
		return "None"
	case dot11CipherAlgorithmWEP40, dot11CipherAlgorithmWEP104, dot11CipherAlgorithmWEP:
		return "WEP"
	case dot11CipherAlgorithmTKIP:
		return "TKIP"
	case dot11CipherAlgorithmCCMP:
		return "CCMP"
	case dot11CipherAlgorithmGCMP:
		return "GCMP"
	case dot11CipherAlgorithmGCMP256:
		return "GCMP-256"
	default:
		return fmt.Sprintf("Cipher(%d)", algo)
	}
}

func phyLabel(phy uint32) string {
	switch phy {
	case dot11PhyTypeHRDSSS:
		return "802.11b"
	case dot11PhyTypeOFDM:
		return "802.11a"
	case dot11PhyTypeERPOFDM:
		return "802.11g"
	case dot11PhyTypeHT:
		return "802.11n"
	case dot11PhyTypeVHT:
		return "802.11ac"
	case dot11PhyTypeHE:
		return "802.11ax (Wi-Fi 6)"
	case dot11PhyTypeEHT:
		return "802.11be (Wi-Fi 7)"
	default:
		return fmt.Sprintf("PHY(%d)", phy)
	}
}

func freqKHzToBand(khz uint32) string {
	mhz := khz / 1000
	switch {
	case mhz >= 5925:
		return "6 GHz"
	case mhz >= 5000:
		return "5 GHz"
	case mhz >= 2400:
		return "2.4 GHz"
	default:
		return ""
	}
}

func freqKHzToChannel(khz uint32) string {
	mhz := int(khz / 1000)
	switch {
	case mhz >= 2412 && mhz <= 2484:
		if mhz == 2484 {
			return "14"
		}
		return fmt.Sprintf("%d", (mhz-2407)/5)
	case mhz >= 5180 && mhz <= 5825:
		return fmt.Sprintf("%d", (mhz-5000)/5)
	case mhz >= 5955 && mhz <= 7115:
		return fmt.Sprintf("%d", (mhz-5950)/5)
	}
	return ""
}

func rssiToPercent(rssi int32) int {
	if rssi >= -30 {
		return 100
	}
	if rssi <= -90 {
		return 0
	}
	return int((rssi + 90) * 100 / 60)
}

// ════════════════════════════════════════════════════════════════════════════
//  IP / gateway collection
// ════════════════════════════════════════════════════════════════════════════

type adapterIP struct {
	friendlyName string
	ipv4         string
	ipv6         string
	gateway      string
}

func getAdapterIPs() (byGUID map[string]adapterIP, byName map[string]adapterIP) {
	byGUID = map[string]adapterIP{}
	byName = map[string]adapterIP{}

	var bufSize uint32 = 16 * 1024
	var buf []byte
	for {
		buf = make([]byte, bufSize)
		ret, _, _ := procGetAdaptersAddresses.Call(
			afUnspec,
			gaaMFlagIncludeGateways,
			0,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&bufSize)),
		)
		if ret == noError {
			break
		}
		if ret == errorBufferOverflow {
			continue
		}
		return
	}

	for p := (*ipAdaptersAddresses)(unsafe.Pointer(&buf[0])); p != nil; p = p.Next {
		if p.IfType != ifTypeIeee80211 {
			continue
		}

		guidStr := ""
		if p.AdapterName != nil {
			raw := bytePtrToString(p.AdapterName)
			guidStr = strings.ToUpper(strings.Trim(raw, "{}"))
		}
		friendly := utf16PtrToString(p.FriendlyName)

		entry := adapterIP{friendlyName: friendly}

		for ua := p.FirstUnicastAddress; ua != nil; ua = ua.Next {
			if ua.Address.lpSockaddr == 0 {
				continue
			}
			family := *(*uint16)(unsafe.Pointer(ua.Address.lpSockaddr))
			switch family {
			case afInet:
				b := (*[4]byte)(unsafe.Pointer(ua.Address.lpSockaddr + 4))
				entry.ipv4 = fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
			case afInet6:
				b := (*[16]byte)(unsafe.Pointer(ua.Address.lpSockaddr + 8))
				ip := net.IP(b[:])
				if !ip.IsLinkLocalUnicast() {
					entry.ipv6 = ip.String()
				}
			}
		}

		if p.FirstGatewayAddress != nil && p.FirstGatewayAddress.Address.lpSockaddr != 0 {
			family := *(*uint16)(unsafe.Pointer(p.FirstGatewayAddress.Address.lpSockaddr))
			if family == afInet {
				b := (*[4]byte)(unsafe.Pointer(p.FirstGatewayAddress.Address.lpSockaddr + 4))
				entry.gateway = fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
			}
		}

		if guidStr != "" {
			byGUID[guidStr] = entry
		}
		if friendly != "" {
			byName[friendly] = entry
		}
	}
	return
}

// ════════════════════════════════════════════════════════════════════════════
//  Main WiFi enumeration
// ════════════════════════════════════════════════════════════════════════════

func getWindows() Info {
	now := time.Now()

	var clientHandle syscall.Handle
	var negotiatedVersion uint32
	ret, _, _ := procWlanOpenHandle.Call(
		wlanAPIVersion2, 0,
		uintptr(unsafe.Pointer(&negotiatedVersion)),
		uintptr(unsafe.Pointer(&clientHandle)),
	)
	if ret != 0 {
		return Info{Timestamp: now, Error: fmt.Sprintf("WlanOpenHandle failed: 0x%X", ret)}
	}
	defer procWlanCloseHandle.Call(uintptr(clientHandle), 0)

	var ifaceList *wlanInterfaceInfoList
	ret, _, _ = procWlanEnumInterfaces.Call(
		uintptr(clientHandle), 0,
		uintptr(unsafe.Pointer(&ifaceList)),
	)
	if ret != 0 || ifaceList == nil {
		return Info{Timestamp: now, HasWifi: false, Error: "no Wi-Fi adapter detected"}
	}
	defer procWlanFreeMemory.Call(uintptr(unsafe.Pointer(ifaceList)))

	count := int(ifaceList.dwNumberOfItems)
	if count == 0 {
		return Info{Timestamp: now, HasWifi: false, Error: "no Wi-Fi adapter detected"}
	}

	ifaceSlice := (*[128]wlanInterfaceInfo)(
		unsafe.Pointer(uintptr(unsafe.Pointer(ifaceList)) + unsafe.Sizeof(*ifaceList)),
	)[:count:count]

	ipByGUID, ipByName := getAdapterIPs()

	adapters := make([]Adapter, 0, count)
	activeIndex := 0
	hasInternet := false

	for i := 0; i < count; i++ {
		iface := ifaceSlice[i]
		guidStr := strings.ToUpper(strings.Trim(iface.InterfaceGuid.String(), "{}"))

		// Prefer FriendlyName from GetAdaptersAddresses ("Wi-Fi", "Wi-Fi 3", …).
		// strInterfaceDescription is the long hardware driver string — not what users see.
		friendlyName := utf16ArrayToString(iface.strInterfaceDescription[:]) // fallback only
		if ip, ok := ipByGUID[guidStr]; ok && ip.friendlyName != "" {
			friendlyName = ip.friendlyName
		}

		adapter := Adapter{
			Name:  friendlyName,
			GUID:  iface.InterfaceGuid.String(),
			State: "disconnected",
		}

		// ── Connection attributes ──────────────────────────────────────────
		var dataSize uint32
		var data uintptr
		var opCode uint32
		ret, _, _ = procWlanQueryInterface.Call(
			uintptr(clientHandle),
			uintptr(unsafe.Pointer(&iface.InterfaceGuid)),
			wlanIntfOpcodeConnectionAttributes,
			0,
			uintptr(unsafe.Pointer(&dataSize)),
			uintptr(unsafe.Pointer(&data)),
			uintptr(unsafe.Pointer(&opCode)),
		)
		if ret == 0 && data != 0 {
			conn := (*wlanConnectionAttributes)(unsafe.Pointer(data))
			if conn.isState == wlanConnected || conn.isState == wlanAdHocNetwork {
				adapter.State = "connected"
				assoc := conn.wlanAssociationAttributes
				adapter.SSID = assoc.dot11Ssid.String()
				adapter.BSSID = assoc.dot11Bssid.String()
				adapter.SignalPercent = int(assoc.wlanSignalQuality)
				adapter.SignalText = fmt.Sprintf("%d%%", assoc.wlanSignalQuality)
				adapter.RadioType = phyLabel(assoc.dot11PhyType)
				adapter.Profile = utf16ArrayToString(conn.strProfileName[:])
				if assoc.ulTxRate > 0 {
					adapter.TransmitRateMbps = fmt.Sprintf("%.1f", float64(assoc.ulTxRate)/1000.0)
				}
				if assoc.ulRxRate > 0 {
					adapter.ReceiveRateMbps = fmt.Sprintf("%.1f", float64(assoc.ulRxRate)/1000.0)
				}
				sec := conn.wlanSecurityAttributes
				adapter.Authentication = authLabel(sec.dot11AuthAlgorithm)
				adapter.Cipher = cipherLabel(sec.dot11CipherAlgorithm)
			}
			procWlanFreeMemory.Call(data)
		}

		// ── BSS list: accurate band, channel, RSSI, PHY ───────────────────
		if adapter.State == "connected" {
			var bssList *wlanBssList
			ret, _, _ = procWlanGetNetworkBssList.Call(
				uintptr(clientHandle),
				uintptr(unsafe.Pointer(&iface.InterfaceGuid)),
				0, // NULL ssid → current association
				1, // dot11BssTypeInfrastructure
				0,
				0,
				uintptr(unsafe.Pointer(&bssList)),
			)
			if ret == 0 && bssList != nil && bssList.dwNumberOfItems > 0 {
				n := int(bssList.dwNumberOfItems)
				// Walk entries by pointer arithmetic — each entry has variable
				// trailing data, but we only read the fixed head.
				ptr := uintptr(unsafe.Pointer(bssList)) + unsafe.Sizeof(*bssList)
				entrySize := unsafe.Sizeof(wlanBssEntry{})
				for j := 0; j < n; j++ {
					bss := (*wlanBssEntry)(unsafe.Pointer(ptr))
					if bss.dot11Bssid.String() == adapter.BSSID || n == 1 {
						if bss.ulChCenterFrequency > 0 {
							adapter.Band = freqKHzToBand(bss.ulChCenterFrequency)
							adapter.Channel = freqKHzToChannel(bss.ulChCenterFrequency)
						}
						if bss.lRssi != 0 {
							if pct := rssiToPercent(bss.lRssi); pct > 0 {
								adapter.SignalPercent = pct
								adapter.SignalText = fmt.Sprintf("%d%%", pct)
							}
						}
						if bss.dot11BssPhyType > 0 {
							adapter.RadioType = phyLabel(bss.dot11BssPhyType)
						}
						break
					}
					ptr += entrySize
				}
				procWlanFreeMemory.Call(uintptr(unsafe.Pointer(bssList)))
			}
		}

		// ── IP / gateway ──────────────────────────────────────────────────
		if ip, ok := ipByGUID[guidStr]; ok {
			adapter.IPv4 = ip.ipv4
			adapter.IPv6 = ip.ipv6
			adapter.Gateway = ip.gateway
		} else if ip, ok := ipByName[friendlyName]; ok {
			adapter.IPv4 = ip.ipv4
			adapter.IPv6 = ip.ipv6
			adapter.Gateway = ip.gateway
		}

		if adapter.Connected() {
			activeIndex = i
			if adapter.Gateway != "" {
				hasInternet = true
			}
		}
		adapters = append(adapters, adapter)
	}

	return Info{
		Timestamp:   now,
		HasWifi:     true,
		HasInternet: hasInternet,
		ActiveIndex: activeIndex,
		Adapters:    adapters,
	}
}

// ════════════════════════════════════════════════════════════════════════════
//  Native ICMP ping
// ════════════════════════════════════════════════════════════════════════════

func pingGatewayWindows(gateway string) PingResult {
	now := time.Now()
	gw := strings.TrimSpace(gateway)
	if gw == "" {
		return PingResult{Error: "no gateway", CheckedAt: now}
	}

	// Resolve gateway to uint32 IPv4 address in network byte order.
	// FIX: inet_addr takes ANSI bytes, NOT UTF-16. Use BytePtrFromString.
	var gwAddr uint32
	if ip4 := net.ParseIP(gw).To4(); ip4 != nil {
		gwAddr = uint32(ip4[0]) | uint32(ip4[1])<<8 | uint32(ip4[2])<<16 | uint32(ip4[3])<<24
	} else {
		// Not a bare IP — resolve via Go's resolver
		addrs, err := net.LookupHost(gw)
		if err != nil || len(addrs) == 0 {
			return PingResult{Error: "cannot resolve gateway: " + gw, CheckedAt: now}
		}
		ip4 = net.ParseIP(addrs[0]).To4()
		if ip4 == nil {
			return PingResult{Error: "gateway has no IPv4 address", CheckedAt: now}
		}
		gwAddr = uint32(ip4[0]) | uint32(ip4[1])<<8 | uint32(ip4[2])<<16 | uint32(ip4[3])<<24
	}

	icmpHandle, _, _ := procIcmpCreateFile.Call()
	if icmpHandle == 0 || icmpHandle == ^uintptr(0) {
		return PingResult{Error: "IcmpCreateFile failed", CheckedAt: now}
	}
	defer procIcmpCloseHandle.Call(icmpHandle)

	const (
		nPings    = 4
		timeoutMs = 1200
	)

	payload := []byte("wifimon-ping-data")
	replySize := unsafe.Sizeof(icmpEchoReply{}) + uintptr(len(payload)) + 8
	replyBuf := make([]byte, replySize)

	sent, received, totalLatency := 0, 0, 0

	for i := 0; i < nPings; i++ {
		sent++
		ret, _, _ := procIcmpSendEcho.Call(
			icmpHandle,
			uintptr(gwAddr),
			uintptr(unsafe.Pointer(&payload[0])),
			uintptr(len(payload)),
			0, // RequestOptions = NULL
			uintptr(unsafe.Pointer(&replyBuf[0])),
			uintptr(len(replyBuf)),
			timeoutMs,
		)
		if ret > 0 {
			reply := (*icmpEchoReply)(unsafe.Pointer(&replyBuf[0]))
			if reply.Status == 0 { // IP_SUCCESS
				received++
				totalLatency += int(reply.RoundTripTime)
			}
		}
	}

	result := PingResult{CheckedAt: now, Sent: sent, Received: received}
	if received > 0 {
		result.Reachable = true
		result.LatencyMS = totalLatency / received
		result.PacketLoss = float64(sent-received) / float64(sent) * 100.0
	} else {
		result.PacketLoss = 100
		result.Error = "request timed out"
	}
	return result
}
