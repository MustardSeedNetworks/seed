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

import { LoginForm } from "./app/LoginForm";
import { CapabilityWarnings } from "./components/app/CapabilityWarnings";
import { HeaderBar } from "./components/app/HeaderBar";
import { CableCard } from "./components/cards/CableCard";
import { DnsCard } from "./components/cards/DnsCard";
import { GatewayCard } from "./components/cards/GatewayCard";
import { HealthCheckCard } from "./components/cards/HealthCheckCard";
import { LinkCard } from "./components/cards/LinkCard";
import { LogViewerCard } from "./components/cards/LogViewerCard";
import { NetworkCard } from "./components/cards/NetworkCard";
import {
  NetworkDiscoveryCard,
  type NetworkDiscoveryData,
} from "./components/cards/NetworkDiscoveryCard";
import { PathDiscoveryCard } from "./components/cards/PathDiscoveryCard";
import { PerformanceCard } from "./components/cards/PerformanceCard";
import { PublicIpCard } from "./components/cards/PublicIpCard";
import { SLADashboardCard } from "./components/cards/SlaDashboardCard";
import { SwitchCard } from "./components/cards/SwitchCard";
import { SystemHealthCard } from "./components/cards/SystemHealthCard";
import { WiFiCard } from "./components/cards/WiFiCard";
import { WifiChannelGraph } from "./components/cards/WiFiChannelGraph";
import { WiFiSurveyCard } from "./components/cards/WiFiSurveyCard";
import { ImprovedHelpModal } from "./components/help/ImprovedHelpModal";
import { ProfileManagement } from "./components/profiles/ProfileManagement";
import { SettingsDrawer } from "./components/settings/SettingsDrawer";
import { SetupWizard } from "./components/setup/SetupWizard";
import { Fab } from "./components/ui/Fab";
import { useProfileContext } from "./contexts/profile-context";
import { useSettings } from "./contexts/useSettings";
import { useAppDrawers } from "./hooks/useAppDrawers";
import { useAuth } from "./hooks/useAuth";
import { useCapabilities } from "./hooks/useCapabilities";
import { useCardState } from "./hooks/useCardState";
import { useChannelGraph } from "./hooks/useChannelGraph";
import { useInterfaceState } from "./hooks/useInterfaceState";
import { useNetworkFetchers } from "./hooks/useNetworkFetchers";
import { useSetupState } from "./hooks/useSetupState";
import { useSse } from "./hooks/useSse";
import { useSsePolling } from "./hooks/useSsePolling";
import { useTheme } from "./hooks/useTheme";
import { api, setSessionExpiredCallback } from "./lib/api";
import { LogComponents, logger } from "./lib/logger";
import { cn, layout, radius, section, spacing } from "./styles/theme";

// ============================================================================
// Helper types for interface management
// ============================================================================

interface InterfaceInfo {
  name: string;
  type: string;
  up: boolean;
}

/** Profile interface config from backend (uses snake_case) */
interface ProfileInterfacesConfig {
  // biome-ignore lint/style/useNamingConvention: Backend API uses snake_case
  active_ethernet?: string;
  // biome-ignore lint/style/useNamingConvention: Backend API uses snake_case
  active_wifi?: string;
}

interface InterfaceRestorationResult {
  restoredEthernet: boolean;
  restoredWifi: boolean;
  savedEthernetName: string;
  savedWifiName: string;
}

// ============================================================================
// Helper functions for interface management (extracted to reduce complexity)
// ============================================================================

/**
 * Find the best interface of a given type from available interfaces.
 * Prefers link-up interfaces, otherwise returns the first one.
 */
function findBestInterface(
  interfaces: InterfaceInfo[],
  type: "ethernet" | "wifi",
): InterfaceInfo | null {
  const candidates = interfaces.filter((iface) => iface.type === type);
  if (candidates.length === 0) {
    return null;
  }
  return candidates.find((iface) => iface.up) ?? candidates[0];
}

/**
 * Check if a saved interface exists in the available interfaces.
 */
function interfaceExistsWithType(interfaces: InterfaceInfo[], name: string, type: string): boolean {
  return interfaces.some((i) => i.name === name && i.type === type);
}

/**
 * Parse profile interfaces config and determine which interfaces can be restored.
 */
