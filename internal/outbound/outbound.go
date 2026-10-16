// Package outbound holds what every opt-in outbound integration shares: the
// operator's answer to whether Seed may call out, the endpoint rule, and the
// HTTP client the call goes through.
//
// Hard rule #3 (owner 2026-09-12, 2026-10-14): a feature that calls out ships
// off, the operator is asked once with "no" as the default answer, and the
// answer stays toggleable in Settings. Until it is "on", the feature builds no
// request at all, so a network-blocked install makes zero outbound attempts.
package outbound

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

// Choice is the operator's answer to whether Seed may call out for one
// integration.
type Choice string

// The answers. Unanswered calls nothing, exactly like Off; the difference is
// only that the operator has not been asked yet.
const (
	ChoiceUnanswered Choice = "unanswered"
	ChoiceOff        Choice = "off"
	ChoiceOn         Choice = "on"
)

// ValidAnswer reports whether c is an answer an operator can give. The
// question cannot be un-asked, so unanswered is not one.
func (c Choice) ValidAnswer() bool {
	return c == ChoiceOn || c == ChoiceOff
}

// ValidateEndpoint accepts an absolute https URL with no credentials, query or
// fragment. Every integration sends a secret with its requests, so plain http
// is refused.
func ValidateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("not a URL")
	}
	switch {
	case u.Scheme != "https":
		return errors.New("must be https")
	case u.Hostname() == "":
		return errors.New("must name a host")
	case u.User != nil:
		return errors.New("must not carry credentials")
	case u.RawQuery != "" || u.Fragment != "":
		return errors.New("must not carry a query or fragment")
	}
	return nil
}

// NewClient returns a client with a per-request timeout that follows no
// redirects: the secret a request carries is for the configured host and must
// not travel further.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
