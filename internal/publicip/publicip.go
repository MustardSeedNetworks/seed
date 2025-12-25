// Package publicip provides public IP address detection with caching.
package publicip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// cacheDuration is how long to cache public IP results.
	cacheDuration = 5 * time.Minute
	// requestTimeout is the timeout for external API requests.
	requestTimeout = 10 * time.Second
)

// HistoryEntry represents a previous IP address with geo info.
type HistoryEntry struct {
	IP        string    `json:"ip"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	City      string    `json:"city,omitempty"`
	Country   string    `json:"country,omitempty"`
}

// Result contains the public IP information with optional geolocation.
type Result struct {
	IPv4        string    `json:"ipv4,omitempty"`
	IPv6        string    `json:"ipv6,omitempty"`
	LastChecked time.Time `json:"lastChecked"`
	Error       string    `json:"error,omitempty"`
	// Geo fields from ip-api.com
	ISP         string  `json:"isp,omitempty"`
	ASN         string  `json:"asn,omitempty"`
	Org         string  `json:"org,omitempty"`
	City        string  `json:"city,omitempty"`
	Region      string  `json:"region,omitempty"`
	Country     string  `json:"country,omitempty"`
	CountryCode string  `json:"countryCode,omitempty"`
	Lat         float64 `json:"lat,omitempty"`
	Lon         float64 `json:"lon,omitempty"`
	// History of previous IP addresses
	History []HistoryEntry `json:"history,omitempty"`
}

// geoResponse represents the ip-api.com response format.
type geoResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message,omitempty"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
}

// Checker fetches and caches public IP addresses with geolocation.
type Checker struct {
	mu         sync.RWMutex
	cache      *Result
	cacheTime  time.Time
	httpClient *http.Client
	// geoCache stores geo data by IP to avoid redundant lookups
	geoCache     map[string]*geoResponse
	geoCacheTime map[string]time.Time
	// history stores previous IP addresses (max 10 entries)
	history  []HistoryEntry
	lastIPv4 string
}

// NewChecker creates a new public IP checker with geolocation support.
func NewChecker() *Checker {
	return &Checker{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		geoCache:     make(map[string]*geoResponse),
		geoCacheTime: make(map[string]time.Time),
		history:      make([]HistoryEntry, 0, 10),
	}
}

// GetPublicIP returns the cached public IP or fetches a new one if cache expired.
func (c *Checker) GetPublicIP(ctx context.Context) *Result {
	c.mu.RLock()
	if c.cache != nil && time.Since(c.cacheTime) < cacheDuration {
		result := *c.cache
		c.mu.RUnlock()
		return &result
	}
	c.mu.RUnlock()

	return c.refresh(ctx)
}

// Refresh forces a refresh of the public IP cache.
func (c *Checker) Refresh(ctx context.Context) *Result {
	return c.refresh(ctx)
}

func (c *Checker) refresh(ctx context.Context) *Result {
	result := &Result{
		LastChecked: time.Now(),
	}

	// Fetch IPv4 and IPv6 in parallel
	var wg sync.WaitGroup
	var ipv4, ipv6 string
	var ipv4Err, ipv6Err error

	wg.Add(2)

	go func() {
		defer wg.Done()
		ipv4, ipv4Err = c.fetchIPv4(ctx)
	}()

	go func() {
		defer wg.Done()
		ipv6, ipv6Err = c.fetchIPv6(ctx)
	}()

	wg.Wait()

	result.IPv4 = ipv4
	result.IPv6 = ipv6

	// Only set error if both failed
	if ipv4Err != nil && ipv6Err != nil {
		result.Error = fmt.Sprintf("IPv4: %v; IPv6: %v", ipv4Err, ipv6Err)
	}

	// Fetch geolocation data for IPv4 (primary IP for geo)
	if ipv4 != "" {
		geo := c.fetchGeoData(ctx, ipv4)
		if geo != nil {
			result.ISP = geo.ISP
			result.Org = geo.Org
			result.City = geo.City
			result.Region = geo.RegionName
			result.Country = geo.Country
			result.CountryCode = geo.CountryCode
			result.Lat = geo.Lat
			result.Lon = geo.Lon
			// Parse ASN from the "AS" field (format: "AS15169 Google LLC")
			if geo.AS != "" {
				parts := strings.SplitN(geo.AS, " ", 2)
				if len(parts) > 0 {
					result.ASN = strings.TrimPrefix(parts[0], "AS")
				}
			}
		}
	}

	// Update history if IP changed
	c.mu.Lock()
	c.updateHistory(ipv4)
	result.History = c.getHistoryCopy()
	c.cache = result
	c.cacheTime = time.Now()
	c.mu.Unlock()

	return result
}

// fetchGeoData fetches geolocation data from ip-api.com with caching.
// Uses 1-hour cache per IP to respect rate limits (free tier: 45 req/min).
func (c *Checker) fetchGeoData(ctx context.Context, ip string) *geoResponse {
	const geoCacheDuration = 1 * time.Hour

	c.mu.RLock()
	if cached, ok := c.geoCache[ip]; ok {
		if time.Since(c.geoCacheTime[ip]) < geoCacheDuration {
			c.mu.RUnlock()
			return cached
		}
	}
	c.mu.RUnlock()

	// Fetch from ip-api.com (free, no API key required)
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,lat,lon,isp,org,as", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "The Seed/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil
	}

	var geo geoResponse
	if err := json.Unmarshal(body, &geo); err != nil {
		return nil
	}

	if geo.Status != "success" {
		return nil
	}

	// Cache the result
	c.mu.Lock()
	c.geoCache[ip] = &geo
	c.geoCacheTime[ip] = time.Now()
	c.mu.Unlock()

	return &geo
}

// updateHistory updates the IP history when IP changes.
// Must be called with c.mu held.
func (c *Checker) updateHistory(ipv4 string) {
	if ipv4 == "" {
		return
	}

	now := time.Now()

	// Check if IP changed
	if c.lastIPv4 != "" && c.lastIPv4 != ipv4 {
		// IP changed - add old IP to history if not already there
		found := false
		for i := range c.history {
			if c.history[i].IP == c.lastIPv4 {
				c.history[i].LastSeen = now
				found = true
				break
			}
		}
		if !found && c.lastIPv4 != "" {
			// Prepend to history (most recent first)
			entry := HistoryEntry{
				IP:        c.lastIPv4,
				FirstSeen: now,
				LastSeen:  now,
			}
			// Get geo for old IP if we have it cached
			if geo, ok := c.geoCache[c.lastIPv4]; ok {
				entry.City = geo.City
				entry.Country = geo.Country
			}
			c.history = append([]HistoryEntry{entry}, c.history...)
			// Keep max 10 entries
			if len(c.history) > 10 {
				c.history = c.history[:10]
			}
		}
	}

	c.lastIPv4 = ipv4
}

// getHistoryCopy returns a copy of the history slice.
// Must be called with c.mu held.
func (c *Checker) getHistoryCopy() []HistoryEntry {
	if len(c.history) == 0 {
		return nil
	}
	hist := make([]HistoryEntry, len(c.history))
	copy(hist, c.history)
	return hist
}

func (c *Checker) fetchIPv4(ctx context.Context) (string, error) {
	// Try multiple services for redundancy
	services := []struct {
		url    string
		parser func([]byte) (string, error)
	}{
		{"https://api.ipify.org?format=json", parseIpifyJSON},
		{"https://api4.my-ip.io/ip.json", parseMyIPJSON},
		{"https://ipv4.icanhazip.com", parseTextIP},
	}

	var lastErr error
	for _, svc := range services {
		ip, err := c.fetchFromService(ctx, svc.url, svc.parser)
		if err == nil && ip != "" {
			return ip, nil
		}
		lastErr = err
	}

	return "", lastErr
}

func (c *Checker) fetchIPv6(ctx context.Context) (string, error) {
	// Try multiple services for redundancy
	services := []struct {
		url    string
		parser func([]byte) (string, error)
	}{
		{"https://api64.ipify.org?format=json", parseIpifyJSON},
		{"https://api6.my-ip.io/ip.json", parseMyIPJSON},
		{"https://ipv6.icanhazip.com", parseTextIP},
	}

	var lastErr error
	for _, svc := range services {
		ip, err := c.fetchFromService(ctx, svc.url, svc.parser)
		if err == nil && ip != "" {
			// Validate it's actually IPv6 (contains colons)
			if strings.Contains(ip, ":") {
				return ip, nil
			}
		}
		lastErr = err
	}

	return "", lastErr
}

func (c *Checker) fetchFromService(ctx context.Context, url string, parser func([]byte) (string, error)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "The Seed/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}

	return parser(body)
}

func parseIpifyJSON(body []byte) (string, error) {
	var result struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.IP), nil
}

func parseMyIPJSON(body []byte) (string, error) {
	var result struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.IP), nil
}

func parseTextIP(body []byte) (string, error) {
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty response")
	}
	return ip, nil
}
