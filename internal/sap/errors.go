package sap

import "errors"

// Sap module errors.
var (
	// ErrNotImplemented is returned when a feature is not yet implemented.
	ErrNotImplemented = errors.New("not implemented: pending migration")

	// ErrNotInitialized is returned when a service is accessed before initialization.
	ErrNotInitialized = errors.New("service not initialized")

	// ErrNotSupported is returned when a feature is not supported on this platform.
	ErrNotSupported = errors.New("feature not supported on this platform")

	// ErrTestFailed is returned when a test operation fails.
	ErrTestFailed = errors.New("test failed")
)
