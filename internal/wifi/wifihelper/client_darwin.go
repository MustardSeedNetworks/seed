//go:build darwin

package wifihelper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"
)

// Serve connects to the daemon's socket and answers its requests until the
// connection ends or ctx is cancelled.
//
// This is the helper side: it runs in the user's login session, where the
// Location Services grant lives, and does the CoreWLAN work the daemon cannot.
func Serve(ctx context.Context, socketPath string, log *slog.Logger) error {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return fmt.Errorf("wifihelper: dial daemon: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Unblock the read loop when the caller cancels.
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	log.InfoContext(ctx, "connected to seed daemon", slog.String("event", "wifi.helper.serving"))

	rd := bufio.NewReader(conn)
	for {
		line, readErr := rd.ReadBytes('\n')
		if readErr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("wifihelper: read request: %w", readErr)
		}

		resp := answer(ctx, line, log)

		payload, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return fmt.Errorf("wifihelper: encode reply: %w", marshalErr)
		}
		if _, writeErr := conn.Write(append(payload, '\n')); writeErr != nil {
			return fmt.Errorf("wifihelper: send reply: %w", writeErr)
		}
	}
}

// answer performs one request. Failures are returned to the daemon in the reply
// rather than ending the session: a missing Location grant is a condition an
// operator must fix, not a reason to drop the connection and retry forever.
func answer(ctx context.Context, line []byte, log *slog.Logger) Response {
	req, err := DecodeRequest(line)
	if err != nil {
		log.WarnContext(ctx, "rejected malformed request",
			slog.String("event", "wifi.helper.bad_request"),
			slog.String("error", err.Error()))
		return Response{Error: err.Error()}
	}

	switch req.Op {
	case OpScan:
		return scanResponse()
	case OpCurrent:
		return currentResponse()
	case OpSaved:
		return savedResponse()
	default:
		// DecodeRequest already refuses unknown ops.
		return Response{Error: fmt.Sprintf("unsupported op %q", req.Op)}
	}
}

func scanResponse() Response {
	found, err := corewlan.Scan()
	if err != nil {
		return Response{Error: err.Error()}
	}

	networks := make([]Network, 0, len(found))
	for _, n := range found {
		networks = append(networks, fromCoreWLAN(n))
	}
	return Response{Networks: networks}
}

func currentResponse() Response {
	current, err := corewlan.Current()
	if err != nil {
		if errors.Is(err, corewlan.ErrNotAssociated) {
			return Response{} // an absence, not a failure
		}
		return Response{Error: err.Error()}
	}
	return Response{Networks: []Network{fromCoreWLAN(*current)}}
}

func savedResponse() Response {
	names, err := corewlan.SavedNetworks()
	if err != nil {
		return Response{Error: err.Error()}
	}
	return Response{Names: names}
}

func fromCoreWLAN(n corewlan.Network) Network {
	return Network{
		SSID:         n.SSID,
		BSSID:        n.BSSID,
		RSSI:         n.RSSI,
		Noise:        n.Noise,
		Channel:      n.Channel,
		ChannelWidth: n.ChannelWidth,
		Band:         int(n.Band),
		PHYMode:      n.PHYMode,
		Security:     n.Security,
	}
}
