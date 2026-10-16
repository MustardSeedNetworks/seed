package enumerate

// bluetooth_ad.go decodes the advertising data an LE device broadcasts and the
// extended inquiry response (EIR) a classic device answers with. Both use the
// same length-type-value layout (Core Specification Supplement, Part A), so one
// parser fills name, services, TX power, class, appearance and manufacturer for
// either kind of device.

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
)

// Advertising data types, from the Bluetooth assigned numbers.
const (
	adTypeIncompleteUUID16  = 0x02
	adTypeCompleteUUID16    = 0x03
	adTypeIncompleteUUID128 = 0x06
	adTypeCompleteUUID128   = 0x07
	adTypeShortName         = 0x08
	adTypeCompleteName      = 0x09
	adTypeTxPower           = 0x0A
	adTypeClassOfDevice     = 0x0D
	adTypeAppearance        = 0x19
	adTypeManufacturerData  = 0xFF
)

const (
	adUUID16Len      = 2
	adUUID128Len     = 16
	adClassLen       = 3
	adAppearanceLen  = 2
	adCompanyIDLen   = 2
	adClassByte2Bits = 16
	adClassByte1Bits = 8
)

// applyAdvertisingData decodes an advertising or EIR payload onto dev. A field
// whose length runs past the payload ends the walk, and a zero length is the
// padding that closes an EIR block.
func applyAdvertisingData(dev *BluetoothDevice, data []byte) {
	for len(data) > 1 {
		n := int(data[0])
		if n == 0 || n >= len(data) {
			return
		}
		applyADField(dev, data[1], data[2:n+1])
		data = data[n+1:]
	}
}

func applyADField(dev *BluetoothDevice, adType byte, v []byte) {
	switch adType {
	case adTypeIncompleteUUID16, adTypeCompleteUUID16:
		for ; len(v) >= adUUID16Len; v = v[adUUID16Len:] {
			appendServiceUUID(dev, fmt.Sprintf("%04x", binary.LittleEndian.Uint16(v)))
		}
	case adTypeIncompleteUUID128, adTypeCompleteUUID128:
		for ; len(v) >= adUUID128Len; v = v[adUUID128Len:] {
			appendServiceUUID(dev, formatUUID128(v[:adUUID128Len]))
		}
	case adTypeCompleteName:
		dev.Name = string(v)
	case adTypeShortName:
		if dev.Name == "" {
			dev.Name = string(v)
		}
	case adTypeTxPower:
		if len(v) == 1 {
			dev.TxPower = signedByte(v[0])
		}
	case adTypeClassOfDevice:
		if len(v) == adClassLen {
			dev.ClassOfDev = uint32(v[2])<<adClassByte2Bits | uint32(v[1])<<adClassByte1Bits | uint32(v[0])
			dev.DeviceClass = ClassOfDeviceToClass(dev.ClassOfDev)
		}
	case adTypeAppearance:
		if len(v) == adAppearanceLen {
			dev.Appearance = binary.LittleEndian.Uint16(v)
			if dev.ClassOfDev == 0 {
				dev.DeviceClass = BLEAppearanceToClass(dev.Appearance)
			}
		}
	case adTypeManufacturerData:
		if len(v) >= adCompanyIDLen {
			dev.ManufacturerID = binary.LittleEndian.Uint16(v)
			dev.ManufacturerData = slices.Clone(v[adCompanyIDLen:])
		}
	}
}

// signedByte reads a two's-complement byte, as TX power and RSSI are sent.
func signedByte(b byte) int {
	if b > math.MaxInt8 {
		return int(b) - (math.MaxUint8 + 1)
	}
	return int(b)
}

func appendServiceUUID(dev *BluetoothDevice, uuid string) {
	if !slices.Contains(dev.ServiceUUIDs, uuid) {
		dev.ServiceUUIDs = append(dev.ServiceUUIDs, uuid)
	}
}

// formatUUID128 renders a little-endian 128-bit UUID in its canonical form.
func formatUUID128(le []byte) string {
	b := slices.Clone(le)
	slices.Reverse(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
