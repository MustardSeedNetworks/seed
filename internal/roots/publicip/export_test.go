// Package publicip exports internal functions for testing.
package publicip

import (
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

// FetchFromService is exported for testing.
var FetchFromService = (*Checker).fetchFromService

// ParseIpifyJSON is exported for testing.
var ParseIpifyJSON = parseIpifyJSON

// ParseMyIPJSON is exported for testing.
var ParseMyIPJSON = parseMyIPJSON

// ParseTextIP is exported for testing.
var ParseTextIP = parseTextIP

// RequestTimeout is exported for testing.
const RequestTimeout = requestTimeout

// GeoResponse is exported for testing.
type GeoResponse = geoResponse
