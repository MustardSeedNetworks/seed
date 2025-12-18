// Package testutil provides centralized test utilities for The Seed project.
//
// This package offers:
//   - TestDefaults: Singleton test defaults derived from config.DefaultConfig()
//   - ConfigBuilder: Fluent builder for creating test configurations
//   - Fixtures: Pre-built configuration fixtures for common test scenarios
//
// All test defaults are derived from the production DefaultConfig() to ensure
// tests stay in sync with production defaults. The package uses lazy initialization
// to compute expensive values (like bcrypt password hashes) only once.
//
// Example usage:
//
//	// Get test defaults
//	defaults := testutil.GetTestDefaults()
//	username := defaults.Auth.Username
//
//	// Build custom config
//	cfg := testutil.NewConfigBuilder().
//		WithPort(8080).
//		WithInterface("eth0").
//		Build()
//
//	// Use pre-built fixtures
//	minimalCfg := testutil.Fixtures.MinimalConfig()
package testutil
