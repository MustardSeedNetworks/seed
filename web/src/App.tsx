/**
 * Main Application Component
 *
 * The root component for The Seed network monitoring application by Mustard Seed Networks.
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

import { useCallback, useEffect, useRef, useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useWebSocket, Message, CardUpdate } from "./hooks/useWebSocket";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { useAuth } from "./hooks/useAuth";
import { useTheme } from "./hooks/useTheme";
import { useSettings } from "./contexts/useSettings";
import { SettingsDrawer } from "./components/settings/SettingsDrawer";
import { ImprovedHelpModal } from "./components/help/ImprovedHelpModal";
import { SetupWizard } from "./components/setup/SetupWizard";
import { checkSetupStatus } from "./components/setup/setupApi";
import { logger, LogComponents } from "./lib/logger";
import { setSessionExpiredCallback } from "./lib/api";

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
import { ProfileManagement } from "./components/profiles/ProfileManagement";
import { useProfileContext } from "./contexts/ProfileContext";
import { HeaderBar } from "./components/app/HeaderBar";
import {
  radius,
  layout,
  spacing,
  button,
  input,
  section,
  cn,
} from "./styles/theme";

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

const CARD_IDS = [
  "link",
  "cable",
  "vlan",
  "switch",
  "wifi",
  "dhcp",
  "dns",
  "gateway",
  "publicip",
] as const;
type CardId = (typeof CARD_IDS)[number];

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isCardId(value: unknown): value is CardId {
  return (
    typeof value === "string" && (CARD_IDS as readonly string[]).includes(value)
  );
}

/**
 * Main App Component
 *
 * Orchestrates the entire application, managing authentication,
 * real-time data updates, and the dashboard interface.
 */
