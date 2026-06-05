package api

import (
	"fmt"
	"strings"
)

// bluetooth_decode.go enriches discovered Bluetooth devices with human-readable
// decodings of the raw protocol values the scanner reports: the manufacturer
// (company-identifier) ID, advertised GATT service UUIDs, and the BLE
// appearance value. The tables are curated to the common Bluetooth-SIG
// assigned numbers (https://www.bluetooth.com/specifications/assigned-numbers/);
// unknown values fall back to a raw hex rendering so nothing is hidden.
//
// The tables are returned from functions (not package-level vars) to satisfy
// the no-globals lint policy, matching the discovery package's lookup pattern.

// btCompanyNames maps common Bluetooth-SIG company identifiers to vendor names.
func btCompanyNames() map[uint16]string {
	return map[uint16]string{
		0x0001: "Ericsson",
		0x0002: "Intel",
		0x0006: "Microsoft",
		0x000D: "Texas Instruments",
		0x000F: "Broadcom",
		0x004C: "Apple",
		0x0059: "Nordic Semiconductor",
		0x0075: "Samsung Electronics",
		0x0078: "Nike",
		0x0087: "Garmin",
		0x00C4: "LG Electronics",
		0x00D7: "Qualcomm",
		0x00E0: "Google",
		0x0118: "Sony",
		0x0131: "Cypress Semiconductor",
		0x0157: "Huami (Amazfit/Xiaomi)",
		0x0171: "Amazon",
		0x0399: "Nintendo",
		0x05A7: "Sonos",
		0x0822: "Adafruit",
		0x0A12: "Espressif (ESP32)",
	}
}

// btGATTServiceNames maps standard 16-bit GATT service UUIDs (lowercase hex) to
// their assigned names.
func btGATTServiceNames() map[string]string {
	return map[string]string{
		"1800": "Generic Access",
		"1801": "Generic Attribute",
		"1802": "Immediate Alert",
		"1803": "Link Loss",
		"1804": "Tx Power",
		"1805": "Current Time",
		"1809": "Health Thermometer",
		"180a": "Device Information",
		"180d": "Heart Rate",
		"180e": "Phone Alert Status",
		"180f": "Battery",
		"1810": "Blood Pressure",
		"1812": "Human Interface Device",
		"1816": "Cycling Speed and Cadence",
		"1818": "Cycling Power",
		"1819": "Location and Navigation",
		"181a": "Environmental Sensing",
		"181b": "Body Composition",
		"181c": "User Data",
		"181d": "Weight Scale",
		"1820": "Internet Protocol Support",
		"1826": "Fitness Machine",
		"110a": "Audio Source (A2DP)",
		"110b": "Audio Sink (A2DP)",
		"1108": "Headset",
		"111e": "Hands-Free",
		"1124": "HID over BR/EDR",
		"feaa": "Google Eddystone",
	}
}

// btAppearanceLabels maps common full BLE appearance values to labels.
func btAppearanceLabels() map[uint16]string {
	return map[uint16]string{
		64:   "Phone",
		128:  "Computer",
		192:  "Watch",
		193:  "Sports Watch",
		256:  "Clock",
		320:  "Display",
		384:  "Remote Control",
		448:  "Eye-glasses",
		512:  "Tag",
		576:  "Keyring",
		640:  "Media Player",
		704:  "Barcode Scanner",
		768:  "Thermometer",
		832:  "Heart Rate Sensor",
		896:  "Blood Pressure",
		960:  "Human Interface Device",
		961:  "Keyboard",
		962:  "Mouse",
		963:  "Joystick",
		964:  "Gamepad",
		1024: "Glucose Meter",
		1088: "Running/Walking Sensor",
		1152: "Cycling",
		1344: "Pulse Oximeter",
		1472: "Weight Scale",
		5184: "Outdoor (GPS)",
	}
}

// appearanceCategoryBits is the bit shift that isolates the appearance
// category (the top 10 bits) from the sub-type (bottom 6).
const appearanceCategoryBits = 6

// btAppearanceCategories labels the appearance category (value >> 6) so a value
// without an exact match still decodes to its family.
func btAppearanceCategories() map[uint16]string {
	return map[uint16]string{
		1:  "Phone",
		2:  "Computer",
		3:  "Watch",
		4:  "Clock",
		5:  "Display",
		6:  "Remote Control",
		7:  "Eye-glasses",
		8:  "Tag",
		9:  "Keyring",
		10: "Media Player",
		11: "Barcode Scanner",
		12: "Thermometer",
		13: "Heart Rate Sensor",
		14: "Blood Pressure",
		15: "Human Interface Device",
		16: "Glucose Meter",
		17: "Running/Walking Sensor",
		18: "Cycling",
		21: "Pulse Oximeter",
		81: "Outdoor Sports",
	}
}

// decodeBTCompany returns the vendor name for a company identifier, or a raw
// "0x...." rendering when it is unknown. Returns "" for the zero ID.
func decodeBTCompany(id uint16) string {
	if id == 0 {
		return ""
	}
	if name, ok := btCompanyNames()[id]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (0x%04X)", id)
}

// decodeBTServices maps advertised service UUIDs to their GATT service names.
// Both 16-bit short UUIDs ("180f") and full 128-bit UUIDs using the Bluetooth
// base ("0000180f-0000-1000-8000-00805f9b34fb") resolve; unknown UUIDs pass
// through unchanged so nothing is dropped.
func decodeBTServices(uuids []string) []string {
	if len(uuids) == 0 {
		return nil
	}
	table := btGATTServiceNames()
	out := make([]string, 0, len(uuids))
	for _, u := range uuids {
		if name, ok := table[shortServiceUUID(u)]; ok {
			out = append(out, name)
		} else {
			out = append(out, u)
		}
	}
	return out
}

// shortServiceUUID lowercases a UUID and, if it is a full 128-bit UUID on the
// Bluetooth base (xxxxxxxx-0000-1000-8000-00805f9b34fb), returns its 16-bit
// short form; otherwise returns the (lowercased) input.
func shortServiceUUID(u string) string {
	s := strings.ToLower(strings.TrimSpace(u))
	const btBaseSuffix = "-0000-1000-8000-00805f9b34fb"
	if len(s) == 36 && strings.HasSuffix(s, btBaseSuffix) {
		// 0000XXXX-.... → strip the leading 0000 of the 8-hex group.
		return strings.TrimPrefix(s[:8], "0000")
	}
	return s
}

// decodeBTAppearance returns a human label for a BLE appearance value: an exact
// match first, then the category family, else a raw rendering.
func decodeBTAppearance(a uint16) string {
	if label, ok := btAppearanceLabels()[a]; ok {
		return label
	}
	if label, ok := btAppearanceCategories()[a>>appearanceCategoryBits]; ok {
		return label
	}
	if a == 0 {
		return ""
	}
	return fmt.Sprintf("Unknown (%d)", a)
}
