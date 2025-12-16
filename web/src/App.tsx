/**
 * Main Application Component
 *
 * The root component for the LuminetIQ/NetScope network monitoring application.
 *
 * Responsibilities:
 * - Authentication management and session handling
 * - WebSocket connection for real-time data updates
 * - Network interface monitoring and status tracking
 * - Card-based dashboard state management
 * - User settings and theme management
 * - Setup wizard for first-time configuration
 * - Floating Action Button (FAB) for quick actions
 *
 * Architecture:
 * - Uses WebSocket for real-time updates from backend
 * - Card-based UI with independent data components
 * - Persistent settings stored in localStorage via SettingsContext
 * - JWT authentication with automatic session expiration
 *
 * State Management:
 * - Local state for cards, interface selection, and UI
 * - Context-based settings (SettingsContext)
 * - Custom hooks for auth, WebSocket, and theme
 *
 * The component supports both initial setup flow and normal operation,
 * automatically detecting if the system needs configuration.
 */

import { useCallback, useEffect, useState } from "react";
import { useWebSocket, Message, CardUpdate } from "./hooks/useWebSocket";
import { useAuth, getAuthHeaders } from "./hooks/useAuth";
import { useTheme } from "./hooks/useTheme";
import { useSettings } from "./contexts/SettingsContext";
import { SettingsDrawer } from "./components/settings/SettingsDrawer";
import { ImprovedHelpModal } from "./components/help/ImprovedHelpModal";
import { SetupWizard, checkSetupStatus } from "./components/setup/SetupWizard";

// API base URL - configurable via environment variable
const API_BASE = import.meta.env.VITE_API_BASE || "";
import {
  LinkCard,
  LinkData,
  SwitchCard,
  SwitchData,
  NetworkCard,
  DHCPData,
  DNSCard,
  type DNSData,
  GatewayCard,
  GatewayData,
  VLANData,
  WiFiCard,
  WiFiData,
  CableCard,
  CableData,
  NetworkDiscoveryCard,
  NetworkDiscoveryData,
  PublicIPData,
} from "./components/cards";
import { PerformanceCard } from "./components/cards/PerformanceCard";
import { HealthCheckCard } from "./components/cards/HealthCheckCard";
import { SystemHealthCard } from "./components/cards/SystemHealthCard";
import { WiFiSurveyCard } from "./components/cards/WiFiSurveyCard";
import { FAB } from "./components/ui/FAB";
import { radius } from "./styles/theme";

/**
 * Centralized state for all network monitoring cards.
 * Each card can be null if not yet loaded or unavailable.
 */
interface CardState {
  link: LinkData | null; // Network interface link status
  cable: CableData | null; // Ethernet cable diagnostics
  vlan: VLANData | null; // VLAN configuration and status
  switch: SwitchData | null; // Network switch information (LLDP/CDP)
  wifi: WiFiData | null; // WiFi connection and signal info
  dhcp: DHCPData | null; // DHCP configuration
  dns: DNSData | null; // DNS server and resolution info
  gateway: GatewayData | null; // Gateway reachability
  publicip: PublicIPData | null; // Public IP and location info
}

/**
 * Main App Component
 *
 * Orchestrates the entire application, managing authentication,
 * real-time data updates, and the dashboard interface.
 */
