//go:build darwin

package detection

// Layout is the byte layout of the if_msghdr2 fields the speed reader uses.
type Layout struct {
	Msglen, Type, Index, Baudrate, Size int
}

// IfMsghdr2Layout exposes the offsets baudrates reads at, so a test can hold
// them against the kernel ABI.
func IfMsghdr2Layout() Layout {
	return Layout{
		Msglen:   offMsglen,
		Type:     offType,
		Index:    offIndex,
		Baudrate: offBaudrate,
		Size:     sizeofIfMsghdr2,
	}
}

// Baudrates exposes baudrates for testing.
func Baudrates(rib []byte) map[int]uint64 {
	return baudrates(rib)
}

// InterfaceSpeed exposes getInterfaceSpeed for testing.
func InterfaceSpeed(name string) int64 {
	return getInterfaceSpeed(nil, name)
}
