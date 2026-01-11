// Package alerts provides alerting functionality for health checks and system events.
package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Alert severity levels.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Alert types for health checks.
const (
	AlertTypeHealthCheckFailed    = "health_check_failed"
	AlertTypeHealthCheckRecovered = "health_check_recovered"
	AlertTypeSLAViolation         = "sla_violation"
	AlertTypeAnomalyDetected      = "anomaly_detected"
	AlertTypeDependencyBlocked    = "dependency_blocked"
)

// Alert states.
const (
	AlertStateActive       = "active"
	AlertStateAcknowledged = "acknowledged"
	AlertStateResolved     = "resolved"
)

// Default configuration values.
const (
	// DefaultConsecutiveFailures is the number of failures before alerting.
	DefaultConsecutiveFailures = 3

	// DefaultCooldownPeriod prevents alert spam.
	DefaultCooldownPeriod = 5 * time.Minute

	// DefaultDigestInterval is the interval for batched alert digests.
	DefaultDigestInterval = 15 * time.Minute

	// DefaultAlertRetention is how long to keep resolved alerts.
	DefaultAlertRetention = 24 * time.Hour

	// CriticalityCriticalThreshold is the criticality level for critical severity.
	CriticalityCriticalThreshold = 8

	// CriticalityInfoThreshold is the criticality level for info severity.
	CriticalityInfoThreshold = 3

	// DeviationWarningThreshold is the stddev threshold for warning severity.
	DeviationWarningThreshold = 3.0

	// DeviationCriticalThreshold is the stddev threshold for critical severity.
	DeviationCriticalThreshold = 4.0
)

// HealthAlert represents a health check alert.
type HealthAlert struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	Severity       string         `json:"severity"`
	EndpointName   string         `json:"endpointName"`
	CheckType      string         `json:"checkType,omitempty"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	State          string         `json:"state"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	AcknowledgedAt time.Time      `json:"acknowledgedAt,omitzero"`
	AcknowledgedBy string         `json:"acknowledgedBy,omitempty"`
	ResolvedAt     time.Time      `json:"resolvedAt,omitzero"`
	FailureCount   int            `json:"failureCount"`
	Criticality    int            `json:"criticality,omitempty"` // From endpoint config (1-10)
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// AlertHandler is a function that handles alert events.
type AlertHandler func(*HealthAlert)

// EndpointState tracks the failure state of an endpoint.
type EndpointState struct {
	EndpointName        string
	ConsecutiveFailures int
	LastFailureTime     time.Time
	LastAlertTime       time.Time
	IsInFailedState     bool
}

// AlertManager manages health check alerts.
type AlertManager struct {
	mu                   sync.RWMutex
	alerts               map[string]*HealthAlert // alert ID -> alert
	endpointStates       map[string]*EndpointState
	consecutiveThreshold int
	cooldownPeriod       time.Duration
	digestMode           bool
	digestInterval       time.Duration
	digestBuffer         []*HealthAlert
	handlers             []AlertHandler
	logger               *slog.Logger
}

