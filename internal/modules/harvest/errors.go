package harvest

import "errors"

// ErrNotImplemented is returned by stub functions pending migration.
var ErrNotImplemented = errors.New("not implemented: pending migration")
