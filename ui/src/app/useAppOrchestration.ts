/**
 * useAppOrchestration — the application's data/interface/runtime wiring.
 *
 * Extracted from the former 814-line `App` god component (B1). It owns every
 * non-gating concern: theme, capabilities, settings/profile context, drawer
 * state, network interface + card + fetcher wiring, SSE + polling, channel
 * graph, device scan, and the run-all-tests orchestration. `App` calls this
 * hook UNCONDITIONALLY (regardless of auth) so hook/effect timing and global
 * side effects (theme on the login screen, logger auth toggle) are byte-for-byte
 * identical to before — this is a pure structural split, not a behaviour change.
 *
 * The authenticated UI is rendered by <AppShell>, which consumes this hook's
 * return value. Gating (setup wizard / loading / login) stays in `App`.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api } from '../api';
import type { NetworkDiscoveryData } from '../components/cards/NetworkDiscoveryCard';
import { useProfileContext } from '../contexts/profileContext';
import { useSettings } from '../contexts/useSettings';
import { useAppDrawers } from '../hooks/useAppDrawers';
import { useCapabilities } from '../hooks/useCapabilities';
import { useCardState } from '../hooks/useCardState';
import { useChannelGraph } from '../hooks/useChannelGraph';
import { useDeviceScan } from '../hooks/useDeviceScan';
import { useInterfaceState } from '../hooks/useInterfaceState';
import { useNetworkFetchers } from '../hooks/useNetworkFetchers';
import { useSse } from '../hooks/useSse';
import { useSsePolling } from '../hooks/useSsePolling';
import { useTheme } from '../hooks/useTheme';
import { LogComponents, logger } from '../lib/logger';
import { useTestRunSignal, useTestRunStore } from '../stores/testRunStore';
import {
  applyInterfaceRestoration,
  findBestInterface,
  type ProfileInterfacesConfig,
  parseProfileInterfaces,
} from './appInterfaceHelpers';

interface UseAppOrchestrationArgs {
  isAuthenticated: boolean;
}

export function useAppOrchestration({ isAuthenticated }: UseAppOrchestrationArgs) {
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

  // Command palette open state (Cmd+K / Ctrl+K toggles).
  const [paletteOpen, setPaletteOpen] = useState(false);

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
  const [appVersion, setAppVersion] = useState('dev');
  // #756: Auto-detected recommended interfaces (most capable)
  const [recommendedEthernet, setRecommendedEthernet] = useState<string | undefined>();
  const [recommendedWifi, setRecommendedWifi] = useState<string | undefined>();

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
      logger.info(LogComponents.CONFIG, 'Profile changed, refreshing settings', {
        from: prevActiveProfileRef.current,
        to: currentProfileId,
      });
      refreshSettings().catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to refresh settings', { error: err });
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

  // Trigger network device scan (hook owns poll/timeout refs and cleanup)
  const triggerDeviceScan = useDeviceScan({
    fetchNetworkDiscovery,
    setNetworkDiscovery,
  });

  // Change interface on backend
  const changeInterface = useCallback(
    async (interfaceName: string) => {
      try {
        // Use api.put() which handles CSRF tokens automatically
        const data = await api.put<{ isWireless?: boolean }>('/api/v1/interface', {
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
        logger.error(LogComponents.NETWORK, 'Failed to change interface', err);
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
    async (type: 'ethernet' | 'wifi') => {
      // Mark that user explicitly selected this mode - prevents API responses from flipping back
      userSetWifiModeRef.current = true;
      setActiveMode(type);

      // Check if we already have a stored interface for this mode
      const storedInterface = type === 'wifi' ? wifiInterface : ethernetInterface;
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
      const setInterfaceState = type === 'wifi' ? setWifiInterfaceState : setEthernetInterfaceState;
      setInterfaceState(target.name);
      await changeInterface(target.name);
      // Persist interface selection - use Promise.resolve to satisfy linter
      if (type === 'wifi') {
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
      logger.info(LogComponents.CONFIG, 'Restoring ethernet interface from profile', {
        interface: restoration.savedEthernetName,
      });
    }
    if (restoration.restoredWifi) {
      logger.info(LogComponents.CONFIG, 'Restoring WiFi interface from profile', {
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

  // React to a run start (FAB click or link-up auto-run) via the testRunStore.
  // This replaces the former `window` runAllTests/cardTestComplete/testsComplete
  // event bus (seed#1568, SEED_UI_ARCH_PLAN.md A2/H2). The handler closure reads
  // the latest render scope each time (useTestRunSignal captures it by ref), so
  // no dependency array is needed.
  useTestRunSignal((runId: number): void => {
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
          logger.error(LogComponents.NETWORK, 'Failed to trigger device scan', { error: err });
        });
      }

      // Wait for all fetches to complete
      // Note: runSpeedtest/runIperf and runHealthChecks are handled by their
      // respective card components reacting to the same testRunStore start signal.
      await Promise.all(fetchPromises);

      // Determine how many card-managed tests we need to wait for
      const cardTestsToWait: string[] = [];
      if (runOpts.runPerformance && runOpts.runSpeedtest) {
        cardTestsToWait.push('speedtest');
      }
      if (runOpts.runPerformance && runOpts.runIperf) {
        cardTestsToWait.push('iperf');
      }
      if (runOpts.runHealthChecks) {
        cardTestsToWait.push('healthchecks');
      }

      // Declare the card-managed tests this run must await. Completion
      // accounting lives in the store: each card calls reportComplete() and the
      // last one settles the run to idle — no listener-lifecycle race. An empty
      // set settles the run immediately.
      useTestRunStore.getState().awaitTests(cardTestsToWait);
      if (cardTestsToWait.length === 0) {
        return;
      }

      // Failsafe (90s) in case a card never reports completion. The run is then
      // PARTIAL — some checks did not finish — surfaced as such, never as a clean
      // completion. Presenting partial results as final was the C2 defect.
      // Scoped to this run id so a stale timeout cannot clobber a later run.
      setTimeout(() => {
        const { pending, expected } = useTestRunStore.getState();
        if (pending.length > 0) {
          logger.warn(LogComponents.UI, 'run timeout: not all card tests completed', {
            completed: expected.length - pending.length,
            expected: expected.length,
          });
          useTestRunStore.getState().settlePartial(runId);
        }
      }, 90000);
    };
    handleRunAllTests().catch((err: unknown) => {
      logger.error(LogComponents.UI, 'run-all-tests orchestration failed', { error: err });
    });
  });

  // SSE connection for real-time updates (simpler than WebSocket)
  const { status: sseStatus, reconnect } = useSse({
    url: '/api/events',
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
        logger.error(LogComponents.NETWORK, 'Failed to fetch channel graph data', { error: err });
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
          logger.error(LogComponents.NETWORK, 'Failed to trigger device scan', { error: err });
        });
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [isAuthenticated, triggerDeviceScan, runOpts.runNetworkDiscovery]);

  return {
    // theme + capabilities
    isDark,
    toggleTheme,
    capabilities,
    // settings
    cardSettings,
    displayOptions,
    // profiles
    profiles,
    activeProfile,
    profilesLoading,
    switchProfile,
    // drawers + palette
    profilesOpen,
    settingsOpen,
    helpOpen,
    openProfiles,
    closeProfiles,
    openSettings,
    closeSettings,
    openHelp,
    closeHelp,
    paletteOpen,
    setPaletteOpen,
    // network state
    interfaces,
    networkDiscovery,
    appVersion,
    recommendedEthernet,
    recommendedWifi,
    // interface selection
    currentInterface,
    isWifi,
    hasEthernet,
    hasWifiInterface,
    changeInterface,
    switchToInterfaceType,
    // cards
    cards,
    loading,
    registerTraceHopHandler,
    // device scan
    triggerDeviceScan,
    // channel graph
    channelGraphData,
    channelGraphLoading,
    // sse
    sseStatus,
    reconnect,
  };
}

export type AppOrchestration = ReturnType<typeof useAppOrchestration>;
