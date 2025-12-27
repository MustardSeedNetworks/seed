/**
 * Pipeline Status Hook
 *
 * Provides real-time pipeline status via REST API and WebSocket events.
 * Used by NetworkDiscoveryCard to show multi-phase discovery progress.
 */
import { useState, useEffect, useCallback, useRef } from "react";

// ============================================================================
// Pipeline Types (matching backend internal/discovery/pipeline.go)
// ============================================================================

export type PipelineState =
  | "idle"
  | "enumerating"
  | "resolving"
  | "scanning"
  | "assessing"
  | "complete"
  | "failed"
  | "canceled";

export type PortScanIntensity =
  | "off"
  | "quick"
  | "standard"
  | "comprehensive"
  | "custom";

export type ScanTimingProfile = "polite" | "normal" | "aggressive";

// Phase configuration
export interface PipelinePhaseConfig {
  enumeration: boolean;
  nameResolution: boolean;
  serviceDiscovery: boolean;
  vulnAssessment: boolean;
}

// Timing configuration
export interface PipelineTiming {
  probeDelay: number; // milliseconds
  hostDelay: number;
  maxConcurrentHosts: number;
  phaseTimeout: number;
  profile: ScanTimingProfile;
}

// Port scan configuration
export interface PipelinePortScanConfig {
  intensity: PortScanIntensity;
  customPorts?: number[];
  bannerGrab: boolean;
  connectTimeout: number;
}

// SNMP MIB selection
export interface SNMPMIBSelection {
  system: boolean;
  interfaces: boolean;
  ipAddresses: boolean;
  routing: boolean;
  bridge: boolean;
  entity: boolean;
  lldp: boolean;
  vlan: boolean;
}

// SNMP collection configuration
export interface SNMPCollectionConfig {
  enabled: boolean;
  mibs: SNMPMIBSelection;
  walkTimeout: number;
  maxOidsPerRequest: number;
}

// Persistence configuration
export interface PipelinePersistenceConfig {
  storeHistory: boolean;
  stalenessThreshold: number;
  purgeAfter: number;
}

// Full pipeline configuration
export interface PipelineConfig {
  phases: PipelinePhaseConfig;
  timing: PipelineTiming;
  portScan: PipelinePortScanConfig;
  snmpCollection: SNMPCollectionConfig;
  persistence: PipelinePersistenceConfig;
}

// Pipeline run status
export interface PipelineRun {
  id: string;
  startedAt: string;
  completedAt?: string;
  status: PipelineState;
  trigger: string;
  config: PipelineConfig;
  currentPhase?: string;
  phaseDurations?: Record<string, number>;
  devicesFound: number;
  errors?: string[];
}

// Phase progress payload from WebSocket
export interface PhaseProgress {
  phase: string;
  phaseNumber: number;
  totalPhases: number;
  processedCount: number;
  totalCount: number;
  percentComplete: number;
  currentTarget?: string;
  elapsedMs: number;
  estimatedRemainMs?: number;
}

// Phase completed payload
export interface PhaseCompleted {
  phase: string;
  devicesFound?: number;
  namesResolved?: number;
  portsOpen?: number;
  vulnsFound?: number;
  duration: number;
  errors?: string[];
}

// Port intensity info from API
export interface PortIntensityInfo {
  level: PortScanIntensity;
  portCount: number;
  description: string;
  idsRisk: string;
  warning?: string;
}

// Timing profile info from API
export interface TimingProfileInfo {
  profile: ScanTimingProfile;
  probeDelayMs: number;
  hostDelayMs: number;
  maxConcurrentHosts: number;
  phaseTimeoutMins: number;
  description: string;
  useCase: string;
}

// ============================================================================
// WebSocket Event Types
// ============================================================================

export type PipelineEventType =
  | "pipeline_started"
  | "phase_started"
  | "phase_progress"
  | "phase_completed"
  | "phase_failed"
  | "device_discovered"
  | "device_updated"
  | "pipeline_completed"
  | "pipeline_failed"
  | "pipeline_canceled";

