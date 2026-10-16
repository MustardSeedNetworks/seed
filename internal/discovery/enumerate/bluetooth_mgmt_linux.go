//go:build linux

package enumerate

// bluetooth_mgmt_linux.go discovers Bluetooth devices through the kernel's
// Bluetooth management interface, the HCI control channel bluetoothd itself
// drives. It needs CAP_NET_ADMIN, which the packaged service holds, and no
// BlueZ userspace tools. Every device-found event carries the address type,
// the RSSI and the raw advertising or EIR payload, so signal and vendor come
// from the radio rather than from a tool's text output.
//
// A headless host without bluetoothd leaves its controllers powered off with
// LE disabled. A scan turns on only what it needs and puts it back afterwards,
// so the adapter is left as the scan found it.

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Management protocol values (BlueZ doc/mgmt-api.txt).
const (
	mgmtIndexNone = 0xFFFF
	mgmtHeaderLen = 6
	mgmtMaxPacket = 4096

	mgmtOpReadIndexList  = 0x0003
	mgmtOpReadInfo       = 0x0004
	mgmtOpSetPowered     = 0x0005
	mgmtOpSetLE          = 0x000D
	mgmtOpStartDiscovery = 0x0023
	mgmtOpStopDiscovery  = 0x0024

	mgmtEvCmdComplete = 0x0001
	mgmtEvCmdStatus   = 0x0002
	mgmtEvDeviceFound = 0x0012
	mgmtEvDiscovering = 0x0013

	mgmtStatusSuccess          = 0x00
	mgmtStatusBusy             = 0x0A
	mgmtStatusRejected         = 0x0B
	mgmtStatusNotSupported     = 0x0C
	mgmtStatusInvalidIndex     = 0x11
	mgmtStatusPermissionDenied = 0x14

	mgmtSettingPowered = 1 << 0
	mgmtSettingLE      = 1 << 9

	mgmtAddrBREDR    = 0
	mgmtAddrLERandom = 2

	mgmtScanBREDR = 1 << 0
	mgmtScanLE    = 1<<1 | 1<<2

	mgmtFoundNotConnectable = 1 << 2
	mgmtRSSIUnavailable     = 127

	// Reply layout offsets. Command complete and status replies start with the
	// command's opcode and a status byte; Read Controller Information puts the
	// current settings after the address, version, manufacturer and supported
	// settings; Device Found is address, type, RSSI, flags and an EIR length.
	mgmtReplyStatusOffset   = 2
	mgmtReplyDataOffset     = 3
	mgmtInfoCurrentOffset   = 13
	mgmtInfoMinLen          = 17
	mgmtIndexListHeaderLen  = 2
	mgmtFoundTypeOffset     = 6
	mgmtFoundRSSIOffset     = 7
	mgmtFoundFlagsOffset    = 8
	mgmtFoundEIRLenOffset   = 12
	mgmtFoundHeaderLen      = 14
	mgmtDiscoveringStateOff = 1
	mgmtDiscoveringLen      = 2

	mgmtReplyTimeout = 3 * time.Second
	mgmtPollInterval = 250 * time.Millisecond

	bdAddrLen = 6
)

// mgmtEvent is one management event as read from the control channel.
type mgmtEvent struct {
	code    uint16
	index   uint16
	payload []byte
}

// mgmtConn is the control channel: the kernel socket in production, a script
// in tests.
type mgmtConn interface {
	send(op, index uint16, params []byte) error
	// recv returns the next event, or false when none arrived: the timeout
	// passed or a signal interrupted the wait.
	recv(timeout time.Duration) (mgmtEvent, bool, error)
}

// mgmtStatusError is a command the kernel answered with a non-success status.
type mgmtStatusError struct {
	op     uint16
	status byte
}

func (e *mgmtStatusError) Error() string {
	msg := fmt.Sprintf("bluetooth management command 0x%04x failed: status 0x%02x", e.op, e.status)
	switch e.status {
	case mgmtStatusPermissionDenied:
		return msg + " (permission denied; the scan needs CAP_NET_ADMIN)"
	case mgmtStatusNotSupported:
		return msg + " (not supported by this controller)"
	case mgmtStatusInvalidIndex:
		return msg + " (no such adapter)"
	}
	return msg
}

