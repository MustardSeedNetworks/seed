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

// Schema returns the schema field for testing.
func (v *SchemaValidator) Schema() any {
	return v.schema
}
