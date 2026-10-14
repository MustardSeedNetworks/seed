//go:build linux

package enumerate

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseLEScanOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   map[string]string // address -> name
	}{
		{
			name:   "header only",
			output: "LE Scan ...\n",
			want:   map[string]string{},
		},
		{
			name: "named and unknown devices",
			output: "LE Scan ...\n" +
				"AA:BB:CC:DD:EE:01 Kitchen Speaker\n" +
				"AA:BB:CC:DD:EE:02 (unknown)\n",
			want: map[string]string{
				"AA:BB:CC:DD:EE:01": "Kitchen Speaker",
				"AA:BB:CC:DD:EE:02": "",
			},
		},
		{
			name: "duplicate fills a missing name",
			output: "AA:BB:CC:DD:EE:01 (unknown)\n" +
				"AA:BB:CC:DD:EE:01 Badge Reader\n",
			want: map[string]string{"AA:BB:CC:DD:EE:01": "Badge Reader"},
		},
		{
			name: "duplicate never replaces a name",
			output: "AA:BB:CC:DD:EE:01 Badge Reader\n" +
				"AA:BB:CC:DD:EE:01 (unknown)\n" +
				"AA:BB:CC:DD:EE:01 Other\n",
			want: map[string]string{"AA:BB:CC:DD:EE:01": "Badge Reader"},
		},
		{
			name: "padding trimmed, malformed lines skipped",
			output: "  AA:BB:CC:DD:EE:03  \n" +
				"AA:BB:CC:DD:EE Short\n" +
				"ZZ:BB:CC:DD:EE:04 NotHex\n" +
				"\n",
			want: map[string]string{"AA:BB:CC:DD:EE:03": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			devices := parseLEScanOutput(strings.NewReader(tt.output))

			got := make(map[string]string, len(devices))
			for _, d := range devices {
				got[d.Address] = d.Name
				checkLEScanDevice(t, d, before)
			}
			if len(devices) != len(got) || !maps.Equal(got, tt.want) {
				t.Errorf("got %d devices %v, want %v", len(devices), got, tt.want)
			}
		})
	}
}

func checkLEScanDevice(t *testing.T, d BluetoothDevice, scanStart time.Time) {
	t.Helper()
	if d.Type != BluetoothTypeBLE || d.DeviceClass != BluetoothClassUncategorized ||
		!d.IsConnectable || d.Metadata == nil {
		t.Errorf("%s: unexpected fields %+v", d.Address, d)
	}
	if d.FirstSeen.Before(scanStart) || d.LastSeen.Before(d.FirstSeen) {
		t.Errorf("%s: FirstSeen %v LastSeen %v, scan began %v",
			d.Address, d.FirstSeen, d.LastSeen, scanStart)
	}
}

func TestLEScanArgs(t *testing.T) {
	if got := lescanArgs(""); !slices.Equal(got, []string{"lescan", "--duplicates"}) {
		t.Errorf("default adapter: got %v", got)
	}
	if got := lescanArgs("hci1"); !slices.Equal(got, []string{"-i", "hci1", "lescan", "--duplicates"}) {
		t.Errorf("hci1: got %v", got)
	}
}

func TestBLEScanDuration(t *testing.T) {
	if got := bleScanDuration(&BluetoothScanConfig{}); got != defaultBLEScanSeconds*time.Second {
		t.Errorf("unset: got %v", got)
	}
	if got := bleScanDuration(&BluetoothScanConfig{ScanDurationSec: 12}); got != 12*time.Second {
		t.Errorf("12 s: got %v", got)
	}
}
