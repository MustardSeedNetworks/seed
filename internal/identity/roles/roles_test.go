package roles_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

func TestIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role string
		want bool
	}{
		{roles.Admin, true},
		{roles.Operator, true},
		{roles.Viewer, true},
		{"", false},
		{"Admin", false},
		{"superuser", false},
		{" viewer", false},
	}
	for _, tt := range tests {
		if got := roles.IsValid(tt.role); got != tt.want {
			t.Errorf("IsValid(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}
