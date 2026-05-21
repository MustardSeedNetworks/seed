// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Coordinates port-scan and vuln-scan automation; mirrors structure of original NetworkDiscoveryCard inline code
/**
 * useNetworkDiscoveryAutoScan
 *
 * Hosts the port-scan + vulnerability-scan automation that previously
 * lived inline in NetworkDiscoveryCard. The hook:
 * - fetches DiscoverySettings.PortScan + Vulnerabilities settings on mount
 * - exposes a manual deep-scan callback used by the discovery modal
 * - watches incoming devices and auto-runs port scans (up to 3 concurrent)
 * - watches devices with useful fingerprint info and auto-triggers vuln scans
 *
 * Returns just the deep-scan callback the parent needs to wire into the
 * full-screen DiscoveryModal; everything else is internal state plus
 * tracked-set refs to prevent double-scanning the same device.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api';
import { COMMON_PORTS } from '../components/cards/NetworkDiscoveryCardHelpers';
import type {
  DeepScanResult,
  DiscoveredDevice,
  DiscoverySettingsForAutoScan,
  NetworkDiscoveryData,
  PortScanApiResponse,
  PortScanResult,
  ServiceInfo,
} from '../components/cards/networkDiscoveryCardTypes';
import { LogComponents, logger } from '../lib/logger';

interface UseNetworkDiscoveryAutoScanResult {
  handleDeepScan: (ip: string) => Promise<void>;
}

export function useNetworkDiscoveryAutoScan(
  data: NetworkDiscoveryData | null,
): UseNetworkDiscoveryAutoScanResult {
  const [scanningDevices, setScanningDevices] = useState<Set<string>>(new Set());
  const [_scanResults, setScanResults] = useState<Map<string, DeepScanResult>>(new Map());
  const [autoScanSettings, setAutoScanSettings] = useState<DiscoverySettingsForAutoScan>({
    portScanEnabled: false,
    vulnScanEnabled: false,
    vulnAutoScan: false,
  });

  // Fetch settings for auto-scan behavior on mount
  useEffect(() => {
    const fetchSettings = async (): Promise<void> => {
      const apiBase = import.meta.env.VITE_API_BASE || '';
      try {
        // Fetch discovery options from correct endpoint
        const discoveryResponse = await fetch(`${apiBase}/api/v1/shell/discovery/options`, {
          credentials: 'include',
        });
        if (discoveryResponse.ok) {
          const discoveryData = await discoveryResponse.json();
          // Backend returns { options: { PortScan: { Enabled: true, ... } } }
          const portScanEnabled = discoveryData?.options?.PortScan?.Enabled ?? false;

          // Fetch vulnerability settings from correct endpoint
          const vulnResponse = await fetch(`${apiBase}/api/v1/shell/vulnerabilities/settings`, {
            credentials: 'include',
          });
          let vulnEnabled = false;
          let vulnAutoScan = false;
          if (vulnResponse.ok) {
            const vulnData = await vulnResponse.json();
            // Backend returns { Enabled: false, AutoScan: false, ... }
            vulnEnabled = vulnData?.Enabled ?? false;
            vulnAutoScan = vulnData?.AutoScan ?? false;
          }

          setAutoScanSettings({
            portScanEnabled,
            vulnScanEnabled: vulnEnabled,
            vulnAutoScan,
          });
        }
      } catch (error) {
        logger.debug(LogComponents.DISCOVERY, 'Failed to fetch auto-scan settings', error);
      }
    };

    fetchSettings().catch(() => {
      // Error already logged in fetchSettings
    });
  }, []);

  // Trigger vulnerability scan for a device based on any good info we have
  const triggerVulnScan = useCallback(
    async (ip: string, device?: DiscoveredDevice, services?: ServiceInfo[]) => {
      if (!(autoScanSettings.vulnScanEnabled && autoScanSettings.vulnAutoScan)) {
        return;
      }

      let hasGoodInfo = false;
      const reasons: string[] = [];

      // Check port scan services
      if (services && services.length > 0) {
        const openServices = services.filter(
          (s) => s.state === 'open' && (s.banner || s.version || s.service !== 'unknown'),
        );
        if (openServices.length > 0) {
          hasGoodInfo = true;
          reasons.push(`${openServices.length} services`);
        }
      }

      // Check device info if provided
      if (device) {
        if (device.osGuess) {
          hasGoodInfo = true;
          reasons.push('OS guess');
        }
        if (device.lldpInfo?.systemDescription) {
          hasGoodInfo = true;
          reasons.push('LLDP system info');
        }
        if (device.cdpInfo?.platform || device.cdpInfo?.softwareVersion) {
          hasGoodInfo = true;
          reasons.push('CDP info');
        }
        if (device.profile?.openPorts?.some((p) => p.isOpen)) {
          hasGoodInfo = true;
          reasons.push('profile ports');
        }
        if (device.profile?.httpInfo?.server) {
          hasGoodInfo = true;
          reasons.push('HTTP server');
        }
      }

      if (!hasGoodInfo) {
        return;
      }

      try {
        logger.info(LogComponents.DISCOVERY, 'Triggering auto vulnerability scan', {
          ip,
          reasons: reasons.join(', '),
        });
        await api.post('/api/v1/shell/vulnerabilities/scan', { targets: [ip] });
      } catch (error) {
        logger.debug(LogComponents.DISCOVERY, 'Failed to trigger vulnerability scan', error);
      }
    },
    [autoScanSettings.vulnScanEnabled, autoScanSettings.vulnAutoScan],
  );

  const handleDeepScan = useCallback(
    async (ip: string) => {
      setScanningDevices((prev) => new Set(prev).add(ip));

      try {
        const apiResponse = await api.post<PortScanApiResponse>(
          '/api/v1/shell/discovery/portscan',
          {
            target: ip,
            ports: COMMON_PORTS,
            timeout: 2000,
          },
        );

        // Transform backend response to frontend format
        const results: PortScanResult[] = apiResponse.services.map((svc) => ({
          port: svc.port,
          state: svc.state,
          service: svc.service,
          banner: svc.banner,
          version: svc.version,
          rtt: 0, // Backend doesn't return individual RTT per port
        }));
        setScanResults((prev) => {
          const next = new Map(prev);
          next.set(ip, {
            target: apiResponse.ip,
            results: results,
            scannedAt: new Date(),
          });
          // Fixes #904: Limit stored scan results to prevent unbounded memory growth
          const MAX_SCAN_RESULTS = 100;
          if (next.size > MAX_SCAN_RESULTS) {
            const entries = [...next.entries()].sort(
              (a, b) => a[1].scannedAt.getTime() - b[1].scannedAt.getTime(),
            );
            while (next.size > MAX_SCAN_RESULTS && entries.length > 0) {
              const oldest = entries.shift();
              if (oldest) {
                next.delete(oldest[0]);
              }
            }
          }
          return next;
        });

        // If vulnerability scanning is enabled with auto-scan, trigger vuln scan
        const device = data?.devices.find((d) => d.ip === ip);
        if (apiResponse.services && apiResponse.services.length > 0) {
          await triggerVulnScan(ip, device, apiResponse.services);
        }
      } catch (error) {
        logger.error(LogComponents.DISCOVERY, 'Deep scan failed', error);
      } finally {
        setScanningDevices((prev) => {
          const next = new Set(prev);
          next.delete(ip);
          return next;
        });
      }
    },
    [triggerVulnScan, data?.devices],
  );

  // Track devices we've already auto-scanned to avoid duplicates
  const autoScannedDevices = useRef<Set<string>>(new Set());

  // Fixes #905: Clear auto-scanned tracking when a new scan cycle starts
  useEffect(() => {
    if (data?.status?.scanning) {
      autoScannedDevices.current.clear();
    }
  }, [data?.status?.scanning]);

  // Auto-scan devices after discovery completes (only if port scanning is enabled)
  useEffect(() => {
    if (!autoScanSettings.portScanEnabled) {
      return;
    }
    if (!data?.status || data.status.scanning) {
      return;
    }
    if (!data.devices || data.devices.length === 0) {
      return;
    }

    const devicesToScan = data.devices.filter((device) => {
      if (!device.ip) {
        return false;
      }
      if (autoScannedDevices.current.has(device.ip)) {
        return false;
      }
      return !scanningDevices.has(device.ip);
    });

    if (devicesToScan.length === 0) {
      return;
    }

    for (const device of devicesToScan) {
      autoScannedDevices.current.add(device.ip);
    }

    logger.info(LogComponents.DISCOVERY, 'Auto-scanning devices for open ports', {
      count: devicesToScan.length,
      portScanEnabled: autoScanSettings.portScanEnabled,
    });

    // Fixes #906: Track all timeout IDs for proper cleanup
    const timeoutIds: ReturnType<typeof setTimeout>[] = [];
    const MAX_CONCURRENT_SCANS = 3;
    let scanIndex = 0;

    const scanNextBatch = (): void => {
      const batch = devicesToScan.slice(scanIndex, scanIndex + MAX_CONCURRENT_SCANS);
      if (batch.length === 0) {
        return;
      }
      for (const device of batch) {
        handleDeepScan(device.ip).catch(() => {
          // Errors handled in handleDeepScan
        });
      }
      scanIndex += MAX_CONCURRENT_SCANS;
      if (scanIndex < devicesToScan.length) {
        const tid = setTimeout(scanNextBatch, 1000);
        timeoutIds.push(tid);
      }
    };

    const initialTimeoutId = setTimeout(scanNextBatch, 500);
    timeoutIds.push(initialTimeoutId);

    return (): void => {
      for (const tid of timeoutIds) {
        clearTimeout(tid);
      }
    };
  }, [
    data?.status,
    data?.devices,
    handleDeepScan,
    scanningDevices,
    autoScanSettings.portScanEnabled,
  ]);

  // Track devices we've already queued for vuln scan to avoid duplicates
  const vulnScannedDevices = useRef<Set<string>>(new Set());

  // Fixes #905: Clear vuln-scanned tracking when a new scan cycle starts
  useEffect(() => {
    if (data?.status?.scanning) {
      vulnScannedDevices.current.clear();
    }
  }, [data?.status?.scanning]);

  // Auto-trigger vulnerability scans based on device discovery info
  useEffect(() => {
    if (!(autoScanSettings.vulnScanEnabled && autoScanSettings.vulnAutoScan)) {
      return;
    }
    if (!data?.status || data.status.scanning) {
      return;
    }
    if (!data.devices || data.devices.length === 0) {
      return;
    }

    const devicesToVulnScan = data.devices.filter((device) => {
      if (!device.ip) {
        return false;
      }
      if (vulnScannedDevices.current.has(device.ip)) {
        return false;
      }

      const hasGoodInfo =
        device.osGuess ||
        device.lldpInfo?.systemDescription ||
        device.cdpInfo?.platform ||
        device.cdpInfo?.softwareVersion ||
        device.profile?.httpInfo?.server ||
        device.profile?.openPorts?.some((p) => p.isOpen);

      return hasGoodInfo;
    });

    if (devicesToVulnScan.length === 0) {
      return;
    }

    for (const device of devicesToVulnScan) {
      vulnScannedDevices.current.add(device.ip);
    }

    logger.info(
      LogComponents.DISCOVERY,
      'Auto-triggering vulnerability scans for devices with discovery info',
      {
        count: devicesToVulnScan.length,
      },
    );

    // Fixes #928: Track ALL timeout IDs to prevent orphaned recursive timeouts
    const timeoutIds: ReturnType<typeof setTimeout>[] = [];
    let index = 0;

    const triggerNext = (): void => {
      if (index >= devicesToVulnScan.length) {
        return;
      }
      const device = devicesToVulnScan[index];
      triggerVulnScan(device.ip, device).catch(() => {
        // Errors handled in triggerVulnScan
      });
      index++;
      if (index < devicesToVulnScan.length) {
        const tid = setTimeout(triggerNext, 200);
        timeoutIds.push(tid);
      }
    };

    const initialId = setTimeout(triggerNext, 300);
    timeoutIds.push(initialId);
    return (): void => {
      for (const tid of timeoutIds) {
        clearTimeout(tid);
      }
    };
  }, [
    data?.status,
    data?.devices,
    autoScanSettings.vulnScanEnabled,
    autoScanSettings.vulnAutoScan,
    triggerVulnScan,
  ]);

  return { handleDeepScan };
}
