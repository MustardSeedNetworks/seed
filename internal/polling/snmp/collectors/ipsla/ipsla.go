// Package ipsla implements the ip_sla SNMP Collector: the Cisco IP-SLA
// operations a device runs and the latest result of each, from
// CISCO-RTTMON-MIB. One pass walks
//
//   - rttMonCtrlAdminTable (1.3.6.1.4.1.9.9.42.1.2.1.1): each operation's
//     tag, type, threshold, frequency and row status;
//   - rttMonEchoAdminTable (1.3.6.1.4.1.9.9.42.1.2.2.1): its target;
//   - rttMonLatestRttOperTable (1.3.6.1.4.1.9.9.42.1.2.10.1): the round-trip
//     time and sense of its most recent run, and when that run happened;
//   - rttMonLatestJitterOperTable (1.3.6.1.4.1.9.9.42.1.5.2.1): for the
//     jitter operation types, the average jitter, packet loss and MOS of
//     that run.
//
// Seed only reads: operations are configured on the device, and SNMP writes
// to rttMonCtrlAdminTable are not something an instrument should do.
package ipsla

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "ip_sla"

const (
	ctrlAdminPrefix   = "1.3.6.1.4.1.9.9.42.1.2.1.1"
	echoAdminPrefix   = "1.3.6.1.4.1.9.9.42.1.2.2.1"
	latestRttPrefix   = "1.3.6.1.4.1.9.9.42.1.2.10.1"
	latestJitterTable = "1.3.6.1.4.1.9.9.42.1.5.2.1"

	colAdminTag       = "3"
	colAdminRttType   = "4"
	colAdminThreshold = "5"
	colAdminFrequency = "6"
	colAdminStatus    = "9"

	colEchoTargetAddress = "2"

	colLatestCompletionTime = "1"
	colLatestSense          = "2"
	colLatestTime           = "5"

	colJitterNumOfRTT  = "1"
	colJitterLossSD    = "26"
	colJitterLossDS    = "27"
	colJitterPacketMIA = "29"
	colJitterMOS       = "42"
	colJitterAvgJitter = "46"

	mosScale  = 100
	ipv4Bytes = 4
	ipv6Bytes = 16
)

// Sense values (RttResponseSense) the trend distinguishes. Every other value
// is a failed run.
const (
	SenseOther         = 0
	SenseOK            = 1
	SenseOverThreshold = 3
	SenseTimeout       = 4
)

// RowStatusActive is rttMonCtrlAdminStatus active(1): the operation is
// scheduled and running.
const RowStatusActive = 1

// Jitter is the latest run of a jitter-type operation.
type Jitter struct {
	// AvgJitterMs is rttMonLatestJitterOperAvgJitter, both directions.
	AvgJitterMs uint32
	// Received counts the probe packets that came back.
	Received uint32
	// Lost counts the packets lost in either direction or missing in action.
	Lost uint32
	// MOS is the mean opinion score, 1.00 to 5.00; 0 when the operation has
	// no codec configured and so computes none.
	MOS float64
}

// Result is the latest run of one operation.
type Result struct {
	// Sense is rttMonLatestRttOperSense.
	Sense int
	// RTTMs is rttMonLatestRttOperCompletionTime; meaningful only when the
	// run completed (Completed).
	RTTMs uint32
	// RunUptime is rttMonLatestRttOperTime, the agent's sysUpTime in
	// hundredths of a second when the run happened. It changes when, and
	// only when, the operation runs again.
	RunUptime uint32
	Jitter    *Jitter
}

// Completed reports whether the run reached its target, within its
// threshold or not.
func (r Result) Completed() bool {
	return r.Sense == SenseOK || r.Sense == SenseOverThreshold
}

// Operation is one rttMonCtrlAdminTable row and its latest result.
type Operation struct {
	Index       int
	Tag         string
	RttType     int
	Target      string
	ThresholdMs int
	FrequencyS  int
	Status      int
	// Latest is nil when the operation has not run since it was configured.
	Latest *Result
}

// Observation is the per-poll set of a device's IP-SLA operations. It is
// empty for a device that runs none or does not implement the MIB.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	Operations []Operation
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishIPSLA(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns an IP-SLA Collector. Pass nil now to use [time.Now] UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks the four tables and publishes one Observation.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("ip_sla: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("ip_sla: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("ip_sla: dial: %w", err)
	}

	columns := []string{
		ctrlAdminPrefix + "." + colAdminTag,
		ctrlAdminPrefix + "." + colAdminRttType,
		ctrlAdminPrefix + "." + colAdminThreshold,
		ctrlAdminPrefix + "." + colAdminFrequency,
		ctrlAdminPrefix + "." + colAdminStatus,
		echoAdminPrefix + "." + colEchoTargetAddress,
		latestRttPrefix + "." + colLatestCompletionTime,
		latestRttPrefix + "." + colLatestSense,
		latestRttPrefix + "." + colLatestTime,
		latestJitterTable + "." + colJitterNumOfRTT,
		latestJitterTable + "." + colJitterLossSD,
		latestJitterTable + "." + colJitterLossDS,
		latestJitterTable + "." + colJitterPacketMIA,
		latestJitterTable + "." + colJitterMOS,
		latestJitterTable + "." + colJitterAvgJitter,
	}
	observedAt := c.now()
	var vbs []snmp.Varbind
	for _, column := range columns {
		got, walkErr := client.Walk(ctx, column)
		if walkErr != nil {
			return fmt.Errorf("ip_sla: walk %s: %w", column, walkErr)
		}
		vbs = append(vbs, got...)
	}

	obs := Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Operations: buildOperations(vbs),
	}
	if pubErr := c.publisher.PublishIPSLA(ctx, obs); pubErr != nil {
		return fmt.Errorf("ip_sla: publish: %w", pubErr)
	}
	return nil
}