function App() {
  const { t } = useTranslation("common");
  const {
    isAuthenticated,
    token,
    login,
    logout,
    refreshToken,
    isLoading,
    error,
  } = useAuth();
  const { isDark, toggleTheme } = useTheme();
  // Use settings from context instead of local state
  const { cardSettings, displayOptions, refreshSettings } = useSettings();
  // Profile management (#754)
  const {
    profiles,
    activeProfile,
    isLoading: profilesLoading,
    switchProfile,
  } = useProfileContext();
  const [profilesOpen, setProfilesOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [suggestedPassword, setSuggestedPassword] = useState<
    string | undefined
  >(undefined);
  const [setupUsername, setSetupUsername] = useState<string | undefined>(
    undefined
  );
  // Security fix #724, #758: Store setup token for secure setup completion
  const [setupToken, setSetupToken] = useState<string | undefined>(undefined);

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
      logger.info(
        LogComponents.CONFIG,
        "Profile changed, refreshing settings",
        {
          from: prevActiveProfileRef.current,
          to: currentProfileId,
        }
      );
      refreshSettings();
    }
    prevActiveProfileRef.current = currentProfileId;
  }, [activeProfile?.id, refreshSettings]);

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
  // Fix #572: Don't hardcode interface name - fetch from backend
  const [currentInterface, setCurrentInterface] = useState("");
  const [isWifi, setIsWifi] = useState(false);
  // Track if user manually selected Wi-Fi/Ethernet mode - prevents auto-switching from API responses
  const userSetWifiModeRef = useRef(false);
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

  // Refs to track device scan polling interval and timeout for cleanup
  const scanPollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(
    null
  );
  const scanTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryAbortRef = useRef<AbortController | null>(null);
  const currentInterfaceRef = useRef(currentInterface);

  useEffect(() => {
    currentInterfaceRef.current = currentInterface;
  }, [currentInterface]);

  useEffect(() => {
    return () => {
      networkDiscoveryAbortRef.current?.abort();
    };
  }, []);

  const handleMessage = useCallback((message: Message) => {
    if (message.type === "initial_state") {
      setLoading(false);
      if (!isPlainObject(message.payload)) {
        logger.warn(LogComponents.WEBSOCKET, "Invalid initial_state payload", {
          payload: message.payload,
        });
        return;
      }

      const payload = message.payload;
      if (typeof payload.interface === "string" && payload.interface) {
        setCurrentInterface(payload.interface);
      }

      // Only auto-set WiFi mode if user hasn't manually selected
      if (
        typeof payload.isWireless === "boolean" &&
        !userSetWifiModeRef.current
      ) {
        setIsWifi(payload.isWireless);
      }

      if (isPlainObject(payload.cards)) {
        const updates: Partial<CardState> = {};
        for (const [key, value] of Object.entries(payload.cards)) {
          if (!isCardId(key)) continue;

          const normalized =
            value === null ? null : isPlainObject(value) ? value : undefined;
          if (normalized === undefined) continue;

          switch (key) {
            case "link":
              updates.link = normalized as CardState["link"];
              break;
            case "cable":
              updates.cable = normalized as CardState["cable"];
              break;
            case "vlan":
              updates.vlan = normalized as CardState["vlan"];
              break;
            case "switch":
              updates.switch = normalized as CardState["switch"];
              break;
            case "wifi":
              updates.wifi = normalized as CardState["wifi"];
              break;
            case "dhcp":
              updates.dhcp = normalized as CardState["dhcp"];
              break;
            case "dns":
              updates.dns = normalized as CardState["dns"];
              break;
            case "gateway":
              updates.gateway = normalized as CardState["gateway"];
              break;
            case "publicip":
              updates.publicip = normalized as CardState["publicip"];
              break;
          }
        }

        if (Object.keys(updates).length > 0) {
          setCards((prev) => ({ ...prev, ...updates }));
        }
      }
    }
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

  const handleCardUpdate = useCallback((update: CardUpdate) => {
    if (!update || typeof update !== "object") {
      return;
    }

    const { cardId, data } = update as { cardId?: unknown; data?: unknown };

    if (!isCardId(cardId)) {
      logger.warn(
        LogComponents.WEBSOCKET,
        "Ignoring card_update for unknown cardId",
        { cardId }
      );
      return;
    }

    if (data === undefined || (data !== null && !isPlainObject(data))) {
      logger.warn(
        LogComponents.WEBSOCKET,
        "Ignoring card_update with invalid data",
        {
          cardId,
          data,
        }
      );
      return;
    }

    setCards((prev) => ({
      ...prev,
      [cardId]: data as CardState[typeof cardId],
    }));
  }, []);

  // Fetch link data (Layer 2 only)
  const fetchLinkData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/link`, {
        credentials: "include",
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
      logger.error(LogComponents.NETWORK, "Failed to fetch link data", err);
    }
  }, []);

  // Fetch IP configuration (DHCP card - Layer 3)
  const fetchIPConfig = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/ipconfig`, {
        credentials: "include",
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
      logger.error(LogComponents.NETWORK, "Failed to fetch IP config", err);
    }
  }, []);

  // Fetch interfaces
  const fetchInterfaces = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/interfaces`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setInterfaces(data);
      }
    } catch (err) {
      logger.error(LogComponents.NETWORK, "Failed to fetch interfaces", err);
    }
  }, []);

  // Fetch app version from status endpoint
  const fetchVersion = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/status`, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        if (data.version) {
          setAppVersion(data.version);
        }
      }
    } catch (err) {
      logger.error(LogComponents.SYSTEM, "Failed to fetch version", err);
    }
  }, []);

  // Fetch discovery data (LLDP/CDP/EDP neighbors)
  const fetchDiscoveryData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/discovery`, {
        credentials: "include",
      });
      if (response.ok) {
        const data: unknown = await response.json();
        const neighbors =
          isPlainObject(data) && Array.isArray(data.neighbors)
            ? data.neighbors
            : [];

        // Use the first neighbor as the "nearest switch"
        if (neighbors.length > 0 && isPlainObject(neighbors[0])) {
          const neighbor = neighbors[0];
          const rawProtocol =
            typeof neighbor.protocol === "string"
              ? neighbor.protocol.toLowerCase()
              : "unknown";
          const protocol: SwitchData["protocol"] =
            rawProtocol === "lldp" ||
            rawProtocol === "cdp" ||
            rawProtocol === "edp" ||
            rawProtocol === "fdp"
              ? rawProtocol
              : "unknown";

          const systemName =
            typeof neighbor.systemName === "string" ? neighbor.systemName : "";
          const chassisId =
            typeof neighbor.chassisId === "string" ? neighbor.chassisId : "";

          setCards((prev) => ({
            ...prev,
            switch: {
              protocol,
              switchName: systemName || chassisId || null,
              portId:
                typeof neighbor.portId === "string" ? neighbor.portId : null,
              portDescription:
                typeof neighbor.portDescription === "string"
                  ? neighbor.portDescription
                  : null,
              managementIp:
                typeof neighbor.managementAddress === "string"
                  ? neighbor.managementAddress
                  : null,
              systemDescription:
                typeof neighbor.systemDescription === "string"
                  ? neighbor.systemDescription
                  : null,
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
      logger.error(
        LogComponents.DISCOVERY,
        "Failed to fetch discovery data",
        err
      );
    }
  }, []);

  // Fetch DNS test data
  const fetchDNSData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/dns`, {
        credentials: "include",
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
      logger.error(LogComponents.DNS, "Failed to fetch DNS data", err);
    }
  }, []);

  // Fetch VLAN data
  const fetchVLANData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/vlan`, {
        credentials: "include",
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
      logger.error(LogComponents.VLAN, "Failed to fetch VLAN data", err);
    }
  }, []);

  // Fetch Gateway ping data
  const fetchGatewayData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/gateway`, {
        credentials: "include",
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
      logger.error(LogComponents.GATEWAY, "Failed to fetch Gateway data", err);
    }
  }, []);

  // Fetch Wi-Fi data
  const fetchWiFiData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/wifi`, {
        credentials: "include",
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
          // Only auto-set WiFi mode if user hasn't manually selected
          if (!userSetWifiModeRef.current) {
            setIsWifi(true);
          }
        } else {
          setCards((prev) => ({ ...prev, wifi: null }));
          // Only auto-set WiFi mode if user hasn't manually selected
          if (!userSetWifiModeRef.current) {
            setIsWifi(data.wireless === true);
          }
        }
      }
    } catch (err) {
      logger.error(LogComponents.WIFI, "Failed to fetch Wi-Fi data", err);
    }
  }, []);

  // Fetch Cable test data
  const fetchCableData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/cable`, {
        credentials: "include",
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
      logger.error(LogComponents.CABLE, "Failed to fetch Cable data", err);
    }
  }, []);

  // Fetch Public IP data
  const fetchPublicIP = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/publicip`, {
        credentials: "include",
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
      logger.error(
        LogComponents.PUBLICIP,
        "Failed to fetch Public IP data",
        err
      );
    }
  }, []);

  // Fetch Network Discovery data (devices and status)
  const fetchNetworkDiscovery = useCallback(async () => {
    try {
      networkDiscoveryAbortRef.current?.abort();
      const controller = new AbortController();
      networkDiscoveryAbortRef.current = controller;
      const requestedInterface = currentInterface;

      const [devicesRes, statusRes] = await Promise.all([
        fetch(`${API_BASE}/api/devices`, {
          credentials: "include",
          signal: controller.signal,
        }),
        fetch(`${API_BASE}/api/devices/status`, {
          credentials: "include",
          signal: controller.signal,
        }),
      ]);

      if (devicesRes.ok && statusRes.ok) {
        const devicesData = await devicesRes.json();
        const status = await statusRes.json();

        if (
          controller.signal.aborted ||
          currentInterfaceRef.current !== requestedInterface
        ) {
          return;
        }

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
            interface: requestedInterface,
          },
        });
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") {
        return;
      }
      logger.error(
        LogComponents.DEVICES,
        "Failed to fetch network discovery data",
        err
      );
    }
  }, [currentInterface]);

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
          : null
      );

      const response = await fetch(`${API_BASE}/api/devices/scan`, {
        method: "POST",
        credentials: "include",
      });

      if (response.ok) {
        // Poll for completion
        scanPollIntervalRef.current = setInterval(async () => {
          const statusRes = await fetch(`${API_BASE}/api/devices/status`, {
            credentials: "include",
          });
          if (statusRes.ok) {
            const status = await statusRes.json();
            if (!status.scanning) {
              if (scanPollIntervalRef.current) {
                clearInterval(scanPollIntervalRef.current);
                scanPollIntervalRef.current = null;
              }
              fetchNetworkDiscovery();
            }
          }
        }, 1000);

        // Safety timeout - stop polling after 60 seconds
        scanTimeoutRef.current = setTimeout(() => {
          if (scanPollIntervalRef.current) {
            clearInterval(scanPollIntervalRef.current);
            scanPollIntervalRef.current = null;
          }
        }, 60000);
      }
    } catch (err) {
      logger.error(LogComponents.DEVICES, "Failed to trigger device scan", err);
      setNetworkDiscovery((prev) =>
        prev
          ? {
              ...prev,
              status: { ...prev.status, scanning: false },
            }
          : null
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
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({ interface: interfaceName }),
        });
        if (response.ok) {
          const data = await response.json();
          setCurrentInterface(interfaceName);
          // Only auto-set WiFi mode if user hasn't manually selected via Ethernet/WiFi buttons
          if (!userSetWifiModeRef.current) {
            setIsWifi(data.isWireless === true);
          }
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
        logger.error(LogComponents.NETWORK, "Failed to change interface", err);
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
    ]
  );

  // Quick helpers for interface groups and fast switching between Ethernet/Wi‑Fi views
  const hasEthernet = useMemo(
    () => interfaces.some((iface) => iface.type === "ethernet"),
    [interfaces]
  );
  const hasWifiInterface = useMemo(
    () => interfaces.some((iface) => iface.type === "wifi"),
    [interfaces]
  );
  const switchToInterfaceType = useCallback(
    (type: "ethernet" | "wifi") => {
      // Mark that user explicitly selected this mode - prevents API responses from flipping back
      userSetWifiModeRef.current = true;

      const candidates = interfaces.filter((iface) => iface.type === type);
      // If switching to WiFi with no WiFi interfaces, still allow UI to show WiFi view
      // for planning/survey purposes (Fix #572 extension)
      if (candidates.length === 0) {
        if (type === "wifi") {
          setIsWifi(true);
        } else {
          setIsWifi(false);
        }
        return;
      }
      // Set the mode immediately for responsive UI
      setIsWifi(type === "wifi");
      // Prefer a link-up interface, otherwise first in list
      const target = candidates.find((iface) => iface.up) ?? candidates[0];
      if (target) {
        changeInterface(target.name);
      }
    },
    [interfaces, changeInterface]
  );

  // Memoize run options to prevent unnecessary re-computation (fixes #671)
  const runOpts = useMemo(
    () => ({
      runLink: cardSettings.link.autoRunOnLink,
      runSwitch: cardSettings.switch.autoRunOnLink,
      runVLAN: cardSettings.vlan.autoRunOnLink,
      runIPConfig: cardSettings.network.autoRunOnLink,
      runGateway: cardSettings.gateway.autoRunOnLink,
      runDNS: cardSettings.dns.autoRunOnLink,
      runHealthChecks: cardSettings.healthChecks.autoRunOnLink,
      runPerformance: cardSettings.performance.autoRunOnLink,
      runSpeedtest:
        cardSettings.performance.autoRunOnLink &&
        cardSettings.performance.speedtest.autoRunOnLink,
      runIperf:
        cardSettings.performance.autoRunOnLink &&
        cardSettings.performance.iperf.autoRunOnLink,
      runNetworkDiscovery: cardSettings.networkDiscovery.autoRunOnLink,
    }),
    [cardSettings]
  );

  // Listen for FAB "run all tests" event with per-card autoRunOnLink settings
  useEffect(() => {
    const handleRunAllTests = async () => {
      // Use per-card autoRunOnLink settings to determine which tests to run

      // Build array of fetch promises based on card settings
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
              handleCardComplete as EventListener
            );
            window.dispatchEvent(new CustomEvent("testsComplete"));
          }
        }
      };

      // Listen for card test completions
      window.addEventListener(
        "cardTestComplete",
        handleCardComplete as EventListener
      );

      // Failsafe timeout (90s) in case a card doesn't report completion
      setTimeout(() => {
        window.removeEventListener(
          "cardTestComplete",
          handleCardComplete as EventListener
        );
        if (completed.size < cardTestsToWait.length) {
          logger.warn(
            LogComponents.UI,
            "FAB timeout: Not all card tests completed, signaling done anyway"
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
    runOpts,
  ]);

  // WebSocket connection for real-time updates
  const { status: wsStatus, reconnect } = useWebSocket({
    url: "/ws",
    token,
    isAuthenticated,
    onRefreshToken: refreshToken,
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

  // Fallback REST polling when WebSocket is not connected (fixes #672)
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
      }, 60000); // 60 second interval for non-WS data (increased from 30s)

      return () => clearInterval(slowInterval);
    }

    // Fallback: Poll when WebSocket disconnected with exponential backoff
    let attempts = 0;
    const maxAttempts = 5;

    const scheduleNextPoll = () => {
      // Exponential backoff: 15s, 30s, 60s, 120s, 240s (capped)
      const baseDelay = 15000;
      const delay = Math.min(baseDelay * Math.pow(2, attempts), 240000);

      return setTimeout(() => {
        fetchLinkData();
        fetchIPConfig();
        fetchDiscoveryData();
        fetchDNSData();
        fetchGatewayData();
        fetchVLANData();
        fetchWiFiData();

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
    [login]
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
    return (
      <LoginForm
        onLogin={handleLogin}
        isLoading={isLoading}
        error={authError}
      />
    );
  }

  return (
    <div className="min-h-screen text-text-primary font-body">
      <HeaderBar
        wsStatus={wsStatus}
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

      {/* Main content */}
      <main className={spacing.mainPadding.y}>
        <div className={cn(section.width.xl, "mx-auto", spacing.mainPadding.x)}>
          {/* Section: Primary Connectivity - cards differ by interface type */}
          <section
            aria-labelledby="connectivity-heading"
            className={spacing.margin.bottom.section}
          >
            <h2
              id="connectivity-heading"
              className={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.connectivity")}
            </h2>
            <div className={layout.grid.cards}>
              {/* WiFi-only cards */}
              {isWifi && (
                <WiFiCard data={cards.wifi} loading={loading} visible={true} />
              )}

              {/* Ethernet-only cards */}
              {!isWifi && (
                <>
                  <LinkCard data={cards.link} loading={loading} />
                  {cards.cable?.supported && (
                    <CableCard data={cards.cable} loading={loading} />
                  )}
                  <SwitchCard
                    data={cards.switch}
                    vlanData={cards.vlan}
                    loading={loading}
                  />
                </>
              )}
            </div>
          </section>

          {/* Section: Network Services */}
          <section
            aria-labelledby="network-heading"
            className={spacing.margin.bottom.section}
          >
            <h2
              id="network-heading"
              className={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.network")}
            </h2>
            <div className={layout.grid.cards}>
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

          {/* Section: Testing & Discovery - cards differ by interface type */}
          <section
            aria-labelledby="performance-heading"
            className={spacing.margin.bottom.section}
          >
            <h2
              id="performance-heading"
              className={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.testingDiscovery")}
            </h2>
            <div className={layout.grid.cards}>
              {/* Common cards for both interface types */}
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

              {/* Ethernet-only: Network Discovery (ARP/LLDP/SNMP) */}
              {!isWifi && cardSettings.networkDiscovery.enabled && (
                <NetworkDiscoveryCard
                  data={networkDiscovery}
                  loading={loading}
                  onScan={triggerDeviceScan}
                />
              )}

              {/* WiFi-only: WiFi Survey for heatmaps and site surveys */}
              {/* Fix #572: Pass current interface to avoid hardcoded "wlan0" */}
              {isWifi && (
                <WiFiSurveyCard
                  isWifi={isWifi}
                  currentInterface={currentInterface}
                />
              )}
            </div>
          </section>

          {/* Section: System */}
          <section
            aria-labelledby="system-heading"
            className={spacing.margin.bottom.section}
          >
            <h2
              id="system-heading"
              className={cn("section-title", spacing.margin.bottom.heading)}
            >
              {t("sections.system")}
            </h2>
            <div className={layout.grid.cards}>
              <SystemHealthCard />
            </div>
          </section>

          {/* Footer */}
          <footer
            className={cn(
              spacing.margin.top.section,
              radius.lg,
              "border border-surface-border bg-surface-raised",
              spacing.pad.lg
            )}
          >
            <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
              {/* Product Info */}
              <div>
                <h3 className="heading-4 text-text-primary mb-2">
                  {t("app.title")}
                </h3>
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
                  <a
                    href="/terms"
                    className="body-small text-text-muted hover:text-brand-primary"
                  >
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
                {t(
                  "footer.copyright",
                  "Mustard Seed Networks. All rights reserved."
                )}
              </p>
            </div>
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

      {/* Profile Management Modal (#754) */}
      {profilesOpen && (
        <ProfileManagement onClose={() => setProfilesOpen(false)} />
      )}

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

// Helper to extract and clear SSO error from URL
function getAndClearSsoError(): string | null {
  const params = new URLSearchParams(window.location.search);
  const errorParam = params.get("sso_error");
  if (errorParam) {
    // Clean URL without reload
    window.history.replaceState({}, "", window.location.pathname);
    return decodeURIComponent(errorParam.replace(/%20/g, " "));
  }
  return null;
}

// SSO provider info from backend (fixes #769)
interface SSOProvider {
  name: string;
  enabled: boolean;
}

function LoginForm({ onLogin, isLoading, error }: LoginFormProps) {
  const { t } = useTranslation("common");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  // Initialize SSO error from URL params using lazy initialization
  const [ssoError] = useState<string | null>(getAndClearSsoError);
  // Fetch SSO providers to conditionally show buttons (fixes #769)
  const [ssoProviders, setSsoProviders] = useState<SSOProvider[]>([]);

  // Fetch enabled SSO providers on mount (fixes #769)
  useEffect(() => {
    fetch(`${API_BASE}/api/sso/providers`)
      .then((res) => (res.ok ? res.json() : { providers: [] }))
      .then((data) => setSsoProviders(data.providers || []))
      .catch(() => setSsoProviders([]));
  }, []);

  // Helper to check if a provider is enabled
  const isProviderEnabled = (name: string) =>
    ssoProviders.some(
      (p) => p.name.toLowerCase() === name.toLowerCase() && p.enabled
    );

  // Check if any SSO provider is enabled
  const hasEnabledSSO = ssoProviders.some((p) => p.enabled);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onLogin(username, password);
  };

  return (
    <div className={cn("min-h-screen", layout.flex.center, "pad")}>
      <div className="w-full max-w-sm">
        <div className={cn("text-center", spacing.margin.bottom.sectionLg)}>
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
          <h1 className={cn("heading-1", spacing.margin.top.heading)}>
            {t("app.title")}
          </h1>
          <p className={cn("body-small", spacing.margin.top.inline)}>
            {t("app.tagline")}
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className={cn(
            "bg-surface-raised",
            radius.md,
            "border border-surface-border pad-lg stack-lg"
          )}
        >
          <div>
            <label
              htmlFor="login-username"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("labels.username")}
            </label>
            <input
              id="login-username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className={cn(
                "w-full",
                input.size.md,
                radius.md,
                "border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              )}
              placeholder="admin"
              required
            />
          </div>

          <div>
            <label
              htmlFor="login-password"
              className={cn("label block", spacing.margin.bottom.inline)}
            >
              {t("labels.password")}
            </label>
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={cn(
                "w-full",
                input.size.md,
                radius.md,
                "border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              )}
              placeholder="••••••••"
              required
            />
          </div>

          {(error || ssoError) && (
            <div
              role="alert"
              aria-live="assertive"
              className={cn(
                "pad-sm bg-status-error/10 border border-status-error/20",
                radius.md,
                "text-status-error body-small"
              )}
            >
              {error || ssoError}
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className={cn(
              "w-full",
              button.size.md,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "font-medium hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
            )}
          >
            {isLoading ? t("status.loggingIn") : t("buttons.login")}
          </button>

          <p className="caption text-text-muted text-center">
            {t("login.defaultCredentials")}
          </p>

          {/* SSO Options - only show if any provider is enabled (fixes #769) */}
          {hasEnabledSSO && (
            <div className="flex flex-col space-y-3">
              {isProviderEnabled("google") && (
                <button
                  type="button"
                  onClick={() =>
                    (window.location.href = `${API_BASE}/api/sso/login?provider=google`)
                  }
                  className={cn(
                    "w-full",
                    button.size.md,
                    "bg-status-info text-text-inverse",
                    radius.md,
                    "font-medium hover:bg-status-info-dark focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
                  )}
                >
                  {t("buttons.signInWithGoogle")}
                </button>
              )}
              {isProviderEnabled("microsoft") && (
                <button
                  type="button"
                  onClick={() =>
                    (window.location.href = `${API_BASE}/api/sso/login?provider=microsoft`)
                  }
                  className={cn(
                    "w-full",
                    button.size.md,
                    "bg-brand-secondary text-text-inverse",
                    radius.md,
                    "font-medium hover:bg-brand-secondary-dark focus:outline-none focus:ring-2 focus:ring-brand-secondary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
                  )}
                >
                  {t("buttons.signInWithMicrosoft")}
                </button>
              )}
              {isProviderEnabled("github") && (
                <button
                  type="button"
                  onClick={() =>
                    (window.location.href = `${API_BASE}/api/sso/login?provider=github`)
                  }
                  className={cn(
                    "w-full",
                    button.size.md,
                    "bg-surface-sunken text-text-primary",
                    radius.md,
                    "font-medium hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-surface-border focus:ring-offset-2 focus:ring-offset-surface-base border border-surface-border disabled:opacity-50"
                  )}
                >
                  {t("buttons.signInWithGitHub")}
                </button>
              )}
            </div>
          )}
        </form>
      </div>
    </div>
  );
}

export default App;
