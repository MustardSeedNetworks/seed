//go:build darwin

package detection_test

import (
	"encoding/binary"
	"net"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

// record builds one routing-socket message of msglen bytes with the header
// fields the parser reads, laid out where the kernel puts them.
func record(msgType uint8, index uint16, baud uint64, msglen int) []byte {
	l := detection.IfMsghdr2Layout()
	b := make([]byte, msglen)
	binary.NativeEndian.PutUint16(b[l.Msglen:], uint16(msglen))
	b[l.Type] = msgType
	if msglen > l.Index+2 {
		binary.NativeEndian.PutUint16(b[l.Index:], index)
	}
	if msglen >= l.Baudrate+8 {
		binary.NativeEndian.PutUint64(b[l.Baudrate:], baud)
	}
	return b
}

func ifinfo2(index uint16, baud uint64) []byte {
	return record(syscall.RTM_IFINFO2, index, baud, detection.IfMsghdr2Layout().Size)
}

func TestIfMsghdr2LayoutMatchesKernelABI(t *testing.T) {
	var h unix.IfMsghdr2
	got := detection.IfMsghdr2Layout()
	want := detection.Layout{
		Msglen:   int(unsafe.Offsetof(h.Msglen)),
		Type:     int(unsafe.Offsetof(h.Type)),
		Index:    int(unsafe.Offsetof(h.Index)),
		Baudrate: int(unsafe.Offsetof(h.Data) + unsafe.Offsetof(h.Data.Baudrate)),
		Size:     int(unsafe.Sizeof(h)),
	}
	if got != want {
		t.Fatalf("IfMsghdr2Layout() = %+v, want %+v", got, want)
	}
}

func TestBaudrates(t *testing.T) {
	size := detection.IfMsghdr2Layout().Size
	concat := func(parts ...[]byte) []byte {
		var out []byte
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}

	tests := []struct {
		name string
		rib  []byte
		want map[int]uint64
	}{
		{
			name: "empty dump",
			rib:  nil,
			want: map[int]uint64{},
		},
		{
			name: "single interface",
			rib:  ifinfo2(14, 1_000_000_000),
			want: map[int]uint64{14: 1_000_000_000},
		},
		{
			name: "address records interleaved are skipped by length",
			rib: concat(
				ifinfo2(6, 0),
				record(syscall.RTM_NEWADDR, 6, 0, 40),
				record(syscall.RTM_NEWMADDR2, 6, 0, 24),
				ifinfo2(11, 100_000_000),
			),
			want: map[int]uint64{6: 0, 11: 100_000_000},
		},
		{
			name: "zero msglen terminates the walk",
			rib: concat(
				ifinfo2(1, 10_000_000),
				make([]byte, size),
				ifinfo2(2, 20_000_000),
			),
			want: map[int]uint64{1: 10_000_000},
		},
		{
			name: "msglen past the end of the buffer is not read",
			rib: concat(
				ifinfo2(1, 10_000_000),
				ifinfo2(2, 20_000_000)[:size-1],
			),
			want: map[int]uint64{1: 10_000_000},
		},
		{
			name: "short RTM_IFINFO2 record is skipped",
			rib: concat(
				record(syscall.RTM_IFINFO2, 3, 0, size/2),
				ifinfo2(4, 2_500_000_000),
			),
			want: map[int]uint64{4: 2_500_000_000},
		},
		{
			name: "trailing bytes shorter than a header are ignored",
			rib:  concat(ifinfo2(5, 40_000_000_000), []byte{1, 0}),
			want: map[int]uint64{5: 40_000_000_000},
		},
		{
			name: "same index twice keeps the later record",
			rib:  concat(ifinfo2(7, 100_000_000), ifinfo2(7, 1_000_000_000)),
			want: map[int]uint64{7: 1_000_000_000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detection.Baudrates(tt.rib)
			if len(got) != len(tt.want) {
				t.Fatalf("Baudrates() = %v, want %v", got, tt.want)
			}
			for idx, baud := range tt.want {
				if got[idx] != baud {
					t.Errorf("Baudrates()[%d] = %d, want %d (all: %v)", idx, got[idx], baud, got)
				}
			}
		})
	}
}

// TestInterfaceSpeedReadsTheKernel exercises the real sysctl path: every
// interface on the host resolves without starting a process, loopback has no
// link speed, and an unknown name reports nothing rather than failing.
func TestInterfaceSpeedReadsTheKernel(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces() error = %v", err)
	}
	for _, iface := range ifaces {
		if speed := detection.InterfaceSpeed(iface.Name); speed < 0 {
			t.Errorf("InterfaceSpeed(%q) = %d, want >= 0", iface.Name, speed)
		}
	}
	if speed := detection.InterfaceSpeed("lo0"); speed != 0 {
		t.Errorf("InterfaceSpeed(lo0) = %d, want 0", speed)
	}
	if speed := detection.InterfaceSpeed("no-such-interface"); speed != 0 {
		t.Errorf("InterfaceSpeed(no-such-interface) = %d, want 0", speed)
	}
}

// TestIdentifyByPlatformReportsNothing pins the darwin chipset path to "unknown".
// The system_profiler keyword scan it replaced matched "ice" inside "device" and
// so reported every interface, utun and bridge included, as an Intel E810.
func TestIdentifyByPlatformReportsNothing(t *testing.T) {
	db := detection.NewChipsetDatabase()
	for _, name := range []string{"en0", "utun0", "bridge0"} {
		if info := db.IdentifyByInterface(name, ""); info != nil {
			t.Errorf("IdentifyByInterface(%q) = %s %s, want nil", name, info.Vendor, info.Model)
		}
	}
}
