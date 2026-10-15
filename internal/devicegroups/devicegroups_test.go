package devicegroups_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// memStore keeps the last saved group so Get echoes what Save stored, and
// serves fixed members and open alerts to Summary.
type memStore struct {
	saved   devicegroups.Group
	members []polling.Target
	open    []devicegroups.OpenAlert
}

func (*memStore) List(context.Context, string) ([]devicegroups.Group, error) { return nil, nil }

func (m *memStore) Get(context.Context, string, string) (devicegroups.Group, error) {
	return m.saved, nil
}

func (m *memStore) Lookup(context.Context, string) (devicegroups.Group, error) {
	return m.saved, nil
}

func (m *memStore) Save(_ context.Context, g *devicegroups.Group) error {
	if g.ID == "" {
		g.ID = "grp-1"
	}
	m.saved = *g
	return nil
}

func (*memStore) Delete(context.Context, string, string) error { return nil }

func (m *memStore) MemberTargets(context.Context, string) ([]polling.Target, error) {
	return m.members, nil
}

func (m *memStore) OpenAlerts(context.Context) ([]devicegroups.OpenAlert, error) { return m.open, nil }

func TestCreateNormalizes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		in          devicegroups.Group
		wantName    string
		wantMembers []string
		wantErr     string
	}{
		{
			name:        "trims and sorts",
			in:          devicegroups.Group{Name: " core ", Members: []string{"tgt-b", " tgt-a", "tgt-b"}},
			wantName:    "core",
			wantMembers: []string{"tgt-a", "tgt-b"},
		},
		{name: "no members", in: devicegroups.Group{Name: "empty"}, wantName: "empty", wantMembers: []string{}},
		{name: "blank name", in: devicegroups.Group{Name: " "}, wantErr: "name is required"},
		{name: "long name", in: devicegroups.Group{Name: strings.Repeat("é", 65)}, wantErr: "at most 64"},
		{
			name: "long description", in: devicegroups.Group{Name: "x", Description: strings.Repeat("d", 513)},
			wantErr: "at most 512",
		},
		{
			name:    "empty member",
			in:      devicegroups.Group{Name: "x", Members: []string{" "}},
			wantErr: "member id is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := devicegroups.NewService(&memStore{}).Create(context.Background(), tt.in)
			if tt.wantErr != "" {
				var ve *devicegroups.ValidationError
				if !errors.As(err, &ve) || !strings.Contains(ve.Msg, tt.wantErr) {
					t.Fatalf("err = %v, want a validation error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if got.Name != tt.wantName || strings.Join(got.Members, ",") != strings.Join(tt.wantMembers, ",") {
				t.Errorf("got %q %v, want %q %v", got.Name, got.Members, tt.wantName, tt.wantMembers)
			}
		})
	}
}

func TestCreateIgnoresACallerSuppliedID(t *testing.T) {
	t.Parallel()
	got, err := devicegroups.NewService(&memStore{}).Create(context.Background(),
		devicegroups.Group{ID: "grp-theirs", Name: "x"})
	if err != nil || got.ID != "grp-1" {
		t.Fatalf("Create = %+v, %v; want a fresh id", got, err)
	}
}

