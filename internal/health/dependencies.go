package health

import (
	"context"
	"slices"
	"sync"
	"time"
)

// Dependency relationship types.
const (
	// DependencyTypeDNS indicates DNS is required for the endpoint.
	DependencyTypeDNS = "dns"

	// DependencyTypeGateway indicates gateway/router access is required.
	DependencyTypeGateway = "gateway"

	// DependencyTypeAuth indicates authentication service is required.
	DependencyTypeAuth = "auth"

	// DependencyTypeDatabase indicates database connectivity is required.
	DependencyTypeDatabase = "database"

	// DependencyTypeCustom indicates a custom endpoint dependency.
	DependencyTypeCustom = "custom"
)

// Blocked status reasons.
const (
	BlockedByDNS      = "blocked_by_dns"
	BlockedByGateway  = "blocked_by_gateway"
	BlockedByAuth     = "blocked_by_auth"
	BlockedByDatabase = "blocked_by_database"
	BlockedByCustom   = "blocked_by_dependency"
)

// Dependency priority levels (lower = checked first).
const (
	// PriorityDNS is the priority for DNS dependencies (checked first).
	PriorityDNS = 1

	// PriorityGateway is the priority for gateway dependencies.
	PriorityGateway = 2

	// PriorityAuth is the priority for authentication dependencies.
	PriorityAuth = 3
)

// DependencyChain defines the dependencies for an endpoint.
type DependencyChain struct {
	// PrimaryEndpoint is the endpoint that has dependencies.
	PrimaryEndpoint string `json:"primaryEndpoint"`

	// Dependencies is an ordered list of dependent endpoints.
	// If any dependency fails, the primary endpoint is marked as blocked.
	Dependencies []Dependency `json:"dependencies"`
}

// Dependency represents a single dependency relationship.
type Dependency struct {
	// EndpointName is the name of the dependent endpoint.
	EndpointName string `json:"endpointName"`

	// Type categorizes the dependency (dns, gateway, auth, database, custom).
	Type string `json:"type"`

	// Required indicates if this dependency must be healthy for the primary to be considered testable.
	Required bool `json:"required"`

	// Priority determines the order of checking (lower = checked first).
	Priority int `json:"priority"`
}

// DependencyStatus represents the current status of a dependency check.
type DependencyStatus struct {
	EndpointName  string    `json:"endpointName"`
	IsHealthy     bool      `json:"isHealthy"`
	LastCheck     time.Time `json:"lastCheck"`
	BlockedReason string    `json:"blockedReason,omitempty"`
}

// EndpointDependencyStatus combines endpoint status with dependency information.
type EndpointDependencyStatus struct {
	EndpointName     string             `json:"endpointName"`
	IsBlocked        bool               `json:"isBlocked"`
	BlockedBy        []string           `json:"blockedBy,omitempty"`
	BlockedReason    string             `json:"blockedReason,omitempty"`
	DependencyStatus []DependencyStatus `json:"dependencyStatus,omitempty"`
	LastEvaluated    time.Time          `json:"lastEvaluated"`
}

// DependencyManager tracks and evaluates endpoint dependencies.
type DependencyManager struct {
	mu           sync.RWMutex
	chains       map[string]*DependencyChain // endpoint name -> chain
	statusCache  map[string]*DependencyStatus
	cacheTimeout time.Duration
}

// DependencyManagerConfig configures the dependency manager.
type DependencyManagerConfig struct {
	// CacheTimeout is how long to cache dependency status checks.
	CacheTimeout time.Duration
}

// DefaultDependencyCacheTimeout is the default cache timeout for dependency status.
const DefaultDependencyCacheTimeout = 30 * time.Second

// NewDependencyManager creates a new dependency manager.
func NewDependencyManager(cfg DependencyManagerConfig) *DependencyManager {
	timeout := cfg.CacheTimeout
	if timeout == 0 {
		timeout = DefaultDependencyCacheTimeout
	}

	return &DependencyManager{
		chains:       make(map[string]*DependencyChain),
		statusCache:  make(map[string]*DependencyStatus),
		cacheTimeout: timeout,
	}
}

// RegisterChain registers a dependency chain for an endpoint.
func (dm *DependencyManager) RegisterChain(chain DependencyChain) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.chains[chain.PrimaryEndpoint] = &chain
}

// RegisterChains registers multiple dependency chains.
func (dm *DependencyManager) RegisterChains(chains []DependencyChain) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for i := range chains {
		dm.chains[chains[i].PrimaryEndpoint] = &chains[i]
	}
}

// UpdateDependencyStatus updates the health status of a dependency endpoint.
func (dm *DependencyManager) UpdateDependencyStatus(endpointName string, isHealthy bool) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.statusCache[endpointName] = &DependencyStatus{
		EndpointName: endpointName,
		IsHealthy:    isHealthy,
		LastCheck:    time.Now(),
	}
}