export interface PipelineEvent {
  type: PipelineEventType;
  timestamp: string;
  runId: string;
  payload: unknown;
}

// ============================================================================
// Hook State
// ============================================================================

export interface PipelineStatus {
  // Current run state
  state: PipelineState;
  runId: string;

  // Current phase info
  currentPhase: string;
  phaseNumber: number;
  totalPhases: number;
  enabledPhases: string[];

  // Progress within current phase
  processedCount: number;
  totalCount: number;
  percentComplete: number;
  currentTarget: string;

  // Timing
  elapsedMs: number;
  estimatedRemainMs: number;

  // Results
  devicesFound: number;
  phaseDurations: Record<string, number>;

  // Errors
  errors: string[];
}

export interface UsePipelineStatusReturn {
  status: PipelineStatus;
  config: PipelineConfig | null;
  portIntensityInfo: PortIntensityInfo[];
  timingProfiles: TimingProfileInfo[];
  isLoading: boolean;
  error: string | null;

  // Actions
  startPipeline: (configOverride?: Partial<PipelineConfig>) => Promise<void>;
  cancelPipeline: () => Promise<void>;
  updateConfig: (
    config: PipelineConfig,
    acknowledgeIdsRisk?: boolean
  ) => Promise<void>;
  refreshStatus: () => Promise<void>;
}

// ============================================================================
// Default Values
// ============================================================================

const defaultStatus: PipelineStatus = {
  state: "idle",
  runId: "",
  currentPhase: "",
  phaseNumber: 0,
  totalPhases: 0,
  enabledPhases: [],
  processedCount: 0,
  totalCount: 0,
  percentComplete: 0,
  currentTarget: "",
  elapsedMs: 0,
  estimatedRemainMs: 0,
  devicesFound: 0,
  phaseDurations: {},
  errors: [],
};

const defaultConfig: PipelineConfig = {
  phases: {
    enumeration: true,
    nameResolution: true,
    serviceDiscovery: false,
    vulnAssessment: false,
  },
  timing: {
    probeDelay: 50,
    hostDelay: 20,
    maxConcurrentHosts: 20,
    phaseTimeout: 600000, // 10 minutes in ms
    profile: "normal",
  },
  portScan: {
    intensity: "off",
    bannerGrab: true,
    connectTimeout: 2000,
  },
  snmpCollection: {
    enabled: true,
    mibs: {
      system: true,
      interfaces: true,
      ipAddresses: true,
      routing: false,
      bridge: false,
      entity: false,
      lldp: true,
      vlan: false,
    },
    walkTimeout: 30000,
    maxOidsPerRequest: 10,
  },
  persistence: {
    storeHistory: true,
    stalenessThreshold: 86400000, // 24 hours in ms
    purgeAfter: 2592000000, // 30 days in ms
  },
};

// ============================================================================
// Hook Implementation
// ============================================================================

/**
 *
 */
