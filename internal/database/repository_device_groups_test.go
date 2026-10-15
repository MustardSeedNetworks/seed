package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

func newGroupTarget(t *testing.T, db *database.DB, clientID, ip string) string {
	t.Helper()
	target := &polling.Target{ClientID: clientID, Name: "dev-" + ip, IPAddress: ip}
	require.NoError(t, db.PollingTargets().Create(context.Background(), target))
	return target.ID
}

func TestDeviceGroupsRoundTripMembers(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	a := newGroupTarget(t, db, database.DefaultClientID, "10.0.0.1")
	b := newGroupTarget(t, db, database.DefaultClientID, "10.0.0.2")

	g := &devicegroups.Group{ClientID: database.DefaultClientID, Name: "core", Members: []string{a, b}}
	require.NoError(t, db.DeviceGroups().Save(ctx, g))
	require.Regexp(t, `^grp-[0-9a-f]{12}$`, g.ID)

	got, err := db.DeviceGroups().Get(ctx, database.DefaultClientID, g.ID)
	require.NoError(t, err)
	require.Equal(t, "core", got.Name)
	require.ElementsMatch(t, []string{a, b}, got.Members)
	require.False(t, got.CreatedAt.IsZero())

	g.Name, g.Description, g.Members = "core switches", "the two cores", []string{b}
	require.NoError(t, db.DeviceGroups().Save(ctx, g))
	list, err := db.DeviceGroups().List(ctx, database.DefaultClientID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "core switches", list[0].Name)
	require.Equal(t, "the two cores", list[0].Description)
	require.Equal(t, []string{b}, list[0].Members)

	addrs, err := db.DeviceGroups().MemberAddresses(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{g.ID: {"10.0.0.2"}}, addrs)

	// Deleting a target takes it out of every group.
	require.NoError(t, db.PollingTargets().Delete(ctx, database.DefaultClientID, b))
	got, err = db.DeviceGroups().Get(ctx, database.DefaultClientID, g.ID)
	require.NoError(t, err)
	require.Empty(t, got.Members)
}

func TestDeviceGroupsStayInsideTheClient(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	require.NoError(t, db.Clients().Create(ctx, &database.Client{ID: "tenant-b", Name: "Tenant B", Slug: "tenant-b"}))
	foreign := newGroupTarget(t, db, "tenant-b", "10.9.9.9")
	own := newGroupTarget(t, db, database.DefaultClientID, "10.0.0.1")

	for name, members := range map[string][]string{
		"another client's target": {own, foreign},
		"no such target":          {"tgt-missing"},
	} {
		g := &devicegroups.Group{ClientID: database.DefaultClientID, Name: name, Members: members}
		require.ErrorIs(t, db.DeviceGroups().Save(ctx, g), devicegroups.ErrUnknownMember, name)
	}
	// The failed saves left nothing behind.
	list, err := db.DeviceGroups().List(ctx, database.DefaultClientID)
	require.NoError(t, err)
	require.Empty(t, list)

	g := &devicegroups.Group{ClientID: "tenant-b", Name: "b", Members: []string{foreign}}
	require.NoError(t, db.DeviceGroups().Save(ctx, g))
	_, err = db.DeviceGroups().Get(ctx, database.DefaultClientID, g.ID)
	require.ErrorIs(t, err, devicegroups.ErrNotFound)
	require.ErrorIs(t, db.DeviceGroups().Delete(ctx, database.DefaultClientID, g.ID), devicegroups.ErrNotFound)
	g.ClientID = database.DefaultClientID
	require.ErrorIs(t, db.DeviceGroups().Save(ctx, g), devicegroups.ErrNotFound)
}

func TestDeviceGroupsRefuseADuplicateName(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	require.NoError(
		t,
		db.DeviceGroups().Save(ctx, &devicegroups.Group{ClientID: database.DefaultClientID, Name: "edge"}),
	)
	err := db.DeviceGroups().Save(ctx, &devicegroups.Group{ClientID: database.DefaultClientID, Name: "edge"})
	require.ErrorIs(t, err, devicegroups.ErrNameTaken)
}

func TestDeviceGroupNamedByARuleCannotBeDeleted(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	g := &devicegroups.Group{ClientID: database.DefaultClientID, Name: "core"}
	require.NoError(t, db.DeviceGroups().Save(ctx, g))

	rule := &alerts.Rule{
		Name: "core syslog", Enabled: true, AlertType: "syslog", AlertSeverity: "warning",
		AlertTitle: "core", GroupID: g.ID,
	}
	require.NoError(t, db.AlertRules().Create(ctx, rule))
	got, err := db.AlertRules().Get(ctx, rule.ID)
	require.NoError(t, err)
	require.Equal(t, g.ID, got.GroupID)

	require.ErrorIs(t, db.DeviceGroups().Delete(ctx, database.DefaultClientID, g.ID), devicegroups.ErrInUse)

	got.GroupID = ""
	require.NoError(t, db.AlertRules().Update(ctx, got))
	require.NoError(t, db.DeviceGroups().Delete(ctx, database.DefaultClientID, g.ID))
}

func TestAlertRuleRefusesAnUnknownGroup(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	rule := &alerts.Rule{
		Name: "x", Enabled: true, AlertType: "syslog", AlertSeverity: "warning", AlertTitle: "x",
		GroupID: "grp-missing",
	}
	err := db.AlertRules().Create(ctx, rule)
	require.ErrorContains(t, err, "alert_rules: groupId does not name a device group")
}
