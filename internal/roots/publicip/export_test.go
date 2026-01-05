// Package publicip exports internal functions for testing.
package publicip

import (
	"context"
	"net/http"
	"time"
)

// CheckerHTTPClient returns the HTTP client for testing.
func (c *Checker) CheckerHTTPClient() *http.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.httpClient
}

// CheckerCache returns the cache for testing.
func (c *Checker) CheckerCache() *Result {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache
}

// CheckerSetCache sets the cache for testing.
func (c *Checker) CheckerSetCache(cache *Result, cacheTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = cache
	c.cacheTime = cacheTime
}

// CheckerHistory returns the history for testing.
func (c *Checker) CheckerHistory() []HistoryEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.history
}

// CheckerSetHistory sets the history for testing.
func (c *Checker) CheckerSetHistory(history []HistoryEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = history
}

// CheckerLastIPv4 returns the last IPv4 for testing.
func (c *Checker) CheckerLastIPv4() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastIPv4
}

// CheckerSetLastIPv4 sets the last IPv4 for testing.
func (c *Checker) CheckerSetLastIPv4(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastIPv4 = ip
}

// CheckerGeoCache returns the geo cache for testing.
func (c *Checker) CheckerGeoCache() map[string]*geoResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.geoCache
}

// CheckerSetGeoCache sets the geo cache for testing.
func (c *Checker) CheckerSetGeoCache(cache map[string]*geoResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.geoCache = cache
}

// UpdateHistory is exported for testing.
func (c *Checker) UpdateHistory(ipv4 string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updateHistory(ipv4)
}

// GetHistoryCopy is exported for testing.
func (c *Checker) GetHistoryCopy() []HistoryEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getHistoryCopy()
}

// ExportFetchFromService exposes fetchFromService for testing.
// It's a standalone function that matches the old var export signature.
func ExportFetchFromService(
	c *Checker,
	ctx context.Context,
	url string,
	parser func([]byte) (string, error),
) (string, error) {
	return c.fetchFromService(ctx, url, parser)
}

// ExportParseIpifyJSON exposes parseIpifyJSON for testing.
func ExportParseIpifyJSON(data []byte) (string, error) {
	return parseIpifyJSON(data)
}

// ExportParseMyIPJSON exposes parseMyIPJSON for testing.
func ExportParseMyIPJSON(data []byte) (string, error) {
	return parseMyIPJSON(data)
}

// ExportParseTextIP exposes parseTextIP for testing.
func ExportParseTextIP(data []byte) (string, error) {
	return parseTextIP(data)
}

// RequestTimeout is exported for testing.
const RequestTimeout = requestTimeout

// GeoResponse is exported for testing.
type GeoResponse = geoResponse
