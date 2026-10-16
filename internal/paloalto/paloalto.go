// Package paloalto reads device state from the PAN-OS XML API of a Palo Alto
// Networks firewall or Panorama (P-D12). It is an opt-in outbound
// integration: owner decision 2026-10-14, off by default, asked at first run
// with "no" as the default answer, toggleable in Settings. PAN-OS has no
// default endpoint; the operator names the management address. See package
// outbound.
package paloalto

import (
	"cmp"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

const (
	// RefreshInterval is how often an enabled integration reads device state.
	// A read is one request, or two against Panorama.
	RefreshInterval = 5 * time.Minute

	// maxResponseBytes bounds one response body. Panorama's device listing
	// runs to a few KiB per managed firewall.
	maxResponseBytes = 16 << 20
	// requestTimeout bounds one request.
	requestTimeout = 30 * time.Second

	cmdSystemInfo = "<show><system><info></info></system></show>"
	cmdDevicesAll = "<show><devices><all></all></devices></show>"
)

// Device statuses. The endpoint itself is connected because it answered;
// Panorama reports each managed firewall's connection to it.
const (
	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"
)

// Device is the endpoint, or a firewall the endpoint manages as Panorama.
type Device struct {
	Serial    string `json:"serial"`
	Hostname  string `json:"hostname,omitempty"`
	Model     string `json:"model,omitempty"`
	Family    string `json:"family,omitempty"`
	SWVersion string `json:"swVersion,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	// ManagedBy is the serial of the Panorama that reported the firewall;
	// empty for the endpoint itself.
	ManagedBy string `json:"managedBy,omitempty"`
	Status    string `json:"status"              jsonschema:"enum=connected,enum=disconnected"`
	// HAState is the firewall's high-availability state as Panorama last saw
	// it (active, passive, ...); empty when HA is not configured or unknown.
	HAState string `json:"haState,omitempty"`
}

// Client reads the XML API.
type Client struct {
	http *http.Client
}

// NewClient returns a client over hc, or over [outbound.NewClient] when hc is
// nil: the API key is for the configured host and must not travel further.
func NewClient(hc *http.Client) *Client {
	if hc == nil {
		hc = outbound.NewClient(requestTimeout)
	}
	return &Client{http: hc}
}

type systemInfo struct {
	Hostname   string `xml:"hostname"`
	IPAddress  string `xml:"ip-address"`
	Model      string `xml:"model"`
	Family     string `xml:"family"`
	Serial     string `xml:"serial"`
	SWVersion  string `xml:"sw-version"`
	SystemMode string `xml:"system-mode"`
}

type managedDevice struct {
	Serial    string `xml:"serial"`
	Hostname  string `xml:"hostname"`
	IPAddress string `xml:"ip-address"`
	Model     string `xml:"model"`
	Family    string `xml:"family"`
	SWVersion string `xml:"sw-version"`
	Connected string `xml:"connected"`
	HAState   string `xml:"ha>state"`
}

// Devices reads the endpoint's own identity and, when it is Panorama in a
// mode that manages firewalls, every firewall it manages, in serial order
// after the endpoint.
func (c *Client) Devices(ctx context.Context, creds outbound.Credentials) ([]Device, error) {
	base, err := url.Parse(strings.TrimRight(creds.Endpoint, "/"))
	if err != nil {
		return nil, fmt.Errorf("endpoint: %w", err)
	}
	target := base.JoinPath("api") // PAN-OS serves the XML API at /api/.
	target.Path += "/"

	info, err := op[struct {
		System systemInfo `xml:"system"`
	}](ctx, c, creds.APIKey, target, cmdSystemInfo)
	if err != nil {
		return nil, fmt.Errorf("show system info: %w", err)
	}
	self := info.System
	out := []Device{{
		Serial: self.Serial, Hostname: self.Hostname, Model: self.Model, Family: self.Family,
		SWVersion: self.SWVersion, IPAddress: self.IPAddress, Status: StatusConnected,
	}}
	if self.SystemMode != "panorama" && self.SystemMode != "management-only" {
		return out, nil
	}

	managed, err := op[struct {
		Devices []managedDevice `xml:"devices>entry"`
	}](ctx, c, creds.APIKey, target, cmdDevicesAll)
	if err != nil {
		return nil, fmt.Errorf("show devices all: %w", err)
	}
	firewalls := make([]Device, 0, len(managed.Devices))
	for _, d := range managed.Devices {
		status := StatusDisconnected
		if d.Connected == "yes" {
			status = StatusConnected
		}
		firewalls = append(firewalls, Device{
			Serial: d.Serial, Hostname: d.Hostname, Model: d.Model, Family: d.Family,
			SWVersion: d.SWVersion, IPAddress: d.IPAddress, ManagedBy: self.Serial,
			Status: status, HAState: d.HAState,
		})
	}
	slices.SortFunc(firewalls, func(a, b Device) int { return cmp.Compare(a.Serial, b.Serial) })
	return append(out, firewalls...), nil
}

// response is the XML API envelope. Status is "success" or "error"; an error
// carries a numeric code.
type response[T any] struct {
	Status string `xml:"status,attr"`
	Code   string `xml:"code,attr"`
	Result T      `xml:"result"`
}

// op runs one operational command and returns its decoded result. Errors name
// the HTTP status or API error code, never the body.
func op[T any](ctx context.Context, c *Client, key string, target *url.URL, cmd string) (T, error) {
	var zero T
	u := *target
	u.RawQuery = url.Values{"type": {"op"}, "cmd": {cmd}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Pan-Key", key)
	resp, err := c.http.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var env response[T]
	if decodeErr := xml.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&env); decodeErr != nil {
		return zero, fmt.Errorf("decode response: %w", decodeErr)
	}
	if env.Status != "success" {
		// The code is the only part of an error quoted: it is a number.
		if code, convErr := strconv.Atoi(env.Code); convErr == nil {
			return zero, fmt.Errorf("API error code %d", code)
		}
		return zero, errors.New("API error")
	}
	return env.Result, nil
}
