package fdp_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/cdp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/fdp"
)

type fakeClient struct {
	prefixSeen string
	vbs        []snmp.Varbind
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by fdp")
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	f.prefixSeen = prefix
	return f.vbs, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []cdp.Observation
}

func (p *fakePublisher) PublishCDP(_ context.Context, obs cdp.Observation) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.got = append(p.got, obs)
	return nil
}

func at() time.Time { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

func TestNew_NameIsFDP(t *testing.T) {
	t.Parallel()
	c := fdp.New(nil, nil, at)
	if c.Name() != fdp.Name {
		t.Errorf("Name() = %q, want %q", c.Name(), fdp.Name)
	}
}

func TestNew_WalksFoundryTablePrefix(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		vbs: []snmp.Varbind{
			{OID: fdp.TablePrefix + ".6.1.1", Value: "foundry-edge"},
			{OID: fdp.TablePrefix + ".8.1.1", Value: "Ruckus ICX 7150"},
		},
	}
	pub := &fakePublisher{}
	c := fdp.New(
		func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
			return fc, nil
		},
		pub,
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if fc.prefixSeen != fdp.TablePrefix {
		t.Errorf("walked %q, want %q", fc.prefixSeen, fdp.TablePrefix)
	}
	if len(pub.got) != 1 || len(pub.got[0].Neighbors) != 1 {
		t.Fatalf("expected one neighbor, got %+v", pub.got)
	}
	n := pub.got[0].Neighbors[0]
	if n.DeviceID != "foundry-edge" {
		t.Errorf("DeviceID = %q", n.DeviceID)
	}
	if n.Platform != "Ruckus ICX 7150" {
		t.Errorf("Platform = %q", n.Platform)
	}
	if pub.got[0].TablePrefix != fdp.TablePrefix {
		t.Errorf("Observation.TablePrefix = %q, want %q",
			pub.got[0].TablePrefix, fdp.TablePrefix)
	}
}
