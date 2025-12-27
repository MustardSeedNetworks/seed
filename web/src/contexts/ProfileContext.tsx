/**
 * ProfileContext - MSP Profile Management & Settings
 *
 * Provides a React Context for managing MSP profiles and ALL user settings.
 * Profiles are the SINGLE SOURCE OF TRUTH for all configuration.
 *
 * Features:
 * - Load and cache profiles
 * - Track active profile
 * - Handle profile switching
 * - Listen for WebSocket profile change events
 * - Manage ALL user settings (cardSettings, displayOptions, thresholds, etc.)
 * - Auto-save settings to active profile with debouncing
 */

import {
  createContext,
  useState,
  useCallback,
  useEffect,
  useContext,
  useRef,
  useMemo,
  ReactNode,
} from "react";
import { api } from "../lib/api";
import { logger, LogComponents } from "../lib/logger";
import type {
  Profile,
  ProfileListResponse,
  ProfileRequest,
  ProfileImportRequest,
  ProfileImportResponse,
  ProfileExportResponse,
  ActiveProfileResponse,
  ProfileInterfaceSelection,
  ProfileSettings,
  CardSettingsConfig,
  DisplayOptionsConfig,
  AppearanceConfig,
  IperfConfig,
  ProfileThresholdsConfig,
  SpeedtestConfig,
  TestsConfig,
  NetworkDiscoveryConfig,
  SNMPConfig,
  WiFiSettingsConfig,
  LinkConfig,
  CableTestConfig,
  VulnerabilityConfig,
  DNSSettingsConfig,
} from "../types/profile";
import type { DefaultSettings } from "../types/defaults";

// ============================================================================
// Context Type Definition
// ============================================================================

/** Save status for settings operations */
export type SettingsSaveStatus = "idle" | "saving" | "saved" | "error";

export interface ProfileContextValue {
  // Profile State
  profiles: Profile[];
  activeProfile: Profile | null;
  isLoading: boolean;
  error: string | null;

  // Settings State (derived from active profile, merged with defaults)
  // Order matches Settings Drawer UI for consistency
  linkSettings: LinkConfig;
  cableTestSettings: CableTestConfig;
  displayOptions: DisplayOptionsConfig;
  wifiSettings: WiFiSettingsConfig;
  dnsSettings: DNSSettingsConfig;
  testsSettings: TestsConfig;
  speedtestSettings: SpeedtestConfig;
  iperfSettings: IperfConfig;
  networkDiscoverySettings: NetworkDiscoveryConfig;
  snmpSettings: SNMPConfig;
  vulnerabilitySettings: VulnerabilityConfig;
  thresholds: ProfileThresholdsConfig;
  appearanceSettings: AppearanceConfig;
  cardSettings: CardSettingsConfig;
  settingsStatus: SettingsSaveStatus;
  isSettingsLoaded: boolean;

  // List/fetch operations
  refreshProfiles: () => Promise<void>;
  refreshActiveProfile: () => Promise<void>;

  // CRUD operations
  createProfile: (profile: ProfileRequest) => Promise<Profile | null>;
  updateProfile: (
    id: string,
    profile: ProfileRequest
  ) => Promise<Profile | null>;
  deleteProfile: (id: string) => Promise<boolean>;

  // Profile switching
  switchProfile: (profileId: string) => Promise<boolean>;

  // Duplication
  duplicateProfile: (id: string, newName?: string) => Promise<Profile | null>;

  // Import/Export
  importProfiles: (
    request: ProfileImportRequest
  ) => Promise<ProfileImportResponse | null>;
  exportProfiles: () => Promise<ProfileExportResponse | null>;
  downloadProfiles: () => Promise<boolean>;

  // Settings update methods - auto-save to active profile with debounce
  // Order matches Settings Drawer UI for consistency
  updateLinkSettings: (updates: Partial<LinkConfig>) => void;
  updateCableTestSettings: (updates: Partial<CableTestConfig>) => void;
  updateDisplayOptions: (updates: Partial<DisplayOptionsConfig>) => void;
  updateWifiSettings: (updates: Partial<WiFiSettingsConfig>) => void;
  updateDnsSettings: (updates: Partial<DNSSettingsConfig>) => void;
  updateTestsSettings: (updates: Partial<TestsConfig>) => void;
  updateSpeedtestSettings: (updates: Partial<SpeedtestConfig>) => void;
  updateIperfSettings: (updates: Partial<IperfConfig>) => void;
  updateNetworkDiscoverySettings: (updates: Partial<NetworkDiscoveryConfig>) => void;
  updateSnmpSettings: (updates: Partial<SNMPConfig>) => void;
  updateVulnerabilitySettings: (updates: Partial<VulnerabilityConfig>) => void;
  updateThresholds: (updates: Partial<ProfileThresholdsConfig>) => void;
  updateAppearanceSettings: (updates: Partial<AppearanceConfig>) => void;
  updateCardSettings: (updates: Partial<CardSettingsConfig>) => void;
  /** Update any part of the profile settings */
  updateSettings: (updates: Partial<ProfileSettings>) => void;
  /** Force refresh settings from backend */
  refreshSettings: () => Promise<void>;

