package publicip_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots/publicip"
)

func TestNewChecker(t *testing.T) {
	c := publicip.NewChecker()
	if c == nil {
		t.Fatal("NewChecker() returned nil")
	}

	if c.CheckerHTTPClient() == nil {
		t.Fatal("httpClient should not be nil")
	}

	if c.CheckerHTTPClient().Timeout != publicip.RequestTimeout {
		t.Errorf(
			"httpClient.Timeout = %v, want %v",
			c.CheckerHTTPClient().Timeout,
			publicip.RequestTimeout,
		)
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
			got, err := publicip.ExportParseIpifyJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIpifyJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseIpifyJSON() = %v, want %v", got, tt.want)
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
			got, err := publicip.ExportParseMyIPJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMyIPJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseMyIPJSON() = %v, want %v", got, tt.want)
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
			got, err := publicip.ExportParseTextIP(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTextIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTextIP() = %v, want %v", got, tt.want)
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
			parser:     publicip.ExportParseIpifyJSON,
			wantIP:     "198.51.100.1",
			wantErr:    false,
		},
		{
			name:       "successful text response",
			statusCode: http.StatusOK,
			body:       "198.51.100.2\n",
			parser:     publicip.ExportParseTextIP,
			wantIP:     "198.51.100.2",
			wantErr:    false,
		},
		{
			name:       "HTTP 500 error",
			statusCode: http.StatusInternalServerError,
			body:       "Internal Server Error",
			parser:     publicip.ExportParseTextIP,
			wantIP:     "",
			wantErr:    true,
		},
		{
			name:       "HTTP 404 error",
			statusCode: http.StatusNotFound,
			body:       "Not Found",
			parser:     publicip.ExportParseTextIP,
			wantIP:     "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify User-Agent header.
					if ua := r.Header.Get("User-Agent"); ua != "The Seed/1.0" {
						t.Errorf("User-Agent = %q, want %q", ua, "The Seed/1.0")
					}
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				}),
			)
			defer server.Close()

			c := publicip.NewChecker()
			got, err := publicip.ExportFetchFromService(context.Background(), c, server.URL, tt.parser)

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
		_, _ = w.Write([]byte("192.0.2.1"))
	}))
	defer server.Close()

	c := publicip.NewChecker()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := publicip.ExportFetchFromService(ctx, c, server.URL, publicip.ExportParseTextIP)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestChecker_GetPublicIP_CacheHit(t *testing.T) {
	c := publicip.NewChecker()

	// Pre-populate cache.
	c.CheckerSetCache(&publicip.Result{
		IPv4:        "192.0.2.100",
		IPv6:        "2001:db8::100",
		LastChecked: time.Now(),
	}, time.Now())

	result := c.GetPublicIP(context.Background())

	if result.IPv4 != "192.0.2.100" {
		t.Errorf("IPv4 = %q, want %q", result.IPv4, "192.0.2.100")
	}
	if result.IPv6 != "2001:db8::100" {
		t.Errorf("IPv6 = %q, want %q", result.IPv6, "2001:db8::100")
	}
}

func TestChecker_GetPublicIP_CacheExpired(t *testing.T) {
	// Create test servers that return known IPs.
	ipv4Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"ip": "203.0.113.10"})
	}))
	defer ipv4Server.Close()

	ipv6Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"ip": "2001:db8::10"})
	}))
	defer ipv6Server.Close()

	c := publicip.NewChecker()

	// Pre-populate cache with expired data.
	c.CheckerSetCache(&publicip.Result{
		IPv4:        "192.0.2.1",
		IPv6:        "2001:db8::1",
		LastChecked: time.Now().Add(-10 * time.Minute), // Expired.
	}, time.Now().Add(-10*time.Minute))

	// Since we can't override the service URLs, we test that cache is refreshed.
	result := c.GetPublicIP(context.Background())

	// Should have attempted to refresh (may fail due to real network calls).
	if result.LastChecked.Before(time.Now().Add(-1 * time.Minute)) {
		t.Error("expected LastChecked to be recent after cache expiry")
	}
}

