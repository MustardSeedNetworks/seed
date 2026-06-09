package discovery

import "strings"

// normalizeMAC normalizes a MAC address to uppercase with colons. It is a
// kernel-shared utility (the registry and event log key devices by normalized
// MAC); it lived in wifi.go historically but is not Wi-Fi-specific, so it sits
// in its own file now that the Wi-Fi collector has moved to the enumerate stage.
// The enumerate package keeps its own private copy (it cannot import this
// unexported helper across the package boundary).
func normalizeMAC(mac string) string {
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}
