import { useState, useEffect, useCallback, useRef, memo } from "react";
import { useTheme } from "../../hooks/useTheme";
import { getAuthHeaders } from "../../hooks/useAuth";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import {
  AutoSaveIndicator,
  AppearanceSettings,
  DiscoverySettings,
  DNSSettings,
  FABOptionsSettings,
  HealthChecksSettings,
  PerformanceSettings,
  ThresholdsSettings,
  WiFiSettings,
} from "./sections";

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface ThresholdPair {
  good: number;
  warning: number;
}

interface Thresholds {
  dns: ThresholdPair;
  gateway: ThresholdPair;
  wifi: ThresholdPair;
  customPing: ThresholdPair;
  customTcp: ThresholdPair;
  customHttp: ThresholdPair;
  httpTimings: {
    dns: ThresholdPair;
    tcp: ThresholdPair;
    tls: ThresholdPair;
    ttfb: ThresholdPair;
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
  runLink: boolean;
  runSwitch: boolean;
  runVLAN: boolean;
  runIPConfig: boolean;
  runGateway: boolean;
  runDNS: boolean;
  runHealthChecks: boolean;
  runNetworkDiscovery: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runPerformance: boolean;
  autoScanOnLink: boolean;
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
  runPerformance: boolean; // kept for backwards compatibility
  runSpeedtest: boolean;
  runIperf: boolean;
  runDiscovery: boolean;
  speedtest: {
    serverId: string;
    autoRunOnLink: boolean;
  };
  iperf: {
    autoRunOnLink: boolean;
  };
}

interface IperfSettings {
  server: string;
  port: number;
  protocol: "tcp" | "udp";
  direction: "upload" | "download" | "bidirectional";
  duration: number;
  serverPort: number;
  enableServer: boolean;
}

interface IperfSuggestion {
  host: string;
  hostname?: string;
  latencyMs?: number;
  source?: string;
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

// VLANControl component for creating/deleting VLAN subinterfaces
const VLANControl = memo(function VLANControl() {
  const [vlanId, setVlanId] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleCreate = async () => {
    const id = parseInt(vlanId, 10);
    if (isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: "VLAN ID must be 1-4094", isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/vlan/interface`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ vlanId: id }),
      });
      if (response.ok) {
        setMessage({ text: `VLAN ${id} created`, isError: false });
        setVlanId("");
      } else {
        const text = await response.text();
        setMessage({ text: text || "Failed to create VLAN", isError: true });
      }
    } catch {
      setMessage({ text: "Network error", isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  const handleDelete = async () => {
    const id = parseInt(vlanId, 10);
    if (isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: "VLAN ID must be 1-4094", isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/vlan/interface`, {
        method: "DELETE",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ vlanId: id }),
      });
      if (response.ok) {
        setMessage({ text: `VLAN ${id} deleted`, isError: false });
        setVlanId("");
      } else {
        const text = await response.text();
        setMessage({ text: text || "Failed to delete VLAN", isError: true });
      }
    } catch {
      setMessage({ text: "Network error", isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex gap-2">
        <input
          type="number"
          min="1"
          max="4094"
          value={vlanId}
          onChange={(e) => setVlanId(e.target.value)}
          placeholder="VLAN ID (1-4094)"
          className="flex-1 px-2 py-1.5 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          disabled={loading}
        />
        <button
          onClick={handleCreate}
          disabled={loading || !vlanId}
          className="px-3 py-1.5 bg-brand-primary text-text-inverse rounded text-sm font-medium hover:bg-brand-accent disabled:opacity-50"
        >
          Add
        </button>
        <button
          onClick={handleDelete}
          disabled={loading || !vlanId}
          className="px-3 py-1.5 bg-status-error text-text-inverse rounded text-sm font-medium hover:opacity-80 disabled:opacity-50"
        >
          Remove
        </button>
      </div>
      {message && (
        <p
          className={`text-xs ${message.isError ? "text-status-error" : "text-status-success"}`}
        >
          {message.text}
        </p>
      )}
      <p className="text-xs text-text-muted">
        Creates/removes 802.1Q VLAN subinterface. Requires root.
      </p>
    </div>
  );
});

// MTUControl component for setting interface MTU
const MTUControl = memo(function MTUControl() {
  const [mtu, setMtu] = useState("1500");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleApply = async () => {
    const mtuVal = parseInt(mtu, 10);
    if (isNaN(mtuVal) || mtuVal < 68 || mtuVal > 9000) {
      setMessage({ text: "MTU must be 68-9000", isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/network/mtu`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ mtu: mtuVal }),
      });
      if (response.ok) {
        setMessage({ text: `MTU set to ${mtuVal}`, isError: false });
      } else {
        const text = await response.text();
        setMessage({ text: text || "Failed to set MTU", isError: true });
      }
    } catch {
      setMessage({ text: "Network error", isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex gap-2">
        <input
          type="number"
          min="68"
          max="9000"
          value={mtu}
          onChange={(e) => setMtu(e.target.value)}
          placeholder="1500"
          className="flex-1 px-2 py-1.5 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          disabled={loading}
        />
        <button
          onClick={handleApply}
          disabled={loading}
          className="px-4 py-1.5 bg-brand-primary text-text-inverse rounded text-sm font-medium hover:bg-brand-accent disabled:opacity-50"
        >
          {loading ? "Applying..." : "Apply"}
        </button>
      </div>
      {message && (
        <p
          className={`text-xs ${message.isError ? "text-status-error" : "text-status-success"}`}
        >
          {message.text}
        </p>
      )}
      <p className="text-xs text-text-muted">
        Standard: 1500, Jumbo frames: up to 9000. Requires root.
      </p>
    </div>
  );
});

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  version?: string;
}

export const SettingsDrawer = memo(function SettingsDrawer({
  isOpen,
  onClose,
  version = "dev",
}: SettingsDrawerProps) {
  const { theme, setTheme, isDark } = useTheme();
  const scrollRef = useRef<HTMLDivElement | null>(null);
  // Track if health check settings were modified - dispatch event on drawer close
  const testsSettingsChangedRef = useRef(false);
  const prevIsOpenRef = useRef(isOpen);

  // Smooth scroll to top when opened
  useEffect(() => {
    if (isOpen && scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [isOpen]);

  // When drawer closes, dispatch healthChecksUpdated if test settings changed
  useEffect(() => {
    if (prevIsOpenRef.current && !isOpen && testsSettingsChangedRef.current) {
      // Drawer just closed and test settings were changed
      window.dispatchEvent(new CustomEvent("healthChecksUpdated"));
      testsSettingsChangedRef.current = false;
    }
    prevIsOpenRef.current = isOpen;
  }, [isOpen]);
  const [thresholds, setThresholds] = useState<Thresholds>({
    dns: { good: 50, warning: 100 },
    gateway: { good: 20, warning: 50 },
    wifi: { good: -50, warning: -70 },
    customPing: { good: 50, warning: 100 },
    customTcp: { good: 100, warning: 500 },
    customHttp: { good: 500, warning: 2000 },
    httpTimings: {
      dns: { good: 100, warning: 500 },
      tcp: { good: 100, warning: 500 },
      tls: { good: 150, warning: 500 },
      ttfb: { good: 500, warning: 2000 },
    },
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
    runPerformance: true,
    runSpeedtest: true,
    runIperf: true,
    runDiscovery: true,
    speedtest: {
      serverId: "",
      autoRunOnLink: false,
    },
    iperf: {
      autoRunOnLink: false,
    },
  });

  // FAB Options (stored in localStorage)
  const [fabOptions, setFabOptions] = useState<FABOptions>({
    runLink: true,
    runSwitch: true,
    runVLAN: true,
    runIPConfig: true,
    runGateway: true,
    runDNS: true,
    runHealthChecks: true,
    runNetworkDiscovery: true,
    runSpeedtest: true,
    runIperf: true,
    runPerformance: true,
    autoScanOnLink: true,
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
  const [iperfSuggestions, setIperfSuggestions] = useState<IperfSuggestion[]>(
    [],
  );
  const [iperfSuggestionsStatus, setIperfSuggestionsStatus] = useState<
    "idle" | "loading" | "error"
  >("idle");
  const [iperfSuggestionsError, setIperfSuggestionsError] = useState<
    string | null
  >(null);
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
          runPerformance: data.runPerformance ?? true,
          runSpeedtest: data.runSpeedtest ?? true,
          runIperf: data.runIperf ?? true,
          runDiscovery: data.runDiscovery ?? true,
          speedtest: {
            serverId: data.speedtest?.serverId || "",
            autoRunOnLink: data.speedtest?.autoRunOnLink || false,
          },
          iperf: {
            autoRunOnLink: data.iperf?.autoRunOnLink || false,
          },
        });
      }
    } catch (err) {
      console.error("Failed to fetch tests settings:", err);
    }
  }, []);

  const fetchIperfSuggestions = useCallback(async () => {
    setIperfSuggestionsStatus("loading");
    setIperfSuggestionsError(null);
    try {
      const response = await fetch(`${API_BASE}/api/iperf/suggestions`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setIperfSuggestions(Array.isArray(data) ? data : []);
        setIperfSuggestionsStatus("idle");
      } else {
        setIperfSuggestionsStatus("error");
        setIperfSuggestionsError("No iperf hosts found");
      }
    } catch (err) {
      setIperfSuggestionsStatus("error");
      setIperfSuggestionsError(
        err instanceof Error ? err.message : "Failed to find iperf hosts",
      );
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
    const logToken = import.meta.env.VITE_LOG_ACCESS_TOKEN;
    const logHeader = import.meta.env.VITE_LOG_ACCESS_HEADER || "X-Log-Token";
    try {
      const response = await fetch(`${API_BASE}/api/logs?lines=200`, {
        headers: {
          ...getAuthHeaders(),
          ...(logToken ? { [logHeader]: logToken } : {}),
        },
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
      const payload = { ...testsSettings };
      const response = await fetch(`${API_BASE}/api/tests/settings`, {
        method: "PUT",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });
      if (response.ok) {
        setTestsStatus("saved");
        setTimeout(() => setTestsStatus("idle"), 2000);
        // Mark that test settings changed - event dispatched on drawer close
        testsSettingsChangedRef.current = true;
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
      autoScan: fabOptions.autoScanOnLink,
    }));
  }, [fabOptions.autoScanOnLink]);

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
            {/* Network Configuration */}
            <div className="space-y-3">
              <p className="text-xs uppercase tracking-wide text-text-muted font-semibold">
                Network Configuration
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
                    Display in Network card
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

            {/* VLAN Configuration */}
            <div className="border-t border-surface-border pt-3 mt-3">
              <p className="text-xs uppercase tracking-wide text-text-muted font-semibold mb-2">
                VLAN Tag (802.1Q)
              </p>
              <VLANControl />
            </div>

            {/* MTU Configuration */}
            <div className="border-t border-surface-border pt-3 mt-3">
              <p className="text-xs uppercase tracking-wide text-text-muted font-semibold mb-2">
                MTU Setting
              </p>
              <MTUControl />
            </div>
          </CollapsibleSection>

          <WiFiSettings
            wifiSettings={wifiSettings}
            setWifiSettings={setWifiSettings}
            wifiStatus={wifiStatus}
          />

          <DNSSettings
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
            testsStatus={testsStatus}
          />

          <HealthChecksSettings
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
            testsStatus={testsStatus}
          />

          <PerformanceSettings
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
            iperfSettings={iperfSettings}
            setIperfSettings={setIperfSettings}
            iperfStatus={iperfStatus}
            iperfSuggestions={iperfSuggestions}
            iperfSuggestionsStatus={iperfSuggestionsStatus}
            iperfSuggestionsError={iperfSuggestionsError}
            fetchIperfSuggestions={fetchIperfSuggestions}
          />

          <DiscoverySettings
            networkDiscoverySettings={networkDiscoverySettings}
            setNetworkDiscoverySettings={setNetworkDiscoverySettings}
            networkDiscoveryStatus={networkDiscoveryStatus}
            subnets={subnets}
            subnetsStatus={subnetsStatus}
            newSubnetCidr={newSubnetCidr}
            setNewSubnetCidr={setNewSubnetCidr}
            newSubnetName={newSubnetName}
            setNewSubnetName={setNewSubnetName}
            subnetError={subnetError}
            setSubnetError={setSubnetError}
            addSubnet={addSubnet}
            toggleSubnet={toggleSubnet}
            deleteSubnet={deleteSubnet}
          />

          <ThresholdsSettings
            thresholds={thresholds}
            setThresholds={setThresholds}
            thresholdsStatus={thresholdsStatus}
          />

          {/* FAB Options Section */}
          <FABOptionsSettings
            fabOptions={fabOptions}
            setFabOptions={setFabOptions}
            fabStatus={fabStatus}
          />

          {/* Appearance Section */}
          <AppearanceSettings
            theme={theme}
            setTheme={setTheme}
            isDark={isDark}
          />

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
              <pre className="mt-2 max-h-48 overflow-y-auto text-2xs leading-5 bg-surface-base border border-surface-border rounded px-3 py-2 text-text-primary whitespace-pre-wrap">
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
              NetScope {version}
              <br />
              Network Diagnostic Tool
            </p>
          </section>
        </div>
      </div>
    </>
  );
});
