package iperf

import "time"

// This file is only compiled during testing.

// IperfBinaryPath returns the current cached iperf binary path for testing.
func IperfBinaryPath() string {
	return getIperfBinaryPath()
}

// SetIperfBinaryPath sets the cached iperf binary path for testing.
func SetIperfBinaryPath(path string) {
	setIperfBinaryPath(path)
}

// FindIperf3Binary exposes the internal findIperf3Binary function for testing.
func FindIperf3Binary() (string, error) {
	return findIperf3Binary()
}

// CompareVersions exposes the internal compareVersions function for testing.
func CompareVersions(v1, v2 string) int {
	return compareVersions(v1, v2)
}

// MinSupportedVersion exposes the minimum supported version constant for testing.
const MinSupportedVersion = minSupportedVersion

// IperfJSON exposes the internal iperfJSON type for testing.
type IperfJSON = iperfJSON

// SetManagerClientStatusRunning sets the client status running flag for testing.
func (m *Manager) SetManagerClientStatusRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientStatus.Running = running
}

// SetManagerServerStatusRunning sets the server status running flag for testing.
func (m *Manager) SetManagerServerStatusRunning(running bool, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverStatus.Running = running
	m.serverStatus.Port = port
}

// SetManagerLastResult sets the last result for testing.
func (m *Manager) SetManagerLastResult(result *Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastResult = result
}

// ValidateServer exposes the internal validateServer function for testing.
func ValidateServer(server string) error {
	return validateServer(server)
}

// SetClientDefaults exposes the internal setClientDefaults function for testing.
func SetClientDefaults(config *ClientConfig) {
	setClientDefaults(config)
}

// NormalizeDirection exposes the internal normalizeDirection function for testing.
func NormalizeDirection(config *ClientConfig) string {
	return normalizeDirection(config)
}

// BuildClientArgs exposes the internal buildClientArgs function for testing.
func BuildClientArgs(config *ClientConfig, direction string) []string {
	return buildClientArgs(config, direction)
}

// ParseClientResult exposes the internal parseClientResult function for testing.
func ParseClientResult(iperfOut *IperfJSON, config *ClientConfig, direction string) *Result {
	return parseClientResult(iperfOut, config, direction)
}

// GetLegacyPaths exposes the internal getLegacyPaths function for testing.
func GetLegacyPaths() []string {
	return getLegacyPaths()
}

// ValidateBinary exposes the internal validateBinary function for testing.
func ValidateBinary(path string) bool {
	return validateBinary(path)
}

// NeedsSudo exposes the internal needsSudo function for testing.
func NeedsSudo(packageManager string) bool {
	return needsSudo(packageManager)
}

// GetCacheDir exposes the internal getCacheDir function for testing.
func GetCacheDir() (string, error) {
	return getCacheDir()
}

// IsValidExtractedBinary exposes the internal isValidExtractedBinary function for testing.
func IsValidExtractedBinary(binaryPath, versionFile string) bool {
	return isValidExtractedBinary(binaryPath, versionFile)
}

// GetPlatformBinaryMap exposes the internal getPlatformBinaryMap function for testing.
func GetPlatformBinaryMap() map[string]string {
	return getPlatformBinaryMap()
}

// WaitForPortReady exposes the internal waitForPortReady function for testing.
func WaitForPortReady(port int, timeout time.Duration) error {
	return waitForPortReady(port, timeout)
}

// ExtractEmbeddedBinary exposes the internal extractEmbeddedBinary function for testing.
func ExtractEmbeddedBinary() (string, error) {
	return extractEmbeddedBinary()
}

// FindSystemIperf3 exposes the internal findSystemIperf3 function for testing.
func FindSystemIperf3() (string, error) {
	return findSystemIperf3()
}

// DetectLinuxPackageManager exposes the internal detectLinuxPackageManager function for testing.
func DetectLinuxPackageManager() *PackageManagerInfo {
	return detectLinuxPackageManager()
}

// DetectMacOSPackageManager exposes the internal detectMacOSPackageManager function for testing.
func DetectMacOSPackageManager() *PackageManagerInfo {
	return detectMacOSPackageManager()
}

// DetectWindowsPackageManager exposes the internal detectWindowsPackageManager function for testing.
func DetectWindowsPackageManager() *PackageManagerInfo {
	return detectWindowsPackageManager()
}

// ClearIperfBinaryPath clears the cached binary path for testing.
func ClearIperfBinaryPath() {
	setIperfBinaryPath("")
}

// VersionCheckTimeout exposes the version check timeout constant for testing.
const VersionCheckTimeout = versionCheckTimeout

// ServerStartTimeout exposes the server start timeout constant for testing.
const ServerStartTimeout = serverStartTimeout

// PortCheckTimeout exposes the port check timeout constant for testing.
const PortCheckTimeout = portCheckTimeout

// MaxHostnameLength exposes the max hostname length constant for testing.
const MaxHostnameLength = maxHostnameLength

// DirectionDownload exposes the direction constant for testing.
const DirectionDownload = directionDownload

// DirectionUpload exposes the direction constant for testing.
const DirectionUpload = directionUpload

// DirectionBidirectional exposes the direction constant for testing.
const DirectionBidirectional = directionBidirectional

// BytesToMegabits exposes the conversion constant for testing.
const BytesToMegabits = bytesToMegabits