// row gathers one operation's columns across the four tables.
type row struct {
	op        Operation
	admin     bool
	ran       bool
	latest    Result
	jitterRow bool
	jitter    Jitter
	lossSD    uint32
	lossDS    uint32
	missing   uint32
}

// buildOperations joins the walked columns on rttMonCtrlAdminIndex. A result
// whose operation the admin table does not list is dropped, since there is
// nothing to name it by.
func buildOperations(vbs []snmp.Varbind) []Operation {
	rows := make(map[int]*row)
	for _, vb := range vbs {
		cell, ok := split(vb.OID)
		if !ok {
			continue
		}
		r := rows[cell.index]
		if r == nil {
			r = &row{op: Operation{Index: cell.index}}
			rows[cell.index] = r
		}
		r.apply(cell, vb.Value)
	}

	out := make([]Operation, 0, len(rows))
	for _, r := range rows {
		if r.admin {
			out = append(out, r.operation())
		}
	}
	slices.SortFunc(out, func(a, b Operation) int { return a.Index - b.Index })
	return out
}

func (r *row) apply(c cell, v any) {
	switch c.table {
	case ctrlAdminPrefix:
		r.admin = true
		r.applyAdmin(c.column, v)
	case echoAdminPrefix:
		if c.column == colEchoTargetAddress {
			r.op.Target = address(v)
		}
	case latestRttPrefix:
		r.ran = true
		r.applyLatest(c.column, v)
	case latestJitterTable:
		r.jitterRow = true
		r.applyJitter(c.column, v)
	}
}

func (r *row) applyAdmin(column string, v any) {
	switch column {
	case colAdminTag:
		if b, ok := v.([]byte); ok {
			r.op.Tag = string(b)
		}
	case colAdminRttType:
		r.op.RttType = integer(v)
	case colAdminThreshold:
		r.op.ThresholdMs = integer(v)
	case colAdminFrequency:
		r.op.FrequencyS = integer(v)
	case colAdminStatus:
		r.op.Status = integer(v)
	}
}

func (r *row) applyLatest(column string, v any) {
	switch column {
	case colLatestCompletionTime:
		r.latest.RTTMs = unsigned(v)
	case colLatestSense:
		r.latest.Sense = integer(v)
	case colLatestTime:
		r.latest.RunUptime = unsigned(v)
	}
}

func (r *row) applyJitter(column string, v any) {
	switch column {
	case colJitterNumOfRTT:
		r.jitter.Received = unsigned(v)
	case colJitterLossSD:
		r.lossSD = unsigned(v)
	case colJitterLossDS:
		r.lossDS = unsigned(v)
	case colJitterPacketMIA:
		r.missing = unsigned(v)
	case colJitterMOS:
		r.jitter.MOS = float64(unsigned(v)) / mosScale
	case colJitterAvgJitter:
		r.jitter.AvgJitterMs = unsigned(v)
	}
}

// operation is the row's Operation with its latest result. An operation
// that has never run reports sense other(0) at uptime 0, and has none.
func (r *row) operation() Operation {
	op := r.op
	if !r.ran || (r.latest.Sense == SenseOther && r.latest.RunUptime == 0) {
		return op
	}
	latest := r.latest
	if r.jitterRow {
		jitter := r.jitter
		jitter.Lost = saturatingSum(r.lossSD, r.lossDS, r.missing)
		latest.Jitter = &jitter
	}
	op.Latest = &latest
	return op
}

// cell addresses one walked value: <table>.<column>.<rttMonCtrlAdminIndex>.
type cell struct {
	table  string
	column string
	index  int
}

func split(oid string) (cell, bool) {
	oid = strings.TrimPrefix(oid, ".")
	for _, table := range []string{ctrlAdminPrefix, echoAdminPrefix, latestRttPrefix, latestJitterTable} {
		rest, found := strings.CutPrefix(oid, table+".")
		if !found {
			continue
		}
		column, idx, cut := strings.Cut(rest, ".")
		if !cut {
			return cell{}, false
		}
		index, err := strconv.Atoi(idx)
		if err != nil || index <= 0 {
			return cell{}, false
		}
		return cell{table: table, column: column, index: index}, true
	}
	return cell{}, false
}

// address renders an RttMonTargetAddress. IP operations carry the raw four
// or sixteen octets; other protocols carry text.
func address(v any) string {
	b, ok := v.([]byte)
	if !ok {
		return ""
	}
	if len(b) == ipv4Bytes || len(b) == ipv6Bytes {
		return net.IP(b).String()
	}
	return string(b)
}

func saturatingSum(values ...uint32) uint32 {
	var sum uint64
	for _, v := range values {
		sum += uint64(v)
	}
	return uint32(min(sum, math.MaxUint32))
}

// integer reads an INTEGER column. gosnmp decodes INTEGER as int; anything
// else, or a value outside Integer32, reads as 0.
func integer(v any) int {
	if t, ok := v.(int); ok && t >= math.MinInt32 && t <= math.MaxInt32 {
		return t
	}
	return 0
}

// unsigned reads a Gauge32 or TimeStamp column, which gosnmp decodes as uint
// and uint32 respectively.
func unsigned(v any) uint32 {
	switch t := v.(type) {
	case uint32:
		return t
	case uint:
		if t <= math.MaxUint32 {
			return uint32(t)
		}
	}
	return 0
}
