package lifecycle

// feed.go is the opt-in EoX feed (P-D9): Seed asks the Cisco EoX API about the
// product IDs it has inventoried, with the operator's own API client
// credentials. Owner decision 2026-10-14: off by default, asked at first run
// with "no" as the default answer, toggleable in Settings, endpoint
// configurable.

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

// Cisco's EoX API v5, and the OAuth endpoint that issues its access tokens
// for client credentials.
const (
	DefaultEndpoint = "https://apix.cisco.com/supporttools/eox/rest/5"
	DefaultOAuthURL = "https://id.cisco.com/oauth2/default/v1/token"
)

const (
	// idsPerRequest is the most product IDs the EoX API takes in one query.
	idsPerRequest = 20
	// maxPages bounds one query's pagination; twenty IDs fill a few pages.
	maxPages = 50
	// maxResponseBytes bounds one response body, the import's own cap.
	maxResponseBytes = 8 << 20
	// requestTimeout bounds one request, token or page.
	requestTimeout = 30 * time.Second
)

// FeedSettings is the feed's read model. The client secret is never served:
// HasClientSecret says whether one is stored.
type FeedSettings struct {
	Choice          outbound.Choice
	Endpoint        string
	TokenURL        string
	ClientID        string
	HasClientSecret bool
}

// FeedUpdate is the feed's write model. The operator answers on or off; the
// question cannot be un-asked. Empty Endpoint and TokenURL mean Cisco's. An
// empty ClientSecret keeps the stored one; ClearClientSecret removes it.
type FeedUpdate struct {
	Choice            outbound.Choice
	Endpoint          string
	TokenURL          string
	ClientID          string
	ClientSecret      string
	ClearClientSecret bool
}

// ErrInvalidFeed wraps every reason [FeedUpdate.Validate] refuses an update.
var ErrInvalidFeed = errors.New("invalid EoX feed settings")

// Validate refuses an update that could never fetch. hasSecret says whether a
// secret is already stored, which an update may keep without re-typing it.
func (u FeedUpdate) Validate(hasSecret bool) error {
	if err := u.validate(hasSecret); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidFeed, err)
	}
	return nil
}

func (u FeedUpdate) validate(hasSecret bool) error {
	if !u.Choice.ValidAnswer() {
		return errors.New(`choice must be "on" or "off"`)
	}
	if u.ClearClientSecret && u.ClientSecret != "" {
		return errors.New("clientSecret and clearClientSecret are exclusive")
	}
	for _, field := range []struct{ name, value string }{
		{"endpoint", u.Endpoint}, {"tokenUrl", u.TokenURL},
	} {
		if field.value == "" {
			continue
		}
		if err := outbound.ValidateEndpoint(field.value); err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
	}
	secret := u.ClientSecret != "" || (hasSecret && !u.ClearClientSecret)
	if secret && u.ClientID == "" {
		return errors.New("a client secret needs a client ID")
	}
	if u.ClientID != "" && !secret {
		return errors.New("a client ID needs a client secret")
	}
	if u.Choice == outbound.ChoiceOn && u.ClientID == "" && (u.Endpoint == "" || u.Endpoint == DefaultEndpoint) {
		return errors.New("the Cisco EoX API needs a client ID and secret")
	}
	return nil
}

// Credentials is what one fetch uses. An empty ClientID skips the token
// request, for an operator's mirror of the API that needs none.
type Credentials struct {
	Endpoint     string
	TokenURL     string
	ClientID     string
	ClientSecret string
}

// Fetcher queries the EoX API.
type Fetcher struct {
	client *http.Client
}

// NewFetcher returns a fetcher over client, or over a client with a
// per-request timeout that follows no redirects when client is nil: the
// bearer token is for the configured host and must not travel further.
func NewFetcher(client *http.Client) *Fetcher {
	if client == nil {
		client = outbound.NewClient(requestTimeout)
	}
	return &Fetcher{client: client}
}

// Fetch asks the API about productIDs and returns what it knows of them.
func (f *Fetcher) Fetch(ctx context.Context, creds Credentials, productIDs []string) (*Table, error) {
	token := ""
	if creds.ClientID != "" {
		var err error
		if token, err = f.token(ctx, creds); err != nil {
			return nil, err
		}
	}
	var pages []ciscoResponse
	for batch := range slices.Chunk(productIDs, idsPerRequest) {
		escaped := make([]string, len(batch))
		for i, id := range batch {
			escaped[i] = url.PathEscape(id)
		}
		for index := 1; ; index++ {
			if index > maxPages {
				return nil, fmt.Errorf("EoX query ran past %d pages", maxPages)
			}
			page, err := f.page(ctx, creds.Endpoint, token, index, strings.Join(escaped, ","))
			if err != nil {
				return nil, err
			}
			pages = append(pages, page)
			if page.Pagination.LastIndex <= index {
				break
			}
		}
	}
	return tableFromPages(pages)
}

func (f *Fetcher) token(ctx context.Context, creds Credentials) (string, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {creds.ClientID},
		"client_secret": {creds.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if doErr := f.do(req, &body); doErr != nil {
		return "", fmt.Errorf("token request: %w", doErr)
	}
	if body.AccessToken == "" {
		return "", errors.New("token request: response has no access_token")
	}
	return body.AccessToken, nil
}

func (f *Fetcher) page(ctx context.Context, endpoint, token string, index int, ids string) (ciscoResponse, error) {
	target := strings.TrimRight(endpoint, "/") + "/EOXByProductID/" + strconv.Itoa(index) + "/" + ids
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ciscoResponse{}, fmt.Errorf("build EoX request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	var page ciscoResponse
	if doErr := f.do(req, &page); doErr != nil {
		return ciscoResponse{}, fmt.Errorf("EoX request: %w", doErr)
	}
	if page.EOXRecord == nil {
		return ciscoResponse{}, ErrNotEoX
	}
	return page, nil
}

// do sends req and decodes a 200 JSON response into out. Errors name the
// status, never the body: it may echo the request's credentials.
func (f *Fetcher) do(req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if decodeErr := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); decodeErr != nil {
		return fmt.Errorf("decode response: %w", decodeErr)
	}
	return nil
}
