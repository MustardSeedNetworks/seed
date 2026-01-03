// Package config exports internal functions for testing.
package config

// EncryptedPrefix exports encryptedPrefix for testing.
const EncryptedPrefix = encryptedPrefix

// HasIPv4Address exports hasIPv4Address for testing.
var HasIPv4Address = hasIPv4Address

// DetectActiveInterface exports detectActiveInterface for testing.
var DetectActiveInterface = detectActiveInterface

// ExtractVersion wraps the unexported extractVersion method for testing.
func (b *BackupManager) ExtractVersion(data []byte) int {
	return b.extractVersion(data)
}

// Schema returns the schema field for testing.
func (v *SchemaValidator) Schema() any {
	return v.schema
}
