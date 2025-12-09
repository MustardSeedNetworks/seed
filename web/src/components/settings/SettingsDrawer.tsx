import { useState, useEffect, useCallback, useRef } from "react";
import { useTheme } from "../../hooks/useTheme";
import { getAuthHeaders } from "../../hooks/useAuth";
import { CollapsibleSection } from "../ui/CollapsibleSection";

// Auto-save status indicator component
function AutoSaveIndicator({
  status,
}: {
  status: "idle" | "saving" | "saved" | "error";
}) {
  if (status === "idle") return null;
  return (
    <span
      className={`text-xs ml-2 ${
        status === "saving"
          ? "text-text-muted"
          : status === "saved"
            ? "text-status-success"
            : "text-status-error"
      }`}
    >
      {status === "saving"
        ? "Saving..."
        : status === "saved"
          ? "Saved"
          : "Error"}
    </span>
  );
}

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface Thresholds {
  dns: {
    good: number;
    warning: number;
  };
  gateway: {
    good: number;
    warning: number;
  };
  wifi: {
    good: number;
    warning: number;
  };
  customPing: {
    good: number;
    warning: number;
  };
  customTcp: {
    good: number;
    warning: number;
  };
  customHttp: {
    good: number;
    warning: number;
  };
}

interface WiFiSettings {
  interface: string;
  availableWifi: string[];
  isWireless: boolean;
}

interface IPSettings {
  mode: "dhcp" | "static";
  address: string;
  netmask: string;
  gateway: string;
  dns: string[];
}

interface PingTarget {
  name: string;
  host: string;
  enabled: boolean;
  count?: number; // Number of pings (default 3)
}

interface DNSServer {
  address: string;
  enabled: boolean;
}

interface FABOptions {
  // Order matches card display order
  runLink: boolean; // Link card
  runSwitch: boolean; // Nearest Switch card
  runVLAN: boolean; // VLAN card
  runIPConfig: boolean; // IP Config (DHCP) card
  runGateway: boolean; // Gateway card
  runDNS: boolean; // DNS card
  runHealthChecks: boolean; // Health Checks card
  runSpeedtest: boolean; // Performance: Internet Speed (default OFF)
  runIperf: boolean; // Performance: LAN Speed (default OFF)
  runNetworkDiscovery: boolean; // Network Discovery card (default ON)
  autoScanOnLink: boolean; // Auto-scan network on link up (default ON when discovery enabled)
}

interface DisplayOptions {
  showPublicIP: boolean; // Show Public IP in IP Config card (default ON)
}

