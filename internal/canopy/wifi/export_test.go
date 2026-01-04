// Package wifi exports internals for testing.
package wifi

// GetBand exports getBand for testing.
func GetBand(freq int) string {
	return getBand(freq)
}

// DetectChannelWidth exports detectChannelWidth for testing.
func DetectChannelWidth(freq int, htMode string) int {
	return detectChannelWidth(freq, htMode)
}

// MapSecurityType exports mapSecurityType for testing.
func MapSecurityType(secType string) string {
	return mapSecurityType(secType)
}

// ChannelToFrequency exports channelToFrequency for testing.
func ChannelToFrequency(channel int) int {
	return channelToFrequency(channel)
}

// FrequencyToChannel exports frequencyToChannel for testing.
func FrequencyToChannel(freq int) int {
	return frequencyToChannel(freq)
}

// IsWirelessPlatform exports isWirelessPlatform for testing.
func IsWirelessPlatform(iface string) bool {
	return isWirelessPlatform(iface)
}

// GetInfoPlatform exports getInfoPlatform for testing.
func GetInfoPlatform(iface string) *Info {
	return getInfoPlatform(iface)
}

// InterfaceName returns the interface name for a Manager (getter for testing).
func (m *Manager) InterfaceName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.interfaceName
}
