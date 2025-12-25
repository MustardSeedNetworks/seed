/**
 * Hooks Index
 *
 * Central export point for all custom React hooks.
 * Provides convenient imports for hook consumers.
 */

// Authentication and session management
export { useAuth } from "./useAuth";

// Theme management
export { useTheme } from "./useTheme";

// WebSocket connection
export { useWebSocket } from "./useWebSocket";
export type { Message, CardUpdate } from "./useWebSocket";

// Network data fetching
export { useNetworkData } from "./useNetworkData";

// Performance testing
export { usePerformanceTest } from "./usePerformanceTest";

// Health checks
export { useHealthChecks } from "./useHealthChecks";

// Device discovery
export { useDevices } from "./useDevices";
export { useDiscoveredDevices } from "./useDiscoveredDevices";

// WiFi surveys
export { useSurvey } from "./useSurvey";

// Profiles
export { useProfiles } from "./useProfiles";

// Logs
export { useLogs } from "./useLogs";

// Vulnerabilities
export { useVulnerabilities } from "./useVulnerabilities";

// Card state management (Plan C)
export { useCardState } from "./useCardState";
export type { CardState } from "./useCardState";

// Network fetchers (Plan C)
export { useNetworkFetchers } from "./useNetworkFetchers";

// Interface state management (Plan C)
export { useInterfaceState } from "./useInterfaceState";

// Default settings (Plan E - single source of truth)
export { useDefaults, getDefaultsSync, clearDefaultsCache } from "./useDefaults";
