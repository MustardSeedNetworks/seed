/**
 * Main Application Component
 *
 * The root component for The Seed network monitoring application by Mustard Seed Networks.
 *
 * Responsibilities:
 * - Authentication management and session handling
 * - SSE (Server-Sent Events) connection for real-time data updates
 * - Network interface monitoring and status tracking
 * - Card-based dashboard state management
 * - User settings and theme management
 * - Setup wizard for first-time configuration
 * - Floating Action Button (FAB) for quick actions
 *
 * Architecture:
 * - Uses SSE for real-time updates from backend (simpler than WebSocket)
 * - Card-based UI with independent data components
 * - Persistent settings stored in localStorage via SettingsContext
 * - JWT authentication with automatic session expiration
 *
 * State Management:
 * - Local state for cards, interface selection, and UI
 * - Context-based settings (SettingsContext)
 * - Custom hooks for auth, SSE, and theme
 *
 * The component supports both initial setup flow and normal operation,
 * automatically detecting if the system needs configuration.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { LoginForm } from "./app/login-form";
import { HeaderBar } from "./components/app/header-bar";
import { CableCard } from "./components/cards/cable-card";
import { DnsCard } from "./components/cards/dns-card";
import { GatewayCard } from "./components/cards/gateway-card";
import { HealthCheckCard } from "./components/cards/health-check-card";
import { LinkCard } from "./components/cards/link-card";
import { LogViewerCard } from "./components/cards/log-viewer-card";
import { NetworkCard } from "./components/cards/network-card";
import {
  NetworkDiscoveryCard,
  type NetworkDiscoveryData,
} from "./components/cards/network-discovery-card";
import { PathDiscoveryCard } from "./components/cards/path-discovery-card";
import { PerformanceCard } from "./components/cards/performance-card";
import { PublicIpCard } from "./components/cards/public-ip-card";
import { SwitchCard } from "./components/cards/switch-card";
import { SystemHealthCard } from "./components/cards/system-health-card";
import { WiFiCard } from "./components/cards/wifi-card";
import { WifiChannelGraph } from "./components/cards/wifi-channel-graph";
import { WiFiSurveyCard } from "./components/cards/wifi-survey-card";
import { ImprovedHelpModal } from "./components/help/improved-help-modal";
import { ProfileManagement } from "./components/profiles/profile-management";
import { SettingsDrawer } from "./components/settings/settings-drawer";
import { SetupWizard } from "./components/setup/setup-wizard";
import { checkSetupStatus } from "./components/setup/setupApi";
import { FAB } from "./components/ui/fab";
import { useProfileContext } from "./contexts/profile-context";
import { useSettings } from "./contexts/useSettings";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { useAuth } from "./hooks/useAuth";
import { useCardState } from "./hooks/useCardState";
import { useInterfaceState } from "./hooks/useInterfaceState";
import { useNetworkFetchers } from "./hooks/useNetworkFetchers";
import { useSse } from "./hooks/useSse";
import { useTheme } from "./hooks/useTheme";
import { api, setSessionExpiredCallback } from "./lib/api";
import { LogComponents, logger } from "./lib/logger";
import { cn, layout, radius, section, spacing } from "./styles/theme";

type ChannelGraphNetwork = {
  ssid: string;
  bssid: string;
  channel: number;
  centerFreq: number;
  channelWidth: number;
  signal: number;
  band: string;
  isConnected: boolean;
};

type ChannelGraphData = {
  networks24Ghz: ChannelGraphNetwork[];
  networks5Ghz: ChannelGraphNetwork[];
  networks6Ghz: ChannelGraphNetwork[];
  connectedBssid?: string;
  scanTime: string;
};

type ChannelGraphResponse = {
  available: boolean;
  error?: string;
  data?: ChannelGraphData;
};

type ChannelGraphApiResponse = {
  available: boolean;
  error?: string;
  data?: Record<string, unknown>;
};

const normalizeChannelGraphResponse = (response: ChannelGraphApiResponse): ChannelGraphResponse => {
  if (!response.data) {
    return response;
  }

  const data = response.data as Record<string, unknown>;
  const asNetworkArray = (value: unknown): ChannelGraphNetwork[] =>
    Array.isArray(value) ? (value as ChannelGraphNetwork[]) : [];

  return {
    ...response,
    data: {
      networks24Ghz: asNetworkArray(data.networks_2_4ghz),
      networks5Ghz: asNetworkArray(data.networks_5ghz),
      networks6Ghz: asNetworkArray(data.networks_6ghz),
      connectedBssid: typeof data.connected_bssid === "string" ? data.connected_bssid : undefined,
      scanTime: typeof data.scan_time === "string" ? data.scan_time : "",
    },
  };
};

/**
 * Main App Component
 *
 * Orchestrates the entire application, managing authentication,
 * real-time data updates, and the dashboard interface.
 */
