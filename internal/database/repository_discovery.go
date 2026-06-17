package database

import (
	"errors"
	"strings"
	"time"
)

// Constants for discovery repository.
const (
	// ouiPrefixLength is the length of an OUI (Organizationally Unique Identifier) prefix in hex chars.
	ouiPrefixLength = 6
	// hoursPerDay is the number of hours in a day.
	hoursPerDay = 24
)

// Error definitions for discovery repository.
var (
	ErrWiFiNetworkNotFound        = errors.New("wifi network not found")
	ErrWiFiAccessPointNotFound    = errors.New("wifi access point not found")
	ErrNetworkProblemNotFound     = errors.New("network problem not found")
	ErrOUIVendorNotFound          = errors.New("oui vendor not found")
	ErrChannelUtilizationNotFound = errors.New("channel utilization not found")
)

// DiscoveryRepository provides operations for unified discovery data.
type DiscoveryRepository struct {
	db *DB
}

// Helper functions

func normalizeOUI(oui string) string {
	// Remove common separators and convert to uppercase
	oui = strings.ToUpper(oui)
	oui = strings.ReplaceAll(oui, "-", "")
	oui = strings.ReplaceAll(oui, ":", "")
	oui = strings.ReplaceAll(oui, ".", "")

	// Take first 6 hex chars and format with colons
	if len(oui) >= ouiPrefixLength {
		return oui[0:2] + ":" + oui[2:4] + ":" + oui[4:6]
	}
	return oui
}

func extractOUIPrefix(mac string) string {
	// Normalize MAC address
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, ".", "")

	// Extract first 6 hex chars (OUI)
	if len(mac) >= ouiPrefixLength {
		return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6]
	}
	return ""
}

func nullTimeStr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