// AlertManagerConfig configures the alert manager.
type AlertManagerConfig struct {
	// ConsecutiveFailures is the number of consecutive failures before alerting.
	ConsecutiveFailures int

	// CooldownPeriod prevents alert spam for the same endpoint.
	CooldownPeriod time.Duration

	// DigestMode batches alerts instead of sending immediately.
	DigestMode bool

	// DigestInterval is the interval for batched alert digests.
	DigestInterval time.Duration

	// Handlers are called when alerts are triggered.
	Handlers []AlertHandler

	// Logger for alert events.
	Logger *slog.Logger
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(cfg AlertManagerConfig) *AlertManager {
	consecutive := cfg.ConsecutiveFailures
	if consecutive == 0 {
		consecutive = DefaultConsecutiveFailures
	}

	cooldown := cfg.CooldownPeriod
	if cooldown == 0 {
		cooldown = DefaultCooldownPeriod
	}

	digestInterval := cfg.DigestInterval
	if digestInterval == 0 {
		digestInterval = DefaultDigestInterval
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &AlertManager{
		alerts:               make(map[string]*HealthAlert),
		endpointStates:       make(map[string]*EndpointState),
		consecutiveThreshold: consecutive,
		cooldownPeriod:       cooldown,
		digestMode:           cfg.DigestMode,
		digestInterval:       digestInterval,
		handlers:             cfg.Handlers,
		logger:               logger,
	}
}

// RegisterHandler adds an alert handler.
func (am *AlertManager) RegisterHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.handlers = append(am.handlers, handler)
}

// RecordFailure records a health check failure and potentially triggers an alert.
func (am *AlertManager) RecordFailure(ctx context.Context, endpointName, checkType string, errorMsg string, criticality int) *HealthAlert {
	am.mu.Lock()
	defer am.mu.Unlock()

	state := am.getOrCreateState(endpointName)
	state.ConsecutiveFailures++
	state.LastFailureTime = time.Now()

	// Check if we should trigger an alert
	if state.ConsecutiveFailures >= am.consecutiveThreshold && !state.IsInFailedState {
		// Check cooldown
		if time.Since(state.LastAlertTime) < am.cooldownPeriod {
			return nil
		}

		state.IsInFailedState = true
		state.LastAlertTime = time.Now()

		alert := am.createAlert(ctx, AlertTypeHealthCheckFailed, endpointName, checkType, criticality, errorMsg)
		alert.FailureCount = state.ConsecutiveFailures

		am.processAlert(alert)
		return alert
	}

	// Update existing alert failure count if in failed state
	if state.IsInFailedState {
		for _, alert := range am.alerts {
			if alert.EndpointName == endpointName && alert.State == AlertStateActive {
				alert.FailureCount = state.ConsecutiveFailures
				alert.UpdatedAt = time.Now()
			}
		}
	}

	return nil
}

// RecordSuccess records a health check success and potentially triggers recovery.
func (am *AlertManager) RecordSuccess(_ context.Context, endpointName, checkType string) *HealthAlert {
	am.mu.Lock()
	defer am.mu.Unlock()

	state, exists := am.endpointStates[endpointName]
	if !exists || !state.IsInFailedState {
		// Reset consecutive failures
		if exists {
			state.ConsecutiveFailures = 0
		}
		return nil
	}

	// Endpoint recovered
	state.ConsecutiveFailures = 0
	state.IsInFailedState = false

	// Create recovery alert
	alert := &HealthAlert{
		ID:           fmt.Sprintf("%s-%d", endpointName, time.Now().UnixNano()),
		Type:         AlertTypeHealthCheckRecovered,
		Severity:     SeverityInfo,
		EndpointName: endpointName,
		CheckType:    checkType,
		Title:        fmt.Sprintf("Endpoint Recovered: %s", endpointName),
		Message:      fmt.Sprintf("Health check for %s has recovered", endpointName),
		State:        AlertStateResolved,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ResolvedAt:   time.Now(),
	}

	// Resolve active alerts for this endpoint
	for _, existing := range am.alerts {
		if existing.EndpointName == endpointName && existing.State == AlertStateActive {
			existing.State = AlertStateResolved
			existing.ResolvedAt = time.Now()
			existing.UpdatedAt = time.Now()
		}
	}

	am.processAlert(alert)
	return alert
}

// CreateSLAAlert creates an alert for an SLA violation.
func (am *AlertManager) CreateSLAAlert(_ context.Context, endpointName, violationType string, target, actual float64, criticality int) *HealthAlert {
	am.mu.Lock()
	defer am.mu.Unlock()

	severity := SeverityWarning
	if criticality >= CriticalityCriticalThreshold {
		severity = SeverityCritical
	}

	alert := &HealthAlert{
		ID:           fmt.Sprintf("sla-%s-%d", endpointName, time.Now().UnixNano()),
		Type:         AlertTypeSLAViolation,
		Severity:     severity,
		EndpointName: endpointName,
		Title:        fmt.Sprintf("SLA Violation: %s", endpointName),
		Message:      fmt.Sprintf("%s SLA violated: target %.2f%%, actual %.2f%%", violationType, target, actual),
		State:        AlertStateActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Criticality:  criticality,
		Metadata: map[string]any{
			"violationType": violationType,
			"target":        target,
			"actual":        actual,
		},
	}

	am.processAlert(alert)
	return alert
}

// CreateAnomalyAlert creates an alert for a detected anomaly.
func (am *AlertManager) CreateAnomalyAlert(_ context.Context, endpointName, anomalyType string, value, expected, deviation float64) *HealthAlert {
	am.mu.Lock()
	defer am.mu.Unlock()

	severity := SeverityInfo
	if deviation > DeviationWarningThreshold {
		severity = SeverityWarning
	}
	if deviation > DeviationCriticalThreshold {
		severity = SeverityCritical
	}

	alert := &HealthAlert{
		ID:           fmt.Sprintf("anomaly-%s-%d", endpointName, time.Now().UnixNano()),
		Type:         AlertTypeAnomalyDetected,
		Severity:     severity,
		EndpointName: endpointName,
		Title:        fmt.Sprintf("Anomaly Detected: %s", endpointName),
		Message:      fmt.Sprintf("%s: value %.2f (expected %.2f, %.1f stddev)", anomalyType, value, expected, deviation),
		State:        AlertStateActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata: map[string]any{
			"anomalyType": anomalyType,
			"value":       value,
			"expected":    expected,
			"deviation":   deviation,
		},
	}

	am.processAlert(alert)
	return alert
}

// AcknowledgeAlert marks an alert as acknowledged.
func (am *AlertManager) AcknowledgeAlert(alertID, acknowledgedBy string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists || alert.State != AlertStateActive {
		return false
	}

	alert.State = AlertStateAcknowledged
	alert.AcknowledgedAt = time.Now()
	alert.AcknowledgedBy = acknowledgedBy
	alert.UpdatedAt = time.Now()

	return true
}

// ResolveAlert marks an alert as resolved.
func (am *AlertManager) ResolveAlert(alertID string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists || alert.State == AlertStateResolved {
		return false
	}

	alert.State = AlertStateResolved
	alert.ResolvedAt = time.Now()
	alert.UpdatedAt = time.Now()

	return true
}

// GetActiveAlerts returns all active alerts.
func (am *AlertManager) GetActiveAlerts() []*HealthAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var active []*HealthAlert
	for _, alert := range am.alerts {
		if alert.State == AlertStateActive {
			active = append(active, alert)
		}
	}

	return active
}

