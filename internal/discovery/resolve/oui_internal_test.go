package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

// The startup load caches its result per path, including "not found". A
// refresh that then downloads to that path must serve the new file, both to
// the database it refreshed and to any later load of the same path (#2970).
func TestReloadFromIEEEFormatReplacesTheCachedParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oui.txt")

	startup := NewOUIDatabase()
	if err := startup.LoadFromIEEEFormat(path); err == nil {
		t.Fatal("LoadFromIEEEFormat of a missing file succeeded")
	}

	const downloaded = "FC-FF-AA   (hex)\t\tFreshly Downloaded Vendor\n"
	if err := os.WriteFile(path, []byte(downloaded), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := startup.reloadFromIEEEFormat(path); err != nil {
		t.Fatalf("reloadFromIEEEFormat: %v", err)
	}
	if got := startup.Lookup("FC:FF:AA:00:00:01"); got != "Freshly Downloaded Vendor" {
		t.Errorf("refreshed database Lookup = %q, want the downloaded vendor", got)
	}

	later := NewOUIDatabase()
	if err := later.LoadFromIEEEFormat(path); err != nil {
		t.Fatalf("LoadFromIEEEFormat after reload: %v", err)
	}
	if got := later.Lookup("FC:FF:AA:00:00:01"); got != "Freshly Downloaded Vendor" {
		t.Errorf("later load Lookup = %q, want the downloaded vendor", got)
	}
}
