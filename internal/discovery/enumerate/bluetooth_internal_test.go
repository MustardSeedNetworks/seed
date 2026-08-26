package enumerate

import (
	"math"
	"net"
	"testing"
)

// bleAppearance builds the 16-bit appearance value for a SIG category. The low
// six bits are the subcategory; the category lives above them.
func bleAppearance(category uint16) uint16 { return category << bleAppearanceCategoryBits }

// TestBLEAppearanceToClass covers the Bluetooth SIG appearance categories.
//
// Category 15 is Human Interface Device — a keyboard, a mouse, a barcode
// wedge. It was mapped to Health, so every Bluetooth keyboard in range was
// reported as a medical device. On a product that surveys what is on a network,
// a stray HID is worth noticing and a stray glucose meter is a different
// conversation entirely.
func TestBLEAppearanceToClass(t *testing.T) {
	for _, tc := range []struct {
		name     string
		category uint16
		want     BluetoothDeviceClass
	}{
		{"generic", 0, BluetoothClassMisc},
		{"phone", 1, BluetoothClassPhone},
		{"computer", 2, BluetoothClassComputer},
		{"watch", 3, BluetoothClassWearable},
		{"clock", 4, BluetoothClassWearable},
		{"keyring", 9, BluetoothClassPeripheral},
		{"thermometer", 12, BluetoothClassHealth},
		{"heart rate sensor", 13, BluetoothClassHealth},
		{"blood pressure", 14, BluetoothClassHealth},
		{"human interface device", 15, BluetoothClassPeripheral},
		{"glucose meter", 16, BluetoothClassHealth},
		{"insulin pump", 53, BluetoothClassHealth},
		{"outdoor sports", 81, BluetoothClassMisc},
		{"an unassigned category", 900, BluetoothClassUncategorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := BLEAppearanceToClass(bleAppearance(tc.category)); got != tc.want {
				t.Errorf("category %d = %v, want %v", tc.category, got, tc.want)
			}
		})
	}
}

// TestBLEAppearanceIgnoresSubcategory pins that the low six bits do not change
// the classification. A "keyboard" HID and a "mouse" HID are both peripherals.
func TestBLEAppearanceIgnoresSubcategory(t *testing.T) {
	const hid = 15
	base := BLEAppearanceToClass(bleAppearance(hid))
	for sub := uint16(1); sub < 1<<bleAppearanceCategoryBits; sub += 7 {
		if got := BLEAppearanceToClass(bleAppearance(hid) | sub); got != base {
			t.Errorf("subcategory %d changed the class to %v, want %v", sub, got, base)
		}
	}
}

// TestClassOfDeviceToClass covers the classic Bluetooth Class of Device
// bitfield, whose major class sits in bits 8..12.
func TestClassOfDeviceToClass(t *testing.T) {
	cod := func(major uint32) uint32 { return major << codMajorClassShift }

	for _, tc := range []struct {
		name  string
		major uint32
		want  BluetoothDeviceClass
	}{
		{"miscellaneous", btMajorClassMisc, BluetoothClassMisc},
		{"computer", btMajorClassComputer, BluetoothClassComputer},
		{"phone", btMajorClassPhone, BluetoothClassPhone},
		{"LAN access point", btMajorClassLAN, BluetoothClassLAN},
		{"audio/video", btMajorClassAudioVideo, BluetoothClassAudioVideo},
		{"peripheral", btMajorClassPeripheral, BluetoothClassPeripheral},
		{"imaging", btMajorClassImaging, BluetoothClassImaging},
		{"wearable", btMajorClassWearable, BluetoothClassWearable},
		{"toy", btMajorClassToy, BluetoothClassToy},
		{"health", btMajorClassHealth, BluetoothClassHealth},
		{"an unassigned major class", 0x1E, BluetoothClassUncategorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassOfDeviceToClass(cod(tc.major)); got != tc.want {
				t.Errorf("major class %#x = %v, want %v", tc.major, got, tc.want)
			}
		})
	}
}