function parseProfileInterfaces(
  profileInterfaces: ProfileInterfacesConfig | undefined,
  interfaces: InterfaceInfo[],
): InterfaceRestorationResult {
  const result: InterfaceRestorationResult = {
    restoredEthernet: false,
    restoredWifi: false,
    savedEthernetName: "",
    savedWifiName: "",
  };

  if (!profileInterfaces) {
    return result;
  }

  // Check ethernet interface
  if (profileInterfaces.active_ethernet) {
    result.savedEthernetName = profileInterfaces.active_ethernet;
    if (interfaceExistsWithType(interfaces, result.savedEthernetName, "ethernet")) {
      result.restoredEthernet = true;
    }
  }

  // Check wifi interface
  if (profileInterfaces.active_wifi) {
    result.savedWifiName = profileInterfaces.active_wifi;
    if (interfaceExistsWithType(interfaces, result.savedWifiName, "wifi")) {
      result.restoredWifi = true;
    }
  }

  return result;
}

/**
 * Apply interface state updates for restoration.
 * Handles setting local state and notifying backend.
 */
function applyInterfaceRestoration(
  restoration: InterfaceRestorationResult,
  setEthernetInterfaceState: (name: string) => void,
  setWifiInterfaceState: (name: string) => void,
  changeInterface: (name: string) => Promise<void>,
  setActiveMode: (mode: "ethernet" | "wifi") => void,
): void {
  // Update local state
  if (restoration.restoredEthernet) {
    setEthernetInterfaceState(restoration.savedEthernetName);
  }
  if (restoration.restoredWifi) {
    setWifiInterfaceState(restoration.savedWifiName);
  }

  // Set active interface on backend (prefer ethernet if both exist)
  if (restoration.restoredEthernet) {
    changeInterface(restoration.savedEthernetName).catch((err: unknown) => {
      logger.error(LogComponents.Network, "Failed to change interface", { error: err });
    });
    setActiveMode("ethernet");
  } else if (restoration.restoredWifi) {
    changeInterface(restoration.savedWifiName).catch((err: unknown) => {
      logger.error(LogComponents.Network, "Failed to change interface", { error: err });
    });
    setActiveMode("wifi");
  }
}

