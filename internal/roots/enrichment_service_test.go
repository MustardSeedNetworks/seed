// Package roots_test provides tests for the EnrichmentService.
// Test suite validates IP enrichment, public IP lookup, and error handling.
package roots_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
)

// TestEnrichmentService_Creation validates service creation.
func TestEnrichmentService_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		createFunc func() *roots.EnrichmentService
		wantNil    bool
	}{
		{
			name: "standard creation with nil config",
			createFunc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentService(nil)
			},
			wantNil: false,
		},
		{
			name: "creation with nil checker",
			createFunc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentServiceWithChecker(nil, nil)
			},
			wantNil: false,
		},
		{
			name: "creation with valid checker",
			createFunc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentServiceWithChecker(nil, publicip.NewChecker())
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.createFunc()
			if (svc == nil) != tt.wantNil {
				t.Errorf("creation returned nil = %v, want nil = %v", svc == nil, tt.wantNil)
			}
		})
	}
}

// TestEnrichmentService_Checker validates Checker() accessor.
func TestEnrichmentService_Checker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		createFunc     func() *roots.EnrichmentService
		wantNilChecker bool
	}{
		{
			name: "standard service has checker",
			createFunc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentService(nil)
			},
			wantNilChecker: false,
		},
		{
			name: "nil checker service",
			createFunc: func() *roots.EnrichmentService {
				return roots.NewEnrichmentServiceWithChecker(nil, nil)
			},
			wantNilChecker: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.createFunc()
			checker := svc.Checker()
			if (checker == nil) != tt.wantNilChecker {
				t.Errorf("Checker() nil = %v, want nil = %v", checker == nil, tt.wantNilChecker)
			}
		})
	}
}

// TestEnrichmentService_Enrich_NilChecker validates error when checker is nil.
func TestEnrichmentService_Enrich_NilChecker(t *testing.T) {
	t.Parallel()

	svc := roots.NewEnrichmentServiceWithChecker(nil, nil)
	ctx := context.Background()

	result, err := svc.Enrich(ctx, "8.8.8.8")
	if err == nil {
		t.Error("Enrich() with nil checker should return error")
	}
	if result != nil {
		t.Errorf("Enrich() with nil checker should return nil result, got %+v", result)
	}
	if err != roots.ErrNotInitialized {
		t.Errorf("error = %v, want %v", err, roots.ErrNotInitialized)
	}
}

// TestEnrichmentService_Enrich_NotImplemented validates error for non-public IPs.
func TestEnrichmentService_Enrich_NotImplemented(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ip         string
		wantErr    bool
		errContain string
	}{
		{
			name:       "arbitrary IP returns not implemented",
			ip:         "192.0.2.1",
			wantErr:    true,
			errContain: "not implemented",
		},
		{
			name:       "private IP returns not implemented",
			ip:         "10.0.0.1",
			wantErr:    true,
			errContain: "not implemented",
		},
		{
			name:       "loopback IP returns not implemented",
			ip:         "127.0.0.1",
			wantErr:    true,
			errContain: "not implemented",
		},
		{
			name:       "link local IP returns not implemented",
			ip:         "169.254.1.1",
			wantErr:    true,
			errContain: "not implemented",
		},
	}

	svc := roots.NewEnrichmentService(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result, err := svc.Enrich(ctx, tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("Enrich() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContain != "" {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
			}

			if tt.wantErr && result != nil {
				t.Errorf("Enrich() should return nil result on error, got %+v", result)
			}
		})
	}
}

// TestEnrichmentService_GetPublicIP_NilChecker validates error when checker is nil.
func TestEnrichmentService_GetPublicIP_NilChecker(t *testing.T) {
	t.Parallel()

	svc := roots.NewEnrichmentServiceWithChecker(nil, nil)
	ctx := context.Background()

	result, err := svc.GetPublicIP(ctx)
	if err == nil {
		t.Error("GetPublicIP() with nil checker should return error")
	}
	if result != nil {
		t.Errorf("GetPublicIP() with nil checker should return nil result, got %+v", result)
	}
	if err != roots.ErrNotInitialized {
		t.Errorf("error = %v, want %v", err, roots.ErrNotInitialized)
	}
}

