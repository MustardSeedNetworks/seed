/**
 * useSettingsDrawerSavers
 *
 * Bundles the per-section save callbacks that SettingsDrawer used to
 * declare inline (thresholds, tests, wifi, link, cable, network
 * discovery, snmp). The drawer keeps the state and status setters and
 * passes them in; the hook owns the network calls.
 */

import type React from 'react';
import { useCallback } from 'react';
import { normalizeTestsSettingsForSave } from '../components/settings/settingsDrawerNormalizer';
import type {
  CableTestSettings as CableTestSettingsType,
  LinkSettings as LinkSettingsType,
  NetworkDiscoverySettings,
  SaveStatus,
  SettingsThresholds,
  SnmpSettings as SnmpSettingsType,
  TestsSettings,
  WiFiSettings as WiFiSettingsType,
} from '../types/settings';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

interface UseSettingsDrawerSaversArgs {
  thresholds: SettingsThresholds;
  setThresholdsStatus: (status: SaveStatus) => void;
  testsSettings: TestsSettings;
  setTestsStatus: (status: SaveStatus) => void;
  testsSettingsChangedRef: React.MutableRefObject<boolean>;
  wifiSettings: WiFiSettingsType;
  setWifiStatus: (status: SaveStatus) => void;
  linkSettings: LinkSettingsType;
  setLinkStatus: (status: SaveStatus) => void;
  cableTestSettings: CableTestSettingsType;
  setCableTestStatus: (status: SaveStatus) => void;
  networkDiscoverySettings: NetworkDiscoverySettings;
  setNetworkDiscoveryStatus: (status: SaveStatus) => void;
  snmpSettings: SnmpSettingsType;
  setSnmpStatus: (status: SaveStatus) => void;
}

interface UseSettingsDrawerSaversResult {
  saveThresholds: () => Promise<void>;
  saveTestsSettings: () => Promise<void>;
  saveWifiSettings: () => Promise<void>;
  saveLinkSettings: () => Promise<void>;
  saveCableTestSettings: () => Promise<void>;
  saveNetworkDiscoverySettings: () => Promise<void>;
  saveSnmpSettings: () => Promise<void>;
}

export function useSettingsDrawerSavers({
  thresholds,
  setThresholdsStatus,
  testsSettings,
  setTestsStatus,
  testsSettingsChangedRef,
  wifiSettings,
  setWifiStatus,
  linkSettings,
  setLinkStatus,
  cableTestSettings,
  setCableTestStatus,
  networkDiscoverySettings,
  setNetworkDiscoveryStatus,
  snmpSettings,
  setSnmpStatus,
}: UseSettingsDrawerSaversArgs): UseSettingsDrawerSaversResult {
  const saveThresholds = useCallback(async () => {
    setThresholdsStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ thresholds }),
      });
      if (response.ok) {
        setThresholdsStatus('saved');
        setTimeout(() => setThresholdsStatus('idle'), 2000);
      } else {
        setThresholdsStatus('error');
      }
    } catch {
      setThresholdsStatus('error');
    }
  }, [thresholds, setThresholdsStatus]);

  const saveTestsSettings = useCallback(async () => {
    setTestsStatus('saving');
    try {
      const payload = normalizeTestsSettingsForSave(testsSettings);
      const response = await fetch(`${API_BASE}/api/health-checks/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(payload),
      });
      if (response.ok) {
        setTestsStatus('saved');
        setTimeout(() => setTestsStatus('idle'), 2000);
        testsSettingsChangedRef.current = true;
      } else {
        setTestsStatus('error');
      }
    } catch {
      setTestsStatus('error');
    }
  }, [testsSettings, setTestsStatus, testsSettingsChangedRef]);

  const saveWifiSettings = useCallback(async () => {
    setWifiStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/canopy/wifi/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ interface: wifiSettings.interface }),
      });
      if (response.ok) {
        setWifiStatus('saved');
        setTimeout(() => setWifiStatus('idle'), 2000);
      } else {
        setWifiStatus('error');
      }
    } catch {
      setWifiStatus('error');
    }
  }, [wifiSettings.interface, setWifiStatus]);

  const saveLinkSettings = useCallback(async () => {
    setLinkStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/settings/link`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          mode: linkSettings.mode,
          availableModes: linkSettings.availableModes,
        }),
      });
      if (response.ok) {
        setLinkStatus('saved');
        setTimeout(() => setLinkStatus('idle'), 2000);
      } else {
        setLinkStatus('error');
      }
    } catch {
      setLinkStatus('error');
    }
  }, [linkSettings, setLinkStatus]);

  const saveCableTestSettings = useCallback(async () => {
    setCableTestStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/settings/cable`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ enabled: cableTestSettings.enabled }),
      });
      if (response.ok) {
        setCableTestStatus('saved');
        setTimeout(() => setCableTestStatus('idle'), 2000);
      } else {
        setCableTestStatus('error');
      }
    } catch {
      setCableTestStatus('error');
    }
  }, [cableTestSettings, setCableTestStatus]);

  const saveNetworkDiscoverySettings = useCallback(async () => {
    setNetworkDiscoveryStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(networkDiscoverySettings),
      });
      if (response.ok) {
        setNetworkDiscoveryStatus('saved');
        setTimeout(() => setNetworkDiscoveryStatus('idle'), 2000);
      } else {
        setNetworkDiscoveryStatus('error');
      }
    } catch {
      setNetworkDiscoveryStatus('error');
    }
  }, [networkDiscoverySettings, setNetworkDiscoveryStatus]);

  const saveSnmpSettings = useCallback(async () => {
    setSnmpStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/sap/snmp/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(snmpSettings),
      });
      if (response.ok) {
        setSnmpStatus('saved');
        setTimeout(() => setSnmpStatus('idle'), 2000);
      } else {
        setSnmpStatus('error');
      }
    } catch {
      setSnmpStatus('error');
    }
  }, [snmpSettings, setSnmpStatus]);

  return {
    saveThresholds,
    saveTestsSettings,
    saveWifiSettings,
    saveLinkSettings,
    saveCableTestSettings,
    saveNetworkDiscoverySettings,
    saveSnmpSettings,
  };
}
