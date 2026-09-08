package detection

import (
	"context"
	"fmt"
)

// DetectType exposes detectType for testing.
func DetectType(name string) string {
	return detectType(name)
}

// HasRoutableAddress exposes hasRoutableAddress for testing.
func HasRoutableAddress(addresses []string) bool {
	return hasRoutableAddress(addresses)
}

// FormatSpeed exposes formatSpeed for testing.
func FormatSpeed(bps int64) string {
	return formatSpeed(bps)
}

// CalculateScore exposes calculateScore for testing.
func (d *Detector) CalculateScore(s *InterfaceScore) int {
	return d.calculateScore(s)
}

// GenerateFriendlyName exposes generateFriendlyName for testing.
func (d *Detector) GenerateFriendlyName(s *InterfaceScore) string {
	return d.generateFriendlyName(s)
}

// GenerateDescription exposes generateDescription for testing.
func (d *Detector) GenerateDescription(s *InterfaceScore) string {
	return d.generateDescription(s)
}

// ChipsetsCount returns the number of chipsets in the database.
func (db *ChipsetDatabase) ChipsetsCount() int {
	return len(db.chipsets)
}

// OUIMapCount returns the number of entries in the OUI map.
func (db *ChipsetDatabase) OUIMapCount() int {
	return len(db.ouiMap)
}

// SetPlatformIdentifier replaces platform identification for testing.
func (db *ChipsetDatabase) SetPlatformIdentifier(identify func(string) *ChipsetInfo) {
	db.identifyPlatform = identify
}

// ChipsetDBNil checks if the detector's chipsetDB is nil.
func (d *Detector) ChipsetDBNil() bool {
	return d.chipsetDB == nil
}

// CalculateSpeedBonus exposes calculateSpeedBonus for testing.
func CalculateSpeedBonus(speed int64) int {
	return calculateSpeedBonus(speed)
}

// GetSpeedBonuses exposes getSpeedBonuses for testing.
func GetSpeedBonuses() []struct {
	MinSpeed int64
	Bonus    int
} {
	bonuses := getSpeedBonuses()
	result := make([]struct {
		MinSpeed int64
		Bonus    int
	}, len(bonuses))
	for i, b := range bonuses {
		result[i] = struct {
			MinSpeed int64
			Bonus    int
		}{
			MinSpeed: b.minSpeed,
			Bonus:    b.bonus,
		}
	}
	return result
}

// NewDetectorNoFork builds a detector whose platform helpers are stubbed out, so
// the suite never starts a process.
//
// This is not tidiness. DetectAll calls getInterfaceSpeed once per interface,
// and on darwin each call forks networksetup and often ifconfig; TestDetectAll,
// TestDetectBest and TestScoreInterface between them spent ~9.6s of this
// package's 10.8s doing that against whatever interfaces the developer's
// machine happens to have. A fork out of a cgo/ObjC process is also the
// mechanism behind seed#2420: a child stuck before execve carries the parent's
// argv, so `pgrep -f 'seed --config'` counts it as another daemon.
//
// The stub fails rather than returning empty output: every caller treats an
// error as "this platform did not tell me", which is the path a machine without
// networksetup already takes, so the code under test sees an input it sees in
// production. TestParseMediaSpeed and TestParseIfconfigSpeed cover the parsers
// on real output.
func NewDetectorNoFork() *Detector {
	return newDetector(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("detection: %s is not run under go test", name)
	})
}

// NewChipsetDatabaseNoFork builds a chipset database whose platform lookup is
// stubbed. IdentifyByInterface falls through to identifyByPlatformUncached when
// neither OUI nor keyword matches, which forks system_profiler on darwin — the
// last helper the suite was still starting. Same reasoning as
// [NewDetectorNoFork].
func NewChipsetDatabaseNoFork() *ChipsetDatabase {
	return newChipsetDatabase(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("detection: %s is not run under go test", name)
	})
}