function App() {
  const { t } = useTranslation("common");
  const { isAuthenticated, login, logout, isLoading, error } = useAuth();
  const { isDark, toggleTheme } = useTheme();

  // Sync logger auth state to prevent 401 spam on login screen
  useEffect(() => {
    logger.setAuthenticated(isAuthenticated);
  }, [isAuthenticated]);
  // Use settings from context instead of local state
  const { cardSettings, displayOptions, refreshSettings } = useSettings();
  // Profile management (#754)
  const {
    profiles,
    activeProfile,
    isLoading: profilesLoading,
    switchProfile,
    setEthernetInterface,
    setWifiInterface,
  } = useProfileContext();
  const [profilesOpen, setProfilesOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [suggestedPassword, setSuggestedPassword] = useState<string | undefined>(undefined);
  const [setupUsername, setSetupUsername] = useState<string | undefined>(undefined);
  // Security fix #724, #758: Store setup token for secure setup completion
  const [setupToken, setSetupToken] = useState<string | undefined>(undefined);

  // Network state
  const [interfaces, setInterfaces] = useState<
    Array<{
      name: string;
      friendlyName?: string;
      description?: string;
      type: string;
      up: boolean;
      speedDisplay?: string;
      chipsetVendor?: string;
      chipsetModel?: string;
      hasTdr?: boolean;
      hasDom?: boolean;
      score?: number;
    }>
  >([]);
  const [networkDiscovery, setNetworkDiscovery] = useState<NetworkDiscoveryData | null>(null);
  const [appVersion, setAppVersion] = useState("dev");
  // WiFi channel graph data
  const [channelGraphData, setChannelGraphData] = useState<ChannelGraphResponse | null>(null);
  const [channelGraphLoading, setChannelGraphLoading] = useState(false);

  // Refs to track device scan polling interval and timeout for cleanup
  const scanPollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const scanTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryAbortRef = useRef<AbortController | null>(null);

  // Check if setup is needed on mount
  useEffect(() => {
    checkSetupStatus().then((status) => {
      setNeedsSetup(status.needsSetup);
      if (status.suggestedPassword) {
        setSuggestedPassword(status.suggestedPassword);
      }
      if (status.username) {
        setSetupUsername(status.username); // Fixes #768 - use username from config
      }
      // Security fix #724, #758: Capture setup token for setup completion
      if (status.setupToken) {
        setSetupToken(status.setupToken);
      }
    });
  }, []);

  // Refresh settings when profile changes (fixes #781)
  const prevActiveProfileRef = useRef<string | null>(null);
  useEffect(() => {
    const currentProfileId = activeProfile?.id ?? null;
    // Skip initial render and only refresh when profile actually changes
    if (
      prevActiveProfileRef.current !== null &&
      prevActiveProfileRef.current !== currentProfileId
    ) {
      logger.info(LogComponents.Config, "Profile changed, refreshing settings", {
        from: prevActiveProfileRef.current,
        to: currentProfileId,
      });
      refreshSettings();
    }
    prevActiveProfileRef.current = currentProfileId;
  }, [activeProfile?.id, refreshSettings]);

  // Initialize interface state hook (provides interface switching logic)
  const {
    currentInterface,
    isWifi,
    setCurrentInterface,
    setIsWifi,
    userSetWifiModeRef,
    currentInterfaceRef,
    hasEthernet,
    hasWifiInterface,
    setEthernetInterfaceState,
    setWifiInterfaceState,
    setActiveMode,
    ethernetInterface,
    wifiInterface,
  } = useInterfaceState({
    interfaces,
    activeProfile,
    setEthernetInterface,
    setWifiInterface,
  });

  // Initialize card state hook
  const {
    cards,
    loading,
    setCards,
    setLoading,
    handleMessage,
    handleCardUpdate,
    prevLinkUpRef,
    registerTraceHopHandler,
  } = useCardState({
    setCurrentInterface,
    setIsWifi,
    userSetWifiModeRef,
  });

  // Initialize network fetchers hook
  const {
    fetchLinkData,
    fetchIpConfig,
    fetchInterfaces,
    fetchVersion,
    fetchDiscoveryData,
    fetchDnsData,
    fetchVlanData,
    fetchGatewayData,
    fetchWifiData,
    fetchCableData,
    fetchPublicIp,
    fetchNetworkDiscovery,
  } = useNetworkFetchers({
    currentInterfaceRef,
    setCards,
    setCurrentInterface,
    setInterfaces,
    setAppVersion,
    setNetworkDiscovery,
    setIsWifi,
    userSetWifiModeRef,
    networkDiscoveryAbortRef,
    prevLinkUpRef,
  });

  // Cleanup network discovery on unmount
  useEffect(() => {
    return () => {
      networkDiscoveryAbortRef.current?.abort();
    };
  }, []);

  // Handle session expiration via API client callback
  useEffect(() => {
    setSessionExpiredCallback(() => {
      setSessionExpired(true);
      logout();
    });
    return () => {
      setSessionExpiredCallback(null);
    };
  }, [logout]);

  // Trigger network device scan
  const triggerDeviceScan = useCallback(async () => {
    try {
      // Clear any existing polling interval/timeout
      if (scanPollIntervalRef.current) {
        clearInterval(scanPollIntervalRef.current);
        scanPollIntervalRef.current = null;
      }
      if (scanTimeoutRef.current) {
        clearTimeout(scanTimeoutRef.current);
        scanTimeoutRef.current = null;
      }

      // Update status to show scanning
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: true },
            }
          : null,
      );

      await api.post("/api/shell/devices/scan");

      // Poll for completion
      scanPollIntervalRef.current = setInterval(async () => {
        try {
          const status = await api.get<{ scanning: boolean }>("/api/shell/devices/status");
          if (!status.scanning) {
            if (scanPollIntervalRef.current) {
              clearInterval(scanPollIntervalRef.current);
              scanPollIntervalRef.current = null;
            }
            fetchNetworkDiscovery();
          }
        } catch {
          // Status check failed, keep polling
        }
      }, 1000);

      // Safety timeout - stop polling after 60 seconds
      scanTimeoutRef.current = setTimeout(() => {
        if (scanPollIntervalRef.current) {
          clearInterval(scanPollIntervalRef.current);
          scanPollIntervalRef.current = null;
        }
      }, 60000);
    } catch (err) {
      logger.error(LogComponents.Devices, "Failed to trigger device scan", err);
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: false },
            }
          : null,
      );
    }
  }, [fetchNetworkDiscovery]);

  // Change interface on backend
  const changeInterface = useCallback(
    async (interfaceName: string) => {
      try {
        // Use api.put() which handles CSRF tokens automatically
        const data = await api.put<{ isWireless?: boolean }>("/api/interface", {
          interface: interfaceName,
        });
        if (data) {
          setCurrentInterface(interfaceName);
          // Update ref immediately so fetch functions use the new interface (#754)
          // React state updates are async, but fetch functions read from ref synchronously
          currentInterfaceRef.current = interfaceName;
          // Only auto-set WiFi mode if user hasn't manually selected via Ethernet/WiFi buttons
          if (!userSetWifiModeRef.current) {
            setIsWifi(data.isWireless === true);
          }
          // Refresh data for new interface
          fetchLinkData();
          fetchIpConfig();
          fetchDiscoveryData();
          fetchDnsData();
          fetchGatewayData();
          fetchVlanData();
          fetchWifiData();
          fetchCableData();
        }
      } catch (err) {
        logger.error(LogComponents.Network, "Failed to change interface", err);
      }
    },
    [
      fetchLinkData,
      fetchIpConfig,
      fetchDiscoveryData,
      fetchDnsData,
      fetchGatewayData,
      fetchVlanData,
      fetchWifiData,
      fetchCableData,
      setCurrentInterface,
      setIsWifi,
      userSetWifiModeRef,
      currentInterfaceRef,
    ],
  );

  // Fast switching between Ethernet/Wi-Fi views
  const switchToInterfaceType = useCallback(
    async (type: "ethernet" | "wifi") => {
      // Mark that user explicitly selected this mode - prevents API responses from flipping back
      userSetWifiModeRef.current = true;

      // Set the mode immediately for responsive UI
      setActiveMode(type);

      // Check if we already have a stored interface for this mode
      const storedInterface = type === "wifi" ? wifiInterface : ethernetInterface;
      if (storedInterface) {
        // We already have an interface stored, just switch mode
        // Backend notification happens via changeInterface
        changeInterface(storedInterface);
        return;
      }

      // No stored interface - find one from available interfaces
      const candidates = interfaces.filter((iface) => iface.type === type);
      if (candidates.length === 0) {
        // No interfaces of this type available, just show the view anyway
        // for planning/survey purposes (Fix #572 extension)
        return;
      }

      // Prefer a link-up interface, otherwise first in list
      const target = candidates.find((iface) => iface.up) ?? candidates[0];
      if (target) {
        // Update the appropriate interface state directly
        if (type === "wifi") {
          setWifiInterfaceState(target.name);
        } else {
          setEthernetInterfaceState(target.name);
        }
        changeInterface(target.name);
        // Persist the interface selection to the active profile (#754 multi-interface support)
        if (type === "wifi") {
          await setWifiInterface(target.name, true);
        } else {
          await setEthernetInterface(target.name, true);
        }
      }
    },
    [
      interfaces,
      changeInterface,
      setEthernetInterface,
      setWifiInterface,
      ethernetInterface,
      wifiInterface,
      setActiveMode,
      setEthernetInterfaceState,
      setWifiInterfaceState,
      userSetWifiModeRef,
    ],
  );

  // Load interface selections from active profile (#754 multi-interface support)
  const profileInterfaceLoadedRef = useRef<string | null>(null);
  useEffect(() => {
    // Only load once per profile change, and only if interfaces are available
    if (
      !activeProfile ||
      interfaces.length === 0 ||
      profileInterfaceLoadedRef.current === activeProfile.id
    ) {
      return;
    }

    const profileInterfaces = activeProfile.config?.interfaces;
    let restoredEthernet = false;
    let restoredWifi = false;
    let savedEthernetName = "";
    let savedWifiName = "";

    if (profileInterfaces) {
      // Load ethernet interface if saved in profile (using active_ethernet from array)
      if (profileInterfaces.active_ethernet) {
        savedEthernetName = profileInterfaces.active_ethernet;
        const exists = interfaces.some(
          (i) => i.name === savedEthernetName && i.type === "ethernet",
        );
        if (exists) {
          logger.info(LogComponents.Config, "Restoring ethernet interface from profile", {
            interface: savedEthernetName,
          });
          restoredEthernet = true;
        }
      }

      // Load wifi interface if saved in profile (using active_wifi from array)
      if (profileInterfaces.active_wifi) {
        savedWifiName = profileInterfaces.active_wifi;
        const exists = interfaces.some((i) => i.name === savedWifiName && i.type === "wifi");
        if (exists) {
          logger.info(LogComponents.Config, "Restoring WiFi interface from profile", {
            interface: savedWifiName,
          });
          restoredWifi = true;
        }
      }

      // Batch all state updates in a single setTimeout to avoid cascading renders
      setTimeout(() => {
        if (restoredEthernet && savedEthernetName) {
          setEthernetInterfaceState(savedEthernetName);
        }
        if (restoredWifi && savedWifiName) {
          setWifiInterfaceState(savedWifiName);
        }
        // Set the active interface on the backend
        if (restoredEthernet) {
          changeInterface(savedEthernetName);
          setActiveMode("ethernet");
        } else if (restoredWifi) {
          changeInterface(savedWifiName);
          setActiveMode("wifi");
        }
      }, 0);
    }
    profileInterfaceLoadedRef.current = activeProfile.id;
  }, [
    activeProfile,
    interfaces,
    changeInterface,
    setActiveMode,
    setEthernetInterfaceState,
    setWifiInterfaceState,
  ]);

  // Memoize run options to prevent unnecessary re-computation (fixes #671)
  const runOpts = useMemo(
    () => ({
      runLink: cardSettings.link.autoRunOnLink,
      runSwitch: cardSettings.switch.autoRunOnLink,
      runVlan: cardSettings.vlan.autoRunOnLink,
      runIpConfig: cardSettings.network.autoRunOnLink,
      runGateway: cardSettings.gateway.autoRunOnLink,
      runDns: cardSettings.dns.autoRunOnLink,
      runHealthChecks: cardSettings.healthChecks.autoRunOnLink,
      runPerformance: cardSettings.performance.autoRunOnLink,
      runSpeedtest:
        cardSettings.performance.autoRunOnLink && cardSettings.performance.speedtest.autoRunOnLink,
      runIperf:
        cardSettings.performance.autoRunOnLink && cardSettings.performance.iperf.autoRunOnLink,
      runNetworkDiscovery: cardSettings.networkDiscovery.autoRunOnLink,
    }),
    [cardSettings],
  );

  // Listen for FAB "run all tests" event with per-card autoRunOnLink settings
  useEffect(() => {
    // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Main test orchestration requires handling multiple card types
    const handleRunAllTests = async () => {
      // Use per-card autoRunOnLink settings to determine which tests to run

      // Build array of fetch promises based on card settings
      const fetchPromises: Promise<void>[] = [];

      if (runOpts.runLink) {
        fetchPromises.push(fetchLinkData());
        fetchPromises.push(fetchWifiData()); // WiFi is part of Link layer
        fetchPromises.push(fetchCableData()); // Cable is part of Link layer
      }
      if (runOpts.runSwitch) {
        fetchPromises.push(fetchDiscoveryData());
      }
      if (runOpts.runVlan) {
        fetchPromises.push(fetchVlanData());
      }
      if (runOpts.runIpConfig) {
        fetchPromises.push(fetchIpConfig());
      }
      if (runOpts.runGateway) {
        fetchPromises.push(fetchGatewayData());
      }
      if (runOpts.runDns) {
        fetchPromises.push(fetchDnsData());
      }

      // Trigger network discovery if enabled
      if (runOpts.runNetworkDiscovery) {
        triggerDeviceScan();
      }

      // Wait for all fetches to complete
      // Note: runSpeedtest/runIperf and runHealthChecks are handled by
      // their respective card components listening for the 'runAllTests' event
      await Promise.all(fetchPromises);

      // Determine how many card-managed tests we need to wait for
      const cardTestsToWait: string[] = [];
      if (runOpts.runPerformance && runOpts.runSpeedtest) cardTestsToWait.push("speedtest");
      if (runOpts.runPerformance && runOpts.runIperf) cardTestsToWait.push("iperf");
      if (runOpts.runHealthChecks) cardTestsToWait.push("healthchecks");

      // If no card-managed tests, signal completion immediately
      if (cardTestsToWait.length === 0) {
        window.dispatchEvent(new CustomEvent("testsComplete"));
        return;
      }

      // Wait for all card-managed tests to complete
      const completed = new Set<string>();
      const handleCardComplete = (event: CustomEvent) => {
        const testName = event.detail?.test;
        if (testName && cardTestsToWait.includes(testName)) {
          completed.add(testName);
          // Check if all expected tests are done
          if (completed.size === cardTestsToWait.length) {
            window.removeEventListener("cardTestComplete", handleCardComplete as EventListener);
            window.dispatchEvent(new CustomEvent("testsComplete"));
          }
        }
      };

      // Listen for card test completions
      window.addEventListener("cardTestComplete", handleCardComplete as EventListener);

      // Failsafe timeout (90s) in case a card doesn't report completion
      setTimeout(() => {
        window.removeEventListener("cardTestComplete", handleCardComplete as EventListener);
        if (completed.size < cardTestsToWait.length) {
          logger.warn(
            LogComponents.Ui,
            "FAB timeout: Not all card tests completed, signaling done anyway",
          );
          window.dispatchEvent(new CustomEvent("testsComplete"));
        }
      }, 90000);
    };
    window.addEventListener("runAllTests", handleRunAllTests);
    return () => {
      window.removeEventListener("runAllTests", handleRunAllTests);
    };
  }, [
    fetchLinkData,
    fetchIpConfig,
    fetchDiscoveryData,
    fetchDnsData,
    fetchGatewayData,
    fetchVlanData,
    fetchWifiData,
    fetchCableData,
    triggerDeviceScan,
    runOpts,
  ]);

  // SSE connection for real-time updates (simpler than WebSocket)
  const { status: sseStatus, reconnect } = useSse({
    url: "/api/events",
    isAuthenticated,
    onMessage: handleMessage,
    onCardUpdate: handleCardUpdate,
  });

  // Fetch channel graph data for WiFi visualization
  const fetchChannelGraphData = useCallback(async () => {
    if (!isWifi || !currentInterface) return;
    setChannelGraphLoading(true);
    try {
      const response = await api.get<ChannelGraphApiResponse>(
        `/api/canopy/wifi/channel-graph?interface=${currentInterface}`,
      );
      setChannelGraphData(normalizeChannelGraphResponse(response));
    } catch {
      setChannelGraphData({ available: false, error: "Failed to fetch channel data" });
    } finally {
      setChannelGraphLoading(false);
    }
  }, [isWifi, currentInterface]);

  // Fetch data on mount (initial load) and data not covered by WebSocket
  useEffect(() => {
    if (!isAuthenticated) return;

    // Initial fetch of all data
    setTimeout(() => {
      fetchLinkData();
      fetchIpConfig();
      fetchInterfaces();
      fetchVersion();
      fetchDiscoveryData();
      fetchDnsData();
      fetchGatewayData();
      fetchVlanData();
      fetchWifiData();
      fetchCableData();
      fetchPublicIp();
      fetchNetworkDiscovery();
      fetchChannelGraphData();
      setLoading(false);
    }, 0);
  }, [
    isAuthenticated,
    fetchLinkData,
    fetchIpConfig,
    fetchInterfaces,
    fetchVersion,
    fetchDiscoveryData,
    fetchDnsData,
    fetchGatewayData,
    fetchVlanData,
    fetchWifiData,
    fetchCableData,
    fetchPublicIp,
    fetchNetworkDiscovery,
    fetchChannelGraphData,
    setLoading,
  ]);

  // Fallback REST polling when WebSocket is not connected (fixes #672)
  // When WS is connected, backend pushes updates every 5 seconds via card_update messages
  useEffect(() => {
    if (!isAuthenticated) return;

    // Only poll if WebSocket is not connected
    if (sseStatus === "connected") {
      // WebSocket provides real-time updates, no need for aggressive polling
      // Still poll some endpoints that aren't broadcast (interfaces, wifi details)
      const slowInterval = setInterval(() => {
        fetchInterfaces();
        fetchWifiData(); // WiFi details not broadcast via WS
        fetchCableData(); // Cable test not broadcast via WS
        fetchChannelGraphData(); // Channel graph data for WiFi visualization
      }, 60000); // 60 second interval for non-WS data (increased from 30s)

      return () => clearInterval(slowInterval);
    }

    // Fallback: Poll when WebSocket disconnected with exponential backoff
    let attempts = 0;
    const maxAttempts = 5;

    const scheduleNextPoll = () => {
      // Exponential backoff: 15s, 30s, 60s, 120s, 240s (capped)
      const baseDelay = 15000;
      const delay = Math.min(baseDelay * 2 ** attempts, 240000);

      return setTimeout(() => {
        fetchLinkData();
        fetchIpConfig();
        fetchDiscoveryData();
        fetchDnsData();
        fetchGatewayData();
        fetchVlanData();
        fetchWifiData();

        // Increase attempts up to max, then reset for continuous polling
        attempts = (attempts + 1) % (maxAttempts + 1);
      }, delay);
    };

    const timeoutId = scheduleNextPoll();
    const interval = setInterval(() => {
      scheduleNextPoll();
    }, 240000); // Maximum interval of 4 minutes

    return () => {
      clearTimeout(timeoutId);
      clearInterval(interval);
    };
  }, [
    isAuthenticated,
    sseStatus,
    fetchLinkData,
    fetchIpConfig,
    fetchInterfaces,
    fetchDiscoveryData,
    fetchDnsData,
    fetchGatewayData,
    fetchVlanData,
    fetchWifiData,
    fetchCableData,
    fetchChannelGraphData,
  ]);

  // Auto-scan network devices on mount (respects per-card autoRunOnLink setting)
  useEffect(() => {
    if (!isAuthenticated) return;

    const shouldAutoScan = runOpts.runNetworkDiscovery;

    if (shouldAutoScan) {
      // Small delay to let other data load first
      const timer = setTimeout(() => {
        triggerDeviceScan();
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [isAuthenticated, triggerDeviceScan, runOpts.runNetworkDiscovery]);

  // Cleanup device scan polling on unmount
  useEffect(() => {
    return () => {
      if (scanPollIntervalRef.current) {
        clearInterval(scanPollIntervalRef.current);
      }
      if (scanTimeoutRef.current) {
        clearTimeout(scanTimeoutRef.current);
      }
    };
  }, []);

  // Login form
  const authError = sessionExpired ? "Session expired. Please log in again." : error;

  const handleLogin = useCallback(
    async (username: string, password: string) => {
      const success = await login(username, password);
      if (success) {
        setSessionExpired(false);
      }
      return success;
    },
    [login],
  );

  // Show setup wizard if needed (before auth check)
  if (needsSetup === true) {
    return (
      <SetupWizard
        onComplete={() => setNeedsSetup(false)}
        onLogin={login}
        suggestedPassword={suggestedPassword}
        username={setupUsername}
        setupToken={setupToken} // Security fix #724, #758
      />
    );
  }

  // Show loading while checking setup status
  if (needsSetup === null) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-text-muted">{t("status.loading")}</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginForm onLogin={handleLogin} isLoading={isLoading} error={authError} />;
  }

  return (
    <div className="min-h-screen text-text-primary font-body">
      <HeaderBar
        wsStatus={sseStatus}
        onReconnect={reconnect}
        profiles={profiles}
        activeProfile={activeProfile}
        profilesLoading={profilesLoading}
        onProfileSwitch={switchProfile}
        onProfileManage={() => setProfilesOpen(true)}
        interfaces={interfaces}
        currentInterface={currentInterface}
        isWifi={isWifi}
        onInterfaceChange={changeInterface}
        hasEthernet={hasEthernet}
        hasWifiInterface={hasWifiInterface}
        switchToInterfaceType={switchToInterfaceType}
        toggleTheme={toggleTheme}
        isDark={isDark}
        onHelpOpen={() => setHelpOpen(true)}
        onSettingsOpen={() => setSettingsOpen(true)}
        logout={logout}
      />

      {/* Main content - pb-24 adds bottom padding for fixed FAB */}
      <main className={cn(spacing.mainPadding.y, "pb-24")}>
        <div className={cn(section.width.xl, "mx-auto", spacing.mainPadding.x)}>
          {/* Section: Primary Connectivity - cards differ by interface type */}
          <section aria-labelledby="connectivity-heading" className={spacing.margin.bottom.section}>
            <h2
              id="connectivity-heading"
              className={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.connectivity")}
            </h2>
            <div className={layout.grid.cards}>
              {/* WiFi-only cards */}
              {isWifi && <WiFiCard data={cards.wifi} loading={loading} visible={true} />}

              {/* Ethernet-only cards */}
              {!isWifi && (
                <>
                  <LinkCard data={cards.link} loading={loading} />
                  {cards.cable?.supported && (
                    <CableCard
                      data={cards.cable}
                      loading={loading}
                      unitSystem={displayOptions.unitSystem}
                    />
                  )}
                  <SwitchCard data={cards.switch} vlanData={cards.vlan} loading={loading} />
                </>
              )}
            </div>
          </section>

          {/* Section: Network Services */}
          <section aria-labelledby="network-heading" className={spacing.margin.bottom.section}>
            <h2 id="network-heading" className={cn("section-title", spacing.margin.bottom.heading)}>
              {t("sections.network")}
            </h2>
            <div className={layout.grid.cards}>
              {/* Network info cards - hide when in WiFi mode without WiFi connection */}
              {/* Prevents showing wired interface data when user selected WiFi mode */}
              {(!isWifi || cards.wifi) && (
                <>
                  <NetworkCard
                    data={cards.dhcp}
                    publicip={cards.publicip}
                    loading={loading}
                    showPublicIp={displayOptions.showPublicIp}
                  />
                  <GatewayCard data={cards.gateway} loading={loading} />
                  <DnsCard data={cards.dns} loading={loading} />
                  {/* Public IP Card - shows geolocation, ISP/ASN, and IP history */}
                  <PublicIpCard data={cards.publicip} loading={loading} />
                </>
              )}
            </div>
          </section>

          {/* Section: Testing & Discovery - cards differ by interface type */}
          <section aria-labelledby="performance-heading" className={spacing.margin.bottom.section}>
            <h2
              id="performance-heading"
              className={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.testingDiscovery")}
            </h2>
            <div className={layout.grid.cards}>
              {/* Test cards - only show when connected to the selected interface type */}
              {/* Fix: Don't show test results from wired when in WiFi mode but disconnected */}
              {(!isWifi || cards.wifi) && (
                <>
                  <HealthCheckCard loading={loading} />
                  {cardSettings.performance.enabled && (
                    <PerformanceCard
                      loading={loading}
                      runSpeedtestEnabled={
                        cardSettings.performance.speedtest.enabled &&
                        cardSettings.performance.speedtest.autoRunOnLink
                      }
                      runIperfEnabled={
                        cardSettings.performance.iperf.enabled &&
                        cardSettings.performance.iperf.autoRunOnLink
                      }
                    />
                  )}
                </>
              )}

              {/* Ethernet-only: Network Discovery (ARP/LLDP/SNMP) */}
              {!isWifi && cardSettings.networkDiscovery.enabled && (
                <NetworkDiscoveryCard
                  data={networkDiscovery}
                  loading={loading}
                  onScan={triggerDeviceScan}
                />
              )}

              {/* Path Discovery - only show when connected */}
              {(!isWifi || cards.wifi) && (
                <PathDiscoveryCard
                  gateway={cards.gateway?.gateway}
                  dnsServer={cards.dns?.servers?.[0]?.address}
                  onRegisterTraceHandler={registerTraceHopHandler}
                />
              )}

              {/* WiFi-only: WiFi Survey for heatmaps and site surveys */}
              {/* Fix #572: Pass current interface to avoid hardcoded "wlan0" */}
              {isWifi && <WiFiSurveyCard isWifi={isWifi} currentInterface={currentInterface} />}

              {/* WiFi-only: Channel Graph for visualizing channel overlap */}
              {isWifi && (
                <WifiChannelGraph
                  data={channelGraphData}
                  loading={channelGraphLoading}
                  visible={isWifi}
                />
              )}
            </div>
          </section>

          {/* Section: System */}
          <section aria-labelledby="system-heading" className={spacing.margin.bottom.section}>
            <h2 id="system-heading" className={cn("section-title", spacing.margin.bottom.heading)}>
              {t("sections.system")}
            </h2>
            <div className={layout.grid.cards}>
              <SystemHealthCard />
              <LogViewerCard maxHeight="400px" />
            </div>
          </section>

          {/* Footer */}
          <footer
            className={cn(
              spacing.margin.top.section,
              radius.lg,
              "border border-surface-border bg-surface-raised",
              spacing.pad.lg,
            )}
          >
            <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
              {/* Product Info */}
              <div>
                <h3 className="heading-4 text-text-primary mb-2">{t("app.title")}</h3>
                <p className="body-small text-text-muted mb-1">
                  {t("footer.byCompany", "by Mustard Seed Networks")}
                </p>
                <p className="caption text-text-muted">
                  {t("footer.version", "Version")} {appVersion}
                </p>
              </div>

              {/* Contact */}
              <div>
                <h4 className="body-small font-medium text-text-primary mb-2">
                  {t("footer.contact", "Contact")}
                </h4>
                <div className="space-y-1">
                  <a
                    href="mailto:support@mustardseednetworks.com"
                    className="body-small text-brand-primary hover:underline block"
                  >
                    support@mustardseednetworks.com
                  </a>
                  <a
                    href="tel:+17194403079"
                    className="body-small text-text-muted hover:text-text-primary block"
                  >
                    719.440.3079
                  </a>
                </div>
              </div>

              {/* Website */}
              <div>
                <h4 className="body-small font-medium text-text-primary mb-2">
                  {t("footer.website", "Website")}
                </h4>
                <a
                  href="https://www.mustardseednetworks.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="body-small text-brand-primary hover:underline"
                >
                  www.mustardseednetworks.com
                </a>
              </div>

              {/* Legal */}
              <div>
                <h4 className="body-small font-medium text-text-primary mb-2">
                  {t("footer.legal", "Legal")}
                </h4>
                <div className="flex flex-wrap gap-x-3 gap-y-1">
                  <a href="/terms" className="body-small text-text-muted hover:text-brand-primary">
                    {t("footer.tos", "Terms of Service")}
                  </a>
                  <a
                    href="/privacy"
                    className="body-small text-text-muted hover:text-brand-primary"
                  >
                    {t("footer.privacy", "Privacy")}
                  </a>
                  <a
                    href="/license"
                    className="body-small text-text-muted hover:text-brand-primary"
                  >
                    {t("footer.license", "License")}
                  </a>
                </div>
              </div>
            </div>

            {/* Copyright */}
            <div className="mt-6 pt-4 border-t border-surface-border text-center">
              <p className="caption text-text-muted">
                &copy; {new Date().getFullYear()}{" "}
                {t("footer.copyright", "Mustard Seed Networks. All rights reserved.")}
              </p>
            </div>
          </footer>
        </div>
      </main>

      {/* Settings Drawer - shows interface-specific settings (#754) */}
      <SettingsDrawer
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        version={appVersion}
        isWifi={isWifi}
      />

      {/* Help Modal - improved with TOC, About, and search */}
      <ImprovedHelpModal
        isOpen={helpOpen}
        onClose={() => setHelpOpen(false)}
        version={appVersion}
      />

      {/* Profile Management Modal (#754) */}
      {profilesOpen && <ProfileManagement onClose={() => setProfilesOpen(false)} />}

      {/* FAB - Run All Tests - constrained to content width */}
      <div className="fixed bottom-0 left-0 right-0 pointer-events-none z-50">
        <div className={cn(section.width.xl, "mx-auto", spacing.mainPadding.x, "relative")}>
          <FAB className="pointer-events-auto absolute bottom-6 right-0" />
        </div>
      </div>
    </div>
  );
}

export default App;
