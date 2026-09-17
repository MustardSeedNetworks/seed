//go:build darwin

package wifi

import "github.com/MustardSeedNetworks/foundation/pkg/corewlan"

func ObserveCurrentForTest(read func() (*corewlan.Network, error), h Helper) Observation {
	return observeCurrent(read, h)
}

func ScanCurrentForTest(scan func() ([]corewlan.Network, error), h Helper) ([]*ScannedNetwork, error) {
	return scanCurrent(scan, h)
}
