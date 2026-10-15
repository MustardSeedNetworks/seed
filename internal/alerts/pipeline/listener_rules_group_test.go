package pipeline_test

import (
	"context"
	"errors"
	"testing"

	alertmodel "github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

type fakeGroups struct {
	members map[string][]string
	err     error
}

func (f *fakeGroups) MemberAddresses(context.Context) (map[string][]string, error) {
	return f.members, f.err
}

func groupRule(groupID string) *alertmodel.Rule {
	r := dbRule("scoped", "", "", "", true)
	r.GroupID = groupID
	return r
}

func TestCompileRulesFromDB_GroupScope(t *testing.T) {
	t.Parallel()
	groups := map[string][]string{"grp-core": {"10.0.0.1", "2001:db8::1"}}

	tests := []struct {
		name    string
		groupID string
		source  string
		want    bool
	}{
		{"member", "grp-core", "10.0.0.1:514", true},
		{"non-member", "grp-core", "10.0.0.2:514", false},
		{"member over IPv6, other spelling", "grp-core", "[2001:DB8:0::1]:162", true},
		{"member as v4-mapped IPv6", "grp-core", "[::ffff:10.0.0.1]:514", true},
		{"member without a port", "grp-core", "10.0.0.1", true},
		{"group with no members matches nothing", "grp-empty", "10.0.0.1:514", false},
		{"unscoped rule matches everyone", "", "10.0.0.2:514", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rule := pipeline.CompileRulesFromDB([]*alertmodel.Rule{groupRule(tt.groupID)}, groups)[0]
			evt := &listener.EventRecord{Kind: "syslog-udp", SourceAddr: tt.source}
			if got := rule.Match(evt); got != tt.want {
				t.Errorf("Match(%s) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestScanOnce_GroupScopedRuleAlertsOnlyForMembers(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*listener.EventRecord{
		syslogEvent("warning", "10.0.0.1:514", at(), "member"),
		syslogEvent("warning", "10.0.0.2:514", at(), "outsider"),
	}}
	alerts := &fakeAlerts{}
	p, err := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: &fakeAlertRulesReader{rows: []*alertmodel.Rule{groupRule("grp-core")}},
		Groups:     &fakeGroups{members: map[string][]string{"grp-core": {"10.0.0.1"}}},
	})
	if err != nil {
		t.Fatalf("NewListenerPipeline: %v", err)
	}
	if scanErr := p.ScanOnce(context.Background()); scanErr != nil {
		t.Fatalf("ScanOnce: %v", scanErr)
	}
	if len(alerts.created) != 1 || alerts.created[0].Source != "10.0.0.1:514" {
		t.Fatalf("alerts = %+v, want one from the member 10.0.0.1:514", alerts.created)
	}
}

func TestReloadRules_FailsWhenGroupsCannotBeRead(t *testing.T) {
	t.Parallel()
	groups := &fakeGroups{err: errors.New("database is closed")}
	p, err := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: &fakeEvents{}, Alerts: &fakeAlerts{}, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: &fakeAlertRulesReader{rows: []*alertmodel.Rule{groupRule("grp-core")}},
		Groups:     groups,
	})
	if err != nil {
		t.Fatalf("NewListenerPipeline: %v", err)
	}
	if reloadErr := p.ReloadRules(context.Background()); !errors.Is(reloadErr, groups.err) {
		t.Fatalf("ReloadRules = %v, want the group read error", reloadErr)
	}
}
