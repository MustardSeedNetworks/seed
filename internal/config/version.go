// Package config handles application configuration.
package config

// ConfigVersion is the current configuration schema version.
// Increment this when making breaking changes to the config structure.
//
// Version history:
//   - 1: Initial versioned config (2025-12-16)
const ConfigVersion = 1

// MinSupportedVersion is the minimum config version that can be migrated.
// Configs older than this version cannot be automatically upgraded.
const MinSupportedVersion = 1
