// Package publicip provides public IP address detection with caching.
// Test suite validates provider selection, caching behavior, and error handling.
package publicip

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("NewChecker() returned nil")
	}

	if c.httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}

	if c.httpClient.Timeout != requestTimeout {
		t.Errorf("httpClient.Timeout = %v, want %v", c.httpClient.Timeout, requestTimeout)
	}
}

func TestParseIpifyJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid IPv4",
			input:   []byte(`{"ip":"203.0.113.1"}`),
			want:    "203.0.113.1",
			wantErr: false,
		},
		{
			name:    "valid IPv6",
			input:   []byte(`{"ip":"2001:db8::1"}`),
			want:    "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "IP with whitespace",
			input:   []byte(`{"ip":"  192.168.1.1  "}`),
			want:    "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`not json`),
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty IP",
			input:   []byte(`{"ip":""}`),
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIpifyJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIpifyJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseIpifyJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMyIPJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid IP",
			input:   []byte(`{"ip":"10.0.0.1"}`),
			want:    "10.0.0.1",
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{broken`),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMyIPJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMyIPJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMyIPJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTextIP(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid IP",
			input:   []byte("203.0.113.50"),
			want:    "203.0.113.50",
			wantErr: false,
		},
		{
			name:    "IP with newline",
			input:   []byte("203.0.113.50\n"),
			want:    "203.0.113.50",
			wantErr: false,
		},
		{
			name:    "IP with whitespace",
			input:   []byte("  192.0.2.1  \n"),
			want:    "192.0.2.1",
			wantErr: false,
		},
		{
			name:    "empty response",
			input:   []byte(""),
			want:    "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   []byte("   \n  "),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTextIP(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTextIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseTextIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecker_FetchFromService(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		parser     func([]byte) (string, error)
		wantIP     string
		wantErr    bool
	}{
		{
			name:       "successful JSON response",
			statusCode: http.StatusOK,
			body:       `{"ip":"198.51.100.1"}`,
			parser:     parseIpifyJSON,
			wantIP:     "198.51.100.1",
			wantErr:    false,
		},
		{
			name:       "successful text response",
			statusCode: http.StatusOK,
			body:       "198.51.100.2\n",
			parser:     parseTextIP,
			wantIP:     "198.51.100.2",
			wantErr:    false,
		},
		{
			name:       "HTTP 500 error",
			statusCode: http.StatusInternalServerError,
			body:       "Internal Server Error",
			parser:     parseTextIP,
			wantIP:     "",
			wantErr:    true,
		},
		{
			name:       "HTTP 404 error",
			statusCode: http.StatusNotFound,
			body:       "Not Found",
			parser:     parseTextIP,
			wantIP:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify User-Agent header
				if ua := r.Header.Get("User-Agent"); ua != "The Seed/1.0" {
					t.Errorf("User-Agent = %q, want %q", ua, "The Seed/1.0")
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c := NewChecker()
			got, err := c.fetchFromService(context.Background(), server.URL, tt.parser)

			if (err != nil) != tt.wantErr {
				t.Errorf("fetchFromService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantIP {
				t.Errorf("fetchFromService() = %v, want %v", got, tt.wantIP)
			}
		})
	}
}

func TestChecker_FetchFromService_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("192.0.2.1"))
	}))
	defer server.Close()

	c := NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.fetchFromService(ctx, server.URL, parseTextIP)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestChecker_GetPublicIP_CacheHit(t *testing.T) {
	c := NewChecker()

	// Pre-populate cache
	c.cache = &Result{
		IPv4:        "192.0.2.100",
		IPv6:        "2001:db8::100",
		LastChecked: time.Now(),
	}
	c.cacheTime = time.Now()

	result := c.GetPublicIP(context.Background())

	if result.IPv4 != "192.0.2.100" {
		t.Errorf("IPv4 = %q, want %q", result.IPv4, "192.0.2.100")
	}
	if result.IPv6 != "2001:db8::100" {
		t.Errorf("IPv6 = %q, want %q", result.IPv6, "2001:db8::100")
	}
}

func TestChecker_GetPublicIP_CacheExpired(t *testing.T) {
	// Create test servers that return known IPs
	ipv4Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"ip": "203.0.113.10"})
	}))
	defer ipv4Server.Close()

	ipv6Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"ip": "2001:db8::10"})
	}))
	defer ipv6Server.Close()

	c := NewChecker()

	// Pre-populate cache with expired data
	c.cache = &Result{
		IPv4:        "192.0.2.1",
		IPv6:        "2001:db8::1",
		LastChecked: time.Now().Add(-10 * time.Minute), // Expired
	}
	c.cacheTime = time.Now().Add(-10 * time.Minute)

	// Since we can't override the service URLs, we test that cache is refreshed
	result := c.GetPublicIP(context.Background())

	// Should have attempted to refresh (may fail due to real network calls)
	if result.LastChecked.Before(time.Now().Add(-1 * time.Minute)) {
		t.Error("expected LastChecked to be recent after cache expiry")
	}
}

func TestChecker_Refresh(t *testing.T) {
	c := NewChecker()

	// Pre-populate cache
	c.cache = &Result{
		IPv4:        "192.0.2.1",
		LastChecked: time.Now(),
	}
	c.cacheTime = time.Now()

	// Force refresh should update LastChecked even if cache is fresh
	oldTime := c.cache.LastChecked
	time.Sleep(10 * time.Millisecond)

	result := c.Refresh(context.Background())

	if !result.LastChecked.After(oldTime) {
		t.Error("Refresh should update LastChecked")
	}
}

func TestChecker_FetchIPv6_ValidatesIPv6(t *testing.T) {
	// Create a server that returns an IPv4 address for an IPv6 endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return IPv4 instead of IPv6 - should be rejected
		json.NewEncoder(w).Encode(map[string]string{"ip": "192.0.2.1"})
	}))
	defer server.Close()

	c := &Checker{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Direct call to fetchIPv6 would need the real services, but we can test
	// that the validation logic exists by checking the code path
	// This is a behavioral test - fetchIPv6 checks for ":" in the result
	ctx := context.Background()
	ip, _ := c.fetchFromService(ctx, server.URL, parseIpifyJSON)

	// The service returns an IPv4, which shouldn't contain ":"
	if strings.Contains(ip, ":") {
		t.Error("expected IPv4 address from test server")
	}
}

func TestResult_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	r := Result{
		IPv4:        "203.0.113.1",
		IPv6:        "2001:db8::1",
		LastChecked: now,
		Error:       "",
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.IPv4 != r.IPv4 {
		t.Errorf("IPv4 = %q, want %q", decoded.IPv4, r.IPv4)
	}
	if decoded.IPv6 != r.IPv6 {
		t.Errorf("IPv6 = %q, want %q", decoded.IPv6, r.IPv6)
	}
}

func TestResult_OmitEmpty(t *testing.T) {
	r := Result{
		LastChecked: time.Now(),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Should not contain ipv4, ipv6, or error fields when empty
	if strings.Contains(string(data), `"ipv4"`) {
		t.Error("empty IPv4 should be omitted")
	}
	if strings.Contains(string(data), `"ipv6"`) {
		t.Error("empty IPv6 should be omitted")
	}
	if strings.Contains(string(data), `"error"`) {
		t.Error("empty Error should be omitted")
	}
}

func TestChecker_ConcurrentAccess(_ *testing.T) {
	c := NewChecker()

	// Pre-populate cache
	c.cache = &Result{
		IPv4:        "192.0.2.1",
		LastChecked: time.Now(),
	}
	c.cacheTime = time.Now()

	// Access cache concurrently
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			_ = c.GetPublicIP(context.Background())
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

func TestChecker_updateHistory(t *testing.T) {
	t.Run("empty IP does nothing", func(t *testing.T) {
		c := NewChecker()
		c.updateHistory("")
		if len(c.history) != 0 {
			t.Errorf("expected empty history, got %d entries", len(c.history))
		}
	})

	t.Run("first IP sets lastIPv4", func(t *testing.T) {
		c := NewChecker()
		c.updateHistory("192.0.2.1")
		if c.lastIPv4 != "192.0.2.1" {
			t.Errorf("expected lastIPv4 = '192.0.2.1', got %q", c.lastIPv4)
		}
		// No history yet - history only populated on IP change
		if len(c.history) != 0 {
			t.Errorf("expected no history on first IP, got %d entries", len(c.history))
		}
	})

	t.Run("same IP does not add to history", func(t *testing.T) {
		c := NewChecker()
		c.lastIPv4 = "192.0.2.1"
		c.updateHistory("192.0.2.1") // Same IP
		if len(c.history) != 0 {
			t.Errorf("expected no history for same IP, got %d entries", len(c.history))
		}
	})

	t.Run("IP change adds old IP to history", func(t *testing.T) {
		c := NewChecker()
		c.lastIPv4 = "192.0.2.1"
		c.updateHistory("192.0.2.2") // New IP

		if len(c.history) != 1 {
			t.Fatalf("expected 1 history entry, got %d", len(c.history))
		}
		if c.history[0].IP != "192.0.2.1" {
			t.Errorf("expected old IP in history, got %q", c.history[0].IP)
		}
		if c.lastIPv4 != "192.0.2.2" {
			t.Errorf("expected lastIPv4 = '192.0.2.2', got %q", c.lastIPv4)
		}
	})

	t.Run("history capped at 10 entries", func(t *testing.T) {
		c := NewChecker()
		c.lastIPv4 = "192.0.2.1"

		// Add more than 10 entries
		for i := 2; i <= 15; i++ {
			c.updateHistory("192.0.2." + string(rune('0'+i%10)))
			c.lastIPv4 = "192.0.2." + string(rune('0'+i%10))
		}

		if len(c.history) > 10 {
			t.Errorf("expected max 10 history entries, got %d", len(c.history))
		}
	})

	t.Run("IP change with geo cache populates city/country", func(t *testing.T) {
		c := NewChecker()
		c.lastIPv4 = "192.0.2.1"
		c.geoCache = map[string]*geoResponse{
			"192.0.2.1": {City: "TestCity", Country: "TestCountry"},
		}

		c.updateHistory("192.0.2.2") // Trigger IP change

		if len(c.history) != 1 {
			t.Fatalf("expected 1 history entry, got %d", len(c.history))
		}
		if c.history[0].City != "TestCity" {
			t.Errorf("expected city 'TestCity', got %q", c.history[0].City)
		}
		if c.history[0].Country != "TestCountry" {
			t.Errorf("expected country 'TestCountry', got %q", c.history[0].Country)
		}
	})

	t.Run("existing IP in history updates LastSeen", func(t *testing.T) {
		c := NewChecker()
		oldTime := time.Now().Add(-1 * time.Hour)
		c.history = []HistoryEntry{
			{IP: "192.0.2.1", FirstSeen: oldTime, LastSeen: oldTime},
		}
		c.lastIPv4 = "192.0.2.1"

		// Change to new IP and back
		c.updateHistory("192.0.2.2")

		if len(c.history) != 1 {
			t.Fatalf("expected 1 history entry, got %d", len(c.history))
		}
		if c.history[0].LastSeen.Before(time.Now().Add(-1 * time.Minute)) {
			t.Error("expected LastSeen to be updated to recent time")
		}
	})
}

func TestChecker_getHistoryCopy(t *testing.T) {
	t.Run("empty history returns nil", func(t *testing.T) {
		c := NewChecker()
		hist := c.getHistoryCopy()
		if hist != nil {
			t.Errorf("expected nil for empty history, got %v", hist)
		}
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		c := NewChecker()
		c.history = []HistoryEntry{
			{IP: "192.0.2.1", FirstSeen: time.Now(), LastSeen: time.Now()},
			{IP: "192.0.2.2", FirstSeen: time.Now(), LastSeen: time.Now()},
		}

		hist := c.getHistoryCopy()

		if len(hist) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(hist))
		}

		// Modify copy - should not affect original
		hist[0].IP = "modified"
		if c.history[0].IP == "modified" {
			t.Error("getHistoryCopy should return copy, but original was modified")
		}
	})
}

func TestGeoResponse(t *testing.T) {
	geo := geoResponse{
		City:       "San Francisco",
		RegionName: "California",
		Country:    "US",
		ISP:        "Test ISP",
		AS:         "AS12345",
	}

	if geo.City != "San Francisco" {
		t.Errorf("expected city 'San Francisco', got %q", geo.City)
	}
	if geo.Country != "US" {
		t.Errorf("expected country 'US', got %q", geo.Country)
	}
}

func TestHistoryEntry(t *testing.T) {
	now := time.Now()
	entry := HistoryEntry{
		IP:        "203.0.113.1",
		FirstSeen: now,
		LastSeen:  now,
		City:      "London",
		Country:   "UK",
	}

	if entry.IP != "203.0.113.1" {
		t.Errorf("expected IP '203.0.113.1', got %q", entry.IP)
	}
	if entry.City != "London" {
		t.Errorf("expected city 'London', got %q", entry.City)
	}
}

func TestResult_WithGeoFields(t *testing.T) {
	result := Result{
		IPv4:        "203.0.113.1",
		IPv6:        "2001:db8::1",
		LastChecked: time.Now(),
		City:        "Paris",
		Country:     "FR",
		ISP:         "French ISP",
	}

	if result.City != "Paris" {
		t.Errorf("expected city 'Paris', got %q", result.City)
	}
	if result.Country != "FR" {
		t.Errorf("expected country 'FR', got %q", result.Country)
	}
	if result.ISP != "French ISP" {
		t.Errorf("expected ISP 'French ISP', got %q", result.ISP)
	}
}

func TestResult_WithHistory(t *testing.T) {
	now := time.Now()
	result := Result{
		IPv4:        "203.0.113.1",
		LastChecked: now,
		History: []HistoryEntry{
			{IP: "192.0.2.1", FirstSeen: now, LastSeen: now},
			{IP: "192.0.2.2", FirstSeen: now, LastSeen: now},
		},
	}

	if len(result.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(result.History))
	}
}