// scanMgmt runs one discovery of scanTypes on adapter ("hciN", or "" for the
// first controller) for d and returns every device heard, ordered by address.
func scanMgmt(ctx context.Context, adapter string, scanTypes byte, d time.Duration) ([]BluetoothDevice, error) {
	sock, err := openMgmtSocket()
	if err != nil {
		return nil, err
	}
	defer sock.close()

	s := &mgmtSession{conn: sock, devices: make(map[string]*BluetoothDevice)}
	if s.index, err = s.resolveIndex(adapter); err != nil {
		return nil, err
	}
	if err = s.scan(ctx, scanTypes, d); err != nil {
		return nil, err
	}
	return s.result(), nil
}

// mgmtSession is one scan's state on one controller.
type mgmtSession struct {
	conn        mgmtConn
	index       uint16
	devices     map[string]*BluetoothDevice
	discovering bool
}

func (s *mgmtSession) resolveIndex(adapter string) (uint16, error) {
	if adapter != "" {
		n, err := strconv.ParseUint(strings.TrimPrefix(adapter, "hci"), 10, 16)
		if err != nil || !strings.HasPrefix(adapter, "hci") {
			return 0, fmt.Errorf("bluetooth adapter %q is not an hciN name", adapter)
		}
		return uint16(n), nil
	}
	reply, err := s.command(mgmtOpReadIndexList, mgmtIndexNone, nil)
	if err != nil {
		return 0, err
	}
	if len(reply) < mgmtIndexListHeaderLen+2 || binary.LittleEndian.Uint16(reply) == 0 {
		return 0, errors.New("no Bluetooth adapter found")
	}
	return binary.LittleEndian.Uint16(reply[mgmtIndexListHeaderLen:]), nil
}

// scan prepares the controller, discovers until d elapses and restores the
// settings it changed.
func (s *mgmtSession) scan(ctx context.Context, scanTypes byte, d time.Duration) (err error) {
	restore, err := s.prepare(scanTypes)
	defer func() { err = errors.Join(err, restore()) }()
	if err != nil {
		return err
	}
	return s.discover(ctx, scanTypes, d)
}

// prepare enables LE (when the scan needs it) and power, and returns what
// undoes exactly the changes it made, power first.
func (s *mgmtSession) prepare(scanTypes byte) (func() error, error) {
	var undo []uint16
	restore := func() error {
		var err error
		for _, op := range slices.Backward(undo) {
			err = errors.Join(err, s.setting(op, false))
		}
		return err
	}

	info, err := s.command(mgmtOpReadInfo, s.index, nil)
	if err != nil {
		return restore, err
	}
	if len(info) < mgmtInfoMinLen {
		return restore, errors.New("bluetooth controller information reply is truncated")
	}
	current := binary.LittleEndian.Uint32(info[mgmtInfoCurrentOffset:])

	needLE := scanTypes&mgmtScanLE != 0 && current&mgmtSettingLE == 0
	needPower := current&mgmtSettingPowered == 0
	for _, step := range []struct {
		op   uint16
		need bool
	}{{mgmtOpSetLE, needLE}, {mgmtOpSetPowered, needPower}} {
		if !step.need {
			continue
		}
		if err = s.setting(step.op, true); err != nil {
			return restore, err
		}
		undo = append(undo, step.op)
	}
	return restore, nil
}