  // Interface selection helpers (multi-interface support)
  /** Get the active ethernet interface from the active profile */
  getEthernetInterface: () => ProfileInterfaceSelection | null;
  /** Get the active WiFi interface from the active profile */
  getWifiInterface: () => ProfileInterfaceSelection | null;
  /** Get all ethernet interfaces from the active profile */
  getAllEthernetInterfaces: () => ProfileInterfaceSelection[];
  /** Get all WiFi interfaces from the active profile */
  getAllWiFiInterfaces: () => ProfileInterfaceSelection[];
  /** Add or update an ethernet interface and set it as active */
  setEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add or update a WiFi interface and set it as active */
  setWifiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add an ethernet interface without changing the active one */
  addEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add a WiFi interface without changing the active one */
  addWiFiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Remove an ethernet interface */
  removeEthernetInterface: (name: string) => Promise<boolean>;
  /** Remove a WiFi interface */
  removeWiFiInterface: (name: string) => Promise<boolean>;
  /** Set the active ethernet interface (must already be in the list) */
  setActiveEthernetInterface: (name: string) => Promise<boolean>;
  /** Set the active WiFi interface (must already be in the list) */
  setActiveWiFiInterface: (name: string) => Promise<boolean>;
}

// Create context with undefined default to enforce provider requirement
const ProfileContext = createContext<ProfileContextValue | undefined>(
  undefined
);

// ============================================================================
// Provider Component
// ============================================================================

interface ProfileProviderProps {
  children: ReactNode;
}

// Debounce delay for saving settings to profile
const SETTINGS_DEBOUNCE_MS = 1000;

// Default settings values - loaded from backend, fallback if API fails
const DEFAULT_CARD_SETTINGS: CardSettingsConfig = {
  // Core network cards
  link: { enabled: true, autoRunOnLink: true },
  cable: { enabled: true, autoRunOnLink: false }, // Cable test on demand only
  switch: { enabled: true, autoRunOnLink: true },
  vlan: { enabled: true, autoRunOnLink: true },
  network: { enabled: true, autoRunOnLink: true },
  gateway: { enabled: true, autoRunOnLink: true },
  dns: { enabled: true, autoRunOnLink: true },
  publicIP: { enabled: true, autoRunOnLink: true },

  // WiFi cards
  wifi: { enabled: true, autoRunOnLink: true },
  wifiSurvey: { enabled: true, autoRunOnLink: false }, // Survey on demand

  // Diagnostic/analysis cards
  healthChecks: { enabled: true, autoRunOnLink: true },
  networkDiscovery: { enabled: true, autoRunOnLink: true },
  pathDiscovery: { enabled: true, autoRunOnLink: false }, // Path discovery on demand
  systemHealth: { enabled: true, autoRunOnLink: false }, // System health passive

  // Performance testing
  performance: {
    enabled: true,
    autoRunOnLink: true,
    speedtest: { enabled: true, autoRunOnLink: true },
    iperf: { enabled: false, autoRunOnLink: false },
  },
};

const DEFAULT_DISPLAY_OPTIONS: DisplayOptionsConfig = {
  showPublicIP: true,
  unitSystem: "sae",
};

const DEFAULT_IPERF_SETTINGS: IperfConfig = {
  server: "",
  port: 5201,
  protocol: "tcp",
  direction: "download",
  duration: 10,
  serverPort: 5201,
  enableServer: true,
};

const DEFAULT_THRESHOLDS: ProfileThresholdsConfig = {
  dns: { good: 50, warning: 100 },
  gateway: { good: 20, warning: 50 },
  wifi: { good: -50, warning: -70 },
  customPing: { good: 50, warning: 100 },
  customTcp: { good: 100, warning: 200 },
  customHttp: { good: 500, warning: 1000 },
  httpTimings: {
    dns: { good: 50, warning: 100 },
    tcp: { good: 50, warning: 100 },
    tls: { good: 100, warning: 200 },
    ttfb: { good: 200, warning: 500 },
  },
};

const DEFAULT_SPEEDTEST_SETTINGS: SpeedtestConfig = {
  serverId: "",
  autoRunOnLink: true,
};

const DEFAULT_TESTS_SETTINGS: TestsConfig = {
  dnsHostname: "google.com",
  pingTargets: [
    {
      id: "default-google-dns",
      name: "Google DNS",
      host: "8.8.8.8",
      enabled: true,
      count: 3,
    },
    {
      id: "default-cloudflare-dns",
      name: "Cloudflare",
      host: "1.1.1.1",
      enabled: true,
      count: 3,
    },
  ],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [
    {
      id: "default-google",
      name: "Google",
      url: "https://www.google.com",
      expectedStatus: 200,
      enabled: true,
    },
  ],
  runPerformance: false,
  runSpeedtest: false,
  runIperf: false,
  runDiscovery: false,
};

const DEFAULT_NETWORK_DISCOVERY_SETTINGS: NetworkDiscoveryConfig = {
  enabled: true,
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: true,
  scanIntervalMs: 600000, // 10 minutes
  ipv6Enabled: true,
  options: {
    passiveProtocols: {
      lldp: true,
      cdp: true,
      edp: true,
      ndp: true,
    },
    arpScan: true,
    icmpScan: true,
    portScan: {
      enabled: false, // Port scanning off by default for security
      preset: "common",
      tcpPorts: "22,80,443,8080-8100",
      udpPorts: "53,123,161",
      bannerTimeoutMs: 2000,
    },
    tcpProbe: {
      timeoutMs: 2000,
      workers: 20,
    },
    traceroute: false,
    snmpQuery: false,
  },
  timing: {
    probeIntervalMs: 75,
    rescanIntervalMs: 600000, // 10 minutes
    workers: 50,
  },
  profiler: {
    enabled: true,
    timeoutMs: 2000,
    maxConcurrent: 5,
    quickPorts: [22, 80, 443, 8080],
  },
  fingerprinting: {
    enabled: false,
    osDetection: false,
    serviceProbes: false,
  },
};

