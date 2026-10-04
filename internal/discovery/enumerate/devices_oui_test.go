package enumerate_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// Construction is on the daemon's startup path, ahead of the listener, so it
// reads local files only (seed#2970). A configured file layers over the
// embedded registry; a missing one is left missing, not downloaded.
func TestNewDeviceDiscoveryWithOUIReadsLocalFilesOnly(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present.txt")
	if err := os.WriteFile(present, []byte("FC-FF-AB   (hex)\t\tOn-Disk Vendor\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.txt")

	tests := []struct {
		name, path, mac, want string
	}{
		{"configured file layers over the embedded registry", present, "FC:FF:AB:00:00:01", "On-Disk Vendor"},
		{"missing file keeps the embedded registry", missing, "00:00:0C:00:00:01", "Cisco Systems, Inc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := enumerate.NewDeviceDiscoveryWithOUI(loopbackName(t), tc.path)
			if got := d.GetOUIDatabase().Lookup(tc.mac); got != tc.want {
				t.Errorf("Lookup(%s) = %q, want %q", tc.mac, got, tc.want)
			}
		})
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("construction wrote %s (stat err %v); it must not download", missing, err)
	}
}
