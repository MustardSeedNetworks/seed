/**
 * Interface State Management Hook
 *
 * Manages network interface selection and switching logic.
 * Extracted from App.tsx to improve maintainability and reduce component complexity.
 *
 * Handles:
 * - Dual interface state (ethernet and wifi)
 * - Active mode switching
 * - Interface change API calls
 * - Profile-based interface restoration
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

interface NetworkInterface {
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
}

interface UseInterfaceStateProps {
  interfaces: NetworkInterface[];
  activeProfile: {
    id: string;
    config?: {
      interfaces?: {
        ethernet?: { name: string };
        wifi?: { name: string };
      };
    };
  } | null;
  setEthernetInterface: (name: string, persist: boolean) => Promise<void>;
  setWifiInterface: (name: string, persist: boolean) => Promise<void>;
}

export function useInterfaceState({
  interfaces,
  activeProfile: _activeProfile,
  setEthernetInterface: _setEthernetInterface,
  setWifiInterface: _setWifiInterface,
}: UseInterfaceStateProps): {
  ethernetInterface: string;
  wifiInterface: string;
  activeMode: "ethernet" | "wifi";
  currentInterface: string;
  isWifi: boolean;
  setCurrentInterface: (name: string) => void;
  setIsWifi: (wifi: boolean) => void;
  userSetWifiModeRef: React.MutableRefObject<boolean>;
  currentInterfaceRef: React.MutableRefObject<string>;
  hasEthernet: boolean;
  hasWifiInterface: boolean;
  setEthernetInterfaceState: React.Dispatch<React.SetStateAction<string>>;
  setWifiInterfaceState: React.Dispatch<React.SetStateAction<string>>;
  setActiveMode: React.Dispatch<React.SetStateAction<"ethernet" | "wifi">>;
} {
  // Dual interface state: track both ethernet and WiFi interfaces separately (#754 enhancement)
  // This allows seamless switching between modes without losing the previously selected interface
  const [ethernetInterface, setEthernetInterfaceState] = useState("");
  const [wifiInterface, setWifiInterfaceState] = useState("");
  const [activeMode, setActiveMode] = useState<"ethernet" | "wifi">("ethernet");

  // Computed values for backwards compatibility with existing components
  const currentInterface = activeMode === "wifi" ? wifiInterface : ethernetInterface;
  const isWifi = activeMode === "wifi";

  // Helper to set the appropriate interface based on mode
  const setCurrentInterface = useCallback(
    (name: string) => {
      if (activeMode === "wifi") {
        setWifiInterfaceState(name);
      } else {
        setEthernetInterfaceState(name);
      }
    },
    [activeMode],
  );

  // Helper to set isWifi (actually sets activeMode)
  const setIsWifi = useCallback((wifi: boolean) => {
    setActiveMode(wifi ? "wifi" : "ethernet");
  }, []);

  // Track if user manually selected Wi-Fi/Ethernet mode - prevents auto-switching from API responses
  const userSetWifiModeRef = useRef(false);
  const currentInterfaceRef = useRef(currentInterface);

  useEffect(() => {
    currentInterfaceRef.current = currentInterface;
  }, [currentInterface]);

  // Quick helpers for interface groups
  const hasEthernet = useMemo(
    () => interfaces.some((iface) => iface.type === "ethernet"),
    [interfaces],
  );
  const hasWifiInterface = useMemo(
    () => interfaces.some((iface) => iface.type === "wifi"),
    [interfaces],
  );

  return {
    ethernetInterface,
    wifiInterface,
    activeMode,
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
  };
}
