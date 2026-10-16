// Package meraki reads device state from the Cisco Meraki Dashboard API
// (P-D12). It is an opt-in outbound integration: owner decision 2026-10-14,
// off by default, asked at first run with "no" as the default answer,
// toggleable in Settings, endpoint configurable. See package outbound.
package meraki

import (
	"context"
	"encoding/json"
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

// DefaultEndpoint is the Dashboard API v1 base. Regional dashboards (China,
// India, Canada, FedRAMP) have their own, which the operator configures.
const DefaultEndpoint = "https://api.meraki.com/api/v1"

const (
	// perPage is the largest page the device endpoints serve.
	perPage = 1000
	// maxPages bounds one listing's pagination: a million devices.
	maxPages = 1000
	// maxResponseBytes bounds one response body. A full page of devices is
	// well under 2 MiB.
	maxResponseBytes = 8 << 20
	// requestTimeout bounds one request.
	requestTimeout = 30 * time.Second
)

// Settings is the integration's read model. The API key is never served:
// HasAPIKey says whether one is stored.
type Settings struct {
	Choice    outbound.Choice
	Endpoint  string
	HasAPIKey bool
}

// Update is the integration's write model. The operator answers on or off;
// the question cannot be un-asked. An empty Endpoint means Meraki's. An empty
// APIKey keeps the stored one; ClearAPIKey removes it.
type Update struct {
	Choice      outbound.Choice
	Endpoint    string
	APIKey      string
	ClearAPIKey bool
}

// ErrInvalidSettings wraps every reason [Update.Validate] refuses an update.
var ErrInvalidSettings = errors.New("invalid Meraki settings")

// Validate refuses an update that could never read. hasKey says whether a key
// is already stored, which an update may keep without re-typing it.
func (u Update) Validate(hasKey bool) error {
	if err := u.validate(hasKey); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSettings, err)
	}
	return nil
}

func (u Update) validate(hasKey bool) error {
	if !u.Choice.ValidAnswer() {
		return errors.New(`choice must be "on" or "off"`)
	}
	if u.ClearAPIKey && u.APIKey != "" {
		return errors.New("apiKey and clearApiKey are exclusive")
	}
	if u.Endpoint != "" {
		if err := outbound.ValidateEndpoint(u.Endpoint); err != nil {
			return fmt.Errorf("endpoint: %w", err)
		}
	}
	key := u.APIKey != "" || (hasKey && !u.ClearAPIKey)
	if u.Choice == outbound.ChoiceOn && !key {
		return errors.New("the Dashboard API needs an API key")
	}
	return nil
}

// Credentials is what one read uses.
type Credentials struct {
	Endpoint string
	APIKey   string
}

// Device is one Meraki-managed device and its last reported availability.
type Device struct {
	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	NetworkID        string `json:"networkId,omitempty"`
	Serial           string `json:"serial"`
	Name             string `json:"name,omitempty"`
	Model            string `json:"model,omitempty"`
	ProductType      string `json:"productType,omitempty"`
	MAC              string `json:"mac,omitempty"`
	LANIP            string `json:"lanIp,omitempty"`
	Firmware         string `json:"firmware,omitempty"`
	// Status is online, alerting, offline or dormant as the dashboard last
	// saw the device; empty when the availability listing did not name it.
	Status string `json:"status,omitempty"`
}

// Client reads the Dashboard API.
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

type organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiDevice struct {
	Serial      string `json:"serial"`
	Name        string `json:"name"`
	Model       string `json:"model"`
	ProductType string `json:"productType"`
	MAC         string `json:"mac"`
	LANIP       string `json:"lanIp"`
	Firmware    string `json:"firmware"`
	NetworkID   string `json:"networkId"`
}

type availability struct {
	Serial string `json:"serial"`
	Status string `json:"status"`
}

// Devices lists every device in every organization the key can read, in
// organization then serial order.
func (c *Client) Devices(ctx context.Context, creds Credentials) ([]Device, error) {
	base, err := url.Parse(strings.TrimRight(creds.Endpoint, "/"))
	if err != nil {
		return nil, fmt.Errorf("endpoint: %w", err)
	}
	orgs, err := list[organization](ctx, c, creds.APIKey, base, "/organizations")
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	out := []Device{}
	for _, org := range orgs {
		devices, orgErr := c.orgDevices(ctx, creds.APIKey, base, org)
		if orgErr != nil {
			return nil, fmt.Errorf("organization %s: %w", org.ID, orgErr)
		}
		out = append(out, devices...)
	}
	return out, nil
}

func (c *Client) orgDevices(ctx context.Context, key string, base *url.URL, org organization) ([]Device, error) {
	prefix := "/organizations/" + url.PathEscape(org.ID) + "/devices"
	devices, err := list[apiDevice](ctx, c, key, base, prefix)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	availabilities, err := list[availability](ctx, c, key, base, prefix+"/availabilities")
	if err != nil {
		return nil, fmt.Errorf("list availabilities: %w", err)
	}
	status := make(map[string]string, len(availabilities))
	for _, a := range availabilities {
		status[a.Serial] = a.Status
	}
	out := make([]Device, 0, len(devices))
	for _, d := range devices {
		out = append(out, Device{
			OrganizationID: org.ID, OrganizationName: org.Name, NetworkID: d.NetworkID,
			Serial: d.Serial, Name: d.Name, Model: d.Model, ProductType: d.ProductType,
			MAC: d.MAC, LANIP: d.LANIP, Firmware: d.Firmware, Status: status[d.Serial],
		})
	}
	slices.SortFunc(out, func(a, b Device) int { return strings.Compare(a.Serial, b.Serial) })
	return out, nil
}

// list GETs path under base and follows the Link header's next page. A next
// link to any other host is refused, since following it would send the key
// there.
func list[T any](ctx context.Context, c *Client, key string, base *url.URL, path string) ([]T, error) {
	target := base.JoinPath(path)
	if path != "/organizations" {
		target.RawQuery = url.Values{"perPage": {strconv.Itoa(perPage)}}.Encode()
	}
	var out []T
	for page := 1; ; page++ {
		if page > maxPages {
			return nil, fmt.Errorf("listing ran past %d pages", maxPages)
		}
		var items []T
		next, err := c.page(ctx, key, target.String(), &items)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if next == "" {
			return out, nil
		}
		nextURL, err := url.Parse(next)
		if err != nil || nextURL.Scheme != base.Scheme || nextURL.Host != base.Host {
			return nil, errors.New("next page link leaves the configured endpoint")
		}
		target = nextURL
	}
}

// page GETs one page into out and returns the next page's link. Errors name
// the status, never the body.
func (c *Client) page(ctx context.Context, key, target string, out any) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if decodeErr := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); decodeErr != nil {
		return "", fmt.Errorf("decode response: %w", decodeErr)
	}
	return nextLink(resp.Header.Values("Link")), nil
}

// nextLink returns the rel=next target of an RFC 8288 Link header, or "".
func nextLink(headers []string) string {
	for _, header := range headers {
		for link := range strings.SplitSeq(header, ",") {
			target, params, found := strings.Cut(strings.TrimSpace(link), ";")
			if !found {
				continue
			}
			for param := range strings.SplitSeq(params, ";") {
				name, value, _ := strings.Cut(strings.TrimSpace(param), "=")
				if strings.EqualFold(name, "rel") && strings.Trim(value, `"`) == "next" {
					return strings.Trim(strings.TrimSpace(target), "<>")
				}
			}
		}
	}
	return ""
}
