package discovery

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// A device profiled before any SNMP credential existed has a profile with no
// SNMP answer, and QueueProfile skips a profiled device, so without this it is
// never asked again: the credential the first-run step saves never reaches the
// devices discovery already found (seed#2692). The collector records a
// silent device too, with only its errors, so that record is no answer.
func TestReprofileSNMPSilentAsksOnlyTheSilentDevicesAgain(t *testing.T) {
	cfg := DefaultProfilerConfig()
	cfg.Timeout = 100 * time.Millisecond
	cfg.ConnectTimeout = 100 * time.Millisecond
	p := NewDeviceProfiler(cfg, nil)
	p.Start()
	t.Cleanup(p.Stop)

	const silent, answered, collected = "192.0.2.1", "192.0.2.2", "192.0.2.3"
	const refused = "192.0.2.4"
	before := time.Now().Add(-time.Hour)
	answeredProfile := &DeviceProfile{ProfiledAt: before, SNMPInfo: &SNMPInfo{SysName: "core"}}
	collectedProfile := &DeviceProfile{ProfiledAt: before}
	p.mu.Lock()
	p.profiles[silent] = &DeviceProfile{ProfiledAt: before}
	p.profiles[answered] = answeredProfile
	p.profiles[collected] = collectedProfile
	p.snmpData[collected] = &SNMPFullData{System: &snmp.SystemInfo{SysName: "edge"}}
	p.profiles[refused] = &DeviceProfile{ProfiledAt: before}
	p.snmpData[refused] = &SNMPFullData{Errors: []string{"system: request timeout"}}
	p.mu.Unlock()

	if got := p.ReprofileSNMPSilent(); got != 2 {
		t.Errorf("ReprofileSNMPSilent() queued %d devices, want 2", got)
	}

	deadline := time.Now().Add(10 * time.Second)
	for _, ip := range []string{silent, refused} {
		for {
			if profile := p.GetProfile(ip); profile != nil && profile.ProfiledAt.After(before) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s was not profiled again: %+v", ip, p.GetProfile(ip))
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	if p.GetProfile(answered) != answeredProfile {
		t.Errorf("%s answered SNMP and was profiled again", answered)
	}
	if p.GetProfile(collected) != collectedProfile {
		t.Errorf("%s has collected SNMP data and was profiled again", collected)
	}
}