export function usePipelineStatus(
  onMessage?: (event: PipelineEvent) => void
): UsePipelineStatusReturn {
  const [status, setStatus] = useState<PipelineStatus>(defaultStatus);
  const [config, setConfig] = useState<PipelineConfig | null>(null);
  const [portIntensityInfo, setPortIntensityInfo] = useState<
    PortIntensityInfo[]
  >([]);
  const [timingProfiles, setTimingProfiles] = useState<TimingProfileInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const startTimeRef = useRef<number>(0);
  const elapsedIntervalRef = useRef<ReturnType<typeof setInterval> | null>(
    null
  );
  // Stable handler reference to avoid memory leaks from handler accumulation (fixes #849)
  const handlePipelineEventRef = useRef<((event: PipelineEvent) => void) | null>(
    null
  );

  // Fetch pipeline status from API
  const fetchStatus = useCallback(async () => {
    try {
      const response = await fetch("/api/pipeline/status", {
        credentials: "include", // Required for httpOnly cookie auth
      });
      if (!response.ok) {
        throw new Error(`Failed to fetch pipeline status: ${response.status}`);
      }
      const run: PipelineRun = await response.json();

      // Convert PipelineRun to PipelineStatus
      const enabledPhases = getEnabledPhases(run.config);
      const totalPhases = enabledPhases.length;
      const phaseNumber = run.currentPhase
        ? enabledPhases.indexOf(run.currentPhase) + 1
        : 0;

      setStatus((prev) => ({
        ...prev,
        state: run.status,
        runId: run.id || "",
        currentPhase: run.currentPhase || "",
        phaseNumber,
        totalPhases,
        enabledPhases,
        devicesFound: run.devicesFound || 0,
        phaseDurations: run.phaseDurations || {},
        errors: run.errors || [],
      }));

      // Track start time for elapsed calculation
      if (run.startedAt && isRunning(run.status)) {
        startTimeRef.current = new Date(run.startedAt).getTime();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch status");
    }
  }, []);

  // Fetch pipeline config from API
  const fetchConfig = useCallback(async () => {
    try {
      const response = await fetch("/api/pipeline/config", {
        credentials: "include", // Required for httpOnly cookie auth
      });
      if (!response.ok) {
        throw new Error(`Failed to fetch pipeline config: ${response.status}`);
      }
      const cfg: PipelineConfig = await response.json();
      setConfig(cfg);
    } catch (err) {
      console.error("Failed to fetch pipeline config:", err);
      setConfig(defaultConfig);
    }
  }, []);

  // Fetch port intensity info
  const fetchPortIntensityInfo = useCallback(async () => {
    try {
      const response = await fetch("/api/pipeline/port-intensity", {
        credentials: "include", // Required for httpOnly cookie auth
      });
      if (response.ok) {
        const info: PortIntensityInfo[] = await response.json();
        setPortIntensityInfo(info);
      }
    } catch (err) {
      console.error("Failed to fetch port intensity info:", err);
    }
  }, []);

  // Fetch timing profiles
  const fetchTimingProfiles = useCallback(async () => {
    try {
      const response = await fetch("/api/pipeline/timing-profiles", {
        credentials: "include", // Required for httpOnly cookie auth
      });
      if (response.ok) {
        const profiles: TimingProfileInfo[] = await response.json();
        setTimingProfiles(profiles);
      }
    } catch (err) {
      console.error("Failed to fetch timing profiles:", err);
    }
  }, []);

  // Start pipeline
  const startPipeline = useCallback(
    async (configOverride?: Partial<PipelineConfig>) => {
      try {
        setError(null);
        const body = configOverride ? { config: configOverride } : undefined;

        const response = await fetch("/api/pipeline/start", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include", // Required for httpOnly cookie auth
          body: body ? JSON.stringify(body) : undefined,
        });

        if (!response.ok) {
          const text = await response.text();
          throw new Error(
            text || `Failed to start pipeline: ${response.status}`
          );
        }

        const run: PipelineRun = await response.json();
        startTimeRef.current = Date.now();

        const enabledPhases = getEnabledPhases(run.config);
        setStatus({
          ...defaultStatus,
          state: run.status,
          runId: run.id,
          totalPhases: enabledPhases.length,
          enabledPhases,
        });
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to start pipeline"
        );
        throw err;
      }
    },
    []
  );

  // Cancel pipeline
  const cancelPipeline = useCallback(async () => {
    try {
      setError(null);
      const response = await fetch("/api/pipeline/cancel", {
        method: "POST",
        credentials: "include", // Required for httpOnly cookie auth
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(
          text || `Failed to cancel pipeline: ${response.status}`
        );
      }

      setStatus((prev) => ({
        ...prev,
        state: "canceled",
      }));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to cancel pipeline"
      );
      throw err;
    }
  }, []);

  // Update config
  const updateConfig = useCallback(
    async (newConfig: PipelineConfig, acknowledgeIdsRisk?: boolean) => {
      try {
        setError(null);
        const headers: Record<string, string> = {
          "Content-Type": "application/json",
        };

        if (acknowledgeIdsRisk) {
          headers["X-Acknowledge-IDS-Risk"] = "true";
        }

        const response = await fetch("/api/pipeline/config", {
          method: "PUT",
          headers,
          credentials: "include", // Required for httpOnly cookie auth
          body: JSON.stringify(newConfig),
        });

        if (!response.ok) {
          const text = await response.text();
          throw new Error(
            text || `Failed to update config: ${response.status}`
          );
        }

        const updatedConfig: PipelineConfig = await response.json();
        setConfig(updatedConfig);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to update config"
        );
        throw err;
      }
    },
    []
  );

  // Handle WebSocket pipeline events
  const handlePipelineEvent = useCallback(
    (event: PipelineEvent) => {
      // Forward to external handler
      onMessage?.(event);

      switch (event.type) {
        case "pipeline_started": {
          const payload = event.payload as {
            totalPhases: number;
            phases: string[];
          };
          startTimeRef.current = Date.now();
          // Fixes #950: Copy array to prevent external mutation
          const phasesCopy = [...(payload.phases || [])];
          setStatus({
            ...defaultStatus,
            state: "enumerating",
            runId: event.runId,
            totalPhases: payload.totalPhases,
            enabledPhases: phasesCopy,
            currentPhase: phasesCopy[0] || "enumeration",
            phaseNumber: 1,
          });
          break;
        }

        case "phase_started": {
          const payload = event.payload as {
            phase: string;
            phaseNumber: number;
            totalPhases: number;
            deviceCount: number;
          };
          setStatus((prev) => ({
            ...prev,
            state: phaseToState(payload.phase),
            currentPhase: payload.phase,
            phaseNumber: payload.phaseNumber,
            totalPhases: payload.totalPhases,
            totalCount: payload.deviceCount,
            processedCount: 0,
            percentComplete: 0,
            currentTarget: "",
          }));
          break;
        }

        case "phase_progress": {
          const payload = event.payload as PhaseProgress;
          setStatus((prev) => ({
            ...prev,
            processedCount: payload.processedCount,
            totalCount: payload.totalCount,
            percentComplete: payload.percentComplete,
            currentTarget: payload.currentTarget || "",
            elapsedMs: payload.elapsedMs,
            estimatedRemainMs: payload.estimatedRemainMs || 0,
          }));
          break;
        }

        case "phase_completed": {
          const payload = event.payload as PhaseCompleted;
          setStatus((prev) => ({
            ...prev,
            processedCount: prev.totalCount,
            percentComplete: 100,
            currentTarget: "",
            phaseDurations: {
              ...prev.phaseDurations,
              [payload.phase]: payload.duration,
            },
            devicesFound: payload.devicesFound ?? prev.devicesFound,
          }));
          break;
        }

        case "pipeline_completed": {
          const payload = event.payload as {
            totalDevices: number;
            phaseDurations: Record<string, number>;
          };
          setStatus((prev) => ({
            ...prev,
            state: "complete",
            devicesFound: payload.totalDevices,
            phaseDurations: payload.phaseDurations || prev.phaseDurations,
            percentComplete: 100,
          }));
          break;
        }

        case "pipeline_failed":
        case "phase_failed": {
          const payload = event.payload as { error?: string; phase?: string };
          setStatus((prev) => {
            // Limit errors array to last 100 entries to prevent unbounded growth (fixes #855)
            const maxErrors = 100;
            const newErrors = payload.error
              ? [...prev.errors, payload.error].slice(-maxErrors)
              : prev.errors;
            return {
              ...prev,
              state: "failed",
              errors: newErrors,
            };
          });
          break;
        }

        case "pipeline_canceled":
          setStatus((prev) => ({
            ...prev,
            state: "canceled",
          }));
          break;

        case "device_discovered":
        case "device_updated":
          // These are handled by NetworkDiscoveryCard directly
          break;
      }
    },
    [onMessage]
  );

  // Keep ref in sync with latest handler (fixes #849)
  useEffect(() => {
    handlePipelineEventRef.current = handlePipelineEvent;
  }, [handlePipelineEvent]);

  // Expose handlePipelineEvent for external WebSocket integration
  // Uses a Set of handlers with stable references to support multiple components (fixes #842, #849)
  useEffect(() => {
    type WindowWithHandlers = Window &
      typeof globalThis & {
        __pipelineEventHandlers?: Set<(event: PipelineEvent) => void>;
        __pipelineEventHandler?: (event: PipelineEvent) => void;
      };
    const win = window as WindowWithHandlers;

    // Create stable wrapper that delegates to ref (fixes #849)
    const stableHandler = (event: PipelineEvent) => {
      handlePipelineEventRef.current?.(event);
    };

    // Fixes #937: Only create dispatcher once when Set is first created
    // This prevents stale dispatcher references when components mount/unmount
    if (!win.__pipelineEventHandlers) {
      win.__pipelineEventHandlers = new Set();
      // Create dispatcher only once at initialization
      win.__pipelineEventHandler = (event: PipelineEvent) => {
        win.__pipelineEventHandlers?.forEach((h) => h(event));
      };
    }

    // Add this component's stable handler
    win.__pipelineEventHandlers.add(stableHandler);

    return () => {
      // Remove this component's handler
      win.__pipelineEventHandlers?.delete(stableHandler);

      // Only delete the dispatcher if no handlers remain
      if (win.__pipelineEventHandlers?.size === 0) {
        delete win.__pipelineEventHandler;
        delete win.__pipelineEventHandlers;
      }
    };
  }, []); // Empty deps - stableHandler reference is stable

  // Update elapsed time while running
  // Fixes #970: Track interval ID locally for reliable cleanup
  useEffect(() => {
    let intervalId: ReturnType<typeof setInterval> | null = null;

    if (isRunning(status.state) && startTimeRef.current > 0) {
      intervalId = setInterval(() => {
        setStatus((prev) => ({
          ...prev,
          elapsedMs: Date.now() - startTimeRef.current,
        }));
      }, 1000);
      elapsedIntervalRef.current = intervalId;
    } else if (elapsedIntervalRef.current) {
      clearInterval(elapsedIntervalRef.current);
      elapsedIntervalRef.current = null;
    }

    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
      if (elapsedIntervalRef.current) {
        clearInterval(elapsedIntervalRef.current);
        elapsedIntervalRef.current = null;
      }
    };
  }, [status.state]);

  // Initial fetch
  // Fixes #931: Use try/finally to ensure loading state is always cleared
  useEffect(() => {
    const init = async () => {
      setIsLoading(true);
      try {
        await Promise.all([
          fetchStatus(),
          fetchConfig(),
          fetchPortIntensityInfo(),
          fetchTimingProfiles(),
        ]);
      } finally {
        setIsLoading(false);
      }
    };
    init();
  }, [fetchStatus, fetchConfig, fetchPortIntensityInfo, fetchTimingProfiles]);

  return {
    status,
    config,
    portIntensityInfo,
    timingProfiles,
    isLoading,
    error,
    startPipeline,
    cancelPipeline,
    updateConfig,
    refreshStatus: fetchStatus,
  };
}

// ============================================================================
// Helpers
// ============================================================================

function getEnabledPhases(config: PipelineConfig): string[] {
  const phases: string[] = ["enumeration"];
  if (config.phases.nameResolution) phases.push("resolution");
  if (config.phases.serviceDiscovery) phases.push("scanning");
  if (config.phases.vulnAssessment) phases.push("assessment");
  return phases;
}

/**
 *
 */
function isRunning(state: PipelineState): boolean {
  return (
    state === "enumerating" ||
    state === "resolving" ||
    state === "scanning" ||
    state === "assessing"
  );
}

function phaseToState(phase: string): PipelineState {
  switch (phase) {
    case "enumeration":
      return "enumerating";
    case "resolution":
      return "resolving";
    case "scanning":
      return "scanning";
    case "assessment":
      return "assessing";
    default:
      return "idle";
  }
}

// Export helper for use in other components
export { isRunning as isPipelineRunning };
