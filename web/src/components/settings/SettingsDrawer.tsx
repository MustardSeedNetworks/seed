/**
 * SettingsDrawer Component (~1399 lines)
 *
 * Purpose: Master settings configuration panel managing all application settings.
 * Provides tabbed interface to Appearance, Discovery, Network, Performance, Health,
 * DNS, WiFi, SNMP, and Thresholds configuration sections with persistent storage.
 *
 * Key Features:
 * - Multi-section settings: Appearance, Discovery, Network, Performance, Health, DNS, WiFi, SNMP, Thresholds
 * - Persistent storage: Auto-saves settings to server with save status indication
 * - Real-time updates: Changes immediately reflected in UI
 * - Validation: Input validation before saving
 * - Import/Export: Settings backup and restore functionality
 * - Advanced options: Expandable sections for power users
 * - Save status: Shows success/error/unsaved indicators
 * - iperf3 suggestions: Autocomplete for iperf3 server discovery
 * - Keyboard shortcuts: Enter to save, ESC to close
 *
 * Usage:
 * ```typescript
 * <SettingsDrawer isOpen={showSettings} onClose={handleClose} />
 * ```
 *
 * Dependencies: useTheme, useAuth, useSettings context, all settings section components
 * State: All settings state, save status, validation errors, iperf3 suggestions
 */

import { useState, useEffect, useCallback, useRef, memo } from "react";
import { useTranslation } from "react-i18next";
import { useTheme } from "../../hooks/useTheme";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { useSettings } from "../../contexts/useSettings";
import { logger, LogComponents } from "../../lib/logger";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import {
  icon as iconTokens,
  radius,
  modal,
  layout,
  button,
  input,
  spacing,
} from "../../styles/theme";
import {
  AutoSaveIndicator,
  AppearanceSettings,
  ConfigBackupsSection,
  DiscoverySettings,
  DNSSettings,
  HealthChecksSettings,
  PerformanceSettings,
  ThresholdsSettings,
  WiFiSettings,
} from "./sections";
import type {
  SettingsThresholds,
  WiFiSettings as WiFiSettingsType,
  IPSettings,
  LogsResponse,
  TestsSettings,
  IperfSuggestion,
  NetworkDiscoverySettings,
  SubnetConfig,
  SNMPSettings as SNMPSettingsType,
} from "../../types/settings";
import { generateId } from "../../utils/id";

const API_BASE = import.meta.env.VITE_API_BASE || "";

// Utility: ensure every item in an array has a stable id for React keying/updating.
const withIds = <T extends { id?: string }>(
  items: T[] = []
): Array<T & { id: string }> =>
  items.map((item) => ({ ...item, id: item.id ?? generateId() }));

// Normalize tests/DNS settings payload before sending to the API.
const normalizeTestsSettingsForSave = (settings: TestsSettings) => {
  const dnsHostname = settings.dnsHostname?.trim() || "google.com";

  const dnsServers = (settings.dnsServers || [])
    .map((server) => ({
      address: server.address.trim(),
      enabled: server.enabled !== false,
    }))
    .filter((server) => server.address.length > 0);

  const pingTargets = (settings.pingTargets || [])
    .map((target) => ({
      name: target.name?.trim() || target.host.trim(),
      host: target.host.trim(),
      enabled: target.enabled !== false,
    }))
    .filter((target) => target.host.length > 0);

  const tcpPorts = (settings.tcpPorts || [])
    .map((port) => ({
      name: port.name?.trim() || port.host.trim(),
      host: port.host.trim(),
      port:
        typeof port.port === "number"
          ? port.port
          : parseInt(String(port.port), 10) || 0,
      enabled: port.enabled !== false,
    }))
    .filter((port) => port.host.length > 0 && port.port > 0);

  const udpPorts = (settings.udpPorts || [])
    .map((port) => ({
      name: port.name?.trim() || port.host.trim(),
      host: port.host.trim(),
      port:
        typeof port.port === "number"
          ? port.port
          : parseInt(String(port.port), 10) || 0,
      enabled: port.enabled !== false,
    }))
    .filter((port) => port.host.length > 0 && port.port > 0);

  const httpEndpoints = (settings.httpEndpoints || [])
    .map((endpoint) => ({
      name: endpoint.name?.trim() || endpoint.url.trim(),
      url: endpoint.url.trim(),
      expectedStatus:
        typeof endpoint.expectedStatus === "number" &&
        endpoint.expectedStatus > 0
          ? endpoint.expectedStatus
          : 200,
      enabled: endpoint.enabled !== false,
    }))
    .filter((endpoint) => endpoint.url.length > 0);

  return {
    dnsHostname,
    dnsServers,
    pingTargets,
    tcpPorts,
    udpPorts,
    httpEndpoints,
    runPerformance: settings.runPerformance !== false,
    runSpeedtest: settings.runSpeedtest !== false,
    runIperf: settings.runIperf !== false,
    runDiscovery: settings.runDiscovery !== false,
    speedtest: {
      serverId: settings.speedtest?.serverId?.trim() || "",
      autoRunOnLink: !!settings.speedtest?.autoRunOnLink,
    },
    iperf: {
      autoRunOnLink: !!settings.iperf?.autoRunOnLink,
    },
  };
};