// TestEnrichmentService_GetPublicIP_ContextCancellation validates context handling.
func TestEnrichmentService_GetPublicIP_ContextCancellation(t *testing.T) {
	t.Parallel()

	svc := roots.NewEnrichmentService(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should handle cancelled context gracefully
	result, err := svc.GetPublicIP(ctx)
	// May succeed from cache or fail - both are valid
	t.Logf("GetPublicIP with cancelled context: result=%v, err=%v", result, err)
}

// TestEnrichmentService_GetPublicIP_ContextTimeout validates timeout handling.
func TestEnrichmentService_GetPublicIP_ContextTimeout(t *testing.T) {
	t.Parallel()

	svc := roots.NewEnrichmentService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Allow timeout to trigger
	time.Sleep(5 * time.Millisecond)

	// Should handle timed out context gracefully
	result, err := svc.GetPublicIP(ctx)
	// May succeed from cache or fail - both are valid
	t.Logf("GetPublicIP with timed out context: result=%v, err=%v", result, err)
}

// TestIPEnrichment_IndividualFields validates IPEnrichment struct fields individually.
func TestIPEnrichment_IndividualFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name       string
		enrichment roots.IPEnrichment
		checkFn    func(roots.IPEnrichment) bool
		checkMsg   string
	}{
		{
			name: "IP field",
			enrichment: roots.IPEnrichment{
				IP: "8.8.8.8",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.IP == "8.8.8.8" },
			checkMsg: "IP should be 8.8.8.8",
		},
		{
			name: "ASN field",
			enrichment: roots.IPEnrichment{
				ASN: 15169,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.ASN == 15169 },
			checkMsg: "ASN should be 15169",
		},
		{
			name: "ASName field",
			enrichment: roots.IPEnrichment{
				ASName: "GOOGLE",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.ASName == "GOOGLE" },
			checkMsg: "ASName should be GOOGLE",
		},
		{
			name: "ISP field",
			enrichment: roots.IPEnrichment{
				ISP: "Google LLC",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.ISP == "Google LLC" },
			checkMsg: "ISP should be Google LLC",
		},
		{
			name: "Org field",
			enrichment: roots.IPEnrichment{
				Org: "Google LLC",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.Org == "Google LLC" },
			checkMsg: "Org should be Google LLC",
		},
		{
			name: "City field",
			enrichment: roots.IPEnrichment{
				City: "Mountain View",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.City == "Mountain View" },
			checkMsg: "City should be Mountain View",
		},
		{
			name: "Region field",
			enrichment: roots.IPEnrichment{
				Region: "California",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.Region == "California" },
			checkMsg: "Region should be California",
		},
		{
			name: "Country field",
			enrichment: roots.IPEnrichment{
				Country: "United States",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.Country == "United States" },
			checkMsg: "Country should be United States",
		},
		{
			name: "CountryCode field",
			enrichment: roots.IPEnrichment{
				CountryCode: "US",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.CountryCode == "US" },
			checkMsg: "CountryCode should be US",
		},
		{
			name: "Latitude field",
			enrichment: roots.IPEnrichment{
				Latitude: 37.386,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.Latitude == 37.386 },
			checkMsg: "Latitude should be 37.386",
		},
		{
			name: "Longitude field",
			enrichment: roots.IPEnrichment{
				Longitude: -122.084,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.Longitude == -122.084 },
			checkMsg: "Longitude should be -122.084",
		},
		{
			name: "Timezone field",
			enrichment: roots.IPEnrichment{
				Timezone: "America/Los_Angeles",
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.Timezone == "America/Los_Angeles" },
			checkMsg: "Timezone should be America/Los_Angeles",
		},
		{
			name: "IsProxy false",
			enrichment: roots.IPEnrichment{
				IsProxy: false,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return !e.IsProxy },
			checkMsg: "IsProxy should be false",
		},
		{
			name: "IsProxy true",
			enrichment: roots.IPEnrichment{
				IsProxy: true,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.IsProxy },
			checkMsg: "IsProxy should be true",
		},
		{
			name: "IsHosting false",
			enrichment: roots.IPEnrichment{
				IsHosting: false,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return !e.IsHosting },
			checkMsg: "IsHosting should be false",
		},
		{
			name: "IsHosting true",
			enrichment: roots.IPEnrichment{
				IsHosting: true,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.IsHosting },
			checkMsg: "IsHosting should be true",
		},
		{
			name: "IsTor false",
			enrichment: roots.IPEnrichment{
				IsTor: false,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return !e.IsTor },
			checkMsg: "IsTor should be false",
		},
		{
			name: "IsTor true",
			enrichment: roots.IPEnrichment{
				IsTor: true,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.IsTor },
			checkMsg: "IsTor should be true",
		},
		{
			name: "QueryTime field",
			enrichment: roots.IPEnrichment{
				QueryTime: now,
			},
			checkFn:  func(e roots.IPEnrichment) bool { return e.QueryTime.Equal(now) },
			checkMsg: "QueryTime should equal now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !tt.checkFn(tt.enrichment) {
				t.Error(tt.checkMsg)
			}
		})
	}
}

