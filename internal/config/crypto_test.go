package config_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestIsEncrypted(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{"enc:base64data", true},
		{"plaintext", false},
		{"", false},
		{"enc:", true},
		{"ENC:base64", false}, // case-sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			result := config.IsEncrypted(tc.value)
			if result != tc.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}
