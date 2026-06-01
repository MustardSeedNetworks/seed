package roots

import "errors"

// Roots module errors.
var (
	// ErrNotImplemented is returned by stub functions pending migration.
	ErrNotImplemented = errors.New("not implemented: pending migration")

	// ErrNotInitialized is returned when a service is accessed before initialization.
	ErrNotInitialized = errors.New("service not initialized")
)
