package app

// mibdb.go opens the MIB OID store, the one consumer outside internal/database
// that takes the raw write handle (scripts/check-single-writer.sh, #2453).

import (
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
)

// NewMIBDatabase opens the MIB OID store over db's write connection.
func NewMIBDatabase(db *database.DB) *mibdb.DB {
	return mibdb.New(db.WriteConn())
}
