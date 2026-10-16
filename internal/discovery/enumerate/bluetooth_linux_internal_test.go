//go:build linux

package enumerate

import (
	"context"
	"encoding/binary"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeMgmt answers management commands the way the kernel does and records
// what was sent. Settings commands change its state, so a test can check the
// controller is left as it was found.
type fakeMgmt struct {
	settings  uint32
	indexes   []uint16
	sent      []uint16
	queue     []mgmtEvent
	statuses  map[uint16]byte // a non-success reply per opcode
	onStart   []mgmtEvent     // events delivered once, after Start Discovery succeeds
	interrupt int             // recv calls that return nothing, as a signal-interrupted poll does
}

func (f *fakeMgmt) send(op, index uint16, params []byte) error {
	f.sent = append(f.sent, op)
	status := f.statuses[op]
	reply := binary.LittleEndian.AppendUint16(nil, op)
	reply = append(reply, status)
	if status == mgmtStatusSuccess {
		reply = append(reply, f.replyData(op, params)...)
	}
	f.queue = append(f.queue, mgmtEvent{code: mgmtEvCmdComplete, index: index, payload: reply})
	if op == mgmtOpStartDiscovery && status == mgmtStatusSuccess {
		f.queue = append(f.queue, f.onStart...)
		f.onStart = nil
	}
	return nil
}

func (f *fakeMgmt) replyData(op uint16, params []byte) []byte {
	switch op {
	case mgmtOpReadIndexList:
		out := binary.LittleEndian.AppendUint16(nil, uint16(len(f.indexes)))
		for _, i := range f.indexes {
			out = binary.LittleEndian.AppendUint16(out, i)
		}
		return out
	case mgmtOpReadInfo:
		info := make([]byte, mgmtInfoMinLen)
		binary.LittleEndian.PutUint32(info[mgmtInfoCurrentOffset:], f.settings)
		return info
	case mgmtOpSetPowered:
		f.toggle(mgmtSettingPowered, params[0] == 1)
	case mgmtOpSetLE:
		f.toggle(mgmtSettingLE, params[0] == 1)
	}
	return nil
}

func (f *fakeMgmt) toggle(bit uint32, on bool) {
	if on {
		f.settings |= bit
	} else {
		f.settings &^= bit
	}
}

func (f *fakeMgmt) recv(timeout time.Duration) (mgmtEvent, bool, error) {
	if f.interrupt > 0 {
		f.interrupt--
		return mgmtEvent{}, false, nil
	}
	if len(f.queue) == 0 {
		time.Sleep(timeout)
		return mgmtEvent{}, false, nil
	}
	ev := f.queue[0]
	f.queue = f.queue[1:]
	return ev, true, nil
}

// deviceFoundEvent builds a Device Found event for hci0.
func deviceFoundEvent(addr [6]byte, addrType byte, rssi int8, flags uint32, eir []byte) mgmtEvent {
	p := append([]byte{}, addr[:]...)
	p = append(p, addrType, byte(rssi))
	p = binary.LittleEndian.AppendUint32(p, flags)
	p = binary.LittleEndian.AppendUint16(p, uint16(len(eir)))
	return mgmtEvent{code: mgmtEvDeviceFound, payload: append(p, eir...)}
}

func newTestSession(f *fakeMgmt) *mgmtSession {
	return &mgmtSession{conn: f, devices: make(map[string]*BluetoothDevice)}
}

// Device addresses as they appear on the wire, little-endian: a random LE
// address heard on hci0, and a public one.
func addrRandom() [6]byte { return [6]byte{0x3d, 0xa3, 0x2e, 0x03, 0x8e, 0x7d} }
func addrPublic() [6]byte { return [6]byte{0x01, 0x00, 0x00, 0xcf, 0xda, 0x74} }

// TestMgmtScanEnumeratesWithSignalAndVendor is the acceptance in miniature:
// every device heard comes back with its RSSI, address type and manufacturer.
func TestMgmtScanEnumeratesWithSignalAndVendor(t *testing.T) {
	f := &fakeMgmt{
		settings: mgmtSettingPowered | mgmtSettingLE,
		onStart: []mgmtEvent{
			deviceFoundEvent(addrRandom(), mgmtAddrLERandom, -90, 0,
				mustHex(t, "02011a020a080bff4c0010063f1e18958e24")),
			deviceFoundEvent(addrPublic(), 1, -74, mgmtFoundNotConnectable, mustHex(t, "0a0953656e736f72203132")),
			deviceFoundEvent(addrRandom(), mgmtAddrLERandom, mgmtRSSIUnavailable, 0, nil),
		},
	}
	s := newTestSession(f)
	if err := s.scan(context.Background(), mgmtScanLE, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	got := s.result()
	if len(got) != 2 {
		t.Fatalf("got %d devices, want 2: %+v", len(got), got)
	}

	apple, sensor := got[1], got[0]
	if apple.Address != "7D:8E:03:2E:A3:3D" || apple.AddressType != BluetoothAddressRandom ||
		apple.Type != BluetoothTypeBLE || apple.RSSI != -90 || apple.TxPower != 8 ||
		apple.ManufacturerID != 0x004C || !apple.IsConnectable {
		t.Errorf("random-address device: %+v", apple)
	}
	if sensor.Address != "74:DA:CF:00:00:01" || sensor.AddressType != BluetoothAddressPublic ||
		sensor.Name != "Sensor 12" || sensor.RSSI != -74 || sensor.IsConnectable {
		t.Errorf("public-address device: %+v", sensor)
	}
	if !slices.Contains(f.sent, mgmtOpStopDiscovery) {
		t.Errorf("discovery was not stopped: sent %#x", f.sent)
	}
}

// TestMgmtScanRestoresControllerSettings pins that a scan on a controller
// bluetoothd never touched powers it and enables LE only for the scan.
func TestMgmtScanRestoresControllerSettings(t *testing.T) {
	for _, tc := range []struct {
		name      string
		settings  uint32
		scanTypes byte
		wantSent  []uint16
	}{
		{
			name:      "off, LE disabled",
			settings:  0,
			scanTypes: mgmtScanLE,
			wantSent: []uint16{
				mgmtOpReadInfo, mgmtOpSetLE, mgmtOpSetPowered, mgmtOpStartDiscovery,
				mgmtOpStopDiscovery, mgmtOpSetPowered, mgmtOpSetLE,
			},
		},
		{
			name:      "classic only leaves LE alone",
			settings:  0,
			scanTypes: mgmtScanBREDR,
			wantSent: []uint16{
				mgmtOpReadInfo, mgmtOpSetPowered, mgmtOpStartDiscovery,
				mgmtOpStopDiscovery, mgmtOpSetPowered,
			},
		},
		{
			name:      "already on",
			settings:  mgmtSettingPowered | mgmtSettingLE,
			scanTypes: mgmtScanBREDR | mgmtScanLE,
			wantSent:  []uint16{mgmtOpReadInfo, mgmtOpStartDiscovery, mgmtOpStopDiscovery},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeMgmt{settings: tc.settings}
			if err := newTestSession(f).scan(context.Background(), tc.scanTypes, 20*time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(f.sent, tc.wantSent) {
				t.Errorf("sent %#x, want %#x", f.sent, tc.wantSent)
			}
			if f.settings != tc.settings {
				t.Errorf("settings left %#x, found %#x", f.settings, tc.settings)
			}
		})
	}
}

// TestMgmtScanRestoresSettingsOnFailure: a discovery the controller refuses
// still puts power and LE back.
func TestMgmtScanRestoresSettingsOnFailure(t *testing.T) {
	f := &fakeMgmt{statuses: map[uint16]byte{mgmtOpStartDiscovery: mgmtStatusNotSupported}}
	err := newTestSession(f).scan(context.Background(), mgmtScanLE, 20*time.Millisecond)
	if statusErr, ok := errors.AsType[*mgmtStatusError](err); !ok || statusErr.status != mgmtStatusNotSupported {
		t.Fatalf("err = %v, want not supported", err)
	}
	if f.settings != 0 {
		t.Errorf("settings left %#x, want 0", f.settings)
	}
}

// TestMgmtScanJoinsBusyDiscovery: when bluetoothd is already discovering, the
// scan listens to its results and does not stop a discovery it did not start.
func TestMgmtScanJoinsBusyDiscovery(t *testing.T) {
	f := &fakeMgmt{
		settings: mgmtSettingPowered | mgmtSettingLE,
		statuses: map[uint16]byte{mgmtOpStartDiscovery: mgmtStatusBusy},
		queue:    []mgmtEvent{deviceFoundEvent(addrRandom(), mgmtAddrLERandom, -60, 0, nil)},
	}
	s := newTestSession(f)
	if err := s.scan(context.Background(), mgmtScanLE, 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got := s.result(); len(got) != 1 || got[0].RSSI != -60 {
		t.Errorf("got %+v, want the device bluetoothd's discovery heard", got)
	}
	if slices.Contains(f.sent, mgmtOpStopDiscovery) {
		t.Errorf("stopped another client's discovery: sent %#x", f.sent)
	}
}

// TestMgmtScanRestartsEndedDiscovery: the kernel ends a discovery after about
// ten seconds; a longer scan starts another.
func TestMgmtScanRestartsEndedDiscovery(t *testing.T) {
	f := &fakeMgmt{
		settings: mgmtSettingPowered | mgmtSettingLE,
		onStart:  []mgmtEvent{{code: mgmtEvDiscovering, payload: []byte{mgmtScanLE, 0}}},
	}
	if err := newTestSession(f).scan(context.Background(), mgmtScanLE, 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	starts := 0
	for _, op := range f.sent {
		if op == mgmtOpStartDiscovery {
			starts++
		}
	}
	if starts != 2 {
		t.Errorf("discovery started %d times, want 2", starts)
	}
}

// TestMgmtCommandSurvivesInterruptedWait: the Go runtime's preemption signal
// interrupts poll. That is not a timeout, and a command must keep waiting for
// its reply. Live on hci1 this failed every restarted discovery.
func TestMgmtCommandSurvivesInterruptedWait(t *testing.T) {
	f := &fakeMgmt{indexes: []uint16{0}, interrupt: 3}
	if _, err := newTestSession(f).command(mgmtOpReadIndexList, mgmtIndexNone, nil); err != nil {
		t.Fatal(err)
	}
}

// TestDeviceFoundMergesDualMode: one address heard over BR/EDR and LE is one
// dual-mode device.
func TestDeviceFoundMergesDualMode(t *testing.T) {
	s := newTestSession(&fakeMgmt{})
	now := time.Now()
	s.handle(deviceFoundEvent(addrPublic(), mgmtAddrBREDR, -50, 0, mustHex(t, "040d0c025a")))
	s.handle(deviceFoundEvent(addrPublic(), 1, -55, mgmtFoundNotConnectable, mustHex(t, "0319c103")))
	got := s.result()
	if len(got) != 1 {
		t.Fatalf("got %d devices, want 1", len(got))
	}
	d := got[0]
	if d.Type != BluetoothTypeDual || d.RSSI != -55 || !d.IsConnectable ||
		d.DeviceClass != BluetoothClassPhone || d.FirstSeen.Before(now) {
		t.Errorf("got %+v", d)
	}
}

func TestDeviceFoundIgnoresTruncatedAndForeignEvents(t *testing.T) {
	s := newTestSession(&fakeMgmt{})
	ev := deviceFoundEvent(addrPublic(), 1, -55, 0, mustHex(t, "020af4"))
	s.handle(mgmtEvent{code: ev.code, payload: ev.payload[:len(ev.payload)-1]})
	s.handle(mgmtEvent{code: ev.code, index: 1, payload: ev.payload})
	if got := s.result(); len(got) != 0 {
		t.Errorf("got %+v, want nothing", got)
	}
}

func TestResolveIndex(t *testing.T) {
	for _, tc := range []struct {
		name    string
		adapter string
		indexes []uint16
		want    uint16
		wantErr string
	}{
		{name: "first controller", indexes: []uint16{2, 3}, want: 2},
		{name: "named controller", adapter: "hci1", want: 1},
		{name: "no controller", wantErr: "no Bluetooth adapter"},
		{name: "not an hci name", adapter: "wlan0", wantErr: "not an hciN name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newTestSession(&fakeMgmt{indexes: tc.indexes}).resolveIndex(tc.adapter)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("got %d, %v; want %d", got, err, tc.want)
			}
		})
	}
}

func TestMgmtStatusErrorNamesPermission(t *testing.T) {
	err := &mgmtStatusError{op: mgmtOpStartDiscovery, status: mgmtStatusPermissionDenied}
	if !strings.Contains(err.Error(), "CAP_NET_ADMIN") {
		t.Errorf("error %q does not say what is missing", err)
	}
}

func TestMgmtScanTypes(t *testing.T) {
	for _, tc := range []struct {
		classic, ble bool
		want         byte
	}{
		{true, true, mgmtScanBREDR | mgmtScanLE},
		{true, false, mgmtScanBREDR},
		{false, true, mgmtScanLE},
	} {
		got, err := mgmtScanTypes(&BluetoothScanConfig{IncludeClassic: tc.classic, IncludeBLE: tc.ble})
		if err != nil || got != tc.want {
			t.Errorf("classic=%v ble=%v: got %#x, %v; want %#x", tc.classic, tc.ble, got, err, tc.want)
		}
	}
	if _, err := mgmtScanTypes(&BluetoothScanConfig{}); err == nil {
		t.Error("a scan of nothing was accepted")
	}
}

// TestApplyBluetoothctlState: bluetoothd contributes pairing state to devices
// heard in range, and connected devices, which no longer advertise. Its cache
// of devices seen in the past is not reported as present.
func TestApplyBluetoothctlState(t *testing.T) {
	heard := []BluetoothDevice{{Address: "74:DA:CF:00:00:01", RSSI: -60}}
	known := []BluetoothDevice{
		{Address: "74:da:cf:00:00:01", Name: "Keyboard", Alias: "Desk keyboard", IsPaired: true, IsTrusted: true},
		{Address: "AA:BB:CC:00:00:02", Name: "Headset", IsConnected: true},
		{Address: "AA:BB:CC:00:00:03", Name: "Old phone", IsPaired: true},
	}
	got := applyBluetoothctlState(heard, known)
	if len(got) != 2 {
		t.Fatalf("got %d devices, want 2: %+v", len(got), got)
	}
	if d := got[0]; d.Name != "Keyboard" || d.Alias != "Desk keyboard" || !d.IsPaired || !d.IsTrusted || d.RSSI != -60 {
		t.Errorf("heard device: %+v", d)
	}
	if got[1].Name != "Headset" {
		t.Errorf("connected device missing: %+v", got[1])
	}
}
