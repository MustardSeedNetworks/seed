/**
 * Network Data Fetchers Hook
 *
 * Centralizes all fetch functions for network monitoring cards.
 * Extracted from App.tsx to improve maintainability and reduce component complexity.
 *
 * Provides 12 fetch functions:
 * - fetchLinkData: Layer 2 link status
 * - fetchIPConfig: DHCP/IP configuration
 * - fetchInterfaces: Available network interfaces
 * - fetchVersion: Application version
 * - fetchDiscoveryData: LLDP/CDP/EDP neighbors
 * - fetchDNSData: DNS test results
 * - fetchVLANData: VLAN configuration
 * - fetchGatewayData: Gateway reachability
 * - fetchWiFiData: WiFi connection info
 * - fetchCableData: Cable diagnostics
 * - fetchPublicIP: Public IP and location
 * - fetchNetworkDiscovery: Network device discovery
 */

import { useCallback } from "react";
import { logger, LogComponents } from "../lib/logger";
import type {
  LinkData,
  SwitchData,
  DHCPData,
  DNSData,
  VLANData,
  GatewayData,
  WiFiData,
  CableData,
  PublicIPData,
  NetworkDiscoveryData,
} from "../components/cards";

const API_BASE = import.meta.env.VITE_API_BASE || "";

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

interface NetworkInterface {
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
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

interface UseNetworkFetchersProps {
  currentInterfaceRef: React.MutableRefObject<string>;
  setCards: React.Dispatch<React.SetStateAction<CardState>>;
  setCurrentInterface: (name: string) => void;
  setInterfaces: React.Dispatch<React.SetStateAction<NetworkInterface[]>>;
  setAppVersion: React.Dispatch<React.SetStateAction<string>>;
  setNetworkDiscovery: React.Dispatch<React.SetStateAction<NetworkDiscoveryData | null>>;
  setIsWifi: (wifi: boolean) => void;
  userSetWifiModeRef: React.MutableRefObject<boolean>;
  networkDiscoveryAbortRef: React.MutableRefObject<AbortController | null>;
  prevLinkUpRef: React.MutableRefObject<boolean | null>;
}

export function useNetworkFetchers({
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
}: UseNetworkFetchersProps) {
  // Fetch link data (Layer 2 only)
  const fetchLinkData = useCallback(async () => {
    try {
      // Use ref to get current interface without dependency change (#754)
      const iface = currentInterfaceRef.current;
      const url = iface
        ? `${API_BASE}/api/link?interface=${encodeURIComponent(iface)}`
        : `${API_BASE}/api/link`;
      const response = await fetch(url, {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();

        // Detect link-up transition for auto-run tests (fallback polling case)
        const newLinkUp = data.linkUp === true;
        const wasDown = prevLinkUpRef.current === false;

        // Update previous state
        if (typeof data.linkUp === "boolean") {
          prevLinkUpRef.current = data.linkUp;
        }

        // Trigger auto-run when link transitions from down to up
        if (newLinkUp && wasDown) {
          logger.info(
            LogComponents.NETWORK,
            "Link up detected (poll), triggering auto-run tests"
          );
          setTimeout(() => {
            window.dispatchEvent(new CustomEvent("runAllTests"));
          }, 1500);
        }

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
  }, [currentInterfaceRef, setCards, setCurrentInterface, prevLinkUpRef]);

  // Fetch IP configuration (DHCP card - Layer 3)
  const fetchIPConfig = useCallback(async () => {
    try {
      // Use ref to get current interface without dependency change (#754)
      const iface = currentInterfaceRef.current;
      const url = iface
        ? `${API_BASE}/api/ipconfig?interface=${encodeURIComponent(iface)}`
        : `${API_BASE}/api/ipconfig`;
      const response = await fetch(url, {
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
  }, [currentInterfaceRef, setCards]);

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
  }, [setInterfaces]);

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
  }, [setAppVersion]);

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
  }, [setCards]);

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
  }, [setCards]);

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
  }, [setCards]);

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
  }, [setCards]);

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
  }, [setCards, setIsWifi, userSetWifiModeRef]);

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
  }, [setCards]);

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
  }, [setCards]);

  // Fetch Network Discovery data (devices and status)
  const fetchNetworkDiscovery = useCallback(async () => {
    try {
      networkDiscoveryAbortRef.current?.abort();
      const controller = new AbortController();
      networkDiscoveryAbortRef.current = controller;
      const requestedInterface = currentInterfaceRef.current;

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
  }, [currentInterfaceRef, setNetworkDiscovery, networkDiscoveryAbortRef]);

  return {
    fetchLinkData,
    fetchIPConfig,
    fetchInterfaces,
    fetchVersion,
    fetchDiscoveryData,
    fetchDNSData,
    fetchVLANData,
    fetchGatewayData,
    fetchWiFiData,
    fetchCableData,
    fetchPublicIP,
    fetchNetworkDiscovery,
  };
}
