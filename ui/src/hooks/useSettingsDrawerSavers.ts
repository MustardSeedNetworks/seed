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
import { api } from '../api';
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
      await api.put('/api/v1/settings', { thresholds });
      setThresholdsStatus('saved');
      setTimeout(() => setThresholdsStatus('idle'), 2000);
    } catch {
      setThresholdsStatus('error');
    }
  }, [thresholds, setThresholdsStatus]);

  const saveTestsSettings = useCallback(async () => {
    setTestsStatus('saving');
    try {
      const payload = normalizeTestsSettingsForSave(testsSettings);
      await api.put('/api/v1/telemetry/probes/settings', payload);
      setTestsStatus('saved');
      setTimeout(() => setTestsStatus('idle'), 2000);
      testsSettingsChangedRef.current = true;
    } catch {
      setTestsStatus('error');
    }
  }, [testsSettings, setTestsStatus, testsSettingsChangedRef]);

  const saveWifiSettings = useCallback(async () => {
    setWifiStatus('saving');
    try {
      await api.put('/api/v1/wifi/wifi/settings', { interface: wifiSettings.interface });
      setWifiStatus('saved');
      setTimeout(() => setWifiStatus('idle'), 2000);
    } catch {
      setWifiStatus('error');
    }
  }, [wifiSettings.interface, setWifiStatus]);

  const saveLinkSettings = useCallback(async () => {
    setLinkStatus('saving');
    try {
      await api.put('/api/v1/settings/link', {
        mode: linkSettings.mode,
        availableModes: linkSettings.availableModes,
      });
      setLinkStatus('saved');
      setTimeout(() => setLinkStatus('idle'), 2000);
    } catch {
      setLinkStatus('error');
    }
  }, [linkSettings, setLinkStatus]);

  const saveCableTestSettings = useCallback(async () => {
    setCableTestStatus('saving');
    try {
      await api.put('/api/v1/settings/cable', { enabled: cableTestSettings.enabled });
      setCableTestStatus('saved');
      setTimeout(() => setCableTestStatus('idle'), 2000);
    } catch {
      setCableTestStatus('error');
    }
  }, [cableTestSettings, setCableTestStatus]);

  const saveNetworkDiscoverySettings = useCallback(async () => {
    setNetworkDiscoveryStatus('saving');
    try {
      await api.put('/api/v1/security/devices/settings', networkDiscoverySettings);
      setNetworkDiscoveryStatus('saved');
      setTimeout(() => setNetworkDiscoveryStatus('idle'), 2000);
    } catch {
      setNetworkDiscoveryStatus('error');
    }
  }, [networkDiscoverySettings, setNetworkDiscoveryStatus]);

  const saveSnmpSettings = useCallback(async () => {
    setSnmpStatus('saving');
    try {
      await api.put('/api/v1/telemetry/snmp/settings', snmpSettings);
      setSnmpStatus('saved');
      setTimeout(() => setSnmpStatus('idle'), 2000);
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