const DEFAULT_SNMP_SETTINGS: SNMPConfig = {
  communities: ["public"],
  v3Credentials: [], // No v3 credentials by default
  timeoutMs: 5000, // 5 seconds
  retries: 2,
  port: 161,
};

const DEFAULT_WIFI_SETTINGS: WiFiSettingsConfig = {
  interface: "",
  surveyEnabled: true,
  surveyIntervalMs: 5000, // 5 seconds
  signalThreshold: -70, // dBm
};

const DEFAULT_LINK_SETTINGS: LinkConfig = {
  mode: "auto",
  availableModes: [],
};

const DEFAULT_CABLE_TEST_SETTINGS: CableTestConfig = {
  enabled: true,
};

const DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityConfig = {
  enabled: true, // Enable by default for security visibility
  cveDatabase: "nvd", // NVD works without API key (rate limited)
  nvdApiKey: "",
  updateInterval: 86400, // 24 hours
  severityThreshold: "medium",
  maxConcurrent: 5,
  autoScan: true, // Auto-scan after device discovery
};

const DEFAULT_DNS_SETTINGS: DNSSettingsConfig = {
  testHostname: "google.com",
  servers: [], // Use system DNS by default
};

const DEFAULT_APPEARANCE_SETTINGS: AppearanceConfig = {
  theme: "system", // Respect OS preference
  language: "en", // English default
};

/**
 * Context provider that manages profile state, settings, and API synchronization.
 * Profiles are the SINGLE SOURCE OF TRUTH for all user settings.
 */
