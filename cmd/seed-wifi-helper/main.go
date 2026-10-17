//go:build darwin

// Command seed-wifi-helper answers the seed daemon's Wi-Fi and Bluetooth LE
// requests from within a user's login session.
//
// macOS grants Location Services authorization — which is what un-redacts SSID
// and BSSID in CoreWLAN results — per user, to a signed application bundle, and
// only in a logged-in GUI session. Seed itself runs as root from a LaunchDaemon
// and can hold no such grant. This helper runs as a LaunchAgent inside the
// session, holds the grant, and does the scanning on the daemon's behalf.
// CoreBluetooth is granted the same way, so LE scans run here too.
//
// It ships inside an application bundle because an unbundled executable cannot
// be granted the permission at all.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/corebluetooth"
	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

const (
	defaultSocketPath = "/var/run/seed/wifi-helper.sock"

	// authorizationWait bounds the startup permission request. The prompt
	// appears within a second or two when it appears at all, and a decision may
	// never come — the operator can leave the dialog sitting — so this only
	// bounds how long startup blocks, not how long the grant takes.
	authorizationWait = 5 * time.Second

	// reconnectDelay paces retries when the daemon is not listening yet, which
	// is normal at login: the agent may start before the daemon is ready.
	reconnectDelay = 5 * time.Second
)

func main() {
	socketPath := flag.String("socket", defaultSocketPath, "path to the seed daemon's Wi-Fi helper socket")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	reportAuthorization(ctx, log)
	reportBluetoothAuthorization(ctx, log)
	run(ctx, *socketPath, log)
}

// reportAuthorization asks for the Location grant and records the outcome.
//
// Requesting is what registers this bundle with locationd, which is what makes
// it appear in System Settings at all — so it is called on every start even
// when the answer is already known.
//
// A background agent does get prompted, provided the bundle carries the
// com.apple.security.personal-information.location entitlement. Without it
// locationd registers the client and then declines to ask, which looks
// identical to macOS refusing to prompt agents. A not-determined result here
// therefore means either the operator dismissed the dialog or the bundle was
// signed without that entitlement; both are logged rather than surfacing later
// as an empty scan.
func reportAuthorization(ctx context.Context, log *slog.Logger) {
	status := corewlan.RequestAuthorization(authorizationWait)
	if status == corewlan.AuthAuthorized {
		log.InfoContext(ctx, "location services authorized",
			slog.String("event", "wifi.helper.authorized"))
		return
	}

	log.WarnContext(ctx, "location services not authorized; scans will not report network names",
		slog.String("event", "wifi.helper.unauthorized"),
		slog.String("status", status.String()),
		slog.String("remedy", "enable Seed Wi-Fi Helper in System Settings > Privacy & Security > Location Services"))
}

// reportBluetoothAuthorization records the Bluetooth grant, asking for it first
// when the operator has not yet decided. CoreBluetooth has no request call:
// creating a central manager, which a scan does, is what shows the prompt. So a
// short scan at login puts the question in front of the operator then, rather
// than at the first scan the daemon asks for.
func reportBluetoothAuthorization(ctx context.Context, log *slog.Logger) {
	if corebluetooth.AuthorizationStatus() == corebluetooth.AuthNotDetermined {
		_, _ = corebluetooth.Scan(authorizationWait)
	}

	status := corebluetooth.AuthorizationStatus()
	if status == corebluetooth.AuthAllowedAlways {
		log.InfoContext(ctx, "bluetooth authorized",
			slog.String("event", "ble.helper.authorized"))
		return
	}

	log.WarnContext(ctx, "bluetooth not authorized; LE scans will report the missing grant",
		slog.String("event", "ble.helper.unauthorized"),
		slog.String("status", status.String()),
		slog.String("remedy", "enable Seed Wi-Fi Helper in System Settings > Privacy & Security > Bluetooth"))
}

// run keeps a session with the daemon for as long as the process lives.
func run(ctx context.Context, socketPath string, log *slog.Logger) {
	for ctx.Err() == nil {
		if err := wifihelper.Serve(ctx, socketPath, log); err != nil && ctx.Err() == nil {
			log.WarnContext(ctx, "helper session ended",
				slog.String("event", "wifi.helper.disconnected"),
				slog.String("error", err.Error()))
		}

		select {
		case <-ctx.Done():
		case <-time.After(reconnectDelay):
		}
	}

	log.InfoContext(ctx, "helper stopped", slog.String("event", "wifi.helper.stopped"))
}
