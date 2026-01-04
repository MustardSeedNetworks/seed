// Package detection exports internal functions for testing.
package detection

// DetectType exposes detectType for testing.
func DetectType(name string) string {
	return detectType(name)
}

// HasRoutableAddress exposes hasRoutableAddress for testing.
func HasRoutableAddress(addresses []string) bool {
	return hasRoutableAddress(addresses)
}

// FormatSpeed exposes formatSpeed for testing.
func FormatSpeed(bps int64) string {
	return formatSpeed(bps)
}

// CalculateScore exposes calculateScore for testing.
func (d *Detector) CalculateScore(s *InterfaceScore) int {
	return d.calculateScore(s)
}

// GenerateFriendlyName exposes generateFriendlyName for testing.
func (d *Detector) GenerateFriendlyName(s *InterfaceScore) string {
	return d.generateFriendlyName(s)
}

// GenerateDescription exposes generateDescription for testing.
func (d *Detector) GenerateDescription(s *InterfaceScore) string {
	return d.generateDescription(s)
}

// ChipsetsCount returns the number of chipsets in the database.
func (db *ChipsetDatabase) ChipsetsCount() int {
	return len(db.chipsets)
}

// OUIMapCount returns the number of entries in the OUI map.
func (db *ChipsetDatabase) OUIMapCount() int {
	return len(db.ouiMap)
}

// ChipsetDBNil checks if the detector's chipsetDB is nil.
func (d *Detector) ChipsetDBNil() bool {
	return d.chipsetDB == nil
}