/**
 * Main App Component
 *
 * Orchestrates the entire application, managing authentication,
 * real-time data updates, and the dashboard interface.
 */
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Main App component with necessary orchestration logic
function App(): JSX.Element {
  const { t } = useTranslation("common");
  const { isAuthenticated, login, logout, isLoading, error } = useAuth();
  const { isDark, toggleTheme } = useTheme();
  // Issue #803: Track network capabilities for warning display
  const { capabilities } = useCapabilities();

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

  // App drawers state (extracted to hook #889)
  const {
    profilesOpen,
    settingsOpen,
    helpOpen,
    openProfiles,
    closeProfiles,
    openSettings,
    closeSettings,
    openHelp,
    closeHelp,
  } = useAppDrawers();

  const [sessionExpired, setSessionExpired] = useState(false);

  // Setup wizard state (extracted to hook #889)
  const { needsSetup, suggestedPassword, setupUsername, setupToken, completeSetup } =
    useSetupState();

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
  // #756: Auto-detected recommended interfaces (most capable)
  const [recommendedEthernet, setRecommendedEthernet] = useState<string | undefined>();
  const [recommendedWifi, setRecommendedWifi] = useState<string | undefined>();

  // Refs to track device scan polling interval and timeout for cleanup
  const scanPollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const scanTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryAbortRef = useRef<AbortController | null>(null);

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
      refreshSettings().catch((err: unknown) => {
        logger.error(LogComponents.Config, "Failed to refresh settings", { error: err });
      });
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
    // #756: Pass setters for recommended interfaces
    setRecommendedEthernet,
    setRecommendedWifi,
  });

  // Channel graph data for WiFi visualization (extracted to hook #889)
  const { channelGraphData, channelGraphLoading, fetchChannelGraphData } = useChannelGraph({
    isWifi,
    currentInterface,
  });

  // Cleanup network discovery on unmount
  useEffect(
    (): (() => void) => (): void => {
      networkDiscoveryAbortRef.current?.abort();
    },
    [],
  );

  // Handle session expiration via API client callback
  useEffect(() => {
    setSessionExpiredCallback(() => {
      setSessionExpired(true);
      logout();
    });
    return (): void => {
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

      await api.post("/api/v1/shell/devices/scan");

      // Poll for completion
      scanPollIntervalRef.current = setInterval(async () => {
        try {
          const status = await api.get<{ scanning: boolean }>("/api/v1/shell/devices/status");
          if (!status.scanning) {
            if (scanPollIntervalRef.current) {
              clearInterval(scanPollIntervalRef.current);
              scanPollIntervalRef.current = null;
            }
            await fetchNetworkDiscovery();
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
        const data = await api.put<{ isWireless?: boolean }>("/api/v1/interface", {
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
          fetchLinkData().catch((): void => {
            /* handled */
          });
          fetchIpConfig().catch((): void => {
            /* handled */
          });
          fetchDiscoveryData().catch((): void => {
            /* handled */
          });
          fetchDnsData().catch((): void => {
            /* handled */
          });
          fetchGatewayData().catch((): void => {
            /* handled */
          });
          fetchVlanData().catch((): void => {
            /* handled */
          });
          fetchWifiData().catch((): void => {
            /* handled */
          });
          fetchCableData().catch((): void => {
            /* handled */
          });
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
      setActiveMode(type);

      // Check if we already have a stored interface for this mode
      const storedInterface = type === "wifi" ? wifiInterface : ethernetInterface;
      if (storedInterface) {
        await changeInterface(storedInterface);
        return;
      }

      // No stored interface - find one from available interfaces using helper
      const target = findBestInterface(interfaces, type);
      if (!target) {
        // No interfaces of this type available, just show the view anyway
        return;
      }

      // Update state and persist selection
      const setInterfaceState = type === "wifi" ? setWifiInterfaceState : setEthernetInterfaceState;
      setInterfaceState(target.name);
      await changeInterface(target.name);
      // Persist interface selection - use Promise.resolve to satisfy linter
      if (type === "wifi") {
        await Promise.resolve(setWifiInterface(target.name, true));
      } else {
        await Promise.resolve(setEthernetInterface(target.name, true));
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

    // Use helper function to parse profile interfaces
    const profileInterfaces = activeProfile.config?.interfaces as
      | ProfileInterfacesConfig
      | undefined;
    const restoration = parseProfileInterfaces(profileInterfaces, interfaces);

    // Log restoration if applicable
    if (restoration.restoredEthernet) {
      logger.info(LogComponents.Config, "Restoring ethernet interface from profile", {
        interface: restoration.savedEthernetName,
      });
    }
    if (restoration.restoredWifi) {
      logger.info(LogComponents.Config, "Restoring WiFi interface from profile", {
        interface: restoration.savedWifiName,
      });
    }

    // Apply restoration in batched update using helper function
    if (restoration.restoredEthernet || restoration.restoredWifi) {
      setTimeout(() => {
        applyInterfaceRestoration(
          restoration,
          setEthernetInterfaceState,
          setWifiInterfaceState,
          changeInterface,
          setActiveMode,
        );
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
    const handleRunAllTests = async (): Promise<void> => {
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
        triggerDeviceScan().catch((err: unknown) => {
          logger.error(LogComponents.Network, "Failed to trigger device scan", { error: err });
        });
      }

      // Wait for all fetches to complete
      // Note: runSpeedtest/runIperf and runHealthChecks are handled by
      // their respective card components listening for the 'runAllTests' event
      await Promise.all(fetchPromises);

      // Determine how many card-managed tests we need to wait for
      const cardTestsToWait: string[] = [];
      if (runOpts.runPerformance && runOpts.runSpeedtest) {
        cardTestsToWait.push("speedtest");
      }
      if (runOpts.runPerformance && runOpts.runIperf) {
        cardTestsToWait.push("iperf");
      }
      if (runOpts.runHealthChecks) {
        cardTestsToWait.push("healthchecks");
      }

      // If no card-managed tests, signal completion immediately
      if (cardTestsToWait.length === 0) {
        window.dispatchEvent(new CustomEvent("testsComplete"));
        return;
      }

      // Wait for all card-managed tests to complete
      const completed = new Set<string>();
      const handleCardComplete = (event: CustomEvent): void => {
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

  // Fetch data on mount (initial load) and data not covered by SSE
  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    // Initial fetch of all data
    setTimeout((): void => {
      fetchLinkData().catch((): void => {
        /* handled */
      });
      fetchIpConfig().catch((): void => {
        /* handled */
      });
      fetchInterfaces().catch((): void => {
        /* handled */
      });
      fetchVersion().catch((): void => {
        /* handled */
      });
      fetchDiscoveryData().catch((): void => {
        /* handled */
      });
      fetchDnsData().catch((): void => {
        /* handled */
      });
      fetchGatewayData().catch((): void => {
        /* handled */
      });
      fetchVlanData().catch((): void => {
        /* handled */
      });
      fetchWifiData().catch((): void => {
        /* handled */
      });
      fetchCableData().catch((): void => {
        /* handled */
      });
      fetchPublicIp().catch((): void => {
        /* handled */
      });
      fetchNetworkDiscovery().catch((): void => {
        /* handled */
      });
      fetchChannelGraphData().catch((err: unknown) => {
        logger.error(LogComponents.Network, "Failed to fetch channel graph data", { error: err });
      });
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

  // SSE polling: fallback REST polling when SSE disconnected + supplementary data polling
  // Extracted to useSsePolling hook (#892) - see hook for interval details
  useSsePolling({
    isAuthenticated,
    sseStatus,
    fetchers: {
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
    },
  });

  // Auto-scan network devices on mount (respects per-card autoRunOnLink setting)
  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    const shouldAutoScan = runOpts.runNetworkDiscovery;

    if (shouldAutoScan) {
      // Small delay to let other data load first
      const timer = setTimeout(() => {
        triggerDeviceScan().catch((err: unknown) => {
          logger.error(LogComponents.Network, "Failed to trigger device scan", { error: err });
        });
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [isAuthenticated, triggerDeviceScan, runOpts.runNetworkDiscovery]);

  // Cleanup device scan polling on unmount
  useEffect(
    () => (): void => {
      if (scanPollIntervalRef.current) {
        clearInterval(scanPollIntervalRef.current);
      }
      if (scanTimeoutRef.current) {
        clearTimeout(scanTimeoutRef.current);
      }
    },
    [],
  );

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
        onComplete={completeSetup}
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
      <div class="min-h-screen flex items-center justify-center">
        <div class="text-text-muted">{t("status.loading")}</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginForm onLogin={handleLogin} isLoading={isLoading} error={authError} />;
  }

  return (
    <div class="min-h-screen text-text-primary font-body">
      <HeaderBar
        wsStatus={sseStatus}
        onReconnect={reconnect}
        profiles={profiles}
        activeProfile={activeProfile}
        profilesLoading={profilesLoading}
        onProfileSwitch={switchProfile}
        onProfileManage={openProfiles}
        interfaces={interfaces}
        currentInterface={currentInterface}
        isWifi={isWifi}
        onInterfaceChange={changeInterface}
        hasEthernet={hasEthernet}
        hasWifiInterface={hasWifiInterface}
        switchToInterfaceType={switchToInterfaceType}
        toggleTheme={toggleTheme}
        isDark={isDark}
        onHelpOpen={openHelp}
        onSettingsOpen={openSettings}
        logout={logout}
        // #756: Recommended (most capable) interfaces
        recommendedEthernet={recommendedEthernet}
        recommendedWifi={recommendedWifi}
      />

      {/* Main content - pb-24 adds bottom padding for fixed FAB */}
      <main class={cn(spacing.mainPadding.y, "pb-24")}>
        <div class={cn(section.width.xl, "mx-auto", spacing.mainPadding.x)}>
          {/* Issue #803: Show warning banner when network capabilities are missing */}
          <CapabilityWarnings capabilities={capabilities} />

          {/* Section: Primary Connectivity - cards differ by interface type */}
          <section aria-labelledby="connectivity-heading" class={spacing.margin.bottom.section}>
            <h2
              id="connectivity-heading"
              class={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.connectivity")}
            </h2>
            <div class={layout.grid.cards}>
              {/* WiFi-only cards */}
              {isWifi ? <WiFiCard data={cards.wifi} loading={loading} visible={true} /> : null}

              {/* Ethernet-only cards */}
              {!isWifi && (
                <>
                  <LinkCard data={cards.link} loading={loading} />
                  {cards.cable?.supported ? (
                    <CableCard
                      data={cards.cable}
                      loading={loading}
                      unitSystem={displayOptions.unitSystem}
                    />
                  ) : null}
                  <SwitchCard data={cards.switch} vlanData={cards.vlan} loading={loading} />
                </>
              )}
            </div>
          </section>

          {/* Section: Network Services */}
          <section aria-labelledby="network-heading" class={spacing.margin.bottom.section}>
            <h2 id="network-heading" class={cn("section-title", spacing.margin.bottom.heading)}>
              {t("sections.network")}
            </h2>
            <div class={layout.grid.cards}>
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
          <section aria-labelledby="performance-heading" class={spacing.margin.bottom.section}>
            <h2 id="performance-heading" class={cn("section-title", spacing.margin.bottom.heading)}>
              {t("sections.testingDiscovery")}
            </h2>
            <div class={layout.grid.cards}>
              {/* Test cards - only show when connected to the selected interface type */}
              {/* Fix: Don't show test results from wired when in WiFi mode but disconnected */}
              {(!isWifi || cards.wifi) && (
                <>
                  <HealthCheckCard loading={loading} />
                  {/* SLA Dashboard - aggregates health scores, SLA compliance, and alerts */}
                  <SLADashboardCard />
                  {cardSettings.performance.enabled ? (
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
                  ) : null}
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
              {isWifi ? (
                <WiFiSurveyCard isWifi={isWifi} currentInterface={currentInterface} />
              ) : null}

              {/* WiFi-only: Channel Graph for visualizing channel overlap */}
              {isWifi ? (
                <WifiChannelGraph
                  data={channelGraphData}
                  loading={channelGraphLoading}
                  visible={isWifi}
                />
              ) : null}
            </div>
          </section>

          {/* Section: System */}
          <section aria-labelledby="system-heading" class={spacing.margin.bottom.section}>
            <h2 id="system-heading" class={cn("section-title", spacing.margin.bottom.heading)}>
              {t("sections.system")}
            </h2>
            <div class={layout.grid.cards}>
              <SystemHealthCard />
              <LogViewerCard maxHeight="400px" />
            </div>
          </section>

          {/* Footer */}
          <footer
            class={cn(
              spacing.margin.top.section,
              radius.lg,
              "border border-surface-border bg-surface-raised",
              spacing.pad.lg,
            )}
          >
            <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
              {/* Product Info */}
              <div>
                <h3 class="heading-4 text-text-primary mb-2">{t("app.title")}</h3>
                <p class="body-small text-text-muted mb-1">
                  {t("footer.byCompany", "by Mustard Seed Networks")}
                </p>
                <p class="caption text-text-muted">
                  {t("footer.version", "Version")} {appVersion}
                </p>
              </div>

              {/* Contact */}
              <div>
                <h4 class="body-small font-medium text-text-primary mb-2">
                  {t("footer.contact", "Contact")}
                </h4>
                <div class="space-y-1">
                  <a
                    href="mailto:support@mustardseednetworks.com"
                    class="body-small text-brand-primary hover:underline block"
                  >
                    support@mustardseednetworks.com
                  </a>
                  <a
                    href="tel:+17194403079"
                    class="body-small text-text-muted hover:text-text-primary block"
                  >
                    719.440.3079
                  </a>
                </div>
              </div>

              {/* Website */}
              <div>
                <h4 class="body-small font-medium text-text-primary mb-2">
                  {t("footer.website", "Website")}
                </h4>
                <a
                  href="https://www.mustardseednetworks.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="body-small text-brand-primary hover:underline"
                >
                  www.mustardseednetworks.com
                </a>
              </div>

              {/* Legal */}
              <div>
                <h4 class="body-small font-medium text-text-primary mb-2">
                  {t("footer.legal", "Legal")}
                </h4>
                <div class="flex flex-wrap gap-x-3 gap-y-1">
                  <a href="/terms" class="body-small text-text-muted hover:text-brand-primary">
                    {t("footer.tos", "Terms of Service")}
                  </a>
                  <a href="/privacy" class="body-small text-text-muted hover:text-brand-primary">
                    {t("footer.privacy", "Privacy")}
                  </a>
                  <a href="/license" class="body-small text-text-muted hover:text-brand-primary">
                    {t("footer.license", "License")}
                  </a>
                </div>
              </div>
            </div>

            {/* Copyright */}
            <div class="mt-6 pt-4 border-t border-surface-border text-center">
              <p class="caption text-text-muted">
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
        onClose={closeSettings}
        version={appVersion}
        isWifi={isWifi}
      />

      {/* Help Modal - improved with TOC, About, and search */}
      <ImprovedHelpModal isOpen={helpOpen} onClose={closeHelp} version={appVersion} />

      {/* Profile Management Modal (#754) */}
      {profilesOpen ? <ProfileManagement onClose={closeProfiles} /> : null}

      {/* FAB - Run All Tests - positioned inline with card grid */}
      <div class="fixed bottom-0 left-0 right-0 pointer-events-none z-50">
        <div class={cn(section.width.xl, "mx-auto", spacing.mainPadding.x, "relative")}>
          <Fab class="pointer-events-auto absolute bottom-20 -right-2" />
        </div>
      </div>
    </div>
  );
}

export default App;
