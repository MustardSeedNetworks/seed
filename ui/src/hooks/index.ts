/**
 * Hooks Index
 *
 * Central export point for all custom React hooks.
 * Provides convenient imports for hook consumers.
 */

// Authentication and session management
export { useAuth } from "./useAuth";
export type { CardState } from "./useCardState";
// Card state management (Plan C)
export { useCardState } from "./useCardState";
// Default settings (Plan E - single source of truth)
export { clearDefaultsCache, getDefaultsSync, useDefaults } from "./useDefaults";
// Device discovery
export { useDevices } from "./useDevices";
export { useDiscoveredDevices } from "./useDiscoveredDevices";

// Health checks
export { useHealthChecks } from "./useHealthChecks";
// Interface state management (Plan C)
export { useInterfaceState } from "./useInterfaceState";
// Logs
export { useLogs } from "./useLogs";
// Network data fetching
export { useNetworkData } from "./useNetworkData";
// Network fetchers (Plan C)
export { useNetworkFetchers } from "./useNetworkFetchers";
// Performance testing
export { usePerformanceTest } from "./usePerformanceTest";
// Profiles
export { useProfiles } from "./useProfiles";
// WiFi surveys
export { useSurvey } from "./useSurvey";
// Theme management
export { useTheme } from "./useTheme";
// Vulnerabilities
export { useVulnerabilities } from "./useVulnerabilities";
export type { CardUpdate, Message } from "./useWebSocket";
// WebSocket connection
export { useWebSocket } from "./useWebSocket";
