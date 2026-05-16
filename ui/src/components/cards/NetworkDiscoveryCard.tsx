// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
import type React from 'react';
import { memo, useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNetworkDiscoveryAutoScan } from '../../hooks/useNetworkDiscoveryAutoScan';
import { usePipelineStatus } from '../../hooks/usePipelineStatus';
import { button, cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import { Card, CardValue, type Status } from '../ui/card';
import { Maximize2, RefreshCw, ScanSearch } from '../ui/icons';
import { DiscoveryModal } from './DiscoveryModal';
// biome-ignore lint/correctness/noUnusedImports: discoverySummary referenced via lowercase <discoverySummary> JSX (pre-existing pattern, preserved as-is)
import { categorizeDevices, discoverySummary } from './NetworkDiscoveryCardHelpers';
import type { NetworkDiscoveryData as _NetworkDiscoveryData } from './networkDiscoveryCardTypes';
import { VulnerabilityDetailsModal } from './VulnerabilityDetailsModal';

// Re-export public types so existing import paths still resolve.
export type {
  CdpInfo,
  DeepScanResult,
  DeviceProfile,
  DiscoveredDevice,
  DiscoveryMethod,
  DiscoveryStatus,
  EdpInfo,
  HttpInfo,
  LldpInfo,
  NdpInfo,
  NetworkDiscoveryData,
  OpenPort,
  PortScanResult,
  ServiceInfo,
  SnmpEntity,
  SnmpFullData,
  SnmpInterface,
  SnmpIpAddress,
  SnmpSystemInfo,
  SnmpVlan,
} from './networkDiscoveryCardTypes';

interface NetworkDiscoveryCardProps {
  data: _NetworkDiscoveryData | null;
  loading?: boolean;
  onScan?: () => void;
}

// Sorting types for device list
type SortField = 'ip' | 'hostname' | 'vendor' | 'lastSeen' | null;
type SortDirection = 'asc' | 'desc';

export const NetworkDiscoveryCard: React.NamedExoticComponent<NetworkDiscoveryCardProps> = memo(
  function networkDiscoveryCard({
    data,
    loading,
    onScan,
  }: NetworkDiscoveryCardProps): React.ReactElement | null {
    const { t } = useTranslation('cards');
    // Search and sort state (kept for modal use)
    const [searchQuery, _setSearchQuery] = useState('');
    const [sortField, setSortField] = useState<SortField>(null);
    const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

    // Pipeline status hook for multi-phase progress display
    const { status: pipelineStatus, startPipeline, cancelPipeline } = usePipelineStatus();

    // Check if pipeline is actively running
    const isPipelineRunning =
      pipelineStatus.state !== 'idle' &&
      pipelineStatus.state !== 'complete' &&
      pipelineStatus.state !== 'failed' &&
      pipelineStatus.state !== 'canceled';

    // Auto-scan + vuln-scan orchestration lives in its own hook
    const { handleDeepScan } = useNetworkDiscoveryAutoScan(data);

    // Vulnerability modal state
    const [selectedDeviceForVuln, setSelectedDeviceForVuln] = useState<string | null>(null);

    // Full-screen modal state
    const [isModalOpen, setIsModalOpen] = useState(false);

    // Toggle sort field/direction (kept for modal use)
    const _handleSortChange = useCallback(
      (field: SortField): void => {
        if (sortField === field) {
          // Toggle direction or clear
          if (sortDirection === 'asc') {
            setSortDirection('desc');
          } else {
            setSortField(null);
            setSortDirection('asc');
          }
        } else {
          setSortField(field);
          setSortDirection('asc');
        }
      },
      [sortField, sortDirection],
    );

    // Extract data with safe defaults (must come before any hooks to avoid conditional hook calls)
    const rawDevices = data?.devices;
    const status = data?.status;
    // Ensure devices is an array (defensive check for malformed API responses)
    const devices = useMemo(() => (Array.isArray(rawDevices) ? rawDevices : []), [rawDevices]);
    const deviceCount = devices.length;

    // Helper function for IP to numeric conversion
    // Fixes #953: Handle malformed IPs that would produce NaN
    const ipToNum = useCallback((ip: string) => {
      const parts = ip.split('.').map((s) => Number.parseInt(s, 10) || 0);
      return parts[0] * 16777216 + parts[1] * 65536 + parts[2] * 256 + parts[3];
    }, []);

    // Filter and sort devices
    const filteredDevices = useMemo(() => {
      let result = [...devices];

      // Apply search filter
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase();
        result = result.filter(
          (device) =>
            device.ip?.toLowerCase().includes(query) ||
            device.hostname?.toLowerCase().includes(query) ||
            device.vendor?.toLowerCase().includes(query) ||
            device.mac?.toLowerCase().includes(query) ||
            device.osGuess?.toLowerCase().includes(query),
        );
      }

      // Apply sorting
      if (sortField) {
        result.sort((a, b) => {
          let aVal: string | number | null = null;
          let bVal: string | number | null = null;

          switch (sortField) {
            case 'ip':
              // Sort IP numerically
              aVal = a.ip ? ipToNum(a.ip) : 0;
              bVal = b.ip ? ipToNum(b.ip) : 0;
              break;
            case 'hostname':
              aVal = a.hostname?.toLowerCase() || '';
              bVal = b.hostname?.toLowerCase() || '';
              break;
            case 'vendor':
              aVal = a.vendor?.toLowerCase() || '';
              bVal = b.vendor?.toLowerCase() || '';
              break;
            case 'lastSeen':
              aVal = a.lastSeen ? new Date(a.lastSeen).getTime() : 0;
              bVal = b.lastSeen ? new Date(b.lastSeen).getTime() : 0;
              break;
            default:
              break;
          }

          if (aVal === null && bVal === null) {
            return 0;
          }
          if (aVal === null) {
            return 1;
          }
          if (bVal === null) {
            return -1;
          }

          let comparison = 0;
          if (typeof aVal === 'number' && typeof bVal === 'number') {
            comparison = aVal - bVal;
          } else {
            comparison = String(aVal).localeCompare(String(bVal));
          }

          return sortDirection === 'asc' ? comparison : -comparison;
        });
      }

      return result;
    }, [devices, searchQuery, sortField, sortDirection, ipToNum]);

    const _filteredCount = filteredDevices.length;

    // If no user sort applied, use default sorting: local first, then by discovery methods, then by IP
    const sortedDevices = useMemo(() => {
      // If user has applied search/sort, use filtered devices
      if (searchQuery.trim() || sortField) {
        return filteredDevices;
      }

      // Default sorting when no user filters applied
      return [...devices].sort((a, b) => {
        // Local devices first
        if (a.isLocal !== b.isLocal) {
          return a.isLocal ? -1 : 1;
        }
        // Then by discovery method count
        if (b.discoveryMethod.length !== a.discoveryMethod.length) {
          return b.discoveryMethod.length - a.discoveryMethod.length;
        }
        // Then by IP numerically - compare each octet
        const ipA = a.ip.split('.').map(Number);
        const ipB = b.ip.split('.').map(Number);
        // Compare octets using zip iterator pattern
        const ipIterA = ipA[Symbol.iterator]();
        const ipIterB = ipB[Symbol.iterator]();
        let resultA = ipIterA.next();
        let resultB = ipIterB.next();
        while (!(resultA.done || resultB.done)) {
          if (resultA.value !== resultB.value) {
            return resultA.value - resultB.value;
          }
          resultA = ipIterA.next();
          resultB = ipIterB.next();
        }
        return 0;
      });
    }, [devices, filteredDevices, searchQuery, sortField]);

    // Early returns for loading/error states (after all hooks)
    // Fixes #674: Enable live regions for dynamic content updates
    if (loading) {
      return (
        <Card
          title={t('discovery.title')}
          icon={<ScanSearch class={iconTokens.size.md} />}
          status="loading"
          enableLiveRegion={true}
          ariaLabel="Network discovery scanning in progress"
        >
          <CardValue value={t('discovery.scanning')} size="lg" />
        </Card>
      );
    }

    if (!(data && status)) {
      return (
        <Card
          title={t('discovery.title')}
          icon={<ScanSearch class={iconTokens.size.md} />}
          status="unknown"
          enableLiveRegion={true}
          ariaLabel="Network discovery - no data available"
        >
          <CardValue value={t('discovery.noData')} size="md" />
          {onScan ? (
            <button
              type="button"
              onClick={onScan}
              class={cn(
                spacing.margin.top.heading,
                'w-full',
                button.size.md,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'hover:bg-brand-primary/90 transition-colors font-medium body-small',
              )}
              aria-label="Start network discovery scan"
            >
              {t('discovery.startScan')}
            </button>
          ) : null}
        </Card>
      );
    }

    // Categorize devices for summary
    const categories = categorizeDevices(devices);

    const getOverallStatus = (): Status => {
      if (status.scanning || isPipelineRunning) {
        return 'loading';
      }
      if (deviceCount === 0) {
        return 'warning';
      }
      return 'success';
    };

    const cardStatus = getOverallStatus();

    // Separate into local and extended for display (kept for modal use)
    const _localDevices = sortedDevices.filter((d) => d.isLocal);
    const _extendedDevices = sortedDevices.filter((d) => !d.isLocal);

    return (
      <Card
        title={t('discovery.title')}
        icon={<ScanSearch class={iconTokens.size.md} />}
        status={cardStatus}
        enableLiveRegion={true}
        ariaLabel={`Network discovery - ${deviceCount} devices found`}
        headerAction={
          <div class="flex items-center gap-2">
            {/* Full Screen button */}
            <button
              type="button"
              onClick={(): void => setIsModalOpen(true)}
              class={cn(
                'p-1.5',
                'bg-surface-hover text-text-secondary',
                radius.md,
                'hover:bg-surface-border hover:text-text-primary transition-colors flex items-center justify-center cursor-pointer',
              )}
              aria-label="Open full screen view"
              title={t('discovery.fullScreen', 'Full Screen')}
            >
              <Maximize2 class={iconTokens.size.sm} aria-hidden="true" />
            </button>

            {/* Scan button */}
            {onScan || startPipeline ? (
              <button
                type="button"
                onClick={(): void => {
                  // Use pipeline start with port scanning enabled
                  // This enables the serviceDiscovery phase with quick port scan
                  startPipeline({
                    phases: {
                      enumeration: true,
                      nameResolution: true,
                      serviceDiscovery: true,
                      vulnAssessment: false,
                    },
                    portScan: {
                      intensity: 'quick',
                      bannerGrab: true,
                      connectTimeout: 2000,
                    },
                  }).catch(() => {
                    // Errors handled in usePipelineStatus
                  });
                  // Also call onScan for backwards compatibility
                  onScan?.();
                }}
                disabled={status.scanning || isPipelineRunning}
                class={cn(
                  spacing.chip.sm,
                  'bg-brand-primary text-text-inverse',
                  radius.md,
                  'hover:bg-brand-primary/90 transition-colors font-medium caption disabled:opacity-50 disabled:cursor-not-allowed flex items-center',
                  spacing.inline.sm,
                )}
                aria-label={
                  status.scanning || isPipelineRunning ? 'Scanning network' : 'Start network scan'
                }
              >
                {status.scanning || isPipelineRunning ? (
                  <>
                    <RefreshCw class={cn(iconTokens.size.xs, 'animate-spin')} aria-hidden="true" />
                    {t('discovery.scan')}
                  </>
                ) : (
                  t('discovery.scan')
                )}
              </button>
            ) : null}
          </div>
        }
      >
        {/* Discovery Summary - Minimal view showing status, subnet, device count, and categories */}
        <discoverySummary
          status={status}
          deviceCount={deviceCount}
          categories={categories}
          pipelineStatus={pipelineStatus}
          onCancelPipeline={cancelPipeline}
          t={t}
        />

        {deviceCount === 0 && !status.scanning && !isPipelineRunning ? (
          <p class={cn('body-small text-text-muted text-center', spacing.pad.default)}>
            {t('discovery.noDevices')}
          </p>
        ) : null}

        {/* Vulnerability Details Modal */}
        {selectedDeviceForVuln ? (
          <VulnerabilityDetailsModal
            deviceIp={selectedDeviceForVuln}
            onClose={(): void => setSelectedDeviceForVuln(null)}
          />
        ) : null}

        {/* Full Screen Discovery Modal */}
        <DiscoveryModal
          isOpen={isModalOpen}
          onClose={(): void => setIsModalOpen(false)}
          data={data}
          onScan={onScan}
          onDeepScan={handleDeepScan}
        />
      </Card>
    );
  },
);
