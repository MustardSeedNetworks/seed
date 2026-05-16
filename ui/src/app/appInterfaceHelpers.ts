/**
 * Helper types and functions for App-level interface restoration.
 *
 * Pulled out of App.tsx to keep the orchestration component slim. These
 * pure helpers turn a profile's saved interface config into the next set
 * of interface-selection state updates.
 */

import { LogComponents, logger } from '../lib/logger';

export interface InterfaceInfo {
  name: string;
  type: string;
  up: boolean;
}

/** Profile interface config from backend (uses snake_case) */
export interface ProfileInterfacesConfig {
  active_ethernet?: string;
  active_wifi?: string;
}

export interface InterfaceRestorationResult {
  restoredEthernet: boolean;
  restoredWifi: boolean;
  savedEthernetName: string;
  savedWifiName: string;
}

/**
 * Find the best interface of a given type from available interfaces.
 * Prefers link-up interfaces, otherwise returns the first one.
 */
export function findBestInterface(
  interfaces: InterfaceInfo[],
  type: 'ethernet' | 'wifi',
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
export function interfaceExistsWithType(
  interfaces: InterfaceInfo[],
  name: string,
  type: string,
): boolean {
  return interfaces.some((i) => i.name === name && i.type === type);
}

/**
 * Parse profile interfaces config and determine which interfaces can be restored.
 */
export function parseProfileInterfaces(
  profileInterfaces: ProfileInterfacesConfig | undefined,
  interfaces: InterfaceInfo[],
): InterfaceRestorationResult {
  const result: InterfaceRestorationResult = {
    restoredEthernet: false,
    restoredWifi: false,
    savedEthernetName: '',
    savedWifiName: '',
  };

  if (!profileInterfaces) {
    return result;
  }

  // Check ethernet interface
  if (profileInterfaces.active_ethernet) {
    result.savedEthernetName = profileInterfaces.active_ethernet;
    if (interfaceExistsWithType(interfaces, result.savedEthernetName, 'ethernet')) {
      result.restoredEthernet = true;
    }
  }

  // Check wifi interface
  if (profileInterfaces.active_wifi) {
    result.savedWifiName = profileInterfaces.active_wifi;
    if (interfaceExistsWithType(interfaces, result.savedWifiName, 'wifi')) {
      result.restoredWifi = true;
    }
  }

  return result;
}

/**
 * Apply interface state updates for restoration.
 * Handles setting local state and notifying backend.
 */
export function applyInterfaceRestoration(
  restoration: InterfaceRestorationResult,
  setEthernetInterfaceState: (name: string) => void,
  setWifiInterfaceState: (name: string) => void,
  changeInterface: (name: string) => Promise<void>,
  setActiveMode: (mode: 'ethernet' | 'wifi') => void,
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
      logger.error(LogComponents.NETWORK, 'Failed to change interface', { error: err });
    });
    setActiveMode('ethernet');
  } else if (restoration.restoredWifi) {
    changeInterface(restoration.savedWifiName).catch((err: unknown) => {
      logger.error(LogComponents.NETWORK, 'Failed to change interface', { error: err });
    });
    setActiveMode('wifi');
  }
}
