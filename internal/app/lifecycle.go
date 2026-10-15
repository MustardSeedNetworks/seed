package app

// lifecycle.go wires the composition root to the end-of-life data (P-D9,
// ADR-0020). The table lives in settings; an absent key means nothing has
// been imported or fetched. The opt-in feed's settings live in the config
// file, so an installer can answer the feed question before first start.

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
)

// settingKeyLifecycleEoX holds the EoX table as JSON.
const settingKeyLifecycleEoX = "lifecycle_eox"

// NewLifecycleStore builds the EoX table store over a lazy database accessor.
func NewLifecycleStore(persist func() *Persistence) lifecycle.Store {
	return &lifecycleStore{db: lazyHandle(persist)}
}

type lifecycleStore struct {
	db func() *database.DB
	// mu serialises writes, so a feed merge cannot drop an import that lands
	// between its read and its write.
	mu sync.Mutex
}

// EoX returns the stored table. A stored table that no longer parses is an
// error rather than an empty table, so lost data is reported, not hidden.
func (a *lifecycleStore) EoX(ctx context.Context) (*lifecycle.Table, error) {
	stored, err := a.db().Settings().GetValue(ctx, settingKeyLifecycleEoX)
	if err != nil {
		return nil, fmt.Errorf("read EoX table: %w", err)
	}
	if stored == "" {
		return lifecycle.New(nil)
	}
	return lifecycle.Parse([]byte(stored))
}

func (a *lifecycleStore) SetEoX(ctx context.Context, table *lifecycle.Table) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.set(ctx, table)
}

func (a *lifecycleStore) MergeEoX(ctx context.Context, newer *lifecycle.Table) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	stored, err := a.EoX(ctx)
	if err != nil {
		return err
	}
	merged, err := stored.Merge(newer)
	if err != nil {
		return fmt.Errorf("merge EoX records: %w", err)
	}
	return a.set(ctx, merged)
}

func (a *lifecycleStore) set(ctx context.Context, table *lifecycle.Table) error {
	data, err := json.Marshal(table)
	if err != nil {
		return fmt.Errorf("encode EoX table: %w", err)
	}
	return a.db().Settings().Set(ctx, settingKeyLifecycleEoX, string(data))
}

// LifecycleFeed is the opt-in EoX feed's settings over the live config.
type LifecycleFeed struct {
	store managementStore
}

// NewLifecycleFeed builds the feed settings over the live config, saved to
// path on every write.
func NewLifecycleFeed(cfg *config.Config, path string) *LifecycleFeed {
	return &LifecycleFeed{store: managementStore{cfg: cfg, path: path}}
}

func feedChoice(choice config.EoXFeedChoice) lifecycle.Choice {
	switch choice {
	case config.EoXFeedOn:
		return lifecycle.ChoiceOn
	case config.EoXFeedOff:
		return lifecycle.ChoiceOff
	case config.EoXFeedUnanswered:
	}
	return lifecycle.ChoiceUnanswered
}

// Settings returns the feed settings, with Cisco's endpoints filled in for
// empty ones and the secret reported only as present or not.
func (f *LifecycleFeed) Settings() lifecycle.FeedSettings {
	var out lifecycle.FeedSettings
	f.store.Read(func(cfg *config.Config) {
		feed := cfg.Lifecycle.EoXFeed
		out = lifecycle.FeedSettings{
			Choice:          feedChoice(feed.Choice),
			Endpoint:        orDefault(feed.Endpoint, lifecycle.DefaultEndpoint),
			TokenURL:        orDefault(feed.TokenURL, lifecycle.DefaultOAuthURL),
			ClientID:        feed.ClientID,
			HasClientSecret: feed.ClientSecret != "",
		}
	})
	return out
}

// Update validates and persists the operator's answer and settings. An
// endpoint equal to Cisco's is stored empty, so it follows the default.
func (f *LifecycleFeed) Update(u lifecycle.FeedUpdate) error {
	return f.store.Write(func(cfg *config.Config) error {
		feed := &cfg.Lifecycle.EoXFeed
		if err := u.Validate(feed.ClientSecret != ""); err != nil {
			return err
		}
		next := config.EoXFeedConfig{
			Choice:       config.EoXFeedOff,
			Endpoint:     emptyIfDefault(u.Endpoint, lifecycle.DefaultEndpoint),
			TokenURL:     emptyIfDefault(u.TokenURL, lifecycle.DefaultOAuthURL),
			ClientID:     u.ClientID,
			ClientSecret: feed.ClientSecret,
		}
		if u.Choice == lifecycle.ChoiceOn {
			next.Choice = config.EoXFeedOn
		}
		switch {
		case u.ClearClientSecret:
			next.ClientSecret = ""
		case u.ClientSecret != "":
			ciphertext, err := configKeyring{cfg: cfg}.EncryptValue(u.ClientSecret)
			if err != nil {
				// The message names the field, never the value.
				return fmt.Errorf("encrypt clientSecret: %w", err)
			}
			next.ClientSecret = ciphertext
		}
		*feed = next
		return nil
	})
}

// Credentials implements lifecycle.FeedSource: false unless the operator
// turned the feed on.
func (f *LifecycleFeed) Credentials(context.Context) (lifecycle.Credentials, bool, error) {
	var (
		feed   config.EoXFeedConfig
		secret string
		err    error
	)
	f.store.Read(func(cfg *config.Config) {
		feed = cfg.Lifecycle.EoXFeed
		if feed.Choice == config.EoXFeedOn && feed.ClientSecret != "" {
			secret, err = decryptSecret(cfg, feed.ClientSecret)
		}
	})
	if feed.Choice != config.EoXFeedOn {
		return lifecycle.Credentials{}, false, nil
	}
	if err != nil {
		return lifecycle.Credentials{}, false, fmt.Errorf("decrypt EoX client secret: %w", err)
	}
	return lifecycle.Credentials{
		Endpoint:     orDefault(feed.Endpoint, lifecycle.DefaultEndpoint),
		TokenURL:     orDefault(feed.TokenURL, lifecycle.DefaultOAuthURL),
		ClientID:     feed.ClientID,
		ClientSecret: secret,
	}, true, nil
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func emptyIfDefault(value, fallback string) string {
	if value == fallback {
		return ""
	}
	return value
}