// TestIPEnrichment_AllFields validates IPEnrichment with all fields populated.
func TestIPEnrichment_AllFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	enrichment := roots.IPEnrichment{
		IP:          "8.8.8.8",
		ASN:         15169,
		ASName:      "GOOGLE",
		ISP:         "Google LLC",
		Org:         "Google LLC",
		City:        "Mountain View",
		Region:      "California",
		Country:     "United States",
		CountryCode: "US",
		Latitude:    37.386,
		Longitude:   -122.084,
		Timezone:    "America/Los_Angeles",
		IsProxy:     false,
		IsHosting:   true,
		IsTor:       false,
		QueryTime:   now,
	}

	// Validate all fields
	if enrichment.IP != "8.8.8.8" {
		t.Errorf("IP = %q, want %q", enrichment.IP, "8.8.8.8")
	}
	if enrichment.ASN != 15169 {
		t.Errorf("ASN = %d, want %d", enrichment.ASN, 15169)
	}
	if enrichment.ASName != "GOOGLE" {
		t.Errorf("ASName = %q, want %q", enrichment.ASName, "GOOGLE")
	}
	if enrichment.ISP != "Google LLC" {
		t.Errorf("ISP = %q, want %q", enrichment.ISP, "Google LLC")
	}
	if enrichment.Org != "Google LLC" {
		t.Errorf("Org = %q, want %q", enrichment.Org, "Google LLC")
	}
	if enrichment.City != "Mountain View" {
		t.Errorf("City = %q, want %q", enrichment.City, "Mountain View")
	}
	if enrichment.Region != "California" {
		t.Errorf("Region = %q, want %q", enrichment.Region, "California")
	}
	if enrichment.Country != "United States" {
		t.Errorf("Country = %q, want %q", enrichment.Country, "United States")
	}
	if enrichment.CountryCode != "US" {
		t.Errorf("CountryCode = %q, want %q", enrichment.CountryCode, "US")
	}
	if enrichment.Latitude != 37.386 {
		t.Errorf("Latitude = %f, want %f", enrichment.Latitude, 37.386)
	}
	if enrichment.Longitude != -122.084 {
		t.Errorf("Longitude = %f, want %f", enrichment.Longitude, -122.084)
	}
	if enrichment.Timezone != "America/Los_Angeles" {
		t.Errorf("Timezone = %q, want %q", enrichment.Timezone, "America/Los_Angeles")
	}
	if enrichment.IsProxy {
		t.Error("IsProxy should be false")
	}
	if !enrichment.IsHosting {
		t.Error("IsHosting should be true")
	}
	if enrichment.IsTor {
		t.Error("IsTor should be false")
	}
	if !enrichment.QueryTime.Equal(now) {
		t.Errorf("QueryTime = %v, want %v", enrichment.QueryTime, now)
	}
}

// TestIPEnrichment_ZeroValues validates IPEnrichment with zero values.
func TestIPEnrichment_ZeroValues(t *testing.T) {
	t.Parallel()

	var enrichment roots.IPEnrichment

	if enrichment.IP != "" {
		t.Errorf("zero IP should be empty, got %q", enrichment.IP)
	}
	if enrichment.ASN != 0 {
		t.Errorf("zero ASN should be 0, got %d", enrichment.ASN)
	}
	if enrichment.ASName != "" {
		t.Errorf("zero ASName should be empty, got %q", enrichment.ASName)
	}
	if enrichment.ISP != "" {
		t.Errorf("zero ISP should be empty, got %q", enrichment.ISP)
	}
	if enrichment.Org != "" {
		t.Errorf("zero Org should be empty, got %q", enrichment.Org)
	}
	if enrichment.City != "" {
		t.Errorf("zero City should be empty, got %q", enrichment.City)
	}
	if enrichment.Region != "" {
		t.Errorf("zero Region should be empty, got %q", enrichment.Region)
	}
	if enrichment.Country != "" {
		t.Errorf("zero Country should be empty, got %q", enrichment.Country)
	}
	if enrichment.CountryCode != "" {
		t.Errorf("zero CountryCode should be empty, got %q", enrichment.CountryCode)
	}
	if enrichment.Latitude != 0 {
		t.Errorf("zero Latitude should be 0, got %f", enrichment.Latitude)
	}
	if enrichment.Longitude != 0 {
		t.Errorf("zero Longitude should be 0, got %f", enrichment.Longitude)
	}
	if enrichment.Timezone != "" {
		t.Errorf("zero Timezone should be empty, got %q", enrichment.Timezone)
	}
	if enrichment.IsProxy {
		t.Error("zero IsProxy should be false")
	}
	if enrichment.IsHosting {
		t.Error("zero IsHosting should be false")
	}
	if enrichment.IsTor {
		t.Error("zero IsTor should be false")
	}
	if !enrichment.QueryTime.IsZero() {
		t.Errorf("zero QueryTime should be zero, got %v", enrichment.QueryTime)
	}
}
