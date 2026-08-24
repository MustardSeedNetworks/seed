//go:build darwin

package wifihelper_test

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

// The daemon admits a helper only if it satisfies this requirement, so the
// bundle under test must carry the shipped identifier and signing certificate.
const shippedRequirement = `identifier "net.mustardseed.seed.wifihelper"` +
	` and anchor apple generic` +
	` and certificate leaf[subject.OU] = "X6JWYP43HG"`

// TestSignedTrip drives the real signed helper against a real
// listener, which is the only way to exercise audit-token peer verification:
// the check cannot pass for a peer that is not the actual signed bundle, so no
// in-process fake can stand in for it.
//
// Requires a bundle built by deploy/macos/build-helper.sh; skipped otherwise,
// since CI has no signing identity.
func TestSignedTrip(t *testing.T) {
	bundle := os.Getenv("SEED_HELPER_BUNDLE")
	if bundle == "" {
		t.Skip("set SEED_HELPER_BUNDLE to the built Seed Wi-Fi Helper.app to run this")
	}

	binary := filepath.Join(bundle, "Contents", "MacOS", "seed-wifi-helper")
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("helper binary not present: %v", err)
	}

	// Short name deliberately: t.TempDir() embeds it and unix socket paths are
	// capped near 104 bytes.
	socket := filepath.Join(t.TempDir(), "h.sock")

	srv, err := wifihelper.NewServer(socket, shippedRequirement,
		slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	cmd := exec.Command(binary, "-socket", socket)
	if startErr := cmd.Start(); startErr != nil {
		t.Fatalf("start helper: %v", startErr)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// The helper requests Location authorization before connecting, so allow
	// for that wait plus the connection itself.
	deadline := time.Now().Add(30 * time.Second)
	var scanErr error
	for time.Now().Before(deadline) {
		_, scanErr = srv.Scan()
		if !errors.Is(scanErr, wifihelper.ErrNoHelper) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if errors.Is(scanErr, wifihelper.ErrNoHelper) {
		t.Fatal("signed helper never connected, or was rejected by peer verification")
	}

	// Whether the scan itself succeeds depends on the Location grant, which a
	// freshly built bundle will not hold. Either outcome proves the exchange:
	// what must not happen is the helper being refused, or an empty result
	// standing in for a permission problem.
	if scanErr != nil {
		t.Logf("helper connected and answered with: %v", scanErr)
	} else {
		t.Log("helper connected and returned a scan")
	}
}