// TestClassOfDeviceIgnoresServiceAndMinorBits pins that only bits 8..12 decide
// the major class. The service bits above and the minor-class bits below must
// not leak into it.
func TestClassOfDeviceIgnoresServiceAndMinorBits(t *testing.T) {
	const phone = btMajorClassPhone << codMajorClassShift
	want := BluetoothClassPhone

	for _, noise := range []uint32{
		0x00,       // nothing
		0xFF,       // every minor-class and format bit
		0x1F << 13, // service-class bits above the major class
		0xFFE000 | 0xFF,
	} {
		if got := ClassOfDeviceToClass(phone | noise); got != want {
			t.Errorf("noise %#x changed the class to %v, want %v", noise, got, want)
		}
	}
}

// TestEstimateDistance pins the log-distance path-loss model the function
// implements: d = 10^((txPower - rssi) / (10 * n)), with n = 2.5 indoors.
//
// It used to evaluate that with a decade loop plus repeated multiplication by
// 10^0.1, advancing the fractional part by adding 0.1 each time. Float
// accumulation and the dropped final partial step made every estimate low —
// 15.85 m against 17.38 m at -90 dBm, about 9% short.
func TestEstimateDistance(t *testing.T) {
	const (
		txPower = -59
		n       = 2.5
	)
	model := func(rssi int) float64 {
		return math.Pow(10, float64(txPower-rssi)/(pathLossMultiplier*n))
	}

	for _, rssi := range []int{-62, -65, -70, -80, -90, -100} {
		got := estimateDistance(txPower, rssi)
		want := model(rssi)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("rssi %d: got %v, want %v (delta %v)", rssi, got, want, got-want)
		}
	}
}

// TestEstimateDistanceIsMonotonic pins the property that matters to a reader of
// the number: a weaker signal never reports as nearer.
func TestEstimateDistanceIsMonotonic(t *testing.T) {
	const txPower = -59
	previous := 0.0
	for rssi := txPower - 1; rssi >= -110; rssi-- {
		got := estimateDistance(txPower, rssi)
		if got < previous {
			t.Fatalf("rssi %d reports %v, nearer than the stronger %d at %v",
				rssi, got, rssi+1, previous)
		}
		previous = got
	}
}

// TestEstimateDistanceAtOrAboveTxPower pins the close-proximity shortcut. A
// signal at least as strong as the calibrated reference means the device is
// essentially on top of the receiver, and the model would say 1 m.
func TestEstimateDistanceAtOrAboveTxPower(t *testing.T) {
	for _, rssi := range []int{-59, -50, -20, 0} {
		if got := estimateDistance(-59, rssi); got != closeProximityDistance {
			t.Errorf("rssi %d: got %v, want %v", rssi, got, closeProximityDistance)
		}
	}
}

func TestNormalizeMAC(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"already canonical", "AA:BB:CC:DD:EE:FF", "AA:BB:CC:DD:EE:FF"},
		{"lower case", "aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"hyphen separated", "aa-bb-cc-dd-ee-ff", "AA:BB:CC:DD:EE:FF"},
		{"mixed separators and case", "Aa-bB:cC-dD:eE-fF", "AA:BB:CC:DD:EE:FF"},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeMAC(tc.in); got != tc.want {
				t.Errorf("normalizeMAC(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsInLocalSubnet(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("parse subnet: %v", err)
	}
	scanner := &ARPScanner{subnet: subnet}

	for _, tc := range []struct {
		name string
		ip   string
		want bool
	}{
		{"inside the subnet", "192.168.1.50", true},
		{"the network address", "192.168.1.0", true},
		{"the broadcast address", "192.168.1.255", true},
		{"just outside", "192.168.2.1", false},
		{"a different private range", "10.0.0.1", false},
		{"not an address", "not-an-ip", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := scanner.isInLocalSubnet(tc.ip); got != tc.want {
				t.Errorf("isInLocalSubnet(%q) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}

	t.Run("a scanner with no subnet matches nothing", func(t *testing.T) {
		if (&ARPScanner{}).isInLocalSubnet("192.168.1.50") {
			t.Error("a scanner with no subnet reported an address as local")
		}
	})
}
