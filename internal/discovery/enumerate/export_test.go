package enumerate

// This file is only compiled during testing (due to the _test.go suffix) and
// provides access to enumerate-stage implementation details for in-package and
// external tests. It moved here from the discovery kernel alongside the wired
// collector (ADR-0018 Phase 6).

import (
	"net"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
)

// ExportIncrementIP exposes incrementIP for testing.
func ExportIncrementIP(ip net.IP, n int) net.IP {
	return incrementIP(ip, n)
}

// ExportNormalizeMac exposes normalizeMac for testing.
func ExportNormalizeMac(mac string) string {
	return normalizeMac(mac)
}

// ExportGuessOSFromTTL exposes guessOSFromTTL for testing.
func ExportGuessOSFromTTL(ttl int) string {
	return guessOSFromTTL(ttl)
}

// ExportSplitSubnetIntoChunks exposes splitSubnetIntoChunks for testing.
func ExportSplitSubnetIntoChunks(subnet *net.IPNet, maxChunks int) []*net.IPNet {
	return splitSubnetIntoChunks(subnet, maxChunks)
}

// ExportIsLocallyAdministeredMAC exposes isLocallyAdministeredMAC for testing.
func ExportIsLocallyAdministeredMAC(mac string) bool {
	return isLocallyAdministeredMAC(mac)
}

// ARPScannerTestAccessor provides access to ARPScanner's private fields for testing.
type ARPScannerTestAccessor struct {
	Scanner *ARPScanner
}

// GetInterfaceName returns the scanner's interface name.
func (a *ARPScannerTestAccessor) GetInterfaceName() string {
	return a.Scanner.interfaceName
}

// GetOUI returns the scanner's OUI database.
func (a *ARPScannerTestAccessor) GetOUI() *resolve.OUIDatabase {
	return a.Scanner.oui
}

// GetEntries returns the scanner's entries map.
func (a *ARPScannerTestAccessor) GetEntries() map[string]*ARPEntry {
	return a.Scanner.entries
}

// GetSubnet returns the scanner's subnet.
func (a *ARPScannerTestAccessor) GetSubnet() *net.IPNet {
	return a.Scanner.subnet
}

// SetSubnet sets the scanner's subnet.
func (a *ARPScannerTestAccessor) SetSubnet(subnet *net.IPNet) {
	a.Scanner.subnet = subnet
}

// GetLocalIP returns the scanner's local IP.
func (a *ARPScannerTestAccessor) GetLocalIP() net.IP {
	return a.Scanner.localIP
}

// IsScanning returns the scanner's scanning state.
func (a *ARPScannerTestAccessor) IsScanning() bool {
	return a.Scanner.scanning
}

// SetScanning sets the scanner's scanning state.
func (a *ARPScannerTestAccessor) SetScanning(scanning bool) {
	a.Scanner.scanning = scanning
}

// Lock locks the scanner's mutex.
func (a *ARPScannerTestAccessor) Lock() {
	a.Scanner.mu.Lock()
}

// Unlock unlocks the scanner's mutex.
func (a *ARPScannerTestAccessor) Unlock() {
	a.Scanner.mu.Unlock()
}

// IsInSubnet exposes the private isInSubnet method for testing.
func (s *ARPScanner) IsInSubnet(ip string) bool {
	return s.isInSubnet(ip)
}

// DeviceDiscoveryTestAccessor provides access to DeviceDiscovery's private fields for testing.
type DeviceDiscoveryTestAccessor struct {
	Discovery *DeviceDiscovery
}

// GetProtoManager returns the discovery's protocol manager.
func (d *DeviceDiscoveryTestAccessor) GetProtoManager() *Manager {
	return d.Discovery.protoManager
}

// ICMPPingerTestAccessor exposes the ICMP pinger's socket-free logic.
//
// NewICMPPinger opens a raw socket, which needs root or CAP_NET_RAW, so none
// of this file's 25 functions had any coverage at all (#211). The parsing and
// bookkeeping below do not touch the socket and are the parts most likely to
// be wrong, so they get a constructor that skips it.
type ICMPPingerTestAccessor struct {
	Pinger *ICMPPinger
}

// NewICMPPingerWithoutSocket builds a pinger with no connection, for testing
// the logic that never reaches one.
func NewICMPPingerWithoutSocket(id int) *ICMPPingerTestAccessor {
	return &ICMPPingerTestAccessor{
		Pinger: &ICMPPinger{
			timeout: time.Second,
			id:      id,
			pending: make(map[int]*pendingPing),
			stopCh:  make(chan struct{}),
		},
	}
}

// NextSeq exposes nextSeq.
func (a *ICMPPingerTestAccessor) NextSeq() int { return a.Pinger.nextSeq() }

// HandleReadError exposes handleReadError, returning (shouldExit, continueLoop).
func (a *ICMPPingerTestAccessor) HandleReadError(err error) (bool, bool) {
	res, cont := a.Pinger.handleReadError(err)
	return res.shouldExit, cont
}

// ExtractEchoReplySeq exposes extractEchoReply, returning the sequence number
// and whether the message was a valid echo reply for this pinger.
func (a *ICMPPingerTestAccessor) ExtractEchoReplySeq(data []byte) (int, bool) {
	echo, ok := a.Pinger.extractEchoReply(data)
	if !ok {
		return 0, false
	}
	return echo.Seq, true
}

// AddPending registers a pending ping under seq.
func (a *ICMPPingerTestAccessor) AddPending(seq int, ip string) {
	a.Pinger.pendingMu.Lock()
	defer a.Pinger.pendingMu.Unlock()
	a.Pinger.pending[seq] = &pendingPing{ip: ip, seq: seq, start: time.Now()}
}

// CompletePendingPing exposes completePendingPing, returning the IP it held.
func (a *ICMPPingerTestAccessor) CompletePendingPing(seq int) (string, bool) {
	pp, found := a.Pinger.completePendingPing(seq)
	if !found {
		return "", false
	}
	return pp.ip, true
}

// PendingCount reports how many pings are still outstanding.
func (a *ICMPPingerTestAccessor) PendingCount() int {
	a.Pinger.pendingMu.Lock()
	defer a.Pinger.pendingMu.Unlock()
	return len(a.Pinger.pending)
}