// discover listens until d elapses. The kernel ends an interleaved discovery
// after about ten seconds, so a longer scan restarts it.
func (s *mgmtSession) discover(ctx context.Context, scanTypes byte, d time.Duration) error {
	started := false
	deadline := time.Now().Add(d)
	for remaining := d; remaining > 0 && ctx.Err() == nil; remaining = time.Until(deadline) {
		if !s.discovering {
			ours, err := s.startDiscovery(scanTypes)
			if err != nil {
				return err
			}
			started = started || ours
		}
		ev, ok, err := s.conn.recv(min(remaining, mgmtPollInterval))
		if err != nil {
			return err
		}
		if ok {
			s.handle(ev)
		}
	}
	if started {
		if err := s.stopDiscovery(scanTypes); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// startDiscovery starts a discovery and reports whether it is this scan's.
// Busy means another client (bluetoothd) is already discovering; its results
// reach every control socket, so the scan listens to them instead.
func (s *mgmtSession) startDiscovery(scanTypes byte) (bool, error) {
	_, err := s.command(mgmtOpStartDiscovery, s.index, []byte{scanTypes})
	s.discovering = true
	if statusErr, ok := errors.AsType[*mgmtStatusError](err); ok && statusErr.status == mgmtStatusBusy {
		return false, nil
	}
	return err == nil, err
}

// stopDiscovery ends this scan's discovery. Rejected means it had already
// ended on its own.
func (s *mgmtSession) stopDiscovery(scanTypes byte) error {
	_, err := s.command(mgmtOpStopDiscovery, s.index, []byte{scanTypes})
	if statusErr, ok := errors.AsType[*mgmtStatusError](err); ok && statusErr.status == mgmtStatusRejected {
		return nil
	}
	return err
}

func (s *mgmtSession) setting(op uint16, on bool) error {
	var v byte
	if on {
		v = 1
	}
	_, err := s.command(op, s.index, []byte{v})
	return err
}

// command sends op and waits for its reply, handling the events that arrive
// in the meantime.
func (s *mgmtSession) command(op, index uint16, params []byte) ([]byte, error) {
	if err := s.conn.send(op, index, params); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(mgmtReplyTimeout)
	for remaining := mgmtReplyTimeout; remaining > 0; remaining = time.Until(deadline) {
		ev, ok, err := s.conn.recv(remaining)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if isReplyTo(ev, op, index) {
			if status := ev.payload[mgmtReplyStatusOffset]; status != mgmtStatusSuccess {
				return nil, &mgmtStatusError{op: op, status: status}
			}
			return ev.payload[min(mgmtReplyDataOffset, len(ev.payload)):], nil
		}
		s.handle(ev)
	}
	return nil, fmt.Errorf("bluetooth management command 0x%04x: no reply", op)
}

func isReplyTo(ev mgmtEvent, op, index uint16) bool {
	return (ev.code == mgmtEvCmdComplete || ev.code == mgmtEvCmdStatus) &&
		ev.index == index && len(ev.payload) >= mgmtReplyDataOffset &&
		binary.LittleEndian.Uint16(ev.payload) == op
}

func (s *mgmtSession) handle(ev mgmtEvent) {
	if ev.index != s.index {
		return
	}
	switch ev.code {
	case mgmtEvDeviceFound:
		s.deviceFound(ev.payload, time.Now())
	case mgmtEvDiscovering:
		if len(ev.payload) >= mgmtDiscoveringLen {
			s.discovering = ev.payload[mgmtDiscoveringStateOff] == 1
		}
	}
}

// deviceFound folds one Device Found event into the scan. A device heard
// over both BR/EDR and LE is dual-mode; the latest available RSSI wins.
func (s *mgmtSession) deviceFound(p []byte, now time.Time) {
	if len(p) < mgmtFoundHeaderLen {
		return
	}
	eirLen := int(binary.LittleEndian.Uint16(p[mgmtFoundEIRLenOffset:]))
	if len(p) < mgmtFoundHeaderLen+eirLen {
		return
	}
	addrType := p[mgmtFoundTypeOffset]
	kind := BluetoothTypeBLE
	if addrType == mgmtAddrBREDR {
		kind = BluetoothTypeClassic
	}

	addr := formatBDAddr(p[:bdAddrLen])
	dev, seen := s.devices[addr]
	if !seen {
		dev = &BluetoothDevice{
			Address:     addr,
			AddressType: BluetoothAddressPublic,
			Type:        kind,
			DeviceClass: BluetoothClassUncategorized,
			FirstSeen:   now,
		}
		s.devices[addr] = dev
	} else if dev.Type != kind {
		dev.Type = BluetoothTypeDual
	}
	if addrType == mgmtAddrLERandom {
		dev.AddressType = BluetoothAddressRandom
	}

	dev.LastSeen = now
	if rssi := signedByte(p[mgmtFoundRSSIOffset]); rssi != mgmtRSSIUnavailable {
		dev.RSSI = rssi
	}
	flags := binary.LittleEndian.Uint32(p[mgmtFoundFlagsOffset:])
	dev.IsConnectable = dev.IsConnectable || kind == BluetoothTypeClassic || flags&mgmtFoundNotConnectable == 0
	applyAdvertisingData(dev, p[mgmtFoundHeaderLen:mgmtFoundHeaderLen+eirLen])
}

func (s *mgmtSession) result() []BluetoothDevice {
	out := make([]BluetoothDevice, 0, len(s.devices))
	for _, dev := range s.devices {
		out = append(out, *dev)
	}
	slices.SortFunc(out, func(a, b BluetoothDevice) int { return strings.Compare(a.Address, b.Address) })
	return out
}

// formatBDAddr renders a little-endian device address as AA:BB:CC:DD:EE:FF.
func formatBDAddr(le []byte) string {
	b := slices.Clone(le)
	slices.Reverse(b)
	parts := make([]string, len(b))
	for i, octet := range b {
		parts[i] = fmt.Sprintf("%02X", octet)
	}
	return strings.Join(parts, ":")
}

// mgmtSocket is the kernel control channel.
type mgmtSocket struct {
	fd int
}

func openMgmtSocket() (*mgmtSocket, error) {
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.BTPROTO_HCI)
	if err != nil {
		return nil, fmt.Errorf("open Bluetooth management socket: %w", err)
	}
	if err = unix.Bind(fd, &unix.SockaddrHCI{Dev: mgmtIndexNone, Channel: unix.HCI_CHANNEL_CONTROL}); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("bind Bluetooth management socket: %w", err)
	}
	return &mgmtSocket{fd: fd}, nil
}