function App() {
  const { isAuthenticated, token, login, logout, isLoading, error } = useAuth();
  const { isDark, toggleTheme } = useTheme();
  // Use settings from context instead of local state
  const { fabOptions, displayOptions } = useSettings();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [suggestedPassword, setSuggestedPassword] = useState<
    string | undefined
  >(undefined);

  // Check if setup is needed on mount
  useEffect(() => {
    checkSetupStatus().then((status) => {
      setNeedsSetup(status.needsSetup);
      if (status.suggestedPassword) {
        setSuggestedPassword(status.suggestedPassword);
      }
    });
  }, []);
  const [cards, setCards] = useState<CardState>({
    link: null,
    cable: null,
    vlan: null,
    switch: null,
    wifi: null,
    dhcp: null,
    dns: null,
    gateway: null,
    publicip: null,
  });
  const [loading, setLoading] = useState(true);
  const [currentInterface, setCurrentInterface] = useState("eth0");
  const [isWifi, setIsWifi] = useState(false);
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
      hasTDR?: boolean;
      hasDOM?: boolean;
      score?: number;
    }>
  >([]);
  const [networkDiscovery, setNetworkDiscovery] =
    useState<NetworkDiscoveryData | null>(null);
  const [appVersion, setAppVersion] = useState("dev");

  const handleMessage = useCallback((message: Message) => {
    if (message.type === "initial_state") {
      setLoading(false);
      const payload = message.payload as {
        interface?: string;
        isWireless?: boolean;
        cards?: Partial<CardState>;
      };
      if (payload.interface) {
        setCurrentInterface(payload.interface);
      }
      // Use isWireless from payload if available (works for macOS and Linux)
      if (payload.isWireless !== undefined) {
        setIsWifi(payload.isWireless);
      }
      // Use initial card data from WebSocket
      if (payload.cards) {
        setCards((prev) => ({
          ...prev,
          ...Object.fromEntries(
            Object.entries(payload.cards!).filter(([, v]) => v !== null),
          ),
        }));
      }
    }
  }, []);

  // Handle session expiration via API client callback
  useEffect(() => {
    import("./lib/api").then(({ setSessionExpiredCallback }) => {
      setSessionExpiredCallback(() => {
        setSessionExpired(true);
        logout();
      });
    });
    return () => {
      import("./lib/api").then(({ setSessionExpiredCallback }) => {
        setSessionExpiredCallback(() => {});
      });
    };
  }, [logout]);

  const handleCardUpdate = useCallback((update: CardUpdate) => {
    setCards((prev) => ({
      ...prev,
      [update.cardId]: update.data,
    }));
  }, []);

  // Fetch link data (Layer 2 only)
  const fetchLinkData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/link`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          link: {
            linkUp: data.linkUp,
            carrier: data.carrier ?? data.linkUp, // Fallback for compatibility
            hasIP: data.hasIP ?? data.linkUp, // Fallback for compatibility
            speed: data.speed || "",
            duplex: data.duplex || "",
            advertisedSpeeds: data.advertisedSpeeds || [],
            mtu: data.mtu || 0,
            autoNeg: data.autoNeg,
          },
        }));
        setCurrentInterface(data.interface || "unknown");
        // isWifi is now set by fetchWiFiData which properly detects wireless interfaces
      }
    } catch (err) {
      console.error("Failed to fetch link data:", err);
    }
  }, []);

  // Fetch IP configuration (DHCP card - Layer 3)
  const fetchIPConfig = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/ipconfig`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          dhcp: {
            mac: data.mac || "",
            mode: data.mode || "auto",
            ipv4: data.ipv4 || null,
            ipv6: data.ipv6 || [],
            dns: data.dns || [],
            timing: data.timing || null,
          },
        }));
      }
    } catch (err) {
      console.error("Failed to fetch IP config:", err);
    }
  }, []);

  // Fetch interfaces
  const fetchInterfaces = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/interfaces`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setInterfaces(data);
      }
    } catch (err) {
      console.error("Failed to fetch interfaces:", err);
    }
  }, []);

  // Fetch app version from status endpoint
  const fetchVersion = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/status`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        if (data.version) {
          setAppVersion(data.version);
        }
      }
    } catch (err) {
      console.error("Failed to fetch version:", err);
    }
  }, []);

  // Fetch discovery data (LLDP/CDP/EDP neighbors)
  const fetchDiscoveryData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/discovery`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        // Use the first neighbor as the "nearest switch"
        if (data.neighbors && data.neighbors.length > 0) {
          const neighbor = data.neighbors[0];
          setCards((prev) => ({
            ...prev,
            switch: {
              protocol: neighbor.protocol as SwitchData["protocol"],
              switchName: neighbor.systemName || neighbor.chassisId || null,
              portId: neighbor.portId || null,
              portDescription: neighbor.portDescription || null,
              managementIp: neighbor.managementAddress || null,
              systemDescription: neighbor.systemDescription || null,
            },
          }));
        } else {
          setCards((prev) => ({
            ...prev,
            switch: null,
          }));
        }
      }
    } catch (err) {
      console.error("Failed to fetch discovery data:", err);
    }
  }, []);

  // Fetch DNS test data
  const fetchDNSData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/dns`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          dns: {
            server: data.server || "Unknown",
            servers: data.servers || [],
            testHostname: data.testHostname || "google.com",
            forward: data.forward
              ? {
                  result: data.forward.result,
                  time: data.forward.time || data.forward.timeMs || 0,
                  timeMs: data.forward.timeMs || data.forward.time || 0,
                  status: data.forward.status,
                  error: data.forward.error,
                  resolved: data.forward.resolved,
                }
              : null,
            forwardIpv6: data.forwardIpv6
              ? {
                  result: data.forwardIpv6.result,
                  time: data.forwardIpv6.time || data.forwardIpv6.timeMs || 0,
                  timeMs: data.forwardIpv6.timeMs || data.forwardIpv6.time || 0,
                  status: data.forwardIpv6.status,
                  error: data.forwardIpv6.error,
                  resolved: data.forwardIpv6.resolved,
                }
              : null,
            reverse: data.reverse
              ? {
                  result: data.reverse.result,
                  time: data.reverse.time || data.reverse.timeMs || 0,
                  timeMs: data.reverse.timeMs || data.reverse.time || 0,
                  status: data.reverse.status,
                  error: data.reverse.error,
                  resolved: data.reverse.resolved,
                }
              : null,
            reverseIpv6: data.reverseIpv6
              ? {
                  result: data.reverseIpv6.result,
                  time: data.reverseIpv6.time || data.reverseIpv6.timeMs || 0,
                  timeMs: data.reverseIpv6.timeMs || data.reverseIpv6.time || 0,
                  status: data.reverseIpv6.status,
                  error: data.reverseIpv6.error,
                  resolved: data.reverseIpv6.resolved,
                }
              : null,
          },
        }));
      }
    } catch (err) {
      console.error("Failed to fetch DNS data:", err);
    }
  }, []);

  // Fetch VLAN data
  const fetchVLANData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/vlan`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          vlan: {
            nativeVlan: data.nativeVlan || null,
            taggedVlans: data.taggedVlans || [],
            voiceVlan: data.voiceVlan || null,
            configured: data.configured || { enabled: false, id: 0 },
          },
        }));
      }
    } catch (err) {
      console.error("Failed to fetch VLAN data:", err);
    }
  }, []);

  // Fetch Gateway ping data
  const fetchGatewayData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/gateway`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          gateway: {
            gateway: data.gateway || "",
            reachable: data.reachable || false,
            sent: data.sent || 0,
            received: data.received || 0,
            lossPercent: data.lossPercent || 0,
            minTime: data.minTime || 0,
            maxTime: data.maxTime || 0,
            avgTime: data.avgTime || 0,
            lastTime: data.lastTime || 0,
            status: data.status || "unknown",
            ipv6: data.ipv6
              ? {
                  gateway: data.ipv6.gateway || "",
                  reachable: data.ipv6.reachable || false,
                  sent: data.ipv6.sent || 0,
                  received: data.ipv6.received || 0,
                  lossPercent: data.ipv6.lossPercent || 0,
                  minTime: data.ipv6.minTime || 0,
                  maxTime: data.ipv6.maxTime || 0,
                  avgTime: data.ipv6.avgTime || 0,
                  lastTime: data.ipv6.lastTime || 0,
                  status: data.ipv6.status || "unknown",
                }
              : undefined,
          },
        }));
      }
    } catch (err) {
      console.error("Failed to fetch Gateway data:", err);
    }
  }, []);

  // Fetch Wi-Fi data
  const fetchWiFiData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/wifi`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        // Check if this is a wireless interface with data
        if (data.ssid) {
          setCards((prev) => ({
            ...prev,
            wifi: {
              ssid: data.ssid || "",
              bssid: data.bssid || "",
              signal: data.signal || 0,
              channel: data.channel || 0,
              frequency: data.frequency || 0,
              security: data.security || "Unknown",
            },
          }));
          setIsWifi(true);
        } else {
          setCards((prev) => ({ ...prev, wifi: null }));
          setIsWifi(data.wireless === true);
        }
      }
    } catch (err) {
      console.error("Failed to fetch Wi-Fi data:", err);
    }
  }, []);

  // Fetch Cable test data
  const fetchCableData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/cable`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          cable: {
            supported: data.supported || false,
            length: data.length || null,
            status: data.status || "unknown",
            faults: data.faults || [],
          },
        }));
      }
    } catch (err) {
      console.error("Failed to fetch Cable data:", err);
    }
  }, []);

  // Fetch Public IP data
  const fetchPublicIP = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/publicip`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          publicip: {
            ipv4: data.ipv4 || undefined,
            ipv6: data.ipv6 || undefined,
            lastChecked: data.lastChecked || new Date().toISOString(),
            error: data.error || undefined,
          },
        }));
      }
    } catch (err) {
      console.error("Failed to fetch Public IP data:", err);
    }
  }, []);

  // Fetch Network Discovery data (devices and status)
  const fetchNetworkDiscovery = useCallback(async () => {
    try {
      const [devicesRes, statusRes] = await Promise.all([
        fetch(`${API_BASE}/api/devices`, { headers: getAuthHeaders() }),
        fetch(`${API_BASE}/api/devices/status`, { headers: getAuthHeaders() }),
      ]);

      if (devicesRes.ok && statusRes.ok) {
        const devicesData = await devicesRes.json();
        const status = await statusRes.json();
        // devicesData contains { devices: [...], status: {...} }
        // Extract the devices array from the response
        setNetworkDiscovery({
          devices: devicesData.devices || [],
          status: status || {
            scanning: false,
            deviceCount: 0,
            lastScan: "",
            subnet: "",
            localIP: "",
            interface: currentInterface,
          },
        });
      }
    } catch (err) {
      console.error("Failed to fetch network discovery data:", err);
    }
  }, [currentInterface]);

  // Trigger network device scan
  const triggerDeviceScan = useCallback(async () => {
    try {
      // Update status to show scanning
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: true },
            }
          : null,
      );

      const response = await fetch(`${API_BASE}/api/devices/scan`, {
        method: "POST",
        headers: getAuthHeaders(),
      });

      if (response.ok) {
        // Poll for completion
        const pollInterval = setInterval(async () => {
          const statusRes = await fetch(`${API_BASE}/api/devices/status`, {
            headers: getAuthHeaders(),
          });
          if (statusRes.ok) {
            const status = await statusRes.json();
            if (!status.scanning) {
              clearInterval(pollInterval);
              fetchNetworkDiscovery();
            }
          }
        }, 1000);

        // Safety timeout - stop polling after 60 seconds
        setTimeout(() => clearInterval(pollInterval), 60000);
      }
    } catch (err) {
      console.error("Failed to trigger device scan:", err);
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
        const response = await fetch(`${API_BASE}/api/interface`, {
          method: "PUT",
          headers: {
            ...getAuthHeaders(),
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ interface: interfaceName }),
        });
        if (response.ok) {
          const data = await response.json();
          setCurrentInterface(interfaceName);
          // Use isWireless from API response (works for macOS and Linux)
          setIsWifi(data.isWireless === true);
          // Refresh data for new interface
          fetchLinkData();
          fetchIPConfig();
          fetchDiscoveryData();
          fetchDNSData();
          fetchGatewayData();
          fetchVLANData();
          fetchWiFiData();
          fetchCableData();
        }
      } catch (err) {
        console.error("Failed to change interface:", err);
      }
    },
    [
      fetchLinkData,
      fetchIPConfig,
      fetchDiscoveryData,
      fetchDNSData,
      fetchGatewayData,
      fetchVLANData,
      fetchWiFiData,
      fetchCableData,
    ],
  );

  // Listen for FAB "run all tests" event with options
  useEffect(() => {
    const handleRunAllTests = async () => {
      // Use FAB options to determine which tests to run
      const runOpts = {
        runLink: fabOptions.runLink,
        runSwitch: fabOptions.runSwitch,
        runVLAN: fabOptions.runVLAN,
        runIPConfig: fabOptions.runIPConfig,
        runGateway: fabOptions.runGateway,
        runDNS: fabOptions.runDNS,
        runHealthChecks: fabOptions.runHealthChecks,
        runPerformance: fabOptions.runPerformance,
        runSpeedtest: fabOptions.runPerformance && fabOptions.runSpeedtest,
        runIperf: fabOptions.runPerformance && fabOptions.runIperf,
        runNetworkDiscovery: fabOptions.runNetworkDiscovery,
      };

      // Build array of fetch promises based on FAB options
      const fetchPromises: Promise<void>[] = [];

      if (runOpts.runLink) {
        fetchPromises.push(fetchLinkData());
        fetchPromises.push(fetchWiFiData()); // WiFi is part of Link layer
        fetchPromises.push(fetchCableData()); // Cable is part of Link layer
      }
      if (runOpts.runSwitch) {
        fetchPromises.push(fetchDiscoveryData());
      }
      if (runOpts.runVLAN) {
        fetchPromises.push(fetchVLANData());
      }
      if (runOpts.runIPConfig) {
        fetchPromises.push(fetchIPConfig());
      }
      if (runOpts.runGateway) {
        fetchPromises.push(fetchGatewayData());
      }
      if (runOpts.runDNS) {
        fetchPromises.push(fetchDNSData());
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
      if (runOpts.runPerformance && runOpts.runSpeedtest)
        cardTestsToWait.push("speedtest");
      if (runOpts.runPerformance && runOpts.runIperf)
        cardTestsToWait.push("iperf");
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
            window.removeEventListener(
              "cardTestComplete",
              handleCardComplete as EventListener,
            );
            window.dispatchEvent(new CustomEvent("testsComplete"));
          }
        }
      };

      // Listen for card test completions
      window.addEventListener(
        "cardTestComplete",
        handleCardComplete as EventListener,
      );

      // Failsafe timeout (90s) in case a card doesn't report completion
      setTimeout(() => {
        window.removeEventListener(
          "cardTestComplete",
          handleCardComplete as EventListener,
        );
        if (completed.size < cardTestsToWait.length) {
          console.warn(
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
    fetchIPConfig,
    fetchDiscoveryData,
    fetchDNSData,
    fetchGatewayData,
    fetchVLANData,
    fetchWiFiData,
    fetchCableData,
    triggerDeviceScan,
    fabOptions.runPerformance,
    fabOptions.runLink,
    fabOptions.runSwitch,
    fabOptions.runVLAN,
    fabOptions.runIPConfig,
    fabOptions.runGateway,
    fabOptions.runDNS,
    fabOptions.runHealthChecks,
    fabOptions.runSpeedtest,
    fabOptions.runIperf,
    fabOptions.runNetworkDiscovery,
  ]);

  // WebSocket connection for real-time updates
  const { status: wsStatus, reconnect } = useWebSocket({
    url: "/ws",
    token,
    onMessage: handleMessage,
    onCardUpdate: handleCardUpdate,
  });

  // Fetch data on mount (initial load) and data not covered by WebSocket
  useEffect(() => {
    if (!isAuthenticated) return;

    // Initial fetch of all data
    setTimeout(() => {
      fetchLinkData();
      fetchIPConfig();
      fetchInterfaces();
      fetchVersion();
      fetchDiscoveryData();
      fetchDNSData();
      fetchGatewayData();
      fetchVLANData();
      fetchWiFiData();
      fetchCableData();
      fetchPublicIP();
      fetchNetworkDiscovery();
      setLoading(false);
    }, 0);
  }, [
    isAuthenticated,
    fetchLinkData,
    fetchIPConfig,
    fetchInterfaces,
    fetchVersion,
    fetchDiscoveryData,
    fetchDNSData,
    fetchGatewayData,
    fetchVLANData,
    fetchWiFiData,
    fetchCableData,
    fetchPublicIP,
    fetchNetworkDiscovery,
  ]);

  // Fallback REST polling when WebSocket is not connected
  // When WS is connected, backend pushes updates every 5 seconds via card_update messages
  useEffect(() => {
    if (!isAuthenticated) return;

    // Only poll if WebSocket is not connected
    if (wsStatus === "connected") {
      // WebSocket provides real-time updates, no need for aggressive polling
      // Still poll some endpoints that aren't broadcast (interfaces, wifi details)
      const slowInterval = setInterval(() => {
        fetchInterfaces();
        fetchWiFiData(); // WiFi details not broadcast via WS
        fetchCableData(); // Cable test not broadcast via WS
      }, 30000); // 30 second interval for non-WS data

      return () => clearInterval(slowInterval);
    }

    // Fallback: Poll when WebSocket disconnected
    const interval = setInterval(() => {
      fetchLinkData();
      fetchIPConfig();
      fetchDiscoveryData();
      fetchDNSData();
      fetchGatewayData();
      fetchVLANData();
      fetchWiFiData();
    }, 10000); // 10 second fallback

    return () => clearInterval(interval);
  }, [
    isAuthenticated,
    wsStatus,
    fetchLinkData,
    fetchIPConfig,
    fetchInterfaces,
    fetchDiscoveryData,
    fetchDNSData,
    fetchGatewayData,
    fetchVLANData,
    fetchWiFiData,
    fetchCableData,
  ]);

  // Auto-scan network devices on mount (respects FAB option)
  useEffect(() => {
    if (!isAuthenticated) return;

    const shouldAutoScan = fabOptions.runNetworkDiscovery !== false;

    if (shouldAutoScan) {
      // Small delay to let other data load first
      const timer = setTimeout(() => {
        triggerDeviceScan();
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [isAuthenticated, triggerDeviceScan, fabOptions.runNetworkDiscovery]);

  // Login form
  const authError = sessionExpired
    ? "Session expired. Please log in again."
    : error;

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
      />
    );
  }

  // Show loading while checking setup status
  if (needsSetup === null) {
    return (
      <div className="min-h-screen bg-surface-base flex items-center justify-center">
        <div className="text-text-muted">Loading...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <LoginForm
        onLogin={handleLogin}
        isLoading={isLoading}
        error={authError}
      />
    );
  }

  return (
    <div className="min-h-screen bg-surface-base text-text-primary font-body">
      {/* Header */}
      <header className="border-b border-surface-border bg-surface-raised">
        <div className="max-w-8xl mx-auto px-4 sm:px-6 lg:px-8 py-2 sm:py-3 flex items-center justify-between gap-2">
          {/* Logo and title - hide title on very small screens */}
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-xl font-bold text-brand-primary shrink-0">
              ◉
            </span>
            <h1 className="heading-4 hidden xs:block sm:block">LuminetIQ</h1>
            <div className="hidden sm:block">
              <ConnectionStatus status={wsStatus} onReconnect={reconnect} />
            </div>
          </div>

          {/* Controls */}
          <div className="flex items-center gap-1 sm:gap-2">
            {/* Interface selector */}
            <label htmlFor="interface-select" className="sr-only">
              Select network interface
            </label>
            <select
              id="interface-select"
              className="rounded-md border border-surface-border bg-surface-base px-2 py-1.5 body-small min-w-0 max-w-25 sm:max-w-none focus:outline-none focus:ring-2 focus:ring-brand-primary"
              value={currentInterface}
              onChange={(e) => changeInterface(e.target.value)}
              aria-label="Select network interface"
            >
              {interfaces.length > 0 ? (
                interfaces
                  .filter(
                    (iface) =>
                      iface.type === "ethernet" || iface.type === "wifi",
                  )
                  .sort((a, b) => (b.score || 0) - (a.score || 0)) // Sort by score
                  .map((iface) => (
                    <option
                      key={iface.name}
                      value={iface.name}
                      title={`${iface.name}${iface.description ? ` - ${iface.description}` : ""}`}
                    >
                      {iface.friendlyName || iface.name}
                      {iface.speedDisplay && ` (${iface.speedDisplay})`}
                      {!iface.up && " - down"}
                    </option>
                  ))
              ) : (
                <option value={currentInterface}>{currentInterface}</option>
              )}
            </select>

            {/* Touch-friendly buttons with larger tap targets */}
            <button
              className="rounded-md p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
              onClick={toggleTheme}
              aria-label={
                isDark ? "Switch to light mode" : "Switch to dark mode"
              }
            >
              {isDark ? (
                <svg
                  className="w-5 h-5"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                  aria-hidden="true"
                >
                  <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
                </svg>
              ) : (
                <svg
                  className="w-5 h-5"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                  aria-hidden="true"
                >
                  <path
                    fillRule="evenodd"
                    d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
            </button>
            <button
              className="rounded-md p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
              onClick={() => setHelpOpen(true)}
              aria-label="Open help"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </button>
            <button
              className="rounded-md p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
              onClick={() => setSettingsOpen(true)}
              aria-label="Open settings"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
              </svg>
            </button>
            <button
              className="rounded-md p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary body-small hidden sm:block touch-manipulation"
              onClick={logout}
              aria-label="Logout"
            >
              Logout
            </button>
            {/* Mobile logout icon */}
            <button
              className="rounded-md p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary sm:hidden touch-manipulation"
              onClick={logout}
              aria-label="Logout"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* Mobile connection status - show below header on small screens */}
        <div className="sm:hidden mt-2 flex items-center justify-center px-3 pb-2">
          <ConnectionStatus status={wsStatus} onReconnect={reconnect} />
        </div>
      </header>

      {/* Main content */}
      <main className="py-4 sm:py-6">
        <div className="max-w-8xl mx-auto px-4 sm:px-6 lg:px-8">
          {/* Section: Primary Connectivity */}
          <section aria-labelledby="connectivity-heading" className="mb-6">
            <h2 id="connectivity-heading" className="section-title mb-3">
              Connectivity
            </h2>
            <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              <LinkCard data={cards.link} loading={loading} />
              {cards.cable?.supported && (
                <CableCard data={cards.cable} loading={loading} />
              )}
              {isWifi && cards.wifi?.ssid && (
                <WiFiCard data={cards.wifi} loading={loading} visible={true} />
              )}
              <SwitchCard
                data={cards.switch}
                vlanData={cards.vlan}
                loading={loading}
              />
            </div>
          </section>

          {/* Section: Network Services */}
          <section aria-labelledby="network-heading" className="mb-6">
            <h2 id="network-heading" className="section-title mb-3">
              Network
            </h2>
            <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              <NetworkCard
                data={cards.dhcp}
                publicip={cards.publicip}
                loading={loading}
                showPublicIP={displayOptions.showPublicIP}
              />
              <GatewayCard data={cards.gateway} loading={loading} />
              <DNSCard data={cards.dns} loading={loading} />
            </div>
          </section>

          {/* Section: Testing & Discovery */}
          <section aria-labelledby="performance-heading" className="mb-6">
            <h2 id="performance-heading" className="section-title mb-3">
              Testing & Discovery
            </h2>
            <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              <HealthCheckCard loading={loading} />
              {fabOptions.runPerformance && (
                <PerformanceCard
                  loading={loading}
                  runSpeedtestEnabled={fabOptions.runSpeedtest}
                  runIperfEnabled={fabOptions.runIperf}
                />
              )}
              {fabOptions.runNetworkDiscovery && (
                <NetworkDiscoveryCard
                  data={networkDiscovery}
                  loading={loading}
                  onScan={triggerDeviceScan}
                />
              )}
              <WiFiSurveyCard isWifi={isWifi} />
            </div>
          </section>

          {/* Section: System */}
          <section aria-labelledby="system-heading" className="mb-6">
            <h2 id="system-heading" className="section-title mb-3">
              System
            </h2>
            <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              <SystemHealthCard />
            </div>
          </section>

          {/* Footer notice */}
          <footer className="mt-8 rounded-md border border-surface-border bg-surface-raised p-4 sm:p-6 text-center">
            <h2 className="heading-4 text-text-muted">
              LuminetIQ {appVersion}
            </h2>
            <p className="mt-2 body-small text-text-muted">
              Tap the play button to run all tests.
              <span className="hidden sm:inline">
                <br />
              </span>
              <span className="sm:hidden"> </span>
              Use the Network Discovery card to scan for devices on your
              network.
            </p>
          </footer>
        </div>
      </main>

      {/* Settings Drawer */}
      <SettingsDrawer
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        version={appVersion}
      />

      {/* Help Modal - improved with TOC, About, and search */}
      <ImprovedHelpModal isOpen={helpOpen} onClose={() => setHelpOpen(false)} />

      {/* FAB - Run All Tests */}
      <FAB />
    </div>
  );
}

interface LoginFormProps {
  onLogin: (username: string, password: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

function LoginForm({ onLogin, isLoading, error }: LoginFormProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onLogin(username, password);
  };

  return (
    <div className="min-h-screen bg-surface-base flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="w-16 h-16 mx-auto text-brand-primary">
            <svg viewBox="0 0 48 48" fill="none" className="w-full h-full">
              <circle
                cx="24"
                cy="24"
                r="20"
                stroke="currentColor"
                strokeWidth="2"
                opacity="0.3"
              />
              <circle
                cx="24"
                cy="24"
                r="14"
                stroke="currentColor"
                strokeWidth="2"
                opacity="0.5"
              />
              <circle cx="24" cy="24" r="4" fill="currentColor" />
              <line
                x1="24"
                y1="10"
                x2="24"
                y2="18"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="24"
                y1="30"
                x2="24"
                y2="38"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="10"
                y1="24"
                x2="18"
                y2="24"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="30"
                y1="24"
                x2="38"
                y2="24"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="14.1"
                y1="14.1"
                x2="19.1"
                y2="19.1"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="28.9"
                y1="28.9"
                x2="33.9"
                y2="33.9"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="33.9"
                y1="14.1"
                x2="28.9"
                y2="19.1"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <line
                x1="14.1"
                y1="33.9"
                x2="19.1"
                y2="28.9"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <circle cx="24" cy="8" r="3" fill="currentColor" />
              <circle cx="24" cy="40" r="3" fill="currentColor" />
              <circle cx="8" cy="24" r="3" fill="currentColor" />
              <circle cx="40" cy="24" r="3" fill="currentColor" />
              <circle cx="12.3" cy="12.3" r="2.5" fill="currentColor" />
              <circle cx="35.7" cy="35.7" r="2.5" fill="currentColor" />
              <circle cx="35.7" cy="12.3" r="2.5" fill="currentColor" />
              <circle cx="12.3" cy="35.7" r="2.5" fill="currentColor" />
            </svg>
          </div>
          <h1 className="heading-1 mt-3">LuminetIQ</h1>
          <p className="body-small mt-1">Illuminate Your Network</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-surface-raised rounded-md border border-surface-border p-6"
        >
          <div className="mb-4">
            <label htmlFor="login-username" className="label block mb-1">
              Username
            </label>
            <input
              id="login-username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-3 py-2 rounded-md border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="admin"
              required
            />
          </div>

          <div className="mb-6">
            <label htmlFor="login-password" className="label block mb-1">
              Password
            </label>
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 rounded-md border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="••••••••"
              required
            />
          </div>

          {error && (
            <div
              role="alert"
              aria-live="assertive"
              className="mb-4 p-3 bg-status-error/10 border border-status-error/20 rounded-md text-status-error body-small"
            >
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded-md font-medium hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
          >
            {isLoading ? "Logging in..." : "Login"}
          </button>

          <p className="mt-4 caption text-text-muted text-center">
            Default: admin / luminetiq
          </p>
        </form>
      </div>
    </div>
  );
}

interface ConnectionStatusProps {
  status: "connecting" | "connected" | "disconnected" | "error";
  onReconnect: () => void;
}

function ConnectionStatus({ status, onReconnect }: ConnectionStatusProps) {
  const statusConfig = {
    connecting: {
      color: "text-status-warning",
      label: "Connecting...",
      icon: "spinner",
    },
    connected: {
      color: "text-status-success",
      label: "Connected",
      icon: "dot",
    },
    disconnected: {
      color: "text-status-error",
      label: "Disconnected",
      icon: "dot",
    },
    error: { color: "text-status-error", label: "Error", icon: "dot" },
  };

  const config = statusConfig[status];

  return (
    <div
      className="flex items-center gap-2 ml-4"
      role="status"
      aria-live="polite"
    >
      <span
        className={`inline-flex items-center gap-1.5 caption ${config.color}`}
      >
        <span
          className={`inline-flex items-center justify-center ${radius.full} ${config.color} ${
            config.icon === "spinner"
              ? "bg-status-info/10 p-1"
              : "bg-current/10 p-1"
          }`}
          aria-label={`WebSocket status: ${config.label}`}
        >
          {config.icon === "spinner" ? (
            <svg
              className="w-3 h-3 animate-spin"
              viewBox="0 0 20 20"
              fill="none"
              aria-hidden="true"
            >
              <circle
                className="opacity-25"
                cx="10"
                cy="10"
                r="8"
                stroke="currentColor"
                strokeWidth="3"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M18 10a8 8 0 00-8-8v4a4 4 0 014 4h4z"
              />
            </svg>
          ) : (
            <svg
              className="w-3 h-3"
              viewBox="0 0 8 8"
              fill="currentColor"
              aria-hidden="true"
            >
              <circle cx="4" cy="4" r="4" />
            </svg>
          )}
        </span>
        {config.label}
      </span>
      {(status === "disconnected" || status === "error") && (
        <button
          onClick={onReconnect}
          className="caption text-brand-primary hover:underline focus:outline-none focus:ring-2 focus:ring-brand-primary rounded-md"
        >
          Reconnect
        </button>
      )}
    </div>
  );
}

export default App;
