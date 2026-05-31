// Package sysinfo implements the sys_info SNMP Collector: one
// per-target GET of the system MIB scalars (sysDescr, sysName,
// sysObjectID, sysUpTime, sysLocation, sysContact). The observation
// feeds Stage A4 topology identity merging and the operator-facing
// device pane.
package sysinfo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
)

// Name is the collector key written into polling_targets.collector_chain.
const Name = "sys_info"

// MIB-II system OIDs (RFC 1213 §6.4). Scalar instances end in .0.
const (
	oidSysDescr    = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID = "1.3.6.1.2.1.1.2.0"
	oidSysUpTime   = "1.3.6.1.2.1.1.3.0"
	oidSysContact  = "1.3.6.1.2.1.1.4.0"
	oidSysName     = "1.3.6.1.2.1.1.5.0"
	oidSysLocation = "1.3.6.1.2.1.1.6.0"
)

// Observation carries one sys_info snapshot. ObservedAt is set by the
// collector at the start of the GET. SysUpTimeTicks is the raw
// TimeTicks value (hundredths of a second since reboot) — keep raw so
// the Publisher can derive both uptime-duration and a boot-time
// estimate.
type Observation struct {
	ClientID       string
	TargetID       string
	ObservedAt     time.Time
	SysDescr       string
	SysObjectID    string
	SysUpTimeTicks uint32
	SysContact     string
	SysName        string
	SysLocation    string
}

// Publisher is the consumer-defined seam the sysinfo collector calls
// once per successful GET. Stage A3.5 orchestration wires this to the
// topology reconciler + event log; tests inject a fake.
type Publisher interface {
	PublishSysInfo(ctx context.Context, obs Observation) error
}

// Collector runs the sys_info GET against each target. It is
// stateless across runs — one Collector instance is registered with
// the poller and reused for every target.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns a Collector that dials Clients via factory and ships
// observations to publisher. Pass nil for now to use [time.Now].
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect implements snmp.Collector. It performs one Get for the
// six system MIB scalars and publishes the Observation.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("sysinfo: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("sysinfo: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("sysinfo: dial: %w", err)
	}

	observedAt := c.now()
	vbs, err := client.Get(ctx, []string{
		oidSysDescr, oidSysObjectID, oidSysUpTime,
		oidSysContact, oidSysName, oidSysLocation,
	})
	if err != nil {
		return fmt.Errorf("sysinfo: snmp get: %w", err)
	}

	obs := buildObservation(target, observedAt, vbs)
	if pubErr := c.publisher.PublishSysInfo(ctx, obs); pubErr != nil {
		return fmt.Errorf("sysinfo: publish: %w", pubErr)
	}
	return nil
}

// buildObservation reads the six varbinds returned by Get in the
// order requested. Missing or wrong-typed values fall back to the
// zero value of the corresponding field so a partial response still
// produces a publishable Observation.
func buildObservation(target snmp.Target, observedAt time.Time, vbs []snmp.Varbind) Observation {
	obs := Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
	}
	for _, vb := range vbs {
		switch vb.OID {
		case oidSysDescr:
			obs.SysDescr = stringValue(vb.Value)
		case oidSysObjectID:
			obs.SysObjectID = stringValue(vb.Value)
		case oidSysUpTime:
			obs.SysUpTimeTicks = uint32Value(vb.Value)
		case oidSysContact:
			obs.SysContact = stringValue(vb.Value)
		case oidSysName:
			obs.SysName = stringValue(vb.Value)
		case oidSysLocation:
			obs.SysLocation = stringValue(vb.Value)
		}
	}
	return obs
}

// stringValue extracts a string from a varbind. SNMP OCTET STRINGs
// arrive as []byte; DisplayStrings arrive as string. Any other type
// (or nil) yields "".
func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}

// uint32Value extracts a TimeTicks / Counter32 / Gauge32 / Uint32
// scalar. Negative ints and out-of-range values clamp to the uint32
// representable range so the caller never observes garbage.
func uint32Value(v any) uint32 {
	const maxUint32 uint64 = 1<<32 - 1
	switch t := v.(type) {
	case nil:
		return 0
	case uint32:
		return t
	case uint:
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case uint64:
		if t > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case int:
		if t < 0 {
			return 0
		}
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case int32:
		if t < 0 {
			return 0
		}
		return uint32(t)
	case int64:
		if t < 0 {
			return 0
		}
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	default:
		return 0
	}
}
