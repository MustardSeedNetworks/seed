// Package ifstats is the read side of the interface rates the SNMP pipeline
// stores (P-A, ADR-0033): every interface a polling target has reported,
// with the rates of its most recent rated poll (UI-SEED-21, #3191).
//
// An interface is listed as soon as the target's ifTable has been walked,
// before any rate exists: the first poll only baselines the counters, so a
// newly added target has interfaces and no rates yet, and the caller tells
// that apart from an estate with no interfaces at all.
package ifstats

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"
)

// ErrUnavailable is returned when the store cannot be reached (no database).
var ErrUnavailable = errors.New("interface statistics unavailable")

// OperStatus is an interface's ifOperStatus (RFC 2863).
type OperStatus string

// The ifOperStatus values, in MIB order.
const (
	OperUp             OperStatus = "up"
	OperDown           OperStatus = "down"
	OperTesting        OperStatus = "testing"
	OperUnknown        OperStatus = "unknown"
	OperDormant        OperStatus = "dormant"
	OperNotPresent     OperStatus = "notPresent"
	OperLowerLayerDown OperStatus = "lowerLayerDown"
)

// OperStatusFromMIB maps the ifOperStatus integer to its name. Anything
// outside the MIB's range, including the 0 of a row never walked, is unknown.
func OperStatusFromMIB(v int) OperStatus {
	byValue := [...]OperStatus{
		OperUnknown, OperUp, OperDown, OperTesting, OperUnknown, OperDormant, OperNotPresent, OperLowerLayerDown,
	}
	if v < 0 || v >= len(byValue) {
		return OperUnknown
	}
	return byValue[v]
}

// Rates is one rated poll of an interface. Errors and discards are always
// rated; octets are not when a 32-bit counter could wrap within the
// interval, and utilization needs the line rate as well (ifrate.Octets).
type Rates struct {
	At             time.Time
	InOctets       *float64
	OutOctets      *float64
	InUtilization  *float64
	OutUtilization *float64
	InErrors       float64
	OutErrors      float64
	InDiscards     float64
	OutDiscards    float64
}

// ErrorRate is the interface's errors per second in both directions.
func (r Rates) ErrorRate() float64 {
	return r.InErrors + r.OutErrors
}

// Interface is one interface of one polling target. SpeedBps is 0 when the
// agent reports no line rate; Rates is nil until the first rated poll.
type Interface struct {
	TargetID   string
	TargetName string
	IfIndex    uint32
	Name       string
	Alias      string
	OperStatus OperStatus
	SpeedBps   uint64
	Rates      *Rates
}

// Store reads the discovered interfaces and their latest rates.
type Store interface {
	Interfaces(ctx context.Context) ([]Interface, error)
}

// Service answers interface statistics reads from a Store.
type Service struct {
	store Store
}

// NewService builds the interface statistics read use-case over store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns every discovered interface, the highest error rate first so
// the interfaces worth looking at lead. Interfaces with no rate yet follow
// the rated ones; ties go by target, then ifIndex.
func (s *Service) List(ctx context.Context) ([]Interface, error) {
	ifaces, err := s.store.Interfaces(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(ifaces, func(a, b Interface) int {
		if (a.Rates == nil) != (b.Rates == nil) {
			if a.Rates == nil {
				return 1
			}
			return -1
		}
		if a.Rates != nil {
			if c := cmp.Compare(b.Rates.ErrorRate(), a.Rates.ErrorRate()); c != 0 {
				return c
			}
		}
		return cmp.Or(
			cmp.Compare(a.TargetName, b.TargetName),
			cmp.Compare(a.TargetID, b.TargetID),
			cmp.Compare(a.IfIndex, b.IfIndex),
		)
	})
	return ifaces, nil
}