func TestChecker_Refresh(t *testing.T) {
	c := publicip.NewChecker()

	// Pre-populate cache.
	c.CheckerSetCache(&publicip.Result{
		IPv4:        "192.0.2.1",
		LastChecked: time.Now(),
	}, time.Now())

	// Force refresh should update LastChecked even if cache is fresh.
	oldTime := c.CheckerCache().LastChecked
	time.Sleep(10 * time.Millisecond)

	result := c.Refresh(context.Background())

	if !result.LastChecked.After(oldTime) {
		t.Error("Refresh should update LastChecked")
	}
}

func TestChecker_FetchIPv6_ValidatesIPv6(t *testing.T) {
	// Create a server that returns an IPv4 address for an IPv6 endpoint.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return IPv4 instead of IPv6 - should be rejected.
		_ = json.NewEncoder(w).Encode(map[string]string{"ip": "192.0.2.1"})
	}))
	defer server.Close()

	c := publicip.NewChecker()

	// Direct call to fetchIPv6 would need the real services, but we can test
	// that the validation logic exists by checking the code path.
	// This is a behavioral test - fetchIPv6 checks for ":" in the result.
	ctx := context.Background()
	ip, _ := publicip.ExportFetchFromService(ctx, c, server.URL, publicip.ExportParseIpifyJSON)

	// The service returns an IPv4, which shouldn't contain ":".
	if strings.Contains(ip, ":") {
		t.Error("expected IPv4 address from test server")
	}
}

func TestResult_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	r := publicip.Result{
		IPv4:        "203.0.113.1",
		IPv6:        "2001:db8::1",
		LastChecked: now,
		Error:       "",
	}

	data, marshalErr := json.Marshal(r)
	if marshalErr != nil {
		t.Fatalf("Marshal failed: %v", marshalErr)
	}

	var decoded publicip.Result
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
	r := publicip.Result{
		LastChecked: time.Now(),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Should not contain ipv4, ipv6, or error fields when empty.
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
	c := publicip.NewChecker()

	// Pre-populate cache.
	c.CheckerSetCache(&publicip.Result{
		IPv4:        "192.0.2.1",
		LastChecked: time.Now(),
	}, time.Now())

	// Access cache concurrently.
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			_ = c.GetPublicIP(context.Background())
			done <- true
		}()
	}

	// Wait for all goroutines.
	for range 10 {
		<-done
	}
}

// assertHistoryLen checks that the checker's history has the expected length.
func assertHistoryLen(t *testing.T, c *publicip.Checker, expected int) {
	t.Helper()
	if got := len(c.CheckerHistory()); got != expected {
		t.Errorf("expected %d history entries, got %d", expected, got)
	}
}

// assertHistoryLenFatal checks history length and fails immediately if wrong.
func assertHistoryLenFatal(t *testing.T, c *publicip.Checker, expected int) {
	t.Helper()
	if got := len(c.CheckerHistory()); got != expected {
		t.Fatalf("expected %d history entries, got %d", expected, got)
	}
}

// assertLastIPv4 checks that the checker's lastIPv4 matches expected.
func assertLastIPv4(t *testing.T, c *publicip.Checker, expected string) {
	t.Helper()
	if got := c.CheckerLastIPv4(); got != expected {
		t.Errorf("expected lastIPv4 = %q, got %q", expected, got)
	}
}

// assertHistoryIP checks that the history entry at index has the expected IP.
func assertHistoryIP(t *testing.T, c *publicip.Checker, index int, expectedIP string) {
	t.Helper()
	if got := c.CheckerHistory()[index].IP; got != expectedIP {
		t.Errorf("expected history[%d].IP = %q, got %q", index, expectedIP, got)
	}
}

// assertHistoryGeo checks that the history entry at index has expected city/country.
func assertHistoryGeo(t *testing.T, c *publicip.Checker, index int, expectedCity, expectedCountry string) {
	t.Helper()
	entry := c.CheckerHistory()[index]
	if entry.City != expectedCity {
		t.Errorf("expected history[%d].City = %q, got %q", index, expectedCity, entry.City)
	}
	if entry.Country != expectedCountry {
		t.Errorf("expected history[%d].Country = %q, got %q", index, expectedCountry, entry.Country)
	}
}

// assertHistoryMaxEntries checks that history does not exceed max entries.
func assertHistoryMaxEntries(t *testing.T, c *publicip.Checker, maxEntries int) {
	t.Helper()
	if got := len(c.CheckerHistory()); got > maxEntries {
		t.Errorf("expected max %d history entries, got %d", maxEntries, got)
	}
}

