package app

// mibdb.go opens the MIB OID store, the one consumer outside internal/database
// that takes the raw write handle (scripts/check-single-writer.sh, #2453).

import (
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
)

// NewMIBDatabase opens the MIB OID store over persist's write connection.
func NewMIBDatabase(persist *Persistence) *mibdb.DB {
	return mibdb.New(persist.handle().WriteConn())
}