export function ProfileProvider({ children }: ProfileProviderProps) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [activeProfile, setActiveProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Settings state - derived from active profile, merged with defaults
  const [backendDefaults, setBackendDefaults] = useState<DefaultSettings | null>(null);
  const [settingsStatus, setSettingsStatus] = useState<SettingsSaveStatus>("idle");
  const [isSettingsLoaded, setIsSettingsLoaded] = useState(false);

  const isMountedRef = useRef(true);
  const settingsDebounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const settingsStatusResetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const settingsSaveController = useRef<AbortController | null>(null);

  // ============================================================================
  // Load Profiles
  // ============================================================================

  const refreshProfiles = useCallback(async () => {
    try {
      setError(null);
      setIsLoading(true);
      const data = await api.get<ProfileListResponse>("/api/profiles");
      if (isMountedRef.current) {
        setProfiles(data.profiles || []);
      }
    } catch (err) {
      if (isMountedRef.current) {
        // Don't show session expired errors in profile panel - handled globally
        const message =
          err instanceof Error ? err.message : "Failed to fetch profiles";
        if (!message.toLowerCase().includes("session")) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to fetch profiles", err);
      }
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, []);

  const refreshActiveProfile = useCallback(async () => {
    try {
      setError(null);
      const profile = await api.get<Profile>("/api/profiles/active");
      if (isMountedRef.current) {
        setActiveProfile(profile);
        setIsSettingsLoaded(true);
      }
    } catch (err) {
      if (isMountedRef.current) {
        // Active profile may not exist yet, which is okay
        // Don't show session expired errors - handled globally
        const message =
          err instanceof Error ? err.message : "Failed to fetch active profile";
        if (
          !(err instanceof Error && err.message.includes("404")) &&
          !message.toLowerCase().includes("session")
        ) {
          setError(message);
          logger.error(
            LogComponents.PROFILES,
            "Failed to fetch active profile",
            err
          );
        }
        // Still mark as loaded even if no active profile exists
        setIsSettingsLoaded(true);
      }
    }
  }, []);

  // ============================================================================
  // Load Backend Defaults
  // ============================================================================

  const loadBackendDefaults = useCallback(async () => {
    try {
      const defaults = await api.get<DefaultSettings>("/api/settings/defaults");
      if (isMountedRef.current) {
        setBackendDefaults(defaults);
      }
    } catch (err) {
      logger.warn(
        LogComponents.CONFIG,
        "Failed to fetch defaults from backend, using fallback",
        err
      );
    }
  }, []);

  // ============================================================================
  // Computed Settings (derived from active profile, merged with defaults)
  // ============================================================================

  /**
   * Get card settings from active profile, merged with defaults.
   * Profile settings take priority, defaults fill in missing values.
   */
  const cardSettings = useMemo((): CardSettingsConfig => {
    const profileSettings = activeProfile?.config?.settings?.cardSettings;
    const defaults = backendDefaults?.cardSettings ?? DEFAULT_CARD_SETTINGS;

    if (!profileSettings) return defaults as CardSettingsConfig;

    // Deep merge profile settings with defaults - order matches card display order
    return {
      // Core network cards (top row)
      link: { ...defaults.link, ...profileSettings.link },
      cable: { ...defaults.cable, ...profileSettings.cable },
      switch: { ...defaults.switch, ...profileSettings.switch },
      vlan: { ...defaults.vlan, ...profileSettings.vlan },
      network: { ...defaults.network, ...profileSettings.network },
      gateway: { ...defaults.gateway, ...profileSettings.gateway },
      dns: { ...defaults.dns, ...profileSettings.dns },
      publicIP: { ...defaults.publicIP, ...profileSettings.publicIP },

      // WiFi cards
      wifi: { ...defaults.wifi, ...profileSettings.wifi },
      wifiSurvey: { ...defaults.wifiSurvey, ...profileSettings.wifiSurvey },

      // Diagnostic/analysis cards
      healthChecks: { ...defaults.healthChecks, ...profileSettings.healthChecks },
      networkDiscovery: { ...defaults.networkDiscovery, ...profileSettings.networkDiscovery },
      pathDiscovery: { ...defaults.pathDiscovery, ...profileSettings.pathDiscovery },
      systemHealth: { ...defaults.systemHealth, ...profileSettings.systemHealth },

      // Performance testing (bottom)
      performance: {
        ...defaults.performance,
        ...profileSettings.performance,
        speedtest: {
          ...(defaults.performance as { speedtest: { enabled: boolean; autoRunOnLink: boolean } }).speedtest,
          ...profileSettings.performance?.speedtest,
        },
        iperf: {
          ...(defaults.performance as { iperf: { enabled: boolean; autoRunOnLink: boolean } }).iperf,
          ...profileSettings.performance?.iperf,
        },
      },
    } as CardSettingsConfig;
  }, [activeProfile, backendDefaults]);

  /**
   * Get display options from active profile, merged with defaults.
   */
  const displayOptions = useMemo((): DisplayOptionsConfig => {
    const profileSettings = activeProfile?.config?.settings?.displayOptions;
    const defaults = backendDefaults?.displayOptions ?? DEFAULT_DISPLAY_OPTIONS;

    if (!profileSettings) return defaults as DisplayOptionsConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get iPerf settings from active profile, merged with defaults.
   */
  const iperfSettings = useMemo((): IperfConfig => {
    const profileSettings = activeProfile?.config?.settings?.iperf;
    const defaults = backendDefaults?.iperf ?? DEFAULT_IPERF_SETTINGS;

    if (!profileSettings) return defaults as IperfConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get thresholds from active profile, merged with defaults.
   */
  const thresholds = useMemo((): ProfileThresholdsConfig => {
    const profileSettings = activeProfile?.config?.settings?.thresholds;
    const defaults = backendDefaults?.thresholds ?? DEFAULT_THRESHOLDS;

    if (!profileSettings) return defaults as ProfileThresholdsConfig;

    // Deep merge for httpTimings
    return {
      dns: { ...defaults.dns, ...profileSettings.dns },
      gateway: { ...defaults.gateway, ...profileSettings.gateway },
      wifi: { ...defaults.wifi, ...profileSettings.wifi },
      customPing: { ...defaults.customPing, ...profileSettings.customPing },
      customTcp: { ...defaults.customTcp, ...profileSettings.customTcp },
      customHttp: { ...defaults.customHttp, ...profileSettings.customHttp },
      httpTimings: {
        dns: { ...defaults.httpTimings.dns, ...profileSettings.httpTimings?.dns },
        tcp: { ...defaults.httpTimings.tcp, ...profileSettings.httpTimings?.tcp },
        tls: { ...defaults.httpTimings.tls, ...profileSettings.httpTimings?.tls },
        ttfb: { ...defaults.httpTimings.ttfb, ...profileSettings.httpTimings?.ttfb },
      },
    } as ProfileThresholdsConfig;
  }, [activeProfile, backendDefaults]);

  /**
   * Get speedtest settings from active profile, merged with defaults.
   */
  const speedtestSettings = useMemo((): SpeedtestConfig => {
    const profileSettings = activeProfile?.config?.settings?.speedtest;
    const defaults = backendDefaults?.tests?.speedtest ?? DEFAULT_SPEEDTEST_SETTINGS;

    if (!profileSettings) return defaults as SpeedtestConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get tests settings from active profile, merged with defaults.
   */
  const testsSettings = useMemo((): TestsConfig => {
    const profileSettings = activeProfile?.config?.settings?.tests;
    const defaults = backendDefaults?.tests ?? DEFAULT_TESTS_SETTINGS;

    if (!profileSettings) return defaults as TestsConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get network discovery settings from active profile, merged with defaults.
   */
  const networkDiscoverySettings = useMemo((): NetworkDiscoveryConfig => {
    const profileSettings = activeProfile?.config?.settings?.networkDiscovery;
    const defaults = backendDefaults?.networkDiscovery ?? DEFAULT_NETWORK_DISCOVERY_SETTINGS;

    if (!profileSettings) return defaults as NetworkDiscoveryConfig;

    // Deep merge for nested objects
    return {
      ...defaults,
      ...profileSettings,
      options: {
        ...defaults.options,
        ...profileSettings.options,
        passiveProtocols: {
          ...defaults.options?.passiveProtocols,
          ...profileSettings.options?.passiveProtocols,
        },
        portScan: {
          ...defaults.options?.portScan,
          ...profileSettings.options?.portScan,
        },
        tcpProbe: {
          ...defaults.options?.tcpProbe,
          ...profileSettings.options?.tcpProbe,
        },
      },
      timing: {
        ...defaults.timing,
        ...profileSettings.timing,
      },
      profiler: {
        ...defaults.profiler,
        ...profileSettings.profiler,
      },
      fingerprinting: {
        ...defaults.fingerprinting,
        ...profileSettings.fingerprinting,
      },
    } as NetworkDiscoveryConfig;
  }, [activeProfile, backendDefaults]);

  /**
   * Get SNMP settings from active profile, merged with defaults.
   */
  const snmpSettings = useMemo((): SNMPConfig => {
    const profileSettings = activeProfile?.config?.settings?.snmp;
    const defaults = backendDefaults?.snmp ?? DEFAULT_SNMP_SETTINGS;

    if (!profileSettings) return defaults as SNMPConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get WiFi settings from active profile, merged with defaults.
   */
  const wifiSettings = useMemo((): WiFiSettingsConfig => {
    const profileSettings = activeProfile?.config?.settings?.wifi;
    const defaults = DEFAULT_WIFI_SETTINGS;

    if (!profileSettings) return defaults;
    return { ...defaults, ...profileSettings };
  }, [activeProfile]);

  /**
   * Get link settings from active profile, merged with defaults.
   */
  const linkSettings = useMemo((): LinkConfig => {
    const profileSettings = activeProfile?.config?.settings?.link;
    const defaults = backendDefaults?.link ?? DEFAULT_LINK_SETTINGS;

    if (!profileSettings) return defaults as LinkConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get cable test settings from active profile, merged with defaults.
   */
  const cableTestSettings = useMemo((): CableTestConfig => {
    const profileSettings = activeProfile?.config?.settings?.cableTest;
    const defaults = backendDefaults?.cableTest ?? DEFAULT_CABLE_TEST_SETTINGS;

    if (!profileSettings) return defaults as CableTestConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get vulnerability settings from active profile, merged with defaults.
   */
  const vulnerabilitySettings = useMemo((): VulnerabilityConfig => {
    const profileSettings = activeProfile?.config?.settings?.vulnerability;
    const defaults = backendDefaults?.vulnerability ?? DEFAULT_VULNERABILITY_SETTINGS;

    if (!profileSettings) return defaults as VulnerabilityConfig;
    return { ...defaults, ...profileSettings };
  }, [activeProfile, backendDefaults]);

  /**
   * Get DNS settings from active profile, merged with defaults.
   */
  const dnsSettings = useMemo((): DNSSettingsConfig => {
    const profileSettings = activeProfile?.config?.settings?.dns;
    const defaults = DEFAULT_DNS_SETTINGS;

    if (!profileSettings) return defaults;
    return { ...defaults, ...profileSettings };
  }, [activeProfile]);

  /**
   * Get appearance settings from active profile, merged with defaults.
   */
  const appearanceSettings = useMemo((): AppearanceConfig => {
    const profileSettings = activeProfile?.config?.settings?.appearance;
    const defaults = DEFAULT_APPEARANCE_SETTINGS;

    if (!profileSettings) return defaults;
    return { ...defaults, ...profileSettings };
  }, [activeProfile]);

  // ============================================================================
  // Settings Update Methods - Auto-save to Active Profile
  // ============================================================================

  /**
   * Save settings to the active profile with debouncing.
   * This is the core auto-save mechanism.
   */
  const saveSettingsToProfile = useCallback(
    async (updatedSettings: ProfileSettings) => {
      if (!activeProfile) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot save settings: no active profile"
        );
        return;
      }

      // Clear any existing timers
      if (settingsDebounceTimer.current) {
        clearTimeout(settingsDebounceTimer.current);
      }
      if (settingsStatusResetTimer.current) {
        clearTimeout(settingsStatusResetTimer.current);
      }

      // Show saving status immediately
      setSettingsStatus("saving");

      // Debounce the actual save
      settingsDebounceTimer.current = setTimeout(async () => {
        // Cancel any in-flight request
        if (settingsSaveController.current) {
          settingsSaveController.current.abort();
        }
        settingsSaveController.current = new AbortController();

        try {
          const updatedConfig = {
            ...activeProfile.config,
            settings: {
              ...activeProfile.config?.settings,
              ...updatedSettings,
            },
          };

          await api.put(
            `/api/profiles/${activeProfile.id}`,
            {
              name: activeProfile.name,
              description: activeProfile.description,
              config: updatedConfig,
            },
            { signal: settingsSaveController.current.signal }
          );

          if (isMountedRef.current) {
            // Update local state without full refresh for responsiveness
            setActiveProfile((prev) =>
              prev
                ? {
                    ...prev,
                    config: updatedConfig,
                  }
                : null
            );
            setSettingsStatus("saved");

            // Reset status after 2 seconds
            settingsStatusResetTimer.current = setTimeout(() => {
              if (isMountedRef.current) {
                setSettingsStatus("idle");
              }
            }, 2000);
          }

          logger.debug(LogComponents.PROFILES, "Settings saved to profile", {
            profileId: activeProfile.id,
          });
        } catch (err) {
          if (
            err instanceof Error &&
            err.name === "AbortError"
          ) {
            // Request was cancelled, ignore
            return;
          }

          if (isMountedRef.current) {
            setSettingsStatus("error");
            logger.error(
              LogComponents.PROFILES,
              "Failed to save settings to profile",
              err
            );

            // Reset error status after 5 seconds
            settingsStatusResetTimer.current = setTimeout(() => {
              if (isMountedRef.current) {
                setSettingsStatus("idle");
              }
            }, 5000);
          }
        }
      }, SETTINGS_DEBOUNCE_MS);
    },
    [activeProfile]
  );

  /**
   * Update card settings - triggers auto-save to active profile.
   */
  const updateCardSettings = useCallback(
    (updates: Partial<CardSettingsConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newCardSettings = {
        ...currentSettings.cardSettings,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, cardSettings: newCardSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update display options - triggers auto-save to active profile.
   */
  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptionsConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newDisplayOptions = {
        ...currentSettings.displayOptions,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, displayOptions: newDisplayOptions });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update iPerf settings - triggers auto-save to active profile.
   */
  const updateIperfSettings = useCallback(
    (updates: Partial<IperfConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newIperfSettings = {
        ...currentSettings.iperf,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, iperf: newIperfSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update thresholds - triggers auto-save to active profile.
   */
  const updateThresholds = useCallback(
    (updates: Partial<ProfileThresholdsConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newThresholds = {
        ...currentSettings.thresholds,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, thresholds: newThresholds });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update speedtest settings - triggers auto-save to active profile.
   */
  const updateSpeedtestSettings = useCallback(
    (updates: Partial<SpeedtestConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newSpeedtestSettings = {
        ...currentSettings.speedtest,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, speedtest: newSpeedtestSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update tests settings - triggers auto-save to active profile.
   */
  const updateTestsSettings = useCallback(
    (updates: Partial<TestsConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newTestsSettings = {
        ...currentSettings.tests,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, tests: newTestsSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update network discovery settings - triggers auto-save to active profile.
   */
  const updateNetworkDiscoverySettings = useCallback(
    (updates: Partial<NetworkDiscoveryConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newNetworkDiscoverySettings = {
        ...currentSettings.networkDiscovery,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, networkDiscovery: newNetworkDiscoverySettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update SNMP settings - triggers auto-save to active profile.
   */
  const updateSnmpSettings = useCallback(
    (updates: Partial<SNMPConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newSnmpSettings = {
        ...currentSettings.snmp,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, snmp: newSnmpSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update WiFi settings - triggers auto-save to active profile.
   */
  const updateWifiSettings = useCallback(
    (updates: Partial<WiFiSettingsConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newWifiSettings = {
        ...currentSettings.wifi,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, wifi: newWifiSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update link settings - triggers auto-save to active profile.
   */
  const updateLinkSettings = useCallback(
    (updates: Partial<LinkConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newLinkSettings = {
        ...currentSettings.link,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, link: newLinkSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update cable test settings - triggers auto-save to active profile.
   */
  const updateCableTestSettings = useCallback(
    (updates: Partial<CableTestConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newCableTestSettings = {
        ...currentSettings.cableTest,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, cableTest: newCableTestSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update vulnerability settings - triggers auto-save to active profile.
   */
  const updateVulnerabilitySettings = useCallback(
    (updates: Partial<VulnerabilityConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newVulnerabilitySettings = {
        ...currentSettings.vulnerability,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, vulnerability: newVulnerabilitySettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update DNS settings - triggers auto-save to active profile.
   */
  const updateDnsSettings = useCallback(
    (updates: Partial<DNSSettingsConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newDnsSettings = {
        ...currentSettings.dns,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, dns: newDnsSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update appearance settings - triggers auto-save to active profile.
   */
  const updateAppearanceSettings = useCallback(
    (updates: Partial<AppearanceConfig>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      const newAppearanceSettings = {
        ...currentSettings.appearance,
        ...updates,
      };
      saveSettingsToProfile({ ...currentSettings, appearance: newAppearanceSettings });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Update any part of the profile settings - triggers auto-save.
   */
  const updateSettings = useCallback(
    (updates: Partial<ProfileSettings>) => {
      const currentSettings = activeProfile?.config?.settings ?? {};
      saveSettingsToProfile({ ...currentSettings, ...updates });
    },
    [activeProfile, saveSettingsToProfile]
  );

  /**
   * Force refresh settings from backend.
   */
  const refreshSettings = useCallback(async () => {
    await refreshActiveProfile();
  }, [refreshActiveProfile]);

  // ============================================================================
  // CRUD Operations
  // ============================================================================

  const createProfile = useCallback(
    async (profile: ProfileRequest): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const created = await api.post<Profile>("/api/profiles", profile);
        if (isMountedRef.current) {
          setProfiles((prev) => [...prev, created]);
        }
        logger.info(LogComponents.PROFILES, "Profile created", {
          profileId: created.id,
          name: created.name,
        });
        return created;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to create profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to create profile", err);
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    []
  );

  const updateProfile = useCallback(
    async (id: string, profile: ProfileRequest): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const updated = await api.put<Profile>(`/api/profiles/${id}`, profile);
        if (isMountedRef.current) {
          setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
          if (activeProfile?.id === id) {
            setActiveProfile(updated);
          }
        }
        logger.info(LogComponents.PROFILES, "Profile updated", {
          profileId: id,
          name: updated.name,
        });
        return updated;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to update profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to update profile", err);
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [activeProfile]
  );

  const deleteProfile = useCallback(async (id: string): Promise<boolean> => {
    try {
      setError(null);
      setIsLoading(true);
      await api.delete(`/api/profiles/${id}`);
      if (isMountedRef.current) {
        setProfiles((prev) => prev.filter((p) => p.id !== id));
      }
      logger.info(LogComponents.PROFILES, "Profile deleted", { profileId: id });
      return true;
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to delete profile";
      if (isMountedRef.current) {
        setError(message);
      }
      logger.error(LogComponents.PROFILES, "Failed to delete profile", err);
      return false;
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, []);

  // ============================================================================
  // Profile Switching
  // ============================================================================

  const switchProfile = useCallback(
    async (profileId: string): Promise<boolean> => {
      try {
        setError(null);
        setIsLoading(true);
        const result = await api.post<ActiveProfileResponse>(
          "/api/profiles/active",
          {
            profile_id: profileId,
          }
        );
        if (isMountedRef.current) {
          setActiveProfile(result.profile);
        }
        logger.info(LogComponents.PROFILES, "Profile switched", {
          profileId,
          name: result.profile.name,
        });
        return true;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to switch profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to switch profile", err);
        return false;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    []
  );

  // ============================================================================
  // Duplication
  // ============================================================================

  const duplicateProfile = useCallback(
    async (id: string, newName?: string): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const duplicated = await api.post<Profile>(
          `/api/profiles/${id}/duplicate`,
          {
            name: newName,
          }
        );
        if (isMountedRef.current) {
          setProfiles((prev) => [...prev, duplicated]);
        }
        logger.info(LogComponents.PROFILES, "Profile duplicated", {
          sourceId: id,
          newId: duplicated.id,
          name: duplicated.name,
        });
        return duplicated;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to duplicate profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(
          LogComponents.PROFILES,
          "Failed to duplicate profile",
          err
        );
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    []
  );

  // ============================================================================
  // Import/Export
  // ============================================================================

  const importProfiles = useCallback(
    async (
      request: ProfileImportRequest
    ): Promise<ProfileImportResponse | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const result = await api.post<ProfileImportResponse>(
          "/api/profiles/import",
          request
        );
        // Refresh the profile list after import
        await refreshProfiles();
        logger.info(LogComponents.PROFILES, "Profiles imported", {
          created: result.created,
          updated: result.updated,
          skipped: result.skipped,
        });
        return result;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to import profiles";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to import profiles", err);
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [refreshProfiles]
  );

  const exportProfiles =
    useCallback(async (): Promise<ProfileExportResponse | null> => {
      try {
        setError(null);
        const result = await api.get<ProfileExportResponse>(
          "/api/profiles/export"
        );
        logger.info(LogComponents.PROFILES, "Profiles exported", {
          count: result.profiles.length,
        });
        return result;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to export profiles";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to export profiles", err);
        return null;
      }
    }, []);

  const downloadProfiles = useCallback(async (): Promise<boolean> => {
    try {
      const result = await exportProfiles();
      if (!result) return false;

      // Create and trigger download
      const blob = new Blob([JSON.stringify(result, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `seed-profiles-${new Date().toISOString().split("T")[0]}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);

      return true;
    } catch (err) {
      logger.error(LogComponents.PROFILES, "Failed to download profiles", err);
      return false;
    }
  }, [exportProfiles]);

  // ============================================================================
  // Interface Selection Helpers
  // ============================================================================

  /**
   * Get the currently active ethernet interface from the active profile.
   */
  const getEthernetInterface =
    useCallback((): ProfileInterfaceSelection | null => {
      const interfaces = activeProfile?.config?.interfaces;
      if (!interfaces?.active_ethernet || !interfaces.ethernet) return null;
      return (
        interfaces.ethernet.find((i) => i.name === interfaces.active_ethernet) ??
        null
      );
    }, [activeProfile]);

  /**
   * Get the currently active WiFi interface from the active profile.
   */
  const getWifiInterface = useCallback((): ProfileInterfaceSelection | null => {
    const interfaces = activeProfile?.config?.interfaces;
    if (!interfaces?.active_wifi || !interfaces.wifi) return null;
    return (
      interfaces.wifi.find((i) => i.name === interfaces.active_wifi) ?? null
    );
  }, [activeProfile]);

  /**
   * Get all ethernet interfaces from the active profile.
   */
  const getAllEthernetInterfaces = useCallback((): ProfileInterfaceSelection[] => {
    return activeProfile?.config?.interfaces?.ethernet ?? [];
  }, [activeProfile]);

  /**
   * Get all WiFi interfaces from the active profile.
   */
  const getAllWiFiInterfaces = useCallback((): ProfileInterfaceSelection[] => {
    return activeProfile?.config?.interfaces?.wifi ?? [];
  }, [activeProfile]);

  /**
   * Helper to update interface config on the backend.
   */
  const updateInterfaceConfig = useCallback(
    async (
      updater: (
        interfaces: NonNullable<
          NonNullable<Profile["config"]>["interfaces"]
        >
      ) => NonNullable<NonNullable<Profile["config"]>["interfaces"]>
    ): Promise<boolean> => {
      if (!activeProfile) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot update interfaces: no active profile"
        );
        return false;
      }

      try {
        const currentInterfaces = activeProfile.config?.interfaces ?? {
          ethernet: [],
          wifi: [],
        };
        const updatedInterfaces = updater(currentInterfaces);

        const updatedConfig = {
          ...activeProfile.config,
          interfaces: updatedInterfaces,
        };

        await api.put(`/api/profiles/${activeProfile.id}`, {
          name: activeProfile.name,
          description: activeProfile.description,
          config: updatedConfig,
        });

        await refreshActiveProfile();
        return true;
      } catch (err) {
        logger.error(
          LogComponents.PROFILES,
          "Failed to update interface config",
          err
        );
        return false;
      }
    },
    [activeProfile, refreshActiveProfile]
  );

  /**
   * Add or update an ethernet interface and set it as active.
   * Fixes #868: Use ref to get current activeProfile to avoid stale closure.
   */
  const activeProfileRef = useRef(activeProfile);
  activeProfileRef.current = activeProfile;

  const setEthernetInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = [...(interfaces.ethernet ?? [])];
        const existingIdx = ethernet.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          ethernet[existingIdx] = { ...ethernet[existingIdx], enabled };
        } else {
          ethernet.push({ name, enabled });
        }
        return { ...interfaces, ethernet, active_ethernet: name };
      });
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "Ethernet interface set as active", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Add or update a WiFi interface and set it as active.
   */
  const setWifiInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = [...(interfaces.wifi ?? [])];
        const existingIdx = wifi.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          wifi[existingIdx] = { ...wifi[existingIdx], enabled };
        } else {
          wifi.push({ name, enabled });
        }
        return { ...interfaces, wifi, active_wifi: name };
      });
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "WiFi interface set as active", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Add an ethernet interface without changing the active one.
   */
  const addEthernetInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = [...(interfaces.ethernet ?? [])];
        const existingIdx = ethernet.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          ethernet[existingIdx] = { ...ethernet[existingIdx], enabled };
        } else {
          ethernet.push({ name, enabled });
        }
        return { ...interfaces, ethernet };
      });
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "Ethernet interface added", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Add a WiFi interface without changing the active one.
   */
  const addWiFiInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = [...(interfaces.wifi ?? [])];
        const existingIdx = wifi.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          wifi[existingIdx] = { ...wifi[existingIdx], enabled };
        } else {
          wifi.push({ name, enabled });
        }
        return { ...interfaces, wifi };
      });
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "WiFi interface added", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Remove an ethernet interface.
   */
  const removeEthernetInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = (interfaces.ethernet ?? []).filter(
          (i) => i.name !== name
        );
        const active_ethernet =
          interfaces.active_ethernet === name ? "" : interfaces.active_ethernet;
        return { ...interfaces, ethernet, active_ethernet };
      });
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "Ethernet interface removed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Remove a WiFi interface.
   */
  const removeWiFiInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = (interfaces.wifi ?? []).filter((i) => i.name !== name);
        const active_wifi =
          interfaces.active_wifi === name ? "" : interfaces.active_wifi;
        return { ...interfaces, wifi, active_wifi };
      });
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "WiFi interface removed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Set the active ethernet interface (must already be in the list).
   */
  const setActiveEthernetInterface = useCallback(
    async (name: string): Promise<boolean> => {
      // Check if the interface exists in the list - use ref for current value (fixes #868)
      const currentProfile = activeProfileRef.current;
      const exists = (currentProfile?.config?.interfaces?.ethernet ?? []).some(
        (i) => i.name === name
      );
      if (!exists) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot set active ethernet interface: interface not in list",
          { interface: name }
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        active_ethernet: name,
      }));
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "Active ethernet interface changed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  /**
   * Set the active WiFi interface (must already be in the list).
   */
  const setActiveWiFiInterface = useCallback(
    async (name: string): Promise<boolean> => {
      // Check if the interface exists in the list - use ref for current value (fixes #868)
      const currentProfile = activeProfileRef.current;
      const exists = (currentProfile?.config?.interfaces?.wifi ?? []).some(
        (i) => i.name === name
      );
      if (!exists) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot set active WiFi interface: interface not in list",
          { interface: name }
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        active_wifi: name,
      }));
      if (result) {
        // Use ref to get current value, avoiding stale closure (fixes #868)
        logger.info(LogComponents.PROFILES, "Active WiFi interface changed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig]
  );

  // ============================================================================
  // Initial Load
  // ============================================================================

  useEffect(() => {
    isMountedRef.current = true;

    // Load profiles, active profile, and backend defaults on mount
    const loadInitialData = async () => {
      try {
        await Promise.all([
          refreshProfiles(),
          refreshActiveProfile(),
          loadBackendDefaults(),
        ]);
      } catch (err) {
        logger.error(
          LogComponents.PROFILES,
          "Failed to load initial profile data",
          err
        );
      }
    };

    loadInitialData();

    // Cleanup timers on unmount
    return () => {
      isMountedRef.current = false;
      if (settingsDebounceTimer.current) {
        clearTimeout(settingsDebounceTimer.current);
      }
      if (settingsStatusResetTimer.current) {
        clearTimeout(settingsStatusResetTimer.current);
      }
      if (settingsSaveController.current) {
        settingsSaveController.current.abort();
      }
    };
  }, [refreshProfiles, refreshActiveProfile, loadBackendDefaults]);

  // ============================================================================
  // Context Value
  // ============================================================================

  const contextValue: ProfileContextValue = {
    // Profile state
    profiles,
    activeProfile,
    isLoading,
    error,

    // Settings state (derived from active profile)
    // Order matches Settings Drawer UI for consistency
    linkSettings,
    cableTestSettings,
    displayOptions,
    wifiSettings,
    dnsSettings,
    testsSettings,
    speedtestSettings,
    iperfSettings,
    networkDiscoverySettings,
    snmpSettings,
    vulnerabilitySettings,
    thresholds,
    appearanceSettings,
    cardSettings,
    settingsStatus,
    isSettingsLoaded,

    // Profile list/fetch operations
    refreshProfiles,
    refreshActiveProfile,

    // Profile CRUD operations
    createProfile,
    updateProfile,
    deleteProfile,
    switchProfile,
    duplicateProfile,

    // Profile import/export
    importProfiles,
    exportProfiles,
    downloadProfiles,

    // Settings update methods - order matches Settings Drawer UI
    updateLinkSettings,
    updateCableTestSettings,
    updateDisplayOptions,
    updateWifiSettings,
    updateDnsSettings,
    updateTestsSettings,
    updateSpeedtestSettings,
    updateIperfSettings,
    updateNetworkDiscoverySettings,
    updateSnmpSettings,
    updateVulnerabilitySettings,
    updateThresholds,
    updateAppearanceSettings,
    updateCardSettings,
    updateSettings,
    refreshSettings,

    // Interface selection helpers
    getEthernetInterface,
    getWifiInterface,
    getAllEthernetInterfaces,
    getAllWiFiInterfaces,
    setEthernetInterface,
    setWifiInterface,
    addEthernetInterface,
    addWiFiInterface,
    removeEthernetInterface,
    removeWiFiInterface,
    setActiveEthernetInterface,
    setActiveWiFiInterface,
  };

  return (
    <ProfileContext.Provider value={contextValue}>
      {children}
    </ProfileContext.Provider>
  );
}

// ============================================================================
// Hook
// ============================================================================

/**
 * Hook to access the profile context.
 * Must be used within a ProfileProvider.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useProfileContext(): ProfileContextValue {
  const context = useContext(ProfileContext);
  if (context === undefined) {
    throw new Error("useProfileContext must be used within a ProfileProvider");
  }
  return context;
}

// Note: ProfileContext is intentionally not exported to keep fast-refresh happy.