// VLANControl component for creating/deleting VLAN subinterfaces
const VLANControl = memo(function VLANControl() {
  const { t } = useTranslation("settings");
  const [vlanId, setVlanId] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleCreate = async () => {
    const id = parseInt(vlanId, 10);
    if (isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: t("network.vlan.invalidId"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/vlan/interface`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ vlanId: id }),
      });
      if (response.ok) {
        setMessage({ text: t("network.vlan.created", { id }), isError: false });
        setVlanId("");
      } else {
        const text = await response.text();
        setMessage({
          text: text || t("network.vlan.createFailed"),
          isError: true,
        });
      }
    } catch {
      setMessage({ text: t("network.vlan.networkError"), isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  const handleDelete = async () => {
    const id = parseInt(vlanId, 10);
    if (isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: t("network.vlan.invalidId"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/vlan/interface`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ vlanId: id }),
      });
      if (response.ok) {
        setMessage({ text: t("network.vlan.deleted", { id }), isError: false });
        setVlanId("");
      } else {
        const text = await response.text();
        setMessage({
          text: text || t("network.vlan.deleteFailed"),
          isError: true,
        });
      }
    } catch {
      setMessage({ text: t("network.vlan.networkError"), isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  return (
    <div className="stack-sm">
      <div className={layout.inline.default}>
        <input
          type="number"
          min="1"
          max="4094"
          value={vlanId}
          onChange={(e) => setVlanId(e.target.value)}
          placeholder={t("network.vlan.placeholder")}
          className={`flex-1 ${input.size.sm} bg-surface-base border border-surface-border ${radius.md} body-small text-text-primary`}
          disabled={loading}
        />
        <button
          onClick={handleCreate}
          disabled={loading || !vlanId}
          className={`${button.size.sm} bg-brand-primary text-text-inverse ${radius.md} body-small font-medium hover:bg-brand-accent disabled:opacity-50`}
        >
          {t("network.vlan.add")}
        </button>
        <button
          onClick={handleDelete}
          disabled={loading || !vlanId}
          className={`${button.size.sm} bg-status-error text-text-inverse ${radius.md} body-small font-medium hover:opacity-80 disabled:opacity-50`}
        >
          {t("network.vlan.remove")}
        </button>
      </div>
      {message && (
        <p
          className={`caption ${message.isError ? "text-status-error" : "text-status-success"}`}
        >
          {message.text}
        </p>
      )}
      <p className="caption">{t("network.vlan.description")}</p>
    </div>
  );
});

// MTUControl component for setting interface MTU
const MTUControl = memo(function MTUControl() {
  const { t } = useTranslation("settings");
  const [mtu, setMtu] = useState("1500");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleApply = async () => {
    const mtuVal = parseInt(mtu, 10);
    if (isNaN(mtuVal) || mtuVal < 68 || mtuVal > 9000) {
      setMessage({ text: t("network.mtuControl.invalidRange"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/network/mtu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ mtu: mtuVal }),
      });
      if (response.ok) {
        setMessage({
          text: t("network.mtuControl.setSuccess", { value: mtuVal }),
          isError: false,
        });
      } else {
        const text = await response.text();
        setMessage({
          text: text || t("network.mtuControl.setFailed"),
          isError: true,
        });
      }
    } catch {
      setMessage({ text: t("network.mtuControl.networkError"), isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  return (
    <div className="stack-sm">
      <div className={layout.inline.default}>
        <input
          type="number"
          min="68"
          max="9000"
          value={mtu}
          onChange={(e) => setMtu(e.target.value)}
          placeholder={t("network.mtuControl.placeholder")}
          className={`flex-1 ${input.size.sm} bg-surface-base border border-surface-border ${radius.md} body-small text-text-primary`}
          disabled={loading}
        />
        <button
          onClick={handleApply}
          disabled={loading}
          className={`${button.size.md} bg-brand-primary text-text-inverse ${radius.md} body-small font-medium hover:bg-brand-accent disabled:opacity-50`}
        >
          {loading ? t("network.applying") : t("network.mtuControl.apply")}
        </button>
      </div>
      {message && (
        <p
          className={`caption ${message.isError ? "text-status-error" : "text-status-success"}`}
        >
          {message.text}
        </p>
      )}
      <p className="caption">{t("network.mtuControl.description")}</p>
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
  const { t } = useTranslation("settings");
  const { theme, setTheme, isDark } = useTheme();

  // Get settings from context - single source of truth
  const {
    displayOptions,
    iperfSettings,
    status: settingsStatus,
    updateDisplayOptions,
    updateIperfSettings,
  } = useSettings();

  // Create setter wrappers that use context update methods
  const setDisplayOptions = useCallback(
    (updater: React.SetStateAction<typeof displayOptions>) => {
      const newValue =
        typeof updater === "function" ? updater(displayOptions) : updater;
      updateDisplayOptions(newValue);
    },
    [displayOptions, updateDisplayOptions]
  );

  const setIperfSettings = useCallback(
    (updater: React.SetStateAction<typeof iperfSettings>) => {
      const newValue =
        typeof updater === "function" ? updater(iperfSettings) : updater;
      updateIperfSettings(newValue);
    },
    [iperfSettings, updateIperfSettings]
  );

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
  const [thresholds, setThresholds] = useState<SettingsThresholds>({
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

  // FAB Options, Display Options, and iperf Settings now come from SettingsContext above

  const [wifiSettings, setWifiSettings] = useState<WiFiSettingsType>({
    interface: "",
    availableWifi: [],
    isWireless: false,
  });
  const [dnsInput, setDnsInput] = useState("");
  const [iperfSuggestions, setIperfSuggestions] = useState<IperfSuggestion[]>(
    []
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
      profile: "standard",
      arpScanWorkers: 50,
      pingTimeoutMs: 500,
      scanTimeoutMs: 30000,
      autoScan: false,
      scanIntervalMs: 0,
      ouiFilePath: "oui.txt",
      customOptions: {
        passiveListen: true,
        arpScan: true,
        icmpScan: true,
        portScan: {
          enabled: false,
          ports: [],
          topPorts: 100,
        },
        traceroute: false,
        snmpQuery: false,
      },
    });
  // SNMP settings
  const [snmpSettings, setSnmpSettings] = useState<SNMPSettingsType>({
    communities: ["public"],
    v3Credentials: [],
    timeout: 5000,
    retries: 2,
    port: 161,
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
  const [snmpStatus, setSnmpStatus] = useState<SaveStatus>("idle");
  // Status for display, iperf comes from context (settingsStatus)
  const displayStatus = settingsStatus.display;
  const iperfStatus = settingsStatus.iperf;

  const [networkDiscoveryStatus, setNetworkDiscoveryStatus] =
    useState<SaveStatus>("idle");

  // Refs to track initial load (skip auto-save on first load)
  const initialLoadRef = useRef(true);
  const thresholdsInitRef = useRef(true);
  const testsInitRef = useRef(true);
  const wifiInitRef = useRef(true);
  const networkDiscoveryInitRef = useRef(true);
  const snmpInitRef = useRef(true);

  // Debounce timers (fab, display, iperf now handled by context)
  const thresholdsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const testsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wifiTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(
    null
  );
  const snmpTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Legacy state (keep for IP settings which still needs manual apply)
  const [savingIP, setSavingIP] = useState(false);
  const [ipMessage, setIPMessage] = useState<string | null>(null);

  // Fetch current thresholds
  const fetchThresholds = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        credentials: "include",
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
      logger.error(LogComponents.CONFIG, "Failed to fetch thresholds", err);
    }
  }, []);

  // Fetch current IP settings
  const fetchIPSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        credentials: "include",
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
      logger.error(LogComponents.CONFIG, "Failed to fetch IP settings", err);
    }
  }, []);

  // Fetch current tests settings
  const fetchTestsSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/health-checks/settings`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setTestsSettings({
          dnsHostname: data.dnsHostname || "google.com",
          dnsServers: withIds(data.dnsServers || []).map((server) => ({
            ...server,
            enabled: server.enabled !== false,
          })),
          pingTargets: withIds(data.pingTargets || []).map((target) => ({
            ...target,
            enabled: target.enabled !== false,
          })),
          tcpPorts: withIds(data.tcpPorts || []).map((port) => ({
            ...port,
            port: port.port || 80,
            enabled: port.enabled !== false,
          })),
          udpPorts: withIds(data.udpPorts || []).map((port) => ({
            ...port,
            port: port.port || 53,
            enabled: port.enabled !== false,
          })),
          httpEndpoints: withIds(data.httpEndpoints || []).map((endpoint) => ({
            ...endpoint,
            expectedStatus: endpoint.expectedStatus || 200,
            enabled: endpoint.enabled !== false,
          })),
          runPerformance: data.runPerformance ?? true,
          runSpeedtest: data.runSpeedtest ?? true,
          runIperf: data.runIperf ?? true,
          runDiscovery: data.runDiscovery ?? true,
          speedtest: {
            serverId: data.speedtest?.serverId || "",
            autoRunOnLink: data.speedtest?.autoRunOnLink ?? true, // Default to true
          },
          iperf: {
            autoRunOnLink: data.iperf?.autoRunOnLink || false,
          },
        });
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, "Failed to fetch tests settings", err);
    }
  }, []);

  const fetchIperfSuggestions = useCallback(async () => {
    setIperfSuggestionsStatus("loading");
    setIperfSuggestionsError(null);
    try {
      const response = await fetch(`${API_BASE}/api/iperf/suggestions`, {
        credentials: "include",
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
        err instanceof Error ? err.message : "Failed to find iperf hosts"
      );
    }
  }, []);

  // Fetch WiFi settings
  const fetchWifiSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/wifi/settings`, {
        credentials: "include",
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
      logger.error(LogComponents.WIFI, "Failed to fetch WiFi settings", err);
    }
  }, []);

  // FAB options, display options, and iperf settings now come from SettingsContext
  // (loaded automatically by the context provider)

  // Fetch Network Discovery settings from API
  const fetchNetworkDiscoverySettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/devices/settings`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setNetworkDiscoverySettings({
          enabled: data.enabled ?? true,
          profile: data.profile ?? "standard",
          arpScanWorkers: data.arpScanWorkers ?? 50,
          pingTimeoutMs: data.pingTimeoutMs ?? 500,
          scanTimeoutMs: data.scanTimeoutMs ?? 30000,
          autoScan: data.autoScan ?? false,
          scanIntervalMs: data.scanIntervalMs ?? 0,
          ouiFilePath: data.ouiFilePath ?? "oui.txt",
          customOptions: data.customOptions ?? {
            passiveListen: true,
            arpScan: true,
            icmpScan: true,
            portScan: {
              enabled: false,
              ports: [],
              topPorts: 100,
            },
            traceroute: false,
            snmpQuery: false,
          },
        });
      }
    } catch (err) {
      logger.error(
        LogComponents.DISCOVERY,
        "Failed to fetch network discovery settings",
        err
      );
    }
  }, []);

  // Fetch SNMP settings from API
  const fetchSNMPSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/snmp/settings`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setSnmpSettings({
          communities: data.communities ?? ["public"],
          v3Credentials: data.v3Credentials ?? [],
          timeout: data.timeout ?? 5000,
          retries: data.retries ?? 2,
          port: data.port ?? 161,
        });
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, "Failed to fetch SNMP settings", err);
    }
  }, []);

  // Fetch configured subnets from API
  const fetchSubnets = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/devices/subnets`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setSubnets(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      logger.error(LogComponents.DISCOVERY, "Failed to fetch subnets", err);
    }
  }, []);

  // Fetch a small tail of the application log (debug)
  // Security fix #301: Removed VITE_LOG_ACCESS_TOKEN - JWT authentication is sufficient
  const fetchLogPreview = useCallback(async () => {
    setLogLoading(true);
    setLogError(null);
    try {
      const response = await fetch(`${API_BASE}/api/logs?lines=200`, {
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error("Unable to load logs");
      }
      const data = (await response.json()) as LogsResponse;
      setLogPreview(data.lines || []);
    } catch (err) {
      setLogPreview([]);
      setLogError(
        err instanceof Error ? err.message : "Failed to load log file"
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
      setSubnetError(t("network.cidrRequired"));
      return;
    }

    setSubnetError(null);
    setSubnetsStatus("saving");

    try {
      const response = await fetch(`${API_BASE}/api/devices/subnets`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
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
        err instanceof Error ? err.message : "Network error adding subnet"
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
          "Content-Type": "application/json",
        },
        credentials: "include",
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
      // Backend expects CIDR as query parameter, not in body
      const response = await fetch(
        `${API_BASE}/api/devices/subnets?cidr=${encodeURIComponent(cidr)}`,
        {
          method: "DELETE",
          credentials: "include",
        }
      );

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
          "Content-Type": "application/json",
        },
        credentials: "include",
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

  const saveSNMPSettings = useCallback(async () => {
    setSnmpStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/snmp/settings`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(snmpSettings),
      });
      if (response.ok) {
        setSnmpStatus("saved");
        setTimeout(() => setSnmpStatus("idle"), 2000);
      } else {
        setSnmpStatus("error");
      }
    } catch {
      setSnmpStatus("error");
    }
  }, [snmpSettings]);

  useEffect(() => {
    if (isOpen) {
      // Reset init refs on open
      initialLoadRef.current = true;
      thresholdsInitRef.current = true;
      testsInitRef.current = true;
      wifiInitRef.current = true;
      networkDiscoveryInitRef.current = true;
      snmpInitRef.current = true;

      fetchThresholds();
      fetchIPSettings();
      fetchTestsSettings();
      fetchWifiSettings();
      // FAB options, display options, and iperf settings come from SettingsContext
      fetchNetworkDiscoverySettings();
      fetchSNMPSettings();
      fetchSubnets();

      // Mark initial load as done after a short delay
      setTimeout(() => {
        initialLoadRef.current = false;
        thresholdsInitRef.current = false;
        testsInitRef.current = false;
        wifiInitRef.current = false;
        networkDiscoveryInitRef.current = false;
        snmpInitRef.current = false;
      }, 500);
    }
  }, [
    isOpen,
    fetchThresholds,
    fetchIPSettings,
    fetchTestsSettings,
    fetchWifiSettings,
    fetchNetworkDiscoverySettings,
    fetchSNMPSettings,
    fetchSubnets,
  ]);

  const saveThresholds = useCallback(async () => {
    setThresholdsStatus("saving");
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
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
          "Content-Type": "application/json",
        },
        credentials: "include",
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
      const payload = normalizeTestsSettingsForSave(testsSettings);
      const response = await fetch(`${API_BASE}/api/health-checks/settings`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
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
          "Content-Type": "application/json",
        },
        credentials: "include",
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

  // Display options and iperf settings auto-save is handled by SettingsContext

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

  // Auto-save SNMP settings with debounce
  useEffect(() => {
    if (snmpInitRef.current) return;
    if (snmpTimerRef.current) clearTimeout(snmpTimerRef.current);
    snmpTimerRef.current = setTimeout(() => {
      saveSNMPSettings();
    }, 800);
    return () => {
      if (snmpTimerRef.current) clearTimeout(snmpTimerRef.current);
    };
  }, [snmpSettings, saveSNMPSettings]);

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
        className={`fixed inset-0 ${modal.overlay} z-40`}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer - full width on mobile, 384px on larger screens */}
      <div
        ref={drawerRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-drawer-title"
        className="fixed right-0 top-0 h-full w-full sm:w-96 lg:w-lg bg-surface-raised border-l border-surface-border z-50 overflow-y-auto shadow-xl"
      >
        {/* Header */}
        <div
          className={`${layout.flex.between} pad sm:pad-lg border-b border-surface-border sticky top-0 bg-surface-raised z-10`}
        >
          <div className="stack-xs">
            <h2 id="settings-drawer-title" className="heading-3">
              {t("title")}
            </h2>
            <p className="body-small">{t("subtitle")}</p>
          </div>
          <button
            ref={closeButtonRef}
            onClick={onClose}
            className={`${button.size.md} ${radius.md} hover:bg-surface-hover active:bg-surface-hover text-text-muted touch-manipulation focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-raised`}
            aria-label={t("network.closeSettings")}
          >
            <svg
              className={iconTokens.size.lg}
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

        <div
          className={`${spacing.drawerPad} section-gap body-small leading-relaxed`}
        >
          {/* Drawer content padding includes pt-4 for top spacing */}
          {/* Network Section */}
          <CollapsibleSection title={t("sections.network")}>
            {/* Network Configuration */}
            <div className="stack">
              <p className="section-title">{t("network.title")}</p>
              {/* Mode Toggle */}
              <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
                <button
                  onClick={() =>
                    setIPSettings((prev) => ({ ...prev, mode: "dhcp" }))
                  }
                  className={`${spacing.tab} ${radius.md} body-small font-medium transition-colors ${
                    ipSettings.mode === "dhcp"
                      ? "bg-brand-primary text-text-inverse"
                      : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                  }`}
                >
                  {t("network.dhcp")}
                </button>
                <button
                  onClick={() =>
                    setIPSettings((prev) => ({ ...prev, mode: "static" }))
                  }
                  className={`${spacing.tab} ${radius.md} body-small font-medium transition-colors ${
                    ipSettings.mode === "static"
                      ? "bg-brand-primary text-text-inverse"
                      : "bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover"
                  }`}
                >
                  {t("network.static")}
                </button>
              </div>

              {/* Static IP Fields */}
              {ipSettings.mode === "static" && (
                <div
                  className={`stack ${spacing.padding.top.heading} border-t border-surface-border`}
                >
                  <div>
                    <label className="caption font-medium">
                      {t("network.ipAddress")} *
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
                      className={`w-full ${spacing.margin.top.tight} ${spacing.chip.sm} bg-surface-base border ${radius.md} body-small text-text-primary ${
                        ipSettings.address && !isValidIP(ipSettings.address)
                          ? "border-status-error"
                          : "border-surface-border"
                      }`}
                    />
                  </div>
                  <div>
                    <label className="caption font-medium">
                      {t("network.subnetMask")} *
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
                      className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.md} body-small text-text-primary`}
                    />
                  </div>
                  <div>
                    <label className="caption font-medium">
                      {t("network.gateway")}
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
                      className={`w-full ${spacing.margin.top.tight} ${spacing.chip.sm} bg-surface-base border ${radius.md} body-small text-text-primary ${
                        ipSettings.gateway && !isValidIP(ipSettings.gateway)
                          ? "border-status-error"
                          : "border-surface-border"
                      }`}
                    />
                  </div>
                  <div>
                    <label className="caption font-medium">
                      {t("network.dnsServers")}
                    </label>
                    <input
                      type="text"
                      value={dnsInput}
                      onChange={(e) => setDnsInput(e.target.value)}
                      placeholder="8.8.8.8, 8.8.4.4"
                      className={`w-full ${spacing.margin.top.tight} ${spacing.chip.lg} bg-surface-base border border-surface-border ${radius.md} body-small text-text-primary`}
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
                className={`w-full ${button.size.md} bg-brand-primary text-text-inverse ${radius.md} font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors`}
              >
                {savingIP
                  ? t("network.applying")
                  : t("network.applyIPSettings")}
              </button>

              {ipMessage && (
                <p
                  className={`caption text-center ${
                    ipMessage.includes("Failed") || ipMessage.includes("Error")
                      ? "text-status-error"
                      : "text-status-success"
                  }`}
                >
                  {ipMessage}
                </p>
              )}

              <p className="caption">{t("network.requiresRoot")}</p>
            </div>

            {/* Display Options */}
            <div
              className={`border-t border-surface-border ${spacing.padding.top.heading} ${spacing.margin.top.heading}`}
            >
              <p
                className={`caption font-medium ${spacing.margin.bottom.inline}`}
              >
                {t("network.displayOptions")}{" "}
                <AutoSaveIndicator status={displayStatus} />
              </p>
              <div className="stack-sm">
                {/* Measurement Units */}
                <div
                  className={`flex items-center justify-between ${spacing.pad.xs} bg-surface-base ${radius.md} border border-surface-border`}
                >
                  <div>
                    <span className="body-small text-text-primary font-medium">
                      {t("network.measurementSystem")}
                    </span>
                    <p className="caption text-text-muted">
                      {t("network.measurementDescription")}
                    </p>
                  </div>
                  <select
                    value={displayOptions.unitSystem || "sae"}
                    onChange={(e) =>
                      setDisplayOptions((prev) => ({
                        ...prev,
                        unitSystem: e.target.value as "sae" | "metric",
                      }))
                    }
                    className={`${input.size.sm} bg-surface-base border border-surface-border ${radius.md} body-small text-text-primary`}
                  >
                    <option value="sae">
                      {t("display.unitSae", "SAE (feet)")}
                    </option>
                    <option value="metric">
                      {t("display.unitMetric", "Metric (meters)")}
                    </option>
                  </select>
                </div>

                {/* Show Public IP */}
                <label
                  className={`flex items-center justify-between ${spacing.pad.xs} bg-surface-base ${radius.md} border border-surface-border`}
                >
                  <div>
                    <span className="body-small text-text-primary font-medium">
                      {t("network.showPublicIP")}
                    </span>
                    <p className="caption text-text-muted">
                      {t("network.displayInNetworkCard")}
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
                    className={iconTokens.size.sm}
                  />
                </label>
              </div>
            </div>

            {/* VLAN Configuration */}
            <div
              className={`border-t border-surface-border ${spacing.padding.top.heading} ${spacing.margin.top.heading}`}
            >
              <p className={`section-title ${spacing.margin.bottom.inline}`}>
                {t("network.vlanTag")}
              </p>
              <VLANControl />
            </div>

            {/* MTU Configuration */}
            <div
              className={`border-t border-surface-border ${spacing.padding.top.heading} ${spacing.margin.top.heading}`}
            >
              <p className={`section-title ${spacing.margin.bottom.inline}`}>
                {t("network.mtuSetting")}
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
            snmpSettings={snmpSettings}
            setSnmpSettings={setSnmpSettings}
            snmpStatus={snmpStatus}
          />

          <ThresholdsSettings
            thresholds={thresholds}
            setThresholds={setThresholds}
            thresholdsStatus={thresholdsStatus}
          />

          {/* Appearance Section */}
          <AppearanceSettings
            theme={theme}
            setTheme={setTheme}
            isDark={isDark}
          />

          {/* Config Backups Section (implements #494) */}
          <ConfigBackupsSection />

          {/* Logs (debug) */}
          <section
            className={`${spacing.padding.top.section} border-t border-surface-border`}
          >
            <div className="flex items-start justify-between">
              <div>
                <h3 className="body-small font-medium text-text-muted">
                  {t("logs.title")}
                </h3>
                <p className="caption text-text-muted">
                  {t("logs.description")}
                </p>
              </div>
              <button
                onClick={fetchLogPreview}
                className={`caption ${spacing.chip.sm} border border-surface-border ${radius.md} text-text-muted hover:text-text-primary hover:border-text-muted transition-colors`}
              >
                {logLoading ? t("logs.loading") : t("logs.view")}
              </button>
            </div>
            {logError && (
              <p
                className={`caption text-status-error ${spacing.margin.top.inline}`}
              >
                {logError}
              </p>
            )}
            {!logError && logPreview.length > 0 && (
              <pre
                className={`${spacing.margin.top.inline} max-h-48 overflow-y-auto text-2xs leading-5 bg-surface-base border border-surface-border ${radius.md} ${spacing.chip.lg} text-text-primary whitespace-pre-wrap`}
              >
                {logPreview.join("\n")}
              </pre>
            )}
          </section>

          {/* Export Section */}
          <section
            className={`${spacing.padding.top.section} border-t border-surface-border`}
          >
            <h3
              className={`body-small font-medium text-text-muted ${spacing.margin.bottom.heading}`}
            >
              {t("export.title")}
            </h3>
            <a
              href={`${API_BASE}/api/export`}
              download="seed-export.json"
              className={`w-full ${button.size.md} bg-surface-base border border-surface-border text-text-primary ${radius.md} font-medium hover:bg-surface-hover transition-colors flex items-center justify-center ${spacing.gap.compact} touch-manipulation`}
            >
              <svg
                className={iconTokens.size.sm}
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
              {t("export.download")}
            </a>
            <p
              className={`caption text-text-muted ${spacing.margin.top.inline}`}
            >
              {t("export.description")}
            </p>
          </section>

          {/* About Section */}
          <section
            className={`${spacing.padding.top.section} border-t border-surface-border`}
          >
            <h3
              className={`body-small font-medium text-text-muted ${spacing.margin.bottom.inline}`}
            >
              {t("about.title")}
            </h3>
            <p className="caption text-text-muted">
              {t("about.appName")} {version}
              <br />
              {t("about.description")}
            </p>
          </section>
        </div>
      </div>
    </>
  );
});