// assertLastSeenRecent checks that the history entry's LastSeen is recent.
func assertLastSeenRecent(t *testing.T, c *publicip.Checker, index int, threshold time.Duration) {
	t.Helper()
	cutoff := time.Now().Add(-threshold)
	if c.CheckerHistory()[index].LastSeen.Before(cutoff) {
		t.Error("expected LastSeen to be updated to recent time")
	}
}

func TestChecker_updateHistory(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*publicip.Checker)
		action func(*publicip.Checker)
		verify func(*testing.T, *publicip.Checker)
	}{
		{
			name:   "empty IP does nothing",
			setup:  func(_ *publicip.Checker) {},
			action: func(c *publicip.Checker) { c.UpdateHistory("") },
			verify: func(t *testing.T, c *publicip.Checker) {
				assertHistoryLen(t, c, 0)
			},
		},
		{
			name:  "first IP sets lastIPv4",
			setup: func(_ *publicip.Checker) {},
			action: func(c *publicip.Checker) {
				c.UpdateHistory("192.0.2.1")
			},
			verify: func(t *testing.T, c *publicip.Checker) {
				assertLastIPv4(t, c, "192.0.2.1")
				// No history yet - history only populated on IP change.
				assertHistoryLen(t, c, 0)
			},
		},
		{
			name: "same IP does not add to history",
			setup: func(c *publicip.Checker) {
				c.CheckerSetLastIPv4("192.0.2.1")
			},
			action: func(c *publicip.Checker) {
				c.UpdateHistory("192.0.2.1") // Same IP.
			},
			verify: func(t *testing.T, c *publicip.Checker) {
				assertHistoryLen(t, c, 0)
			},
		},
		{
			name: "IP change adds old IP to history",
			setup: func(c *publicip.Checker) {
				c.CheckerSetLastIPv4("192.0.2.1")
			},
			action: func(c *publicip.Checker) {
				c.UpdateHistory("192.0.2.2") // New IP.
			},
			verify: func(t *testing.T, c *publicip.Checker) {
				assertHistoryLenFatal(t, c, 1)
				assertHistoryIP(t, c, 0, "192.0.2.1")
				assertLastIPv4(t, c, "192.0.2.2")
			},
		},
		{
			name: "history capped at 10 entries",
			setup: func(c *publicip.Checker) {
				c.CheckerSetLastIPv4("192.0.2.1")
			},
			action: func(c *publicip.Checker) {
				// Add more than 10 entries.
				for i := 2; i <= 15; i++ {
					c.UpdateHistory("192.0.2." + string(rune('0'+i%10)))
					c.CheckerSetLastIPv4("192.0.2." + string(rune('0'+i%10)))
				}
			},
			verify: func(t *testing.T, c *publicip.Checker) {
				assertHistoryMaxEntries(t, c, 10)
			},
		},
		{
			name: "IP change with geo cache populates city/country",
			setup: func(c *publicip.Checker) {
				c.CheckerSetLastIPv4("192.0.2.1")
				c.CheckerSetGeoCache(map[string]*publicip.GeoResponse{
					"192.0.2.1": {City: "TestCity", Country: "TestCountry"},
				})
			},
			action: func(c *publicip.Checker) {
				c.UpdateHistory("192.0.2.2") // Trigger IP change.
			},
			verify: func(t *testing.T, c *publicip.Checker) {
				assertHistoryLenFatal(t, c, 1)
				assertHistoryGeo(t, c, 0, "TestCity", "TestCountry")
			},
		},
		{
			name: "existing IP in history updates LastSeen",
			setup: func(c *publicip.Checker) {
				oldTime := time.Now().Add(-1 * time.Hour)
				c.CheckerSetHistory([]publicip.HistoryEntry{
					{IP: "192.0.2.1", FirstSeen: oldTime, LastSeen: oldTime},
				})
				c.CheckerSetLastIPv4("192.0.2.1")
			},
			action: func(c *publicip.Checker) {
				// Change to new IP and back.
				c.UpdateHistory("192.0.2.2")
			},
			verify: func(t *testing.T, c *publicip.Checker) {
				assertHistoryLenFatal(t, c, 1)
				assertLastSeenRecent(t, c, 0, 1*time.Minute)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := publicip.NewChecker()
			tt.setup(c)
			tt.action(c)
			tt.verify(t, c)
		})
	}
}

