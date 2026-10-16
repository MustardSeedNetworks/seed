package outbound

import (
	"errors"
	"fmt"
)

// Credentials is what one read of a vendor management API uses.
type Credentials struct {
	Endpoint string
	APIKey   string
}

// Settings is a vendor API integration's read model. The API key is never
// served: HasAPIKey says whether one is stored.
type Settings struct {
	Choice    Choice
	Endpoint  string
	HasAPIKey bool
}

// Update is a vendor API integration's write model. The operator answers on
// or off; the question cannot be un-asked. An empty Endpoint means the
// vendor's default, where it has one. An empty APIKey keeps the stored one;
// ClearAPIKey removes it.
type Update struct {
	Choice      Choice
	Endpoint    string
	APIKey      string
	ClearAPIKey bool
}

// ErrInvalidSettings wraps every reason [Update.Validate] refuses an update.
var ErrInvalidSettings = errors.New("invalid integration settings")

// Validate refuses an update that could never read. hasKey says whether a key
// is already stored, which an update may keep without re-typing it.
// defaultEndpoint is the vendor's own endpoint, or "" when it has none and the
// operator must name one.
func (u Update) Validate(hasKey bool, defaultEndpoint string) error {
	if err := u.validate(hasKey, defaultEndpoint); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSettings, err)
	}
	return nil
}

func (u Update) validate(hasKey bool, defaultEndpoint string) error {
	if !u.Choice.ValidAnswer() {
		return errors.New(`choice must be "on" or "off"`)
	}
	if u.ClearAPIKey && u.APIKey != "" {
		return errors.New("apiKey and clearApiKey are exclusive")
	}
	if u.Endpoint != "" {
		if err := ValidateEndpoint(u.Endpoint); err != nil {
			return fmt.Errorf("endpoint: %w", err)
		}
	}
	if u.Choice != ChoiceOn {
		return nil
	}
	if u.APIKey == "" && (!hasKey || u.ClearAPIKey) {
		return errors.New("the API needs an API key")
	}
	if u.Endpoint == "" && defaultEndpoint == "" {
		return errors.New("the API needs an endpoint")
	}
	return nil
}
