package app

import "github.com/MustardSeedNetworks/seed/internal/database"

// Persistence is the application's durable store (seed#2750, ADR-0020). It is
// opaque outside internal/app: cmd/seed builds it and internal/api holds and
// passes it, but only the use-cases in this package reach the database behind
// it. A nil *Persistence is the supported no-database mode, and every adapter
// built over one degrades to its package's unavailable error.
type Persistence struct {
	db *database.DB
}

// NewPersistence wraps an opened database.
func NewPersistence(db *database.DB) *Persistence {
	return &Persistence{db: db}
}

// Close closes the database. Closing the no-database mode is a no-op.
func (p *Persistence) Close() error {
	if p == nil {
		return nil
	}
	return p.db.Close()
}

func (p *Persistence) handle() *database.DB {
	if p == nil {
		return nil
	}
	return p.db
}

// lazyHandle reads the database through persist on every call, so a handle
// attached after construction (the api test harness) is seen.
func lazyHandle(persist func() *Persistence) func() *database.DB {
	return func() *database.DB { return persist().handle() }
}