func TestSummaryAttributesAlertsToMembers(t *testing.T) {
	core := polling.Target{ID: "tgt-core", IPAddress: "10.0.0.1"}
	v6 := polling.Target{ID: "tgt-v6", IPAddress: "2001:db8::1"}
	tests := []struct {
		name    string
		members []polling.Target
		open    []devicegroups.OpenAlert
		want    []int // each member's open alerts
		counts  devicegroups.AlertCounts
	}{
		{
			name:    "syslog host:port names the member",
			members: []polling.Target{core},
			open:    []devicegroups.OpenAlert{{Source: "10.0.0.1:514", Severity: alerts.SeverityWarning}},
			want:    []int{1},
			counts:  devicegroups.AlertCounts{Total: 1, Warning: 1},
		},
		{
			name:    "a poller observation names the target id",
			members: []polling.Target{core},
			open:    []devicegroups.OpenAlert{{Source: "tgt-core", Severity: alerts.SeverityCritical}},
			want:    []int{1},
			counts:  devicegroups.AlertCounts{Total: 1, Critical: 1},
		},
		{
			name:    "an IPv6 trap in another spelling",
			members: []polling.Target{v6},
			open:    []devicegroups.OpenAlert{{Source: "[2001:DB8:0::1]:162", Severity: alerts.SeverityError}},
			want:    []int{1},
			counts:  devicegroups.AlertCounts{Total: 1, Error: 1},
		},
		{
			name:    "a bare address",
			members: []polling.Target{core},
			open:    []devicegroups.OpenAlert{{Source: "10.0.0.1", Severity: alerts.SeverityInfo}},
			want:    []int{1},
			counts:  devicegroups.AlertCounts{Total: 1, Info: 1},
		},
		{
			name:    "an outsider's alerts and sourceless ones are not the group's",
			members: []polling.Target{core},
			open: []devicegroups.OpenAlert{
				{Source: "10.0.0.10:514", Severity: alerts.SeverityCritical},
				{Source: "tgt-edge", Severity: alerts.SeverityCritical},
				{Source: "", Severity: alerts.SeverityCritical},
			},
			want: []int{0},
		},
		{
			name:    "a severity outside the four counts only in the total",
			members: []polling.Target{core},
			open:    []devicegroups.OpenAlert{{Source: "10.0.0.1:514", Severity: "notice"}},
			want:    []int{1},
			counts:  devicegroups.AlertCounts{Total: 1},
		},
		{
			name: "one address polled twice: each member's, the group's once",
			members: []polling.Target{
				core, {ID: "tgt-core-v3", IPAddress: "10.0.0.1"},
			},
			open:   []devicegroups.OpenAlert{{Source: "10.0.0.1:514", Severity: alerts.SeverityWarning}},
			want:   []int{1, 1},
			counts: devicegroups.AlertCounts{Total: 1, Warning: 1},
		},
		{
			name: "a target id names only that target, not its address twin",
			members: []polling.Target{
				core, {ID: "tgt-core-v3", IPAddress: "10.0.0.1"},
			},
			open:   []devicegroups.OpenAlert{{Source: "tgt-core-v3", Severity: alerts.SeverityWarning}},
			want:   []int{0, 1},
			counts: devicegroups.AlertCounts{Total: 1, Warning: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &memStore{saved: devicegroups.Group{ID: "grp-1", Name: "core"}, members: tt.members, open: tt.open}
			sum, err := devicegroups.NewService(store).Summary(context.Background(), "default", "grp-1")
			if err != nil {
				t.Fatal(err)
			}
			if sum.Group.Name != "core" {
				t.Errorf("group = %q, want core", sum.Group.Name)
			}
			got := make([]int, len(sum.Members))
			for i, m := range sum.Members {
				got[i] = m.OpenAlerts
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("member open alerts = %v, want %v", got, tt.want)
			}
			if sum.OpenAlerts != tt.counts {
				t.Errorf("counts = %+v, want %+v", sum.OpenAlerts, tt.counts)
			}
		})
	}
}

func TestHostCanonicalizes(t *testing.T) {
	tests := map[string]string{
		"10.0.0.1:514":        "10.0.0.1",
		"10.0.0.1":            "10.0.0.1",
		"[2001:DB8:0::1]:162": "2001:db8::1",
		"::ffff:10.0.0.1":     "10.0.0.1",
		"[fe80::1%eth0]:514":  "fe80::1",
		"Switch-A.lan":        "switch-a.lan",
	}
	for in, want := range tests {
		if got := devicegroups.Host(in); got != want {
			t.Errorf("Host(%q) = %q, want %q", in, got, want)
		}
	}
}
