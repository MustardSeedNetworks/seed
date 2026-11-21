package support_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/support"
)

func TestTailLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		n        int
		maxBytes int64
		want     []string
	}{
		{"fewer lines than asked", "a\nb\n", 5, 1 << 10, []string{"a", "b"}},
		{"last n of many", "a\nb\nc\nd\n", 2, 1 << 10, []string{"c", "d"}},
		{"no trailing newline", "a\nb\nc", 2, 1 << 10, []string{"b", "c"}},
		{"empty file", "", 3, 1 << 10, nil},
		{"zero lines", "a\nb\n", 0, 1 << 10, nil},
		{"window cut starts on a line boundary", "first-line\nsecond\nthird\n", 10, 12, []string{"third"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := support.TailLines(strings.NewReader(tc.content), tc.n, tc.maxBytes)
			if err != nil {
				t.Fatalf("TailLines: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("TailLines = %q, want %q", got, tc.want)
			}
		})
	}
}
