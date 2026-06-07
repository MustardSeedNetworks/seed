package hostresources_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
)

const (
	uptimeOID       = "1.3.6.1.2.1.25.1.1.0"
	storagePrefix   = "1.3.6.1.2.1.25.2.3.1"
	processorPrefix = "1.3.6.1.2.1.25.3.3.1"
)

type fakeClient struct {
	scalar     []snmp.Varbind
	storage    []snmp.Varbind
	processors []snmp.Varbind
	getErr     error
	walkErr    error
}

func (f *fakeClient) Get(_ context.Context, oids []string) ([]snmp.Varbind, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	out := make([]snmp.Varbind, 0)
	for _, want := range oids {
		for _, vb := range f.scalar {
			if vb.OID == want {
				out = append(out, vb)
			}
		}
	}
	return out, nil
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	switch {
	case strings.HasPrefix(prefix, storagePrefix):
		return f.storage, nil
	case strings.HasPrefix(prefix, processorPrefix):
		return f.processors, nil
	}
	return nil, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []hostresources.Observation
	err error
}

func (p *fakePublisher) PublishHostResources(_ context.Context, obs hostresources.Observation) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	p.got = append(p.got, obs)
	return nil
}

func factoryFor(c *fakeClient) snmp.ClientFactory {
	return func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
		return c, nil
	}
}

func at() time.Time { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

func storageOID(col string, idx uint32) string {
	return fmt.Sprintf("%s.%s.%d", storagePrefix, col, idx)
}

func processorOID(idx uint32) string {
	return fmt.Sprintf("%s.2.%d", processorPrefix, idx)
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	if hostresources.New(nil, nil, at).Name() != hostresources.Name {
		t.Error("Name mismatch")
	}
}

func TestCollect_AggregatesUptimeStorageAndProcessors(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		scalar: []snmp.Varbind{
			{OID: uptimeOID, Value: uint32(123456789)},
		},
		storage: []snmp.Varbind{
			{OID: storageOID("2", 1), Value: hostresources.StorageTypeRAM},
			{OID: storageOID("3", 1), Value: "Physical RAM"},
			{OID: storageOID("4", 1), Value: uint32(1024)},
			{OID: storageOID("5", 1), Value: 16777216}, // 16 GiB / 1024 = 16384 KiB units
			{OID: storageOID("6", 1), Value: 8388608},
			{OID: storageOID("2", 2), Value: hostresources.StorageTypeFixedDisk},
			{OID: storageOID("3", 2), Value: "/"},
			{OID: storageOID("4", 2), Value: uint32(4096)},
			{OID: storageOID("5", 2), Value: 1000000},
			{OID: storageOID("6", 2), Value: 500000},
		},
		processors: []snmp.Varbind{
			{OID: processorOID(1), Value: 42},
			{OID: processorOID(2), Value: 7},
		},
	}
	pub := &fakePublisher{}
	c := hostresources.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	obs := pub.got[0]
	if obs.SystemUptime != 123456789 {
		t.Errorf("SystemUptime = %d, want 123456789", obs.SystemUptime)
	}
	if len(obs.Storage) != 2 {
		t.Fatalf("Storage rows = %d, want 2", len(obs.Storage))
	}
	if obs.Storage[0].TypeOID != hostresources.StorageTypeRAM {
		t.Errorf("Storage[0].TypeOID = %q", obs.Storage[0].TypeOID)
	}
	if obs.Storage[0].SizeBytes != 16777216*1024 {
		t.Errorf("Storage[0].SizeBytes = %d, want %d",
			obs.Storage[0].SizeBytes, uint64(16777216)*1024)
	}
	if obs.Storage[1].UsedBytes != 500000*4096 {
		t.Errorf("Storage[1].UsedBytes = %d, want %d",
			obs.Storage[1].UsedBytes, uint64(500000)*4096)
	}
	if len(obs.Processors) != 2 {
		t.Fatalf("Processor rows = %d, want 2", len(obs.Processors))
	}
	if obs.Processors[0].LoadPct != 42 || obs.Processors[1].LoadPct != 7 {
		t.Errorf("Processor loads = %d, %d, want 42, 7",
			obs.Processors[0].LoadPct, obs.Processors[1].LoadPct)
	}
}

