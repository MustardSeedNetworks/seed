// Package config exports internal functions for testing.
package config

// EncryptedPrefix exports encryptedPrefix for testing.
const EncryptedPrefix = encryptedPrefix

// ExportHasIPv4Address exposes hasIPv4Address for testing.
func ExportHasIPv4Address(ifaceName string) bool {
	return hasIPv4Address(ifaceName)
}

// ExportDetectActiveInterface exposes detectActiveInterface for testing.
func ExportDetectActiveInterface() string {
	return detectActiveInterface()
}

// ExtractVersion wraps the unexported extractVersion method for testing.
func (b *BackupManager) ExtractVersion(data []byte) int {
	return b.extractVersion(data)
}

// Schema returns the schema field for testing.
func (v *SchemaValidator) Schema() any {
	return v.schema
}