// GetDependencyStatus returns the cached status of a dependency.
func (dm *DependencyManager) GetDependencyStatus(endpointName string) (*DependencyStatus, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	status, exists := dm.statusCache[endpointName]
	if !exists {
		return nil, false
	}

	// Check if cache is still valid
	if time.Since(status.LastCheck) > dm.cacheTimeout {
		return nil, false
	}

	return status, true
}

// EvaluateEndpoint checks if an endpoint is blocked by any failed dependencies.
func (dm *DependencyManager) EvaluateEndpoint(_ context.Context, endpointName string) *EndpointDependencyStatus {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	result := &EndpointDependencyStatus{
		EndpointName:  endpointName,
		LastEvaluated: time.Now(),
	}

	chain, exists := dm.chains[endpointName]
	if !exists {
		// No dependencies registered, endpoint is not blocked
		return result
	}

	var blockedBy []string
	var depStatuses []DependencyStatus

	for _, dep := range chain.Dependencies {
		status, cached := dm.statusCache[dep.EndpointName]

		depStatus := DependencyStatus{
			EndpointName: dep.EndpointName,
			IsHealthy:    true, // Assume healthy if unknown
		}

		if cached {
			depStatus.IsHealthy = status.IsHealthy
			depStatus.LastCheck = status.LastCheck
		}

		// Check if this dependency blocks the endpoint
		if dep.Required && cached && !status.IsHealthy {
			blockedBy = append(blockedBy, dep.EndpointName)
			depStatus.BlockedReason = getBlockedReason(dep.Type)
		}

		depStatuses = append(depStatuses, depStatus)
	}

	result.DependencyStatus = depStatuses

	if len(blockedBy) > 0 {
		result.IsBlocked = true
		result.BlockedBy = blockedBy
		result.BlockedReason = getBlockedReason(chain.Dependencies[0].Type)

		// Find the most specific blocked reason from the first blocker
		for _, dep := range chain.Dependencies {
			if slices.Contains(blockedBy, dep.EndpointName) {
				result.BlockedReason = getBlockedReason(dep.Type)
				return result
			}
		}
	}

	return result
}

// EvaluateAll evaluates all registered endpoints and returns their dependency status.
func (dm *DependencyManager) EvaluateAll(ctx context.Context) map[string]*EndpointDependencyStatus {
	dm.mu.RLock()
	endpoints := make([]string, 0, len(dm.chains))
	for name := range dm.chains {
		endpoints = append(endpoints, name)
	}
	dm.mu.RUnlock()

	results := make(map[string]*EndpointDependencyStatus, len(endpoints))
	for _, name := range endpoints {
		results[name] = dm.EvaluateEndpoint(ctx, name)
	}

	return results
}

// GetBlockedEndpoints returns all endpoints that are currently blocked by dependencies.
func (dm *DependencyManager) GetBlockedEndpoints(ctx context.Context) []*EndpointDependencyStatus {
	all := dm.EvaluateAll(ctx)

	var blocked []*EndpointDependencyStatus
	for _, status := range all {
		if status.IsBlocked {
			blocked = append(blocked, status)
		}
	}

	return blocked
}

// ClearCache clears the dependency status cache.
func (dm *DependencyManager) ClearCache() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.statusCache = make(map[string]*DependencyStatus)
}

// getBlockedReason returns the blocked reason string for a dependency type.
func getBlockedReason(depType string) string {
	switch depType {
	case DependencyTypeDNS:
		return BlockedByDNS
	case DependencyTypeGateway:
		return BlockedByGateway
	case DependencyTypeAuth:
		return BlockedByAuth
	case DependencyTypeDatabase:
		return BlockedByDatabase
	default:
		return BlockedByCustom
	}
}

// DefaultDependencyChains returns common default dependency chains.
// These can be used as a starting point for configuration.
func DefaultDependencyChains() []DependencyChain {
	return []DependencyChain{
		{
			PrimaryEndpoint: "http_external",
			Dependencies: []Dependency{
				{EndpointName: "dns", Type: DependencyTypeDNS, Required: true, Priority: PriorityDNS},
				{EndpointName: "gateway", Type: DependencyTypeGateway, Required: true, Priority: PriorityGateway},
			},
		},
		{
			PrimaryEndpoint: "https_external",
			Dependencies: []Dependency{
				{EndpointName: "dns", Type: DependencyTypeDNS, Required: true, Priority: PriorityDNS},
				{EndpointName: "gateway", Type: DependencyTypeGateway, Required: true, Priority: PriorityGateway},
			},
		},
		{
			PrimaryEndpoint: "api_service",
			Dependencies: []Dependency{
				{EndpointName: "dns", Type: DependencyTypeDNS, Required: true, Priority: PriorityDNS},
				{EndpointName: "auth_service", Type: DependencyTypeAuth, Required: false, Priority: PriorityAuth},
			},
		},
	}
}
