package config_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestIdentityValidation holds the two places an identity is checked to one
// answer: Validate (the settings write and startup) and the schema (a
// hand-edited file). A value one accepts and the other refuses would save
// through the API and then fail the file check, or the other way round.
func TestIdentityValidation(t *testing.T) {
	cases := []struct {
		name     string
		identity config.IdentityConfig
		wantErr  string
	}{
		{"unset", config.IdentityConfig{}, ""},
		{"typical", config.IdentityConfig{Name: "seed-idf-3b", Location: "Main Office, IDF 3B, Port 21"}, ""},
		{"name at the limit", config.IdentityConfig{Name: strings.Repeat("a", config.IdentityNameMaxRunes)}, ""},
		{
			// The limit counts characters, not bytes: 64 two-byte runes fit.
			"multibyte name at the limit",
			config.IdentityConfig{Name: strings.Repeat("é", config.IdentityNameMaxRunes)},
			"",
		},
		{
			"name over the limit",
			config.IdentityConfig{Name: strings.Repeat("a", config.IdentityNameMaxRunes+1)},
			"identity.name is 65 characters; the limit is 64",
		},
		{
			"location at the limit",
			config.IdentityConfig{Location: strings.Repeat("b", config.IdentityLocationMaxRunes)},
			"",
		},
		{
			"location over the limit",
			config.IdentityConfig{Location: strings.Repeat("b", config.IdentityLocationMaxRunes+1)},
			"identity.location is 129 characters; the limit is 128",
		},
		{"newline in name", config.IdentityConfig{Name: "rack\n12"}, "identity.name must be one line of text"},
		{"tab in location", config.IdentityConfig{Location: "IDF\t3B"}, "identity.location must be one line of text"},
		{"C1 control in name", config.IdentityConfig{Name: "rack\u008512"}, "identity.name must be one line of text"},
	}

	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("NewSchemaValidator: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotErr := tc.identity.Validate()
			switch {
			case tc.wantErr == "" && gotErr != nil:
				t.Fatalf("Validate() = %v, want nil", gotErr)
			case tc.wantErr != "" && (gotErr == nil || gotErr.Error() != tc.wantErr):
				t.Fatalf("Validate() = %v, want %q", gotErr, tc.wantErr)
			}

			cfg := config.DefaultConfig()
			cfg.Identity = tc.identity
			schemaErrs := validator.ValidateConfig(cfg)
			if schemaRefused := len(schemaErrs) > 0; schemaRefused != (tc.wantErr != "") {
				t.Fatalf("schema refused = %v (%v), Validate refused = %v", schemaRefused, schemaErrs, gotErr != nil)
			}

			cfg.Auth.DefaultPasswordHash = "hash"
			if startupErr := cfg.Validate(); (startupErr != nil) != (tc.wantErr != "") {
				t.Fatalf("Config.Validate() = %v, want refusal %v", startupErr, tc.wantErr != "")
			}
		})
	}
}

// TestIdentitySurvivesCopies covers the two copies a live config goes through:
// Clone for a snapshot and CopyFieldsFrom for a backup restore. A section
// either one drops reverts on the next save (#2928 is that, for three others).
func TestIdentitySurvivesCopies(t *testing.T) {
	src := config.DefaultConfig()
	src.Identity = config.IdentityConfig{Name: "seed-idf-3b", Location: "IDF 3B"}

	if got := src.Clone().Identity; got != src.Identity {
		t.Errorf("Clone identity = %+v, want %+v", got, src.Identity)
	}
	restored := config.DefaultConfig()
	restored.CopyFieldsFrom(src)
	if restored.Identity != src.Identity {
		t.Errorf("CopyFieldsFrom identity = %+v, want %+v", restored.Identity, src.Identity)
	}
}
