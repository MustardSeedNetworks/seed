package snmp

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// Browser walks one subtree of a polling target on an operator's request: the
// MIB browser (P-E5). It authenticates exactly as the poller does, and it does
// not need the poller running, since browsing one device is not estate polling.
type Browser struct {
	factory     ClientFactory
	credentials *CredentialResolver
}

// NewBrowser returns a Browser that dials through factory with the credentials
// the resolver decrypts for each target.
func NewBrowser(factory ClientFactory, credentials *CredentialResolver) *Browser {
	return &Browser{factory: factory, credentials: credentials}
}

// Walk returns up to limit varbinds under prefix, in agent order, and whether
// the subtree held more than limit.
func (b *Browser) Walk(ctx context.Context, target *polling.Target, prefix string, limit int) ([]Varbind, bool, error) {
	creds, err := b.credentials.Resolve(ctx, target)
	if err != nil {
		return nil, false, err
	}
	client, err := b.factory(wireTarget(target), creds)
	if err != nil {
		return nil, false, fmt.Errorf("snmp browser: %w", err)
	}
	vbs, err := client.WalkLimit(ctx, prefix, limit+1)
	if err != nil {
		return nil, false, err
	}
	if len(vbs) > limit {
		return vbs[:limit], true, nil
	}
	return vbs, false, nil
}