func TestCollect_StorageWithZeroAllocUnitsLeavesBytesZero(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		storage: []snmp.Varbind{
			{OID: storageOID("4", 1), Value: uint32(0)},
			{OID: storageOID("5", 1), Value: 100},
			{OID: storageOID("6", 1), Value: 50},
		},
	}
	pub := &fakePublisher{}
	c := hostresources.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if pub.got[0].Storage[0].SizeBytes != 0 {
		t.Errorf("zero alloc-units should yield SizeBytes=0, got %d",
			pub.got[0].Storage[0].SizeBytes)
	}
}

func TestCollect_SortedByIndex(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		storage: []snmp.Varbind{
			{OID: storageOID("3", 10), Value: "z"},
			{OID: storageOID("3", 1), Value: "a"},
			{OID: storageOID("3", 5), Value: "m"},
		},
		processors: []snmp.Varbind{
			{OID: processorOID(99), Value: 90},
			{OID: processorOID(1), Value: 10},
		},
	}
	pub := &fakePublisher{}
	c := hostresources.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	storage := pub.got[0].Storage
	if storage[0].Index != 1 || storage[1].Index != 5 || storage[2].Index != 10 {
		t.Errorf("storage sort = %d/%d/%d, want 1/5/10",
			storage[0].Index, storage[1].Index, storage[2].Index)
	}
	procs := pub.got[0].Processors
	if procs[0].Index != 1 || procs[1].Index != 99 {
		t.Errorf("processor sort = %d/%d, want 1/99", procs[0].Index, procs[1].Index)
	}
}

func TestCollect_StorageTypeAcceptsByteOID(t *testing.T) {
	t.Parallel()
	// Some agents emit ObjectIdentifier as []byte.
	fc := &fakeClient{
		storage: []snmp.Varbind{
			{OID: storageOID("2", 1), Value: []byte(hostresources.StorageTypeFixedDisk)},
		},
	}
	pub := &fakePublisher{}
	c := hostresources.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if pub.got[0].Storage[0].TypeOID != hostresources.StorageTypeFixedDisk {
		t.Errorf("byte-slice TypeOID = %q", pub.got[0].Storage[0].TypeOID)
	}
}

func TestCollect_MalformedTableOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		storage: []snmp.Varbind{
			{OID: storageOID("3", 1), Value: "good"},
			{OID: "garbage", Value: nil},
			{OID: storagePrefix + ".3.notanint", Value: nil},
		},
	}
	pub := &fakePublisher{}
	c := hostresources.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if len(pub.got[0].Storage) != 1 || pub.got[0].Storage[0].Description != "good" {
		t.Errorf("malformed should be skipped, got %+v", pub.got[0].Storage)
	}
}

func TestCollect_GetUptimeErrorPropagates(t *testing.T) {
	t.Parallel()
	c := hostresources.New(
		factoryFor(&fakeClient{getErr: errors.New("timeout")}),
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected get error")
	}
}

func TestCollect_StorageWalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := hostresources.New(
		factoryFor(&fakeClient{walkErr: errors.New("timeout")}),
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected walk error")
	}
}

func TestCollect_PublishErrorPropagates(t *testing.T) {
	t.Parallel()
	c := hostresources.New(
		factoryFor(&fakeClient{}),
		&fakePublisher{err: errors.New("topo busy")},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected publish error")
	}
}

func TestCollect_NilDepsReturnError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    *hostresources.Collector
	}{
		{"nil factory", hostresources.New(nil, &fakePublisher{}, at)},
		{"nil publisher", hostresources.New(factoryFor(&fakeClient{}), nil, at)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
				t.Error("expected nil-dep error")
			}
		})
	}
}

func TestCollect_EmptyTablesStillPublish(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := hostresources.New(factoryFor(&fakeClient{}), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if len(pub.got) != 1 {
		t.Fatalf("publisher should be called once for empty tables")
	}
	if len(pub.got[0].Storage) != 0 || len(pub.got[0].Processors) != 0 {
		t.Errorf("expected empty slices, got %+v", pub.got[0])
	}
}
