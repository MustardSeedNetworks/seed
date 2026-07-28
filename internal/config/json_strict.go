package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func decodeJSONStrict(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}

func rejectRemovedTLSEnvironment() error {
	for _, name := range []string{
		"SEED_HTTP_PORT",
		"SEED_HTTPS_ENABLED",
		"SEED_HTTP_REDIRECT_PORT",
		"SEED_ACME_ENABLED",
		"SEED_ACME_DOMAIN",
		"SEED_ACME_EMAIL",
		"SEED_ACME_CACHE_DIR",
		"SEED_ACME_STAGING",
	} {
		if _, exists := os.LookupEnv(name); exists {
			return fmt.Errorf(
				"%s is no longer supported; Seed always serves HTTPS and supports only operator-provided or self-signed certificates",
				name,
			)
		}
	}
	return nil
}

type configJSON Config

// UnmarshalJSON rejects stale or misspelled configuration instead of silently
// ignoring it. Seed is pre-v1, so removed settings are errors rather than
// compatibility aliases.
func (c *Config) UnmarshalJSON(data []byte) error {
	return decodeJSONStrict(data, (*configJSON)(c))
}
