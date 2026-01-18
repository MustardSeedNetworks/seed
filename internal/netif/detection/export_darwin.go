//go:build darwin

package detection

// ParseMediaSpeed exposes parseMediaSpeed for testing.
func ParseMediaSpeed(output string) int64 {
	return parseMediaSpeed(output)
}

// ParseIfconfigSpeed exposes parseIfconfigSpeed for testing.
func ParseIfconfigSpeed(output string) int64 {
	return parseIfconfigSpeed(output)
}