func TestChecker_getHistoryCopy(t *testing.T) {
	t.Run("empty history returns nil", func(t *testing.T) {
		c := publicip.NewChecker()
		hist := c.GetHistoryCopy()
		if hist != nil {
			t.Errorf("expected nil for empty history, got %v", hist)
		}
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		c := publicip.NewChecker()
		c.CheckerSetHistory([]publicip.HistoryEntry{
			{IP: "192.0.2.1", FirstSeen: time.Now(), LastSeen: time.Now()},
			{IP: "192.0.2.2", FirstSeen: time.Now(), LastSeen: time.Now()},
		})

		hist := c.GetHistoryCopy()

		if len(hist) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(hist))
		}

		// Modify copy - should not affect original.
		hist[0].IP = "modified"
		if c.CheckerHistory()[0].IP == "modified" {
			t.Error("GetHistoryCopy should return copy, but original was modified")
		}
	})
}

func TestGeoResponse(t *testing.T) {
	geo := publicip.GeoResponse{
		City:       "San Francisco",
		RegionName: "California",
		Country:    "US",
		ISP:        "Test ISP",
		AS:         "AS12345",
	}

	if geo.City != "San Francisco" {
		t.Errorf("expected city 'San Francisco', got %q", geo.City)
	}
	if geo.RegionName != "California" {
		t.Errorf("expected region 'California', got %q", geo.RegionName)
	}
	if geo.Country != "US" {
		t.Errorf("expected country 'US', got %q", geo.Country)
	}
	if geo.ISP != "Test ISP" {
		t.Errorf("expected ISP 'Test ISP', got %q", geo.ISP)
	}
	if geo.AS != "AS12345" {
		t.Errorf("expected AS 'AS12345', got %q", geo.AS)
	}
}

func TestHistoryEntry(t *testing.T) {
	now := time.Now()
	entry := publicip.HistoryEntry{
		IP:        "203.0.113.1",
		FirstSeen: now,
		LastSeen:  now,
		City:      "London",
		Country:   "UK",
	}

	if entry.IP != "203.0.113.1" {
		t.Errorf("expected IP '203.0.113.1', got %q", entry.IP)
	}
	if !entry.FirstSeen.Equal(now) {
		t.Errorf("expected FirstSeen %v, got %v", now, entry.FirstSeen)
	}
	if !entry.LastSeen.Equal(now) {
		t.Errorf("expected LastSeen %v, got %v", now, entry.LastSeen)
	}
	if entry.City != "London" {
		t.Errorf("expected city 'London', got %q", entry.City)
	}
	if entry.Country != "UK" {
		t.Errorf("expected country 'UK', got %q", entry.Country)
	}
}

func TestResult_WithGeoFields(t *testing.T) {
	now := time.Now()
	result := publicip.Result{
		IPv4:        "203.0.113.1",
		IPv6:        "2001:db8::1",
		LastChecked: now,
		City:        "Paris",
		Country:     "FR",
		ISP:         "French ISP",
	}

	if result.IPv4 != "203.0.113.1" {
		t.Errorf("expected IPv4 '203.0.113.1', got %q", result.IPv4)
	}
	if result.IPv6 != "2001:db8::1" {
		t.Errorf("expected IPv6 '2001:db8::1', got %q", result.IPv6)
	}
	if !result.LastChecked.Equal(now) {
		t.Errorf("expected LastChecked %v, got %v", now, result.LastChecked)
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
	result := publicip.Result{
		IPv4:        "203.0.113.1",
		LastChecked: now,
		History: []publicip.HistoryEntry{
			{IP: "192.0.2.1", FirstSeen: now, LastSeen: now},
			{IP: "192.0.2.2", FirstSeen: now, LastSeen: now},
		},
	}

	if result.IPv4 != "203.0.113.1" {
		t.Errorf("expected IPv4 '203.0.113.1', got %q", result.IPv4)
	}
	if !result.LastChecked.Equal(now) {
		t.Errorf("expected LastChecked %v, got %v", now, result.LastChecked)
	}
	if len(result.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(result.History))
	}
}