// GetAlertsByEndpoint returns all alerts for an endpoint.
func (am *AlertManager) GetAlertsByEndpoint(endpointName string) []*HealthAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []*HealthAlert
	for _, alert := range am.alerts {
		if alert.EndpointName == endpointName {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// GetAlertsBySeverity returns all alerts with the given severity.
func (am *AlertManager) GetAlertsBySeverity(severity string) []*HealthAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []*HealthAlert
	for _, alert := range am.alerts {
		if alert.Severity == severity && alert.State == AlertStateActive {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// GetAlert returns a specific alert by ID.
func (am *AlertManager) GetAlert(alertID string) (*HealthAlert, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alert, exists := am.alerts[alertID]
	return alert, exists
}

// CleanupResolvedAlerts removes resolved alerts older than the retention period.
func (am *AlertManager) CleanupResolvedAlerts(retention time.Duration) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-retention)
	cleaned := 0

	for id, alert := range am.alerts {
		if alert.State == AlertStateResolved && alert.ResolvedAt.Before(cutoff) {
			delete(am.alerts, id)
			cleaned++
		}
	}

	return cleaned
}

// FlushDigest sends any buffered alerts immediately.
func (am *AlertManager) FlushDigest() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if len(am.digestBuffer) == 0 {
		return
	}

	// Send all buffered alerts to handlers
	for _, handler := range am.handlers {
		for _, alert := range am.digestBuffer {
			handler(alert)
		}
	}

	am.digestBuffer = nil
}

// getOrCreateState gets or creates endpoint state.
func (am *AlertManager) getOrCreateState(endpointName string) *EndpointState {
	state, exists := am.endpointStates[endpointName]
	if !exists {
		state = &EndpointState{
			EndpointName: endpointName,
		}
		am.endpointStates[endpointName] = state
	}
	return state
}

// createAlert creates a new alert.
func (am *AlertManager) createAlert(_ context.Context, alertType, endpointName, checkType string, criticality int, message string) *HealthAlert {
	severity := SeverityWarning
	if criticality >= CriticalityCriticalThreshold {
		severity = SeverityCritical
	} else if criticality <= CriticalityInfoThreshold {
		severity = SeverityInfo
	}

	alert := &HealthAlert{
		ID:           fmt.Sprintf("%s-%d", endpointName, time.Now().UnixNano()),
		Type:         alertType,
		Severity:     severity,
		EndpointName: endpointName,
		CheckType:    checkType,
		Title:        fmt.Sprintf("Health Check Failed: %s", endpointName),
		Message:      message,
		State:        AlertStateActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Criticality:  criticality,
	}

	return alert
}

// processAlert stores and dispatches an alert.
func (am *AlertManager) processAlert(alert *HealthAlert) {
	am.alerts[alert.ID] = alert

	am.logger.Info("alert triggered",
		slog.String("id", alert.ID),
		slog.String("type", alert.Type),
		slog.String("severity", alert.Severity),
		slog.String("endpoint", alert.EndpointName),
	)

	if am.digestMode {
		am.digestBuffer = append(am.digestBuffer, alert)
		return
	}

	// Dispatch immediately
	for _, handler := range am.handlers {
		handler(alert)
	}
}

// GetCriticalAlertCount returns the count of critical active alerts.
func (am *AlertManager) GetCriticalAlertCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	count := 0
	for _, alert := range am.alerts {
		if alert.Severity == SeverityCritical && alert.State == AlertStateActive {
			count++
		}
	}

	return count
}

// GetAlertStats returns statistics about alerts.
func (am *AlertManager) GetAlertStats() AlertStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := AlertStats{}

	for _, alert := range am.alerts {
		switch alert.State {
		case AlertStateActive:
			stats.Active++
			switch alert.Severity {
			case SeverityCritical:
				stats.Critical++
			case SeverityWarning:
				stats.Warning++
			case SeverityInfo:
				stats.Info++
			}
		case AlertStateAcknowledged:
			stats.Acknowledged++
		case AlertStateResolved:
			stats.Resolved++
		}
		stats.Total++
	}

	return stats
}

// AlertStats contains alert statistics.
type AlertStats struct {
	Total        int `json:"total"`
	Active       int `json:"active"`
	Acknowledged int `json:"acknowledged"`
	Resolved     int `json:"resolved"`
	Critical     int `json:"critical"`
	Warning      int `json:"warning"`
	Info         int `json:"info"`
}