func (m *mgmtSocket) close() {
	_ = unix.Close(m.fd)
}

func (m *mgmtSocket) send(op, index uint16, params []byte) error {
	n := len(params)
	if n > math.MaxUint16 {
		return fmt.Errorf("bluetooth management command 0x%04x: %d parameter bytes", op, n)
	}
	pkt := make([]byte, mgmtHeaderLen, mgmtHeaderLen+n)
	binary.LittleEndian.PutUint16(pkt[0:], op)
	binary.LittleEndian.PutUint16(pkt[2:], index)
	binary.LittleEndian.PutUint16(pkt[4:], uint16(n))
	if _, err := unix.Write(m.fd, append(pkt, params...)); err != nil {
		return fmt.Errorf("bluetooth management command 0x%04x: %w", op, err)
	}
	return nil
}

func (m *mgmtSocket) recv(timeout time.Duration) (mgmtEvent, bool, error) {
	// A zero receive timeout would block forever.
	tv := unix.NsecToTimeval(max(timeout, time.Millisecond).Nanoseconds())
	if err := unix.SetsockoptTimeval(m.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		return mgmtEvent{}, false, fmt.Errorf("set Bluetooth management socket timeout: %w", err)
	}
	buf := make([]byte, mgmtMaxPacket)
	n, err := unix.Read(m.fd, buf)
	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return mgmtEvent{}, false, nil
	}
	if err != nil {
		return mgmtEvent{}, false, fmt.Errorf("read Bluetooth management socket: %w", err)
	}
	if n < mgmtHeaderLen {
		return mgmtEvent{}, false, fmt.Errorf("bluetooth management event of %d bytes is shorter than its header", n)
	}
	ev := mgmtEvent{
		code:  binary.LittleEndian.Uint16(buf[0:]),
		index: binary.LittleEndian.Uint16(buf[2:]),
	}
	end := min(n, mgmtHeaderLen+int(binary.LittleEndian.Uint16(buf[4:])))
	ev.payload = buf[mgmtHeaderLen:end]
	return ev, true, nil
}
