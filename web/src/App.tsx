import { useCallback, useEffect, useState } from "react";
import { useWebSocket, Message, CardUpdate } from "./hooks/useWebSocket";
import { useAuth, getAuthHeaders } from "./hooks/useAuth";
import { useTheme } from "./hooks/useTheme";
import { SettingsDrawer } from "./components/settings/SettingsDrawer";
import { HelpModal, HelpSection, HelpItem } from "./components/ui/HelpModal";
import {
  HTTP_TIMING_HELP,
  LINK_HELP,
  DNS_HELP,
  PERFORMANCE_HELP,
  DISCOVERY_HELP,
  CABLE_HELP,
  WIFI_HELP,
  SWITCH_HELP,
  VLAN_HELP,
  DHCP_HELP,
  GATEWAY_HELP,
} from "./components/help/HelpContent";

const API_BASE = import.meta.env.VITE_API_BASE || "";
import {
  LinkCard,
  LinkData,
  SwitchCard,
  SwitchData,
  DHCPCard,
  DHCPData,
  DNSCard,
  type DNSData,
  GatewayCard,
  GatewayData,
  VLANCard,
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
import { FAB } from "./components/ui/FAB";

interface CardState {
  link: LinkData | null;
  cable: CableData | null;
  vlan: VLANData | null;
  switch: SwitchData | null;
  wifi: WiFiData | null;
  dhcp: DHCPData | null;
  dns: DNSData | null;
  gateway: GatewayData | null;
  publicip: PublicIPData | null;
}

interface FABOptions {
  runLink: boolean;
  runSwitch: boolean;
  runVLAN: boolean;
  runIPConfig: boolean;
  runGateway: boolean;
  runDNS: boolean;
  runHealthChecks: boolean;
  runPerformance: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runNetworkDiscovery: boolean;
  autoScanOnLink: boolean;
}

function App() {
  const { isAuthenticated, token, login, logout, isLoading, error } = useAuth();
  const { isDark, toggleTheme } = useTheme();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
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
    Array<{ name: string; type: string; up: boolean }>
  >([]);
  const [networkDiscovery, setNetworkDiscovery] =
    useState<NetworkDiscoveryData | null>(null);
  const [showPublicIP, setShowPublicIP] = useState(true);
  const [fabOptions, setFabOptions] = useState<FABOptions>(() => {
    const defaults: FABOptions = {
      runLink: true,
      runSwitch: true,
      runVLAN: true,
      runIPConfig: true,
      runGateway: true,
      runDNS: true,
      runHealthChecks: true,
      runPerformance: true,
      runSpeedtest: true,
      runIperf: true,
      runNetworkDiscovery: true,
      autoScanOnLink: true,
    };

    try {
      const saved = localStorage.getItem("netscope-fab-options");
      if (saved) return { ...defaults, ...JSON.parse(saved) };
    } catch (err) {
      console.error("Failed to load FAB options:", err);
    }
    return { ...defaults };
  });

  // Load display options from localStorage on mount
  useEffect(() => {
    try {
      const saved = localStorage.getItem("netscope-display-options");
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.showPublicIP !== undefined) {
          setTimeout(() => setShowPublicIP(parsed.showPublicIP), 0);
        }
      }
    } catch (err) {
      console.error("Failed to load display options:", err);
    }
  }, []);

  // Listen for display options updates from settings drawer
  useEffect(() => {
    const handleDisplayOptionsUpdate = (event: CustomEvent) => {
      if (event.detail && event.detail.showPublicIP !== undefined) {
        setShowPublicIP(event.detail.showPublicIP);
      }
    };
    window.addEventListener(
      "displayOptionsUpdated",
      handleDisplayOptionsUpdate as EventListener,
    );
    return () => {
      window.removeEventListener(
        "displayOptionsUpdated",
        handleDisplayOptionsUpdate as EventListener,
      );
    };
  }, []);

  // Listen for FAB option updates
  useEffect(() => {
    const handleFabUpdated = (event: Event) => {
      const detail = (event as CustomEvent<Partial<typeof fabOptions>>).detail;
      if (detail) {
        setFabOptions((prev) => ({ ...prev, ...detail }));
      }
    };

    window.addEventListener(
      "fabOptionsUpdated",
      handleFabUpdated as EventListener,
    );
    return () =>
      window.removeEventListener(
        "fabOptionsUpdated",
        handleFabUpdated as EventListener,
      );
  }, []);

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

  // Catch unauthorized API responses once and force logout to stop noisy 401 spam
  useEffect(() => {
    const originalFetch = window.fetch;

    window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
      const response = await originalFetch(input, init);

      try {
        const url =
          typeof input === "string"
            ? input
            : input instanceof Request
              ? input.url
              : input.toString();

        const isApiCall = url.includes("/api/");
        const isAuthEndpoint = url.includes("/api/auth/");

        if (response.status === 401 && isApiCall && !isAuthEndpoint) {
          setSessionExpired(true);
          logout();
        }
      } catch (err) {
        console.warn("Failed to inspect fetch response", err);
      }

      return response;
    };

    return () => {
      window.fetch = originalFetch;
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

  const [testsFlags, setTestsFlags] = useState({
    runPerformance: true,
    runDiscovery: true,
    runSpeedtest: true,
    runIperf: true,
  });

  // Fetch tests flags on load
  useEffect(() => {
    const loadTestsFlags = async () => {
      try {
        const res = await fetch("/api/tests/settings", {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const data = await res.json();
          setTestsFlags({
            runPerformance: data.runPerformance ?? true,
            runDiscovery: data.runDiscovery ?? true,
            runSpeedtest: data.runSpeedtest ?? true,
            runIperf: data.runIperf ?? true,
          });
        }
      } catch (err) {
        console.error("Failed to load tests settings", err);
      }
    };
    loadTestsFlags();

    type TestsSettingsDetail = Partial<{
      runPerformance: boolean;
      runDiscovery: boolean;
      runSpeedtest: boolean;
      runIperf: boolean;
    }>;

    const handleSettingsUpdated = (e: Event) => {
      const detail = (e as CustomEvent<TestsSettingsDetail>).detail;
      if (detail) {
        setTestsFlags((prev) => ({
          runPerformance: detail.runPerformance ?? prev.runPerformance,
          runDiscovery: detail.runDiscovery ?? prev.runDiscovery,
          runSpeedtest: detail.runSpeedtest ?? prev.runSpeedtest,
          runIperf: detail.runIperf ?? prev.runIperf,
        }));
      }
    };
    window.addEventListener(
      "testsSettingsUpdated",
      handleSettingsUpdated as EventListener,
    );
    return () =>
      window.removeEventListener(
        "testsSettingsUpdated",
        handleSettingsUpdated as EventListener,
      );
  }, []);

  // Listen for FAB "run all tests" event with options
  useEffect(() => {
    const handleRunAllTests = async () => {
      // Combine FAB toggles with backend test settings
      const runOpts = {
        runLink: fabOptions.runLink,
        runSwitch: fabOptions.runSwitch,
        runVLAN: fabOptions.runVLAN,
        runIPConfig: fabOptions.runIPConfig,
        runGateway: fabOptions.runGateway,
        runDNS: fabOptions.runDNS,
        runHealthChecks: fabOptions.runHealthChecks,
        runPerformance: fabOptions.runPerformance && testsFlags.runPerformance,
        runSpeedtest:
          fabOptions.runPerformance &&
          fabOptions.runSpeedtest &&
          testsFlags.runSpeedtest,
        runIperf:
          fabOptions.runPerformance &&
          fabOptions.runIperf &&
          testsFlags.runIperf,
        runNetworkDiscovery:
          fabOptions.runNetworkDiscovery && testsFlags.runDiscovery,
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
    testsFlags.runPerformance,
    testsFlags.runDiscovery,
    testsFlags.runSpeedtest,
    testsFlags.runIperf,
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
        <div className="max-w-7xl mx-auto px-3 sm:px-4 lg:px-6 py-2 sm:py-3 flex items-center justify-between gap-2">
          {/* Logo and title - hide title on very small screens */}
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-xl font-bold text-brand-primary flex-shrink-0">
              ◉
            </span>
            <h1 className="text-lg font-semibold hidden xs:block sm:block">
              NetScope
            </h1>
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
              className="rounded border border-surface-border bg-surface-base px-2 py-1.5 text-sm min-w-0 max-w-[100px] sm:max-w-none focus:outline-none focus:ring-2 focus:ring-brand-primary"
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
                  .map((iface) => (
                    <option key={iface.name} value={iface.name}>
                      {iface.name} {!iface.up && "(down)"}
                    </option>
                  ))
              ) : (
                <option value={currentInterface}>{currentInterface}</option>
              )}
            </select>

            {/* Touch-friendly buttons with larger tap targets */}
            <button
              className="rounded p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
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
              className="rounded p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
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
              className="rounded p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-1 focus:ring-offset-surface-raised touch-manipulation"
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
              className="rounded p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary text-sm hidden sm:block touch-manipulation"
              onClick={logout}
              aria-label="Logout"
            >
              Logout
            </button>
            {/* Mobile logout icon */}
            <button
              className="rounded p-2.5 hover:bg-surface-hover active:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-brand-primary sm:hidden touch-manipulation"
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
      <main className="py-3 sm:py-4">
        <div className="max-w-7xl mx-auto px-3 sm:px-4 lg:px-6">
          <div className="grid gap-3 sm:gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {/* Layer 1-2: Physical */}
            <LinkCard data={cards.link} loading={loading} />
            {cards.cable?.supported && (
              <CableCard data={cards.cable} loading={loading} />
            )}
            {isWifi && cards.wifi?.ssid && (
              <WiFiCard data={cards.wifi} loading={loading} visible={true} />
            )}

            {/* Layer 2: Discovery */}
            <SwitchCard data={cards.switch} loading={loading} />
            <VLANCard data={cards.vlan} loading={loading} />

            {/* Layer 3: Network */}
            <DHCPCard
              data={cards.dhcp}
              publicip={cards.publicip}
              loading={loading}
              showPublicIP={showPublicIP}
            />
            <GatewayCard data={cards.gateway} loading={loading} />

            {/* Layer 7: Application */}
            <DNSCard data={cards.dns} loading={loading} />

            {/* Health Checks - tests configured endpoints */}
            <HealthCheckCard loading={loading} />

            {/* Performance Testing */}
            {fabOptions.runPerformance && (
              <PerformanceCard
                loading={loading}
                runSpeedtestEnabled={
                  fabOptions.runSpeedtest && testsFlags.runSpeedtest
                }
                runIperfEnabled={fabOptions.runIperf && testsFlags.runIperf}
              />
            )}

            {/* Network Discovery - device scanning (last) */}
            {fabOptions.runNetworkDiscovery && testsFlags.runDiscovery && (
              <NetworkDiscoveryCard
                data={networkDiscovery}
                loading={loading}
                onScan={triggerDeviceScan}
              />
            )}
          </div>

          {/* Development notice */}
          <div className="mt-6 sm:mt-8 rounded-lg border border-surface-border bg-surface-raised p-4 sm:p-6 text-center">
            <h2 className="text-base sm:text-lg font-semibold text-text-muted">
              NetScope v0.12.4
            </h2>
            <p className="mt-2 text-xs sm:text-sm text-text-muted">
              Tap the play button to run all tests.
              <span className="hidden sm:inline">
                <br />
              </span>
              <span className="sm:hidden"> </span>
              Use the Network Discovery card to scan for devices on your
              network.
            </p>
          </div>
        </div>
      </main>

      {/* Settings Drawer */}
      <SettingsDrawer
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />

      {/* Help Modal - sections ordered to match card/settings order */}
      <HelpModal
        isOpen={helpOpen}
        onClose={() => setHelpOpen(false)}
        title="NetScope Help"
      >
        {/* 1. Link Status (matches LinkCard) */}
        <HelpSection title="Link Status">
          <HelpItem term="Carrier" description={LINK_HELP.Carrier} />
          <HelpItem term="Speed" description={LINK_HELP.Speed} />
          <HelpItem term="Duplex" description={LINK_HELP.Duplex} />
          <HelpItem term="Auto-Neg" description={LINK_HELP.AutoNeg} />
          <HelpItem term="MTU" description={LINK_HELP.MTU} />
        </HelpSection>

        {/* 2. Cable Test (matches CableCard) */}
        <HelpSection title="Cable Test">
          <HelpItem term="TDR" description={CABLE_HELP["TDR Test"]} />
          <HelpItem term="Status" description={CABLE_HELP["Cable Status"]} />
          <HelpItem
            term="Distance"
            description={CABLE_HELP["Fault Distance"]}
          />
          <HelpItem term="Pairs" description={CABLE_HELP.Pairs} />
        </HelpSection>

        {/* 3. WiFi Status (matches WiFiCard) */}
        <HelpSection title="WiFi Status">
          <HelpItem term="SSID" description={WIFI_HELP.SSID} />
          <HelpItem term="BSSID" description={WIFI_HELP.BSSID} />
          <HelpItem term="Signal" description={WIFI_HELP.Signal} />
          <HelpItem term="Channel" description={WIFI_HELP.Channel} />
          <HelpItem term="Security" description={WIFI_HELP.Security} />
          <HelpItem term="Frequency" description={WIFI_HELP.Frequency} />
        </HelpSection>

        {/* 4. Switch / LLDP (matches SwitchCard) */}
        <HelpSection title="Switch (LLDP/CDP)">
          <HelpItem term="Protocol" description={SWITCH_HELP.Protocol} />
          <HelpItem term="Switch" description={SWITCH_HELP["Switch Name"]} />
          <HelpItem term="Port" description={SWITCH_HELP.Port} />
          <HelpItem term="Mgmt IP" description={SWITCH_HELP["Management IP"]} />
        </HelpSection>

        {/* 5. VLAN (matches VLANCard) */}
        <HelpSection title="VLAN">
          <HelpItem term="VLAN ID" description={VLAN_HELP["VLAN ID"]} />
          <HelpItem term="Interface" description={VLAN_HELP.Interface} />
        </HelpSection>

        {/* 6. DHCP (matches DHCPCard) */}
        <HelpSection title="DHCP">
          <HelpItem term="Lease" description={DHCP_HELP.Lease} />
          <HelpItem term="Server" description={DHCP_HELP.Server} />
          <HelpItem term="Gateway" description={DHCP_HELP.Gateway} />
          <HelpItem term="DNS" description={DHCP_HELP.DNS} />
        </HelpSection>

        {/* 7. Gateway (matches GatewayCard) */}
        <HelpSection title="Gateway">
          <HelpItem term="IPv4" description={GATEWAY_HELP["IPv4 Gateway"]} />
          <HelpItem term="IPv6" description={GATEWAY_HELP["IPv6 Gateway"]} />
          <HelpItem term="Reachable" description={GATEWAY_HELP.Reachability} />
          <HelpItem term="Latency" description={GATEWAY_HELP.Latency} />
        </HelpSection>

        {/* 8. DNS Tests (matches DNSCard) */}
        <HelpSection title="DNS Tests">
          <HelpItem term="Forward" description={DNS_HELP["Forward Lookup"]} />
          <HelpItem term="Reverse" description={DNS_HELP["Reverse Lookup"]} />
          <HelpItem term="IPv6" description={DNS_HELP["IPv6 Lookup"]} />
        </HelpSection>

        {/* 9. Health Checks / HTTP Timing (matches HealthCheckCard) */}
        <HelpSection title="Health Checks - HTTP Timing">
          <HelpItem
            term="DNS"
            description={HTTP_TIMING_HELP.DNS}
            color="bg-blue-400"
          />
          <HelpItem
            term="TCP"
            description={HTTP_TIMING_HELP.TCP}
            color="bg-cyan-400"
          />
          <HelpItem
            term="TLS"
            description={HTTP_TIMING_HELP.TLS}
            color="bg-purple-400"
          />
          <HelpItem
            term="Wait"
            description={HTTP_TIMING_HELP.Wait}
            color="bg-amber-400"
          />
          <HelpItem
            term="Download"
            description={HTTP_TIMING_HELP.Download}
            color="bg-green-400"
          />
        </HelpSection>

        {/* 10. Performance Tests (matches PerformanceCard) */}
        <HelpSection title="Performance Tests">
          <HelpItem
            term="Internet"
            description={PERFORMANCE_HELP["Internet Speed"]}
          />
          <HelpItem term="LAN" description={PERFORMANCE_HELP["LAN Speed"]} />
          <HelpItem term="Jitter" description={PERFORMANCE_HELP.Jitter} />
        </HelpSection>

        {/* 11. Network Discovery (matches NetworkDiscoveryCard) */}
        <HelpSection title="Network Discovery">
          <HelpItem term="Scan" description={DISCOVERY_HELP["Network Scan"]} />
          <HelpItem term="MAC" description={DISCOVERY_HELP.MAC} />
          <HelpItem term="Vendor" description={DISCOVERY_HELP.Vendor} />
        </HelpSection>
      </HelpModal>

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
          <span className="text-4xl font-bold text-brand-primary">◉</span>
          <h1 className="text-2xl font-bold text-text-primary mt-2">
            NetScope
          </h1>
          <p className="text-text-muted mt-1">Network Diagnostic Tool</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-surface-raised rounded-lg border border-surface-border p-6"
        >
          <div className="mb-4">
            <label className="block text-sm font-medium text-text-primary mb-1">
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="admin"
              required
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium text-text-primary mb-1">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="••••••••"
              required
            />
          </div>

          {error && (
            <div
              role="alert"
              aria-live="assertive"
              className="mb-4 p-3 bg-status-error/10 border border-status-error/20 rounded text-status-error text-sm"
            >
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
          >
            {isLoading ? "Logging in..." : "Login"}
          </button>

          <p className="mt-4 text-xs text-text-muted text-center">
            Default: admin / netscope
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
        className={`inline-flex items-center gap-1.5 text-xs ${config.color}`}
      >
        <span
          className={`inline-flex items-center justify-center rounded-full ${config.color} ${
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
          className="text-xs text-brand-primary hover:underline focus:outline-none focus:ring-2 focus:ring-brand-primary rounded"
        >
          Reconnect
        </button>
      )}
    </div>
  );
}

export default App;
