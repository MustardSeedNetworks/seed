package enumerate

import "testing"

// copyDevice exists so GetDevices and friends hand callers a snapshot that
// cannot be mutated underneath the scanner. A field that is copied shallowly
// still aliases the scanner's own memory, so a caller writing to it corrupts
// discovery state -- and nothing would report an error. These tests mutate the
// copy and assert the original is untouched, which is the only way that class
// of bug shows up.
func deviceWithEverySliceSet() *DiscoveredDevice {
	return &DiscoveredDevice{
		IP:              "10.44.20.5",
		MAC:             "74:ac:b9:3b:af:40",
		DiscoveryMethod: []Method{MethodARP},
		IPv6Addresses:   []string{"fe80::1"},
		DuplicateMACs:   []string{"10.44.20.6"},
		ConnectionTypes: []ConnectionType{},
	}
}

func TestCopyDevice_SlicesDoNotAliasTheOriginal(t *testing.T) {
	original := deviceWithEverySliceSet()
	original.ConnectionTypes = append(original.ConnectionTypes, ConnectionType("wired"))

	clone := copyDevice(original)

	clone.IPv6Addresses[0] = "fe80::999"
	clone.DuplicateMACs[0] = "mutated"
	clone.DiscoveryMethod[0] = Method("mutated")
	clone.ConnectionTypes[0] = ConnectionType("mutated")

	if original.IPv6Addresses[0] != "fe80::1" {
		t.Errorf("IPv6Addresses aliased: original became %q", original.IPv6Addresses[0])
	}
	if original.DuplicateMACs[0] != "10.44.20.6" {
		t.Errorf("DuplicateMACs aliased: original became %q", original.DuplicateMACs[0])
	}
	if original.DiscoveryMethod[0] != MethodARP {
		t.Errorf("DiscoveryMethod aliased: original became %q", original.DiscoveryMethod[0])
	}
	if original.ConnectionTypes[0] != ConnectionType("wired") {
		t.Errorf("ConnectionTypes aliased: original became %q", original.ConnectionTypes[0])
	}
}

func TestCopyDevice_CopiesScalarFields(t *testing.T) {
	original := deviceWithEverySliceSet()

	clone := copyDevice(original)

	if clone.IP != original.IP || clone.MAC != original.MAC {
		t.Errorf("scalar fields not carried: got IP=%q MAC=%q", clone.IP, clone.MAC)
	}
}

// A nil slice must stay nil rather than becoming an empty one: the two encode
// differently in JSON, and the API's shape is a contract.
func TestCopyDevice_NilSlicesStayNil(t *testing.T) {
	clone := copyDevice(&DiscoveredDevice{IP: "10.44.20.5"})

	if clone.IPv6Addresses != nil {
		t.Errorf("nil IPv6Addresses became %#v", clone.IPv6Addresses)
	}
	if clone.DiscoveryMethod != nil {
		t.Errorf("nil DiscoveryMethod became %#v", clone.DiscoveryMethod)
	}
	if clone.DuplicateMACs != nil {
		t.Errorf("nil DuplicateMACs became %#v", clone.DuplicateMACs)
	}
	if clone.ConnectionTypes != nil {
		t.Errorf("nil ConnectionTypes became %#v", clone.ConnectionTypes)
	}
}

func TestCopyWiFiPresence_NilAndDeepCopy(t *testing.T) {
	if got := copyWiFiPresence(nil); got != nil {
		t.Errorf("copyWiFiPresence(nil) = %#v, want nil", got)
	}

	original := &WiFiPresence{}
	clone := copyWiFiPresence(original)
	if clone == original {
		t.Error("copyWiFiPresence returned the same pointer; a caller could mutate the original")
	}
}

func TestCopyBluetoothPresence_NilAndDeepCopy(t *testing.T) {
	if got := copyBluetoothPresence(nil); got != nil {
		t.Errorf("copyBluetoothPresence(nil) = %#v, want nil", got)
	}

	original := &BluetoothPresence{}
	clone := copyBluetoothPresence(original)
	if clone == original {
		t.Error("copyBluetoothPresence returned the same pointer")
	}
}
