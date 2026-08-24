// Package wifihelper carries Wi-Fi scan results from a per-user helper process
// to the seed daemon over a local socket.
//
// macOS grants Location Services authorization — which is what un-redacts SSID
// and BSSID in CoreWLAN results — per user, to a signed application bundle, and
// only in a logged-in GUI session. The daemon runs as root from a LaunchDaemon
// and can hold no such grant, so it cannot scan for itself. A small signed
// helper runs as a LaunchAgent in the user session, holds the grant, and answers
// the daemon's requests.
//
// The daemon listens and the helper connects, rather than the reverse. The
// privileged side must be the one that verifies its peer, and a socket in a
// root-owned directory cannot be squatted by an unprivileged process.
package wifihelper

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Op names a request the daemon can make of the helper.
type Op string

// Requests the helper answers.
const (
	OpScan    Op = "scan"
	OpCurrent Op = "current"
	OpSaved   Op = "saved"
)

// ErrProtocol means a message could not be understood.
var ErrProtocol = errors.New("wifihelper: malformed message")

// Request is a single daemon-to-helper message.
type Request struct {
	Op Op `json:"op"`
}

// Network mirrors the fields the daemon needs from a CoreWLAN observation. It is
// declared here rather than reusing the corewlan type so the wire format does
// not shift when that package gains fields.
type Network struct {
	SSID         string `json:"ssid"`
	BSSID        string `json:"bssid"`
	RSSI         int    `json:"rssi"`
	Noise        int    `json:"noise"`
	Channel      int    `json:"channel"`
	ChannelWidth int    `json:"width"`
	Band         int    `json:"band"`
	PHYMode      string `json:"phyMode,omitempty"`
	Security     string `json:"security"`
}

// Response is a single helper-to-daemon reply. Error carries a failure the
// helper observed, such as the Location grant being absent, so the daemon can
// report why an operator sees no networks instead of reporting none.
type Response struct {
	Networks []Network `json:"networks,omitempty"`
	Names    []string  `json:"names,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// Err converts a response's error field back into an error.
func (r Response) Err() error {
	if r.Error == "" {
		return nil
	}
	return errors.New(r.Error)
}

// DecodeRequest parses a request message, rejecting operations the helper does
// not implement so an unexpected op is not silently answered with an empty set.
func DecodeRequest(line []byte) (Request, error) {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return Request{}, fmt.Errorf("%w: %w", ErrProtocol, err)
	}

	switch req.Op {
	case OpScan, OpCurrent, OpSaved:
		return req, nil
	default:
		return Request{}, fmt.Errorf("%w: unknown op %q", ErrProtocol, req.Op)
	}
}

// DecodeResponse parses a reply message.
func DecodeResponse(line []byte) (Response, error) {
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	return resp, nil
}