interface TCPPort {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

interface UDPPort {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

interface HTTPEndpoint {
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
}

interface LogsResponse {
  path: string;
  lines: string[];
}

interface TestsSettings {
  dnsHostname: string;
  dnsServers: DNSServer[];
  pingTargets: PingTarget[];
  tcpPorts: TCPPort[];
  udpPorts: UDPPort[];
  httpEndpoints: HTTPEndpoint[];
  speedtest: {
    serverId: string;
    autoRunOnLink: boolean;
  };
}

interface IperfSettings {
  server: string;
  port: number;
  protocol: "tcp" | "udp";
  direction: "upload" | "download";
  duration: number;
  serverPort: number;
  enableServer: boolean;
}

interface NetworkDiscoverySettings {
  enabled: boolean;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  ouiFilePath: string;
}

interface SubnetConfig {
  cidr: string;
  name: string;
  enabled: boolean;
}

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsDrawer({ isOpen, onClose }: SettingsDrawerProps) {
  const { theme, setTheme, isDark } = useTheme();
  const scrollRef = useRef<HTMLDivElement | null>(null);
  // Smooth scroll to top when opened
  useEffect(() => {
    if (isOpen && scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [isOpen]);
  const [thresholds, setThresholds] = useState<Thresholds>({
    dns: { good: 50, warning: 100 },
    gateway: { good: 20, warning: 50 },
    wifi: { good: -50, warning: -70 },
    customPing: { good: 50, warning: 100 },
    customTcp: { good: 100, warning: 500 },
    customHttp: { good: 500, warning: 2000 },
  });
  const [ipSettings, setIPSettings] = useState<IPSettings>({
    mode: "dhcp",
    address: "",
    netmask: "24",
    gateway: "",
    dns: [],
  });
  const [testsSettings, setTestsSettings] = useState<TestsSettings>({
    dnsHostname: "google.com",
    dnsServers: [],
    pingTargets: [],
    tcpPorts: [],
    udpPorts: [],
    httpEndpoints: [],
    speedtest: {
      serverId: "",
      autoRunOnLink: false,
    },
  });

  // FAB Options (stored in localStorage)
  const [fabOptions, setFabOptions] = useState<FABOptions>({
    // Order matches card display order
    runLink: true, // Link card
    runSwitch: true, // Nearest Switch card
    runVLAN: true, // VLAN card
    runIPConfig: true, // IP Config (DHCP) card
    runGateway: true, // Gateway card
    runDNS: true, // DNS card
    runHealthChecks: true, // Health Checks card
    runSpeedtest: false, // Performance: Internet Speed (default OFF)
    runIperf: false, // Performance: LAN Speed (default OFF)
    runNetworkDiscovery: true, // Network Discovery card (default ON)
    autoScanOnLink: true, // Auto-scan network on link up (default ON)
  });

  // Display Options (stored in localStorage)
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>({
    showPublicIP: true, // Show Public IP in IP Config card (default ON)
  });
  const [wifiSettings, setWifiSettings] = useState<WiFiSettings>({
    interface: "",
    availableWifi: [],
    isWireless: false,
  });
  const [dnsInput, setDnsInput] = useState("");

  // iperf3/LAN Speed settings
  const [iperfSettings, setIperfSettings] = useState<IperfSettings>({
    server: "",
    port: 5201,
    protocol: "tcp",
    direction: "download",
    duration: 10,
    serverPort: 5201,
    enableServer: false,
  });
  // Network Discovery settings
  const [networkDiscoverySettings, setNetworkDiscoverySettings] =
    useState<NetworkDiscoverySettings>({
      enabled: true,
      arpScanWorkers: 50,
      pingTimeoutMs: 500,
      scanTimeoutMs: 30000,
      autoScan: false,
      scanIntervalMs: 0,
      ouiFilePath: "oui.txt",
    });
  // Additional subnets for scanning
  const [subnets, setSubnets] = useState<SubnetConfig[]>([]);
  const [newSubnetCidr, setNewSubnetCidr] = useState("");
  const [newSubnetName, setNewSubnetName] = useState("");
  const [subnetError, setSubnetError] = useState<string | null>(null);
  // Log preview (debug)
  const [logPreview, setLogPreview] = useState<string[]>([]);
  const [logLoading, setLogLoading] = useState(false);
  const [logError, setLogError] = useState<string | null>(null);
  const [subnetsStatus, setSubnetsStatus] = useState<SaveStatus>("idle");
  // Auto-save status for each section
  type SaveStatus = "idle" | "saving" | "saved" | "error";
  const [thresholdsStatus, setThresholdsStatus] = useState<SaveStatus>("idle");
  const [testsStatus, setTestsStatus] = useState<SaveStatus>("idle");
  const [wifiStatus, setWifiStatus] = useState<SaveStatus>("idle");
  const [iperfStatus, setIperfStatus] = useState<SaveStatus>("idle");
  const [fabStatus, setFabStatus] = useState<SaveStatus>("idle");
  const [networkDiscoveryStatus, setNetworkDiscoveryStatus] =
    useState<SaveStatus>("idle");
  const [displayStatus, setDisplayStatus] = useState<SaveStatus>("idle");

  // Refs to track initial load (skip auto-save on first load)
  const initialLoadRef = useRef(true);
  const thresholdsInitRef = useRef(true);
  const testsInitRef = useRef(true);
  const wifiInitRef = useRef(true);
  const iperfInitRef = useRef(true);
  const fabInitRef = useRef(true);
  const networkDiscoveryInitRef = useRef(true);
  const displayInitRef = useRef(true);

  // Debounce timers
  const thresholdsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const testsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wifiTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const iperfTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const fabTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );
  const displayTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Legacy state (keep for IP settings which still needs manual apply)
  const [savingIP, setSavingIP] = useState(false);
  const [ipMessage, setIPMessage] = useState<string | null>(null);

  // Fetch current thresholds
  const fetchThresholds = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        if (data.thresholds) {
          setThresholds((prev) => ({
            ...prev,
            ...data.thresholds,
          }));
        }
      }
    } catch (err) {
      console.error("Failed to fetch thresholds:", err);
    }
  }, []);

  // Fetch current IP settings
  const fetchIPSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setIPSettings({
          mode: data.mode || "dhcp",
          address: data.address || "",
          netmask: data.netmask || "24",
          gateway: data.gateway || "",
          dns: data.dns || [],
        });
        setDnsInput((data.dns || []).join(", "));
      }
    } catch (err) {
      console.error("Failed to fetch IP settings:", err);
    }
  }, []);

  // Fetch current tests settings
  const fetchTestsSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/tests/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setTestsSettings({
          dnsHostname: data.dnsHostname || "google.com",
          dnsServers: data.dnsServers || [],
          pingTargets: data.pingTargets || [],
          tcpPorts: data.tcpPorts || [],
          udpPorts: data.udpPorts || [],
          httpEndpoints: data.httpEndpoints || [],
          speedtest: {
            serverId: data.speedtest?.serverId || "",
            autoRunOnLink: data.speedtest?.autoRunOnLink || false,
          },
        });
      }
    } catch (err) {
      console.error("Failed to fetch tests settings:", err);
    }
  }, []);

  // Fetch WiFi settings
  const fetchWifiSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/wifi/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setWifiSettings({
          interface: data.interface || "",
          availableWifi: data.availableWifi || [],
          isWireless: data.isWireless || false,
        });
      }
    } catch (err) {
      console.error("Failed to fetch WiFi settings:", err);
    }
  }, []);

  // Load iperf settings from localStorage
  const loadIperfSettings = useCallback(() => {
    try {
      const saved = localStorage.getItem("netscope-iperf-settings");
      if (saved) {
        const parsed = JSON.parse(saved);
        setIperfSettings((prev) => ({ ...prev, ...parsed }));
      }
    } catch (err) {
      console.error("Failed to load iperf settings:", err);
    }
  }, []);

  // Load FAB options from localStorage
  const loadFabOptions = useCallback(() => {
    try {
      const saved = localStorage.getItem("netscope-fab-options");
      if (saved) {
        const parsed = JSON.parse(saved);
        setFabOptions((prev) => ({ ...prev, ...parsed }));
      }
    } catch (err) {
      console.error("Failed to load FAB options:", err);
    }
  }, []);

  // Load Display options from localStorage
  const loadDisplayOptions = useCallback(() => {
    try {
      const saved = localStorage.getItem("netscope-display-options");
      if (saved) {
        const parsed = JSON.parse(saved);
        setDisplayOptions((prev) => ({ ...prev, ...parsed }));
      }
    } catch (err) {
      console.error("Failed to load display options:", err);
    }
  }, []);

  // Fetch Network Discovery settings from API
  const fetchNetworkDiscoverySettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/devices/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setNetworkDiscoverySettings({
          enabled: data.enabled ?? true,
          arpScanWorkers: data.arpScanWorkers ?? 50,
          pingTimeoutMs: data.pingTimeoutMs ?? 500,
          scanTimeoutMs: data.scanTimeoutMs ?? 30000,
          autoScan: data.autoScan ?? false,
          scanIntervalMs: data.scanIntervalMs ?? 0,
          ouiFilePath: data.ouiFilePath ?? "oui.txt",
        });
      }
    } catch (err) {
      console.error("Failed to fetch network discovery settings:", err);
    }
  }, []);

  // Fetch configured subnets from API
  const fetchSubnets = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/devices/subnets`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setSubnets(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error("Failed to fetch subnets:", err);
    }
  }, []);

  // Fetch a small tail of the application log (debug)
  const fetchLogPreview = useCallback(async () => {
    setLogLoading(true);
    setLogError(null);
    try {
      const response = await fetch(`${API_BASE}/api/logs?lines=200`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        throw new Error("Unable to load logs");
      }
      const data = (await response.json()) as LogsResponse;
      setLogPreview(data.lines || []);
    } catch (err) {
      setLogPreview([]);
      setLogError(
        err instanceof Error ? err.message : "Failed to load log file",
      );
    } finally {
      setLogLoading(false);
    }
  }, []);

  // Fetch subnets when drawer opens
  useEffect(() => {
    if (isOpen) {
      fetchSubnets();
    }
  }, [isOpen, fetchSubnets]);

  // Add a new subnet
  const addSubnet = async () => {
    if (!newSubnetCidr.trim()) {
      setSubnetError("CIDR is required");
      return;
    }

    setSubnetError(null);
    setSubnetsStatus("saving");

    try {
      const response = await fetch(`${API_BASE}/api/devices/subnets`, {
        method: "POST",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          cidr: newSubnetCidr.trim(),
          name: newSubnetName.trim() || newSubnetCidr.trim(),
          enabled: true,
        }),
      });

      if (response.ok) {
        setNewSubnetCidr("");
        setNewSubnetName("");
        setSubnetsStatus("saved");
        setTimeout(() => setSubnetsStatus("idle"), 2000);
        fetchSubnets();
      } else {
        // Handle both JSON and plain text error responses
        const contentType = response.headers.get("content-type");
        if (contentType && contentType.includes("application/json")) {
          const errorData = await response.json();
          setSubnetError(errorData.error || "Failed to add subnet");
        } else {
          const errorText = await response.text();
          setSubnetError(errorText || "Failed to add subnet");
        }
        setSubnetsStatus("error");
      }
    } catch (err) {
      setSubnetError(
        err instanceof Error ? err.message : "Network error adding subnet",
      );
      setSubnetsStatus("error");
    }
  };

  // Toggle subnet enabled state
  const toggleSubnet = async (cidr: string, enabled: boolean) => {
    setSubnetsStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/devices/subnets`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ cidr, enabled }),
      });

      if (response.ok) {
        setSubnetsStatus("saved");
        setTimeout(() => setSubnetsStatus("idle"), 2000);
        fetchSubnets();
      } else {
        setSubnetsStatus("error");
      }
    } catch {
      setSubnetsStatus("error");
    }
  };

  // Delete a subnet
  const deleteSubnet = async (cidr: string) => {
    setSubnetsStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/devices/subnets`, {
        method: "DELETE",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ cidr }),
      });

      if (response.ok) {
        setSubnetsStatus("saved");
        setTimeout(() => setSubnetsStatus("idle"), 2000);
        fetchSubnets();
      } else {
        setSubnetsStatus("error");
      }
    } catch {
      setSubnetsStatus("error");
    }
  };

  // Save Network Discovery settings to API
  const saveNetworkDiscoverySettings = useCallback(async () => {
    setNetworkDiscoveryStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/devices/settings`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(networkDiscoverySettings),
      });
      if (response.ok) {
        setNetworkDiscoveryStatus("saved");
        setTimeout(() => setNetworkDiscoveryStatus("idle"), 2000);
      } else {
        setNetworkDiscoveryStatus("error");
      }
    } catch {
      setNetworkDiscoveryStatus("error");
    }
  }, [networkDiscoverySettings]);

  useEffect(() => {
    if (isOpen) {
      // Reset init refs on open
      initialLoadRef.current = true;
      thresholdsInitRef.current = true;
      testsInitRef.current = true;
      wifiInitRef.current = true;
      iperfInitRef.current = true;
      fabInitRef.current = true;
      networkDiscoveryInitRef.current = true;
      displayInitRef.current = true;

      fetchThresholds();
      fetchIPSettings();
      fetchTestsSettings();
      fetchWifiSettings();
      loadIperfSettings();
      loadFabOptions();
      loadDisplayOptions();
      fetchNetworkDiscoverySettings();
      fetchSubnets();

      // Mark initial load as done after a short delay
      setTimeout(() => {
        initialLoadRef.current = false;
        thresholdsInitRef.current = false;
        testsInitRef.current = false;
        wifiInitRef.current = false;
        iperfInitRef.current = false;
        fabInitRef.current = false;
        networkDiscoveryInitRef.current = false;
        displayInitRef.current = false;
      }, 500);
    }
  }, [
    isOpen,
    fetchThresholds,
    fetchIPSettings,
    fetchTestsSettings,
    fetchWifiSettings,
    loadIperfSettings,
    loadFabOptions,
    loadDisplayOptions,
    fetchNetworkDiscoverySettings,
    fetchSubnets,
  ]);

  const saveThresholds = useCallback(async () => {
    setThresholdsStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ thresholds }),
      });
      if (response.ok) {
        setThresholdsStatus("saved");
        setTimeout(() => setThresholdsStatus("idle"), 2000);
      } else {
        setThresholdsStatus("error");
      }
    } catch {
      setThresholdsStatus("error");
    }
  }, [thresholds]);

  const updateThreshold = (
    category: keyof Thresholds,
    level: "good" | "warning",
    value: number,
  ) => {
    setThresholds((prev) => ({
      ...prev,
      [category]: {
        ...prev[category],
        [level]: value,
      },
    }));
  };

  const saveIPSettings = async () => {
    setSavingIP(true);
    setIPMessage(null);
    try {
      // Parse DNS from input
      const dns = dnsInput
        .split(",")
        .map((s) => s.trim())
        .filter((s) => s.length > 0);

      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          mode: ipSettings.mode,
          address: ipSettings.address,
          netmask: ipSettings.netmask,
          gateway: ipSettings.gateway,
          dns,
        }),
      });
      if (response.ok) {
        setIPMessage("IP settings applied");
        setTimeout(() => setIPMessage(null), 3000);
      } else {
        const error = await response.text();
        setIPMessage(`Failed: ${error}`);
      }
    } catch {
      setIPMessage("Error applying IP settings");
    } finally {
      setSavingIP(false);
    }
  };

  const saveTestsSettings = useCallback(async () => {
    setTestsStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/tests/settings`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(testsSettings),
      });
      if (response.ok) {
        setTestsStatus("saved");
        setTimeout(() => setTestsStatus("idle"), 2000);
        window.dispatchEvent(new CustomEvent("healthChecksUpdated"));
      } else {
        setTestsStatus("error");
      }
    } catch {
      setTestsStatus("error");
    }
  }, [testsSettings]);

  const saveWifiSettings = useCallback(async () => {
    setWifiStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/wifi/settings`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ interface: wifiSettings.interface }),
      });
      if (response.ok) {
        setWifiStatus("saved");
        setTimeout(() => setWifiStatus("idle"), 2000);
      } else {
        setWifiStatus("error");
      }
    } catch {
      setWifiStatus("error");
    }
  }, [wifiSettings.interface]);

  // Auto-save thresholds with debounce
  useEffect(() => {
    if (thresholdsInitRef.current) return;
    if (thresholdsTimerRef.current) clearTimeout(thresholdsTimerRef.current);
    thresholdsTimerRef.current = setTimeout(() => {
      saveThresholds();
    }, 800);
    return () => {
      if (thresholdsTimerRef.current) clearTimeout(thresholdsTimerRef.current);
    };
  }, [thresholds, saveThresholds]);

  // Auto-save tests settings with debounce
  useEffect(() => {
    if (testsInitRef.current) return;
    if (testsTimerRef.current) clearTimeout(testsTimerRef.current);
    testsTimerRef.current = setTimeout(() => {
      saveTestsSettings();
    }, 800);
    return () => {
      if (testsTimerRef.current) clearTimeout(testsTimerRef.current);
    };
  }, [testsSettings, saveTestsSettings]);

  // Auto-save wifi settings with debounce
  useEffect(() => {
    if (wifiInitRef.current) return;
    if (wifiTimerRef.current) clearTimeout(wifiTimerRef.current);
    wifiTimerRef.current = setTimeout(() => {
      saveWifiSettings();
    }, 800);
    return () => {
      if (wifiTimerRef.current) clearTimeout(wifiTimerRef.current);
    };
  }, [wifiSettings.interface, saveWifiSettings]);

  // Auto-save iperf settings with debounce
  useEffect(() => {
    if (iperfInitRef.current) return;
    if (iperfTimerRef.current) clearTimeout(iperfTimerRef.current);
    iperfTimerRef.current = setTimeout(() => {
      setIperfStatus("saving");
      localStorage.setItem(
        "netscope-iperf-settings",
        JSON.stringify(iperfSettings),
      );
      window.dispatchEvent(
        new CustomEvent("iperfSettingsUpdated", { detail: iperfSettings }),
      );
      setIperfStatus("saved");
      setTimeout(() => setIperfStatus("idle"), 2000);
    }, 800);
    return () => {
      if (iperfTimerRef.current) clearTimeout(iperfTimerRef.current);
    };
  }, [iperfSettings]);

  // Auto-save FAB options with debounce
  useEffect(() => {
    if (fabInitRef.current) return;
    if (fabTimerRef.current) clearTimeout(fabTimerRef.current);
    fabTimerRef.current = setTimeout(() => {
      setFabStatus("saving");
      localStorage.setItem("netscope-fab-options", JSON.stringify(fabOptions));
      window.dispatchEvent(
        new CustomEvent("fabOptionsUpdated", { detail: fabOptions }),
      );
      setFabStatus("saved");
      setTimeout(() => setFabStatus("idle"), 2000);
    }, 800);
    return () => {
      if (fabTimerRef.current) clearTimeout(fabTimerRef.current);
    };
  }, [fabOptions]);

  // Sync autoScanOnLink with networkDiscoverySettings.autoScan for backend compatibility
  useEffect(() => {
    setNetworkDiscoverySettings((prev) => ({
      ...prev,
      autoScan: fabOptions.autoScanOnLink && fabOptions.runNetworkDiscovery,
    }));
  }, [fabOptions.autoScanOnLink, fabOptions.runNetworkDiscovery]);

  // Auto-save Display options with debounce
  useEffect(() => {
    if (displayInitRef.current) return;
    if (displayTimerRef.current) clearTimeout(displayTimerRef.current);
    displayTimerRef.current = setTimeout(() => {
      setDisplayStatus("saving");
      localStorage.setItem(
        "netscope-display-options",
        JSON.stringify(displayOptions),
      );
      window.dispatchEvent(
        new CustomEvent("displayOptionsUpdated", { detail: displayOptions }),
      );
      setDisplayStatus("saved");
      setTimeout(() => setDisplayStatus("idle"), 2000);
    }, 800);
    return () => {
      if (displayTimerRef.current) clearTimeout(displayTimerRef.current);
    };
  }, [displayOptions]);

  // Auto-save Network Discovery settings with debounce
  useEffect(() => {
    if (networkDiscoveryInitRef.current) return;
    if (networkDiscoveryTimerRef.current)
      clearTimeout(networkDiscoveryTimerRef.current);
    networkDiscoveryTimerRef.current = setTimeout(() => {
      saveNetworkDiscoverySettings();
    }, 800);
    return () => {
      if (networkDiscoveryTimerRef.current)
        clearTimeout(networkDiscoveryTimerRef.current);
    };
  }, [networkDiscoverySettings, saveNetworkDiscoverySettings]);

  // Validate IP address format
  const isValidIP = (ip: string): boolean => {
    if (!ip) return true; // Empty is OK for optional fields
    const parts = ip.split(".");
    if (parts.length !== 4) return false;
    return parts.every((p) => {
      const n = parseInt(p, 10);
      return !isNaN(n) && n >= 0 && n <= 255 && p === String(n);
    });
  };

  // Add/remove ping target
  const addPingTarget = () => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: [...prev.pingTargets, { name: "", host: "", enabled: true }],
    }));
  };

  const removePingTarget = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: prev.pingTargets.filter((_, i) => i !== index),
    }));
  };

  const updatePingTarget = (
    index: number,
    field: keyof PingTarget,
    value: string | boolean | number,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: prev.pingTargets.map((t, i) =>
        i === index ? { ...t, [field]: value } : t,
      ),
    }));
  };

  // Add/remove TCP port
  const addTCPPort = () => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: [
        ...prev.tcpPorts,
        { name: "", host: "", port: 80, enabled: true },
      ],
    }));
  };

  const removeTCPPort = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: prev.tcpPorts.filter((_, i) => i !== index),
    }));
  };

  const updateTCPPort = (
    index: number,
    field: keyof TCPPort,
    value: string | number | boolean,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: prev.tcpPorts.map((t, i) =>
        i === index ? { ...t, [field]: value } : t,
      ),
    }));
  };

  // Add/remove UDP port
  const addUDPPort = () => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: [
        ...prev.udpPorts,
        { name: "", host: "", port: 53, enabled: true },
      ],
    }));
  };

  const removeUDPPort = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: prev.udpPorts.filter((_, i) => i !== index),
    }));
  };

  const updateUDPPort = (
    index: number,
    field: keyof UDPPort,
    value: string | number | boolean,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: prev.udpPorts.map((u, i) =>
        i === index ? { ...u, [field]: value } : u,
      ),
    }));
  };

  // Add/remove HTTP endpoint
  const addHTTPEndpoint = () => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: [
        ...prev.httpEndpoints,
        { name: "", url: "", expectedStatus: 200, enabled: true },
      ],
    }));
  };

  const removeHTTPEndpoint = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: prev.httpEndpoints.filter((_, i) => i !== index),
    }));
  };

  const updateHTTPEndpoint = (
    index: number,
    field: keyof HTTPEndpoint,
    value: string | number | boolean,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: prev.httpEndpoints.map((t, i) =>
        i === index ? { ...t, [field]: value } : t,
      ),
    }));
  };

  // Add/remove DNS server
  const addDNSServer = () => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: [...prev.dnsServers, { address: "", enabled: true }],
    }));
  };

  const removeDNSServer = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: prev.dnsServers.filter((_, i) => i !== index),
    }));
  };

  const updateDNSServer = (
    index: number,
    field: keyof DNSServer,
    value: string | boolean,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: prev.dnsServers.map((s, i) =>
        i === index ? { ...s, [field]: value } : s,
      ),
    }));
  };

  const drawerRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Handle ESC key to close drawer
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: globalThis.KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    // Focus the close button when drawer opens
    setTimeout(() => closeButtonRef.current?.focus(), 100);

    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer - full width on mobile, 384px on larger screens */}
      <div
        ref={drawerRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-drawer-title"
        className="fixed right-0 top-0 h-full w-full sm:w-[28rem] lg:w-[32rem] bg-surface-raised border-l border-surface-border z-50 overflow-y-auto shadow-xl"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 sm:px-5 sm:py-4 border-b border-surface-border sticky top-0 bg-surface-raised z-10">
          <div className="space-y-0.5">
            <h2
              id="settings-drawer-title"
              className="text-lg font-semibold text-text-primary leading-tight"
            >
              Settings
            </h2>
            <p className="text-xs sm:text-sm text-text-muted">
              Adjust thresholds, network, and display
            </p>
          </div>
          <button
            ref={closeButtonRef}
            onClick={onClose}
            className="p-2.5 rounded hover:bg-surface-hover active:bg-surface-hover text-text-muted touch-manipulation focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-raised"
            aria-label="Close settings"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <div className="px-4 sm:px-5 pb-10 pt-4 space-y-6 text-sm leading-relaxed">
          {/* Network Section */}
          <CollapsibleSection title="Network">
            {/* IP Configuration */}
            <div className="space-y-3">
              <p className="text-xs uppercase tracking-wide text-text-muted font-semibold">
                IP Configuration
              </p>
              {/* Mode Toggle */}
              <div className="grid grid-cols-2 gap-2">
                <button
                  onClick={() =>
                    setIPSettings((prev) => ({ ...prev, mode: "dhcp" }))
                  }
                  className={`py-2.5 px-3 rounded text-sm font-medium transition-colors ${
                    ipSettings.mode === "dhcp"
                      ? "bg-brand-primary text-text-inverse"
                      : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                  }`}
                >
                  DHCP
                </button>
                <button
                  onClick={() =>
                    setIPSettings((prev) => ({ ...prev, mode: "static" }))
                  }
                  className={`py-2.5 px-3 rounded text-sm font-medium transition-colors ${
                    ipSettings.mode === "static"
                      ? "bg-brand-primary text-text-inverse"
                      : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                  }`}
                >
                  Static
                </button>
              </div>

              {/* Static IP Fields */}
              {ipSettings.mode === "static" && (
                <div className="space-y-3 pt-3 border-t border-surface-border">
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      IP Address *
                    </label>
                    <input
                      type="text"
                      value={ipSettings.address}
                      onChange={(e) =>
                        setIPSettings((prev) => ({
                          ...prev,
                          address: e.target.value,
                        }))
                      }
                      placeholder="192.168.1.100"
                      className={`w-full mt-1 px-2 py-1 bg-surface-base border rounded text-sm text-text-primary ${
                        ipSettings.address && !isValidIP(ipSettings.address)
                          ? "border-status-error"
                          : "border-surface-border"
                      }`}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      Subnet Mask *
                    </label>
                    <input
                      type="text"
                      value={ipSettings.netmask}
                      onChange={(e) =>
                        setIPSettings((prev) => ({
                          ...prev,
                          netmask: e.target.value,
                        }))
                      }
                      placeholder="24 or 255.255.255.0"
                      className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      Gateway
                    </label>
                    <input
                      type="text"
                      value={ipSettings.gateway}
                      onChange={(e) =>
                        setIPSettings((prev) => ({
                          ...prev,
                          gateway: e.target.value,
                        }))
                      }
                      placeholder="192.168.1.1"
                      className={`w-full mt-1 px-2 py-1 bg-surface-base border rounded text-sm text-text-primary ${
                        ipSettings.gateway && !isValidIP(ipSettings.gateway)
                          ? "border-status-error"
                          : "border-surface-border"
                      }`}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      DNS Servers (comma-separated)
                    </label>
                    <input
                      type="text"
                      value={dnsInput}
                      onChange={(e) => setDnsInput(e.target.value)}
                      placeholder="8.8.8.8, 8.8.4.4"
                      className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              )}

              {/* Apply Button */}
              <button
                onClick={saveIPSettings}
                disabled={
                  savingIP ||
                  (ipSettings.mode === "static" && !ipSettings.address)
                }
                className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors"
              >
                {savingIP ? "Applying..." : "Apply IP Settings"}
              </button>

              {ipMessage && (
                <p
                  className={`text-xs text-center ${
                    ipMessage.includes("Failed") || ipMessage.includes("Error")
                      ? "text-status-error"
                      : "text-status-success"
                  }`}
                >
                  {ipMessage}
                </p>
              )}

              <p className="text-xs text-text-muted">
                Note: Requires root/admin privileges to apply
              </p>
            </div>

            {/* Display Options */}
            <div className="border-t border-surface-border pt-3 mt-3">
              <p className="text-xs text-text-muted font-medium mb-2">
                Display Options <AutoSaveIndicator status={displayStatus} />
              </p>
              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <div>
                  <span className="text-sm text-text-primary font-medium">
                    Show Public IP
                  </span>
                  <p className="text-xs text-text-muted">
                    Display in IP Config card
                  </p>
                </div>
                <input
                  type="checkbox"
                  checked={displayOptions.showPublicIP}
                  onChange={(e) =>
                    setDisplayOptions((prev) => ({
                      ...prev,
                      showPublicIP: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>
            </div>
          </CollapsibleSection>

          {/* WiFi Section */}
          <CollapsibleSection
            title={
              <>
                WiFi
                <AutoSaveIndicator status={wifiStatus} />
              </>
            }
          >
            <div className="space-y-3">
              <div>
                <label className="text-xs text-text-muted">
                  WiFi Interface
                </label>
                {wifiSettings.availableWifi.length > 0 ? (
                  <select
                    value={wifiSettings.interface}
                    onChange={(e) =>
                      setWifiSettings((prev) => ({
                        ...prev,
                        interface: e.target.value,
                      }))
                    }
                    className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                  >
                    {wifiSettings.availableWifi.map((iface) => (
                      <option key={iface} value={iface}>
                        {iface}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    value={wifiSettings.interface}
                    onChange={(e) =>
                      setWifiSettings((prev) => ({
                        ...prev,
                        interface: e.target.value,
                      }))
                    }
                    placeholder="wlan0 or en0"
                    className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                  />
                )}
                <p className="text-xs text-text-muted mt-1">
                  {wifiSettings.isWireless
                    ? "Currently monitoring a wireless interface"
                    : "No wireless interface detected"}
                </p>
              </div>
            </div>
          </CollapsibleSection>

          {/* DNS Section */}
          <CollapsibleSection
            title={
              <>
                DNS
                <AutoSaveIndicator status={testsStatus} />
              </>
            }
          >
            <div className="space-y-4">
              {/* DNS Hostname */}
              <div>
                <label className="text-xs text-text-muted">Test Hostname</label>
                <input
                  type="text"
                  value={testsSettings.dnsHostname}
                  onChange={(e) =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      dnsHostname: e.target.value,
                    }))
                  }
                  placeholder="google.com"
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  Hostname used for DNS forward/reverse lookups
                </p>
              </div>

              {/* DNS Servers for per-server testing */}
              <div className="border-t border-surface-border pt-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">
                    Additional DNS Servers
                  </span>
                  <button
                    onClick={addDNSServer}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                <p className="text-xs text-text-muted mb-2">
                  Add servers to compare DNS response times (e.g., 8.8.8.8,
                  1.1.1.1)
                </p>
                {testsSettings.dnsServers.map((server, idx) => (
                  <div key={idx} className="flex gap-2 mb-2">
                    <input
                      type="text"
                      value={server.address}
                      onChange={(e) =>
                        updateDNSServer(idx, "address", e.target.value)
                      }
                      placeholder="DNS Server IP"
                      className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <button
                      onClick={() => removeDNSServer(idx)}
                      className="text-status-error hover:text-status-error/80 px-1"
                    >
                      x
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </CollapsibleSection>

          {/* Health Checks Section */}
          <CollapsibleSection
            title={
              <>
                Health Checks
                <AutoSaveIndicator status={testsStatus} />
              </>
            }
          >
            <div className="space-y-4">
              {/* Ping Targets */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">
                    Ping Targets
                  </span>
                  <button
                    onClick={addPingTarget}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                <p className="text-xs text-text-muted mb-2">
                  Default: 3 pings per target
                </p>
                {testsSettings.pingTargets.map((target, idx) => (
                  <div key={idx} className="flex gap-2 mb-2">
                    <input
                      type="text"
                      value={target.name}
                      onChange={(e) =>
                        updatePingTarget(idx, "name", e.target.value)
                      }
                      placeholder="Name"
                      className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="text"
                      value={target.host}
                      onChange={(e) =>
                        updatePingTarget(idx, "host", e.target.value)
                      }
                      placeholder="Host/IP"
                      className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="number"
                      value={target.count || 3}
                      onChange={(e) =>
                        updatePingTarget(
                          idx,
                          "count",
                          parseInt(e.target.value) || 3,
                        )
                      }
                      min={1}
                      max={10}
                      title="Number of pings"
                      className="w-14 px-2 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary text-center"
                    />
                    <button
                      onClick={() => removePingTarget(idx)}
                      className="text-status-error hover:text-status-error/80 px-1"
                    >
                      x
                    </button>
                  </div>
                ))}
              </div>

              {/* TCP Ports */}
              <div className="border-t border-surface-border pt-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">
                    TCP Port Tests
                  </span>
                  <button
                    onClick={addTCPPort}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                {testsSettings.tcpPorts.map((port, idx) => (
                  <div key={idx} className="flex gap-2 mb-2">
                    <input
                      type="text"
                      value={port.name}
                      onChange={(e) =>
                        updateTCPPort(idx, "name", e.target.value)
                      }
                      placeholder="Name"
                      className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="text"
                      value={port.host}
                      onChange={(e) =>
                        updateTCPPort(idx, "host", e.target.value)
                      }
                      placeholder="Host"
                      className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="number"
                      value={port.port}
                      onChange={(e) =>
                        updateTCPPort(
                          idx,
                          "port",
                          parseInt(e.target.value) || 80,
                        )
                      }
                      placeholder="Port"
                      className="w-20 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <button
                      onClick={() => removeTCPPort(idx)}
                      className="text-status-error hover:text-status-error/80 px-1"
                    >
                      x
                    </button>
                  </div>
                ))}
              </div>

              {/* UDP Ports */}
              <div className="border-t border-surface-border pt-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">
                    UDP Port Tests
                  </span>
                  <button
                    onClick={addUDPPort}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                <p className="text-xs text-text-muted mb-2">
                  Test UDP services (DNS:53, NTP:123, etc.)
                </p>
                {testsSettings.udpPorts.map((port, idx) => (
                  <div key={idx} className="flex gap-2 mb-2">
                    <input
                      type="text"
                      value={port.name}
                      onChange={(e) =>
                        updateUDPPort(idx, "name", e.target.value)
                      }
                      placeholder="Name"
                      className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="text"
                      value={port.host}
                      onChange={(e) =>
                        updateUDPPort(idx, "host", e.target.value)
                      }
                      placeholder="Host"
                      className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="number"
                      value={port.port}
                      onChange={(e) =>
                        updateUDPPort(
                          idx,
                          "port",
                          parseInt(e.target.value) || 53,
                        )
                      }
                      placeholder="Port"
                      className="w-20 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <button
                      onClick={() => removeUDPPort(idx)}
                      className="text-status-error hover:text-status-error/80 px-1"
                    >
                      x
                    </button>
                  </div>
                ))}
              </div>

              {/* HTTP Endpoints */}
              <div className="border-t border-surface-border pt-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">
                    HTTP Endpoints
                  </span>
                  <button
                    onClick={addHTTPEndpoint}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                {testsSettings.httpEndpoints.map((endpoint, idx) => (
                  <div
                    key={idx}
                    className="space-y-1 mb-3 p-2 bg-surface-base rounded border border-surface-border"
                  >
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={endpoint.name}
                        onChange={(e) =>
                          updateHTTPEndpoint(idx, "name", e.target.value)
                        }
                        placeholder="Name"
                        className="flex-1 px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                      />
                      <input
                        type="number"
                        value={endpoint.expectedStatus}
                        onChange={(e) =>
                          updateHTTPEndpoint(
                            idx,
                            "expectedStatus",
                            parseInt(e.target.value) || 200,
                          )
                        }
                        placeholder="Status"
                        className="w-20 px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                      />
                      <button
                        onClick={() => removeHTTPEndpoint(idx)}
                        className="text-status-error hover:text-status-error/80 px-1"
                      >
                        x
                      </button>
                    </div>
                    <input
                      type="text"
                      value={endpoint.url}
                      onChange={(e) =>
                        updateHTTPEndpoint(idx, "url", e.target.value)
                      }
                      placeholder="https://example.com/health"
                      className="w-full px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                    />
                  </div>
                ))}
              </div>
            </div>
          </CollapsibleSection>

          {/* Performance Section - matches PerformanceCard */}
          <CollapsibleSection
            title={
              <>
                Performance
                <AutoSaveIndicator status={iperfStatus} />
              </>
            }
          >
            <div className="space-y-4">
              {/* Internet Speed (Speedtest) Subsection */}
              <div className="border-b border-surface-border pb-4">
                <h4 className="text-sm font-semibold text-text-primary mb-2 uppercase tracking-wide">
                  Internet Speed (Speedtest)
                </h4>
                <div className="space-y-3 pl-1">
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      Server ID (optional)
                    </label>
                    <input
                      type="text"
                      value={testsSettings.speedtest.serverId}
                      onChange={(e) =>
                        setTestsSettings((prev) => ({
                          ...prev,
                          speedtest: {
                            ...prev.speedtest,
                            serverId: e.target.value,
                          },
                        }))
                      }
                      placeholder="Auto (closest server)"
                      className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                    <p className="text-xs text-text-muted mt-1">
                      Leave empty for auto-selection
                    </p>
                  </div>

                  <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                    <span className="text-sm text-text-primary">
                      Auto-run on link up
                    </span>
                    <input
                      type="checkbox"
                      checked={testsSettings.speedtest.autoRunOnLink}
                      onChange={(e) =>
                        setTestsSettings((prev) => ({
                          ...prev,
                          speedtest: {
                            ...prev.speedtest,
                            autoRunOnLink: e.target.checked,
                          },
                        }))
                      }
                      className="w-4 h-4"
                    />
                  </label>
                </div>
              </div>

              {/* LAN Speed (iperf3) Subsection */}
              <div>
                <h4 className="text-sm font-semibold text-text-primary mb-2 uppercase tracking-wide">
                  LAN Speed (iperf3)
                </h4>
                <div className="space-y-3 pl-1">
                  <p className="text-xs text-text-muted">
                    Configure iperf3 client settings for LAN speed tests.
                  </p>

                  {/* Server Address */}
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      Server Address
                    </label>
                    <input
                      type="text"
                      value={iperfSettings.server}
                      onChange={(e) =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          server: e.target.value,
                        }))
                      }
                      placeholder="192.168.1.100 or hostname"
                      className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>

                  {/* Port */}
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      Port
                    </label>
                    <input
                      type="number"
                      value={iperfSettings.port}
                      onChange={(e) =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          port: parseInt(e.target.value) || 5201,
                        }))
                      }
                      className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>

                  {/* Protocol Toggle */}
                  <div>
                    <label className="text-xs text-text-muted font-medium block mb-1">
                      Protocol
                    </label>
                    <div className="flex gap-2">
                      <button
                        onClick={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            protocol: "tcp",
                          }))
                        }
                        className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                          iperfSettings.protocol === "tcp"
                            ? "bg-brand-primary text-text-inverse"
                            : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                        }`}
                      >
                        TCP
                      </button>
                      <button
                        onClick={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            protocol: "udp",
                          }))
                        }
                        className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                          iperfSettings.protocol === "udp"
                            ? "bg-brand-primary text-text-inverse"
                            : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                        }`}
                      >
                        UDP
                      </button>
                    </div>
                  </div>

                  {/* Direction Toggle */}
                  <div>
                    <label className="text-xs text-text-muted font-medium block mb-1">
                      Direction
                    </label>
                    <div className="flex gap-2">
                      <button
                        onClick={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            direction: "download",
                          }))
                        }
                        className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                          iperfSettings.direction === "download"
                            ? "bg-brand-primary text-text-inverse"
                            : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                        }`}
                      >
                        Download
                      </button>
                      <button
                        onClick={() =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            direction: "upload",
                          }))
                        }
                        className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                          iperfSettings.direction === "upload"
                            ? "bg-brand-primary text-text-inverse"
                            : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                        }`}
                      >
                        Upload
                      </button>
                    </div>
                  </div>

                  {/* Duration */}
                  <div>
                    <label className="text-xs text-text-muted font-medium">
                      Duration (seconds)
                    </label>
                    <input
                      type="number"
                      value={iperfSettings.duration}
                      onChange={(e) =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          duration: parseInt(e.target.value) || 10,
                        }))
                      }
                      min={1}
                      max={60}
                      className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>

                  {/* Server Mode */}
                  <div className="border-t border-surface-border pt-3">
                    <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border mb-2">
                      <span className="text-sm text-text-primary">
                        Enable iperf3 Server
                      </span>
                      <input
                        type="checkbox"
                        checked={iperfSettings.enableServer}
                        onChange={(e) =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            enableServer: e.target.checked,
                          }))
                        }
                        className="w-4 h-4"
                      />
                    </label>
                    <div>
                      <label className="text-xs text-text-muted font-medium">
                        Server Port
                      </label>
                      <input
                        type="number"
                        value={iperfSettings.serverPort}
                        onChange={(e) =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            serverPort: parseInt(e.target.value) || 5201,
                          }))
                        }
                        className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                      />
                    </div>
                    <p className="text-xs text-text-muted mt-1">
                      When enabled, starts iperf3 server automatically
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </CollapsibleSection>

          {/* Network Discovery Section */}
          <CollapsibleSection
            title={
              <>
                Network Discovery
                <AutoSaveIndicator status={networkDiscoveryStatus} />
              </>
            }
          >
            <div className="space-y-4">
              <p className="text-xs text-text-muted">
                Configure ARP-based device discovery for finding devices on the
                local network.
              </p>

              {/* Enable Discovery */}
              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">
                  Enable Discovery
                </span>
                <input
                  type="checkbox"
                  checked={networkDiscoverySettings.enabled}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      enabled: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              {/* Scan Workers */}
              <div>
                <label className="text-xs text-text-muted font-medium">
                  Concurrent Scan Workers
                </label>
                <input
                  type="number"
                  value={networkDiscoverySettings.arpScanWorkers}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      arpScanWorkers: parseInt(e.target.value) || 50,
                    }))
                  }
                  min={1}
                  max={100}
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  More workers = faster scan (default: 50)
                </p>
              </div>

              {/* Ping Timeout */}
              <div>
                <label className="text-xs text-text-muted font-medium">
                  Ping Timeout (ms)
                </label>
                <input
                  type="number"
                  value={networkDiscoverySettings.pingTimeoutMs}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      pingTimeoutMs: parseInt(e.target.value) || 500,
                    }))
                  }
                  min={100}
                  max={5000}
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  Timeout per host ping (default: 500ms)
                </p>
              </div>

              {/* Scan Timeout */}
              <div>
                <label className="text-xs text-text-muted font-medium">
                  Total Scan Timeout (ms)
                </label>
                <input
                  type="number"
                  value={networkDiscoverySettings.scanTimeoutMs}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      scanTimeoutMs: parseInt(e.target.value) || 30000,
                    }))
                  }
                  min={5000}
                  max={120000}
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  Max time for entire scan (default: 30s)
                </p>
              </div>

              {/* Scan Interval */}
              <div>
                <label className="text-xs text-text-muted font-medium">
                  Auto-Scan Interval (ms)
                </label>
                <input
                  type="number"
                  value={networkDiscoverySettings.scanIntervalMs}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      scanIntervalMs: parseInt(e.target.value) || 0,
                    }))
                  }
                  min={0}
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  0 = disabled, otherwise interval between automatic scans
                </p>
              </div>

              {/* OUI File Path */}
              <div>
                <label className="text-xs text-text-muted font-medium">
                  OUI Database File Path
                </label>
                <input
                  type="text"
                  value={networkDiscoverySettings.ouiFilePath}
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      ouiFilePath: e.target.value,
                    }))
                  }
                  placeholder="oui.txt"
                  className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  Path to IEEE OUI file for vendor lookup (download from{" "}
                  <a
                    href="https://standards-oui.ieee.org/oui/oui.txt"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-brand-primary hover:underline"
                  >
                    IEEE
                  </a>
                  )
                </p>
              </div>

              {/* Additional Subnets */}
              <div className="border-t border-surface-border pt-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">
                    Additional Subnets{" "}
                    <AutoSaveIndicator status={subnetsStatus} />
                  </span>
                </div>
                <p className="text-xs text-text-muted mb-2">
                  Add subnets beyond the local interface to scan for devices
                  (e.g., server VLANs, remote networks).
                </p>

                {/* List of configured subnets */}
                {subnets.length > 0 && (
                  <div className="space-y-2 mb-3">
                    {subnets.map((subnet) => (
                      <div
                        key={subnet.cidr}
                        className="flex items-center justify-between p-2 bg-surface-base rounded border border-surface-border"
                      >
                        <div className="flex-1 min-w-0">
                          <div className="text-sm text-text-primary truncate">
                            {subnet.name || subnet.cidr}
                          </div>
                          <div className="text-xs text-text-muted">
                            {subnet.cidr}
                          </div>
                        </div>
                        <div className="flex items-center gap-2 ml-2">
                          <input
                            type="checkbox"
                            checked={subnet.enabled}
                            onChange={(e) =>
                              toggleSubnet(subnet.cidr, e.target.checked)
                            }
                            className="w-4 h-4"
                            title={
                              subnet.enabled
                                ? "Disable subnet"
                                : "Enable subnet"
                            }
                          />
                          <button
                            onClick={() => deleteSubnet(subnet.cidr)}
                            className="text-status-error hover:text-red-400 text-sm"
                            title="Remove subnet"
                          >
                            X
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* Add new subnet form */}
                <div className="space-y-2">
                  <input
                    type="text"
                    value={newSubnetCidr}
                    onChange={(e) => {
                      setNewSubnetCidr(e.target.value);
                      setSubnetError(null);
                    }}
                    placeholder="CIDR (e.g., 10.0.0.0/24)"
                    className="w-full px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                  />
                  <input
                    type="text"
                    value={newSubnetName}
                    onChange={(e) => setNewSubnetName(e.target.value)}
                    placeholder="Name (optional, e.g., Server VLAN)"
                    className="w-full px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                  />
                  {subnetError && (
                    <p className="text-xs text-status-error">{subnetError}</p>
                  )}
                  <button
                    onClick={addSubnet}
                    className="w-full px-3 py-2 bg-brand-primary hover:bg-brand-accent text-white rounded text-sm"
                  >
                    + Add Subnet
                  </button>
                </div>
              </div>
            </div>
          </CollapsibleSection>

          {/* Thresholds Section */}
          <CollapsibleSection
            title={
              <>
                Thresholds
                <AutoSaveIndicator status={thresholdsStatus} />
              </>
            }
          >
            <div className="space-y-3">
              {/* DNS Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">
                  DNS Lookup (ms)
                </span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">
                      Good (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.dns.good}
                      onChange={(e) =>
                        updateThreshold("dns", "good", Number(e.target.value))
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">
                      Warning (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.dns.warning}
                      onChange={(e) =>
                        updateThreshold(
                          "dns",
                          "warning",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Gateway Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">
                  Gateway Ping (ms)
                </span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">
                      Good (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.gateway.good}
                      onChange={(e) =>
                        updateThreshold(
                          "gateway",
                          "good",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">
                      Warning (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.gateway.warning}
                      onChange={(e) =>
                        updateThreshold(
                          "gateway",
                          "warning",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Wi-Fi Signal Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">
                  Wi-Fi Signal (dBm)
                </span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">
                      Good (&gt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.wifi.good}
                      onChange={(e) =>
                        updateThreshold("wifi", "good", Number(e.target.value))
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">
                      Warning (&gt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.wifi.warning}
                      onChange={(e) =>
                        updateThreshold(
                          "wifi",
                          "warning",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Health Check Ping Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">
                  Health Check: Ping (ms)
                </span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">
                      Good (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.customPing.good}
                      onChange={(e) =>
                        updateThreshold(
                          "customPing",
                          "good",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">
                      Warning (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.customPing.warning}
                      onChange={(e) =>
                        updateThreshold(
                          "customPing",
                          "warning",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Health Check TCP Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">
                  Health Check: TCP (ms)
                </span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">
                      Good (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.customTcp.good}
                      onChange={(e) =>
                        updateThreshold(
                          "customTcp",
                          "good",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">
                      Warning (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.customTcp.warning}
                      onChange={(e) =>
                        updateThreshold(
                          "customTcp",
                          "warning",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Health Check HTTP Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">
                  Health Check: HTTP (ms)
                </span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">
                      Good (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.customHttp.good}
                      onChange={(e) =>
                        updateThreshold(
                          "customHttp",
                          "good",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">
                      Warning (&lt;)
                    </label>
                    <input
                      type="number"
                      value={thresholds.customHttp.warning}
                      onChange={(e) =>
                        updateThreshold(
                          "customHttp",
                          "warning",
                          Number(e.target.value),
                        )
                      }
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>
            </div>
          </CollapsibleSection>

          {/* FAB Options Section */}
          <CollapsibleSection
            title={
              <>
                Run All Tests (FAB)
                <AutoSaveIndicator status={fabStatus} />
              </>
            }
          >
            <div className="space-y-3">
              <p className="text-xs text-text-muted">
                Configure which tests run when the FAB button is pressed. Order
                matches card display.
              </p>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">Link</span>
                <input
                  type="checkbox"
                  checked={fabOptions.runLink}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runLink: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">
                  Nearest Switch
                </span>
                <input
                  type="checkbox"
                  checked={fabOptions.runSwitch}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runSwitch: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">VLAN</span>
                <input
                  type="checkbox"
                  checked={fabOptions.runVLAN}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runVLAN: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">IP Config</span>
                <input
                  type="checkbox"
                  checked={fabOptions.runIPConfig}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runIPConfig: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">Gateway</span>
                <input
                  type="checkbox"
                  checked={fabOptions.runGateway}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runGateway: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">DNS</span>
                <input
                  type="checkbox"
                  checked={fabOptions.runDNS}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runDNS: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">Health Checks</span>
                <input
                  type="checkbox"
                  checked={fabOptions.runHealthChecks}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runHealthChecks: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <p className="text-xs text-text-muted font-medium pt-2">
                Performance Tests
              </p>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border ml-3">
                <div>
                  <span className="text-sm text-text-primary">
                    Internet Speed
                  </span>
                  <p className="text-xs text-text-muted">Uses bandwidth</p>
                </div>
                <input
                  type="checkbox"
                  checked={fabOptions.runSpeedtest}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runSpeedtest: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border ml-3">
                <div>
                  <span className="text-sm text-text-primary">
                    LAN Speed (iperf3)
                  </span>
                  <p className="text-xs text-text-muted">Requires server</p>
                </div>
                <input
                  type="checkbox"
                  checked={fabOptions.runIperf}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runIperf: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border">
                <div>
                  <span className="text-sm text-text-primary">
                    Network Discovery
                  </span>
                  <p className="text-xs text-text-muted">Scan for devices</p>
                </div>
                <input
                  type="checkbox"
                  checked={fabOptions.runNetworkDiscovery}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      runNetworkDiscovery: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <label className="flex items-center justify-between p-2.5 bg-surface-base rounded border border-surface-border ml-3">
                <div>
                  <span className="text-sm text-text-primary">
                    Auto-Scan on Link
                  </span>
                  <p className="text-xs text-text-muted">
                    Scan when interface comes up
                  </p>
                </div>
                <input
                  type="checkbox"
                  checked={fabOptions.autoScanOnLink}
                  onChange={(e) =>
                    setFabOptions((prev) => ({
                      ...prev,
                      autoScanOnLink: e.target.checked,
                    }))
                  }
                  className="w-4 h-4"
                  disabled={!fabOptions.runNetworkDiscovery}
                />
              </label>
            </div>
          </CollapsibleSection>

          {/* Appearance Section */}
          <CollapsibleSection title="Appearance">
            <div className="space-y-2">
              <label className="flex items-center justify-between p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">Theme</span>
                <select
                  value={theme}
                  onChange={(e) =>
                    setTheme(e.target.value as "light" | "dark" | "system")
                  }
                  className="bg-surface-raised border border-surface-border rounded px-2 py-1 text-sm text-text-primary"
                >
                  <option value="light">Light</option>
                  <option value="dark">Dark</option>
                  <option value="system">System</option>
                </select>
              </label>

              <button
                onClick={() => setTheme(isDark ? "light" : "dark")}
                className="w-full flex items-center justify-between p-3 bg-surface-base rounded border border-surface-border hover:bg-surface-hover transition-colors"
              >
                <span className="text-sm text-text-primary">Quick Toggle</span>
                <span className="text-xl">{isDark ? "🌙" : "☀️"}</span>
              </button>
            </div>
          </CollapsibleSection>

          {/* Logs (debug) */}
          <section className="pt-4 border-t border-surface-border">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="text-sm font-medium text-text-muted">
                  Logs (debug)
                </h3>
                <p className="text-xs text-text-muted">
                  Rotating file; use only for troubleshooting.
                </p>
              </div>
              <button
                onClick={fetchLogPreview}
                className="text-xs px-3 py-1 border border-surface-border rounded text-text-muted hover:text-text-primary hover:border-text-muted transition-colors"
              >
                {logLoading ? "Loading…" : "View"}
              </button>
            </div>
            {logError && (
              <p className="text-xs text-status-error mt-2">{logError}</p>
            )}
            {!logError && logPreview.length > 0 && (
              <pre className="mt-2 max-h-48 overflow-y-auto text-[11px] leading-5 bg-surface-base border border-surface-border rounded px-3 py-2 text-text-primary whitespace-pre-wrap">
                {logPreview.join("\n")}
              </pre>
            )}
          </section>

          {/* Export Section */}
          <section className="pt-4 border-t border-surface-border">
            <h3 className="text-sm font-medium text-text-muted mb-3">Export</h3>
            <a
              href={`${API_BASE}/api/export`}
              download="netscope-export.json"
              className="w-full py-2 px-4 bg-surface-base border border-surface-border text-text-primary rounded font-medium hover:bg-surface-hover transition-colors flex items-center justify-center gap-2 touch-manipulation"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              Download JSON Export
            </a>
            <p className="text-xs text-text-muted mt-2">
              Export all diagnostic data as JSON for documentation or analysis.
            </p>
          </section>

          {/* About Section */}
          <section className="pt-4 border-t border-surface-border">
            <h3 className="text-sm font-medium text-text-muted mb-2">About</h3>
            <p className="text-xs text-text-muted">
              NetScope v0.8.7
              <br />
              Network Diagnostic Tool
            </p>
          </section>
        </div>
      </div>
    </>
  );
}
