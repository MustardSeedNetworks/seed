// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
import type React from 'react';
import { memo, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNetworkDiscoveryAutoScan } from '../../hooks/useNetworkDiscoveryAutoScan';
import { usePipelineStatus } from '../../hooks/usePipelineStatus';
import { button, cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import { Card, CardValue, type Status } from '../ui/card';
import { Maximize2, RefreshCw, ScanSearch } from '../ui/icons';
import { DiscoveryModal } from './DiscoveryModal';
import { categorizeDevices, DiscoverySummary } from './NetworkDiscoveryCardHelpers';
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

export const NetworkDiscoveryCard: React.NamedExoticComponent<NetworkDiscoveryCardProps> = memo(
  function networkDiscoveryCard({
    data,
    loading,
    onScan,
  }: NetworkDiscoveryCardProps): React.ReactElement | null {
    const { t } = useTranslation('cards');
    // Search and sort state lived here when the card embedded a sortable
    // device table; those rows now live entirely inside DiscoveryModal.

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

    // Extract data with safe defaults (must come before any hooks to avoid conditional hook calls)
    const rawDevices = data?.devices;
    const status = data?.status;
    // Ensure devices is an array (defensive check for malformed API responses)
    const devices = useMemo(() => (Array.isArray(rawDevices) ? rawDevices : []), [rawDevices]);
    const deviceCount = devices.length;

    // filteredDevices + sortedDevices were used by the now-deleted _localDevices /
    // _extendedDevices sections. All sorting/filtering now lives inside
    // DiscoveryModal where the full table renders. The IP-numeric sort
    // helper moved there too.

    // Early returns for loading/error states (after all hooks)
    // Fixes #674: Enable live regions for dynamic content updates
    if (loading) {
      return (
        <Card
          title={t('discovery.title')}
          icon={<ScanSearch className={iconTokens.size.md} />}
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
          icon={<ScanSearch className={iconTokens.size.md} />}
          status="unknown"
          enableLiveRegion={true}
          ariaLabel="Network discovery - no data available"
        >
          <CardValue value={t('discovery.noData')} size="md" />
          {onScan ? (
            <button
              type="button"
              onClick={onScan}
              className={cn(
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

    return (
      <Card
        title={t('discovery.title')}
        icon={<ScanSearch className={iconTokens.size.md} />}
        status={cardStatus}
        enableLiveRegion={true}
        ariaLabel={`Network discovery - ${deviceCount} devices found`}
        headerAction={
          <div className="flex items-center gap-2">
            {/* Full Screen button */}
            <button
              type="button"
              onClick={(): void => setIsModalOpen(true)}
              className={cn(
                'p-1.5',
                'bg-surface-hover text-text-secondary',
                radius.md,
                'hover:bg-surface-border hover:text-text-primary transition-colors flex items-center justify-center cursor-pointer',
              )}
              aria-label="Open full screen view"
              title={t('discovery.fullScreen', 'Full Screen')}
            >
              <Maximize2 className={iconTokens.size.sm} aria-hidden="true" />
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
                className={cn(
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
                    <RefreshCw
                      className={cn(iconTokens.size.xs, 'animate-spin')}
                      aria-hidden="true"
                    />
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
        <DiscoverySummary
          status={status}
          deviceCount={deviceCount}
          categories={categories}
          pipelineStatus={pipelineStatus}
          onCancelPipeline={cancelPipeline}
          t={t}
        />
        {deviceCount === 0 && !status.scanning && !isPipelineRunning ? (
          <p className={cn('body-small text-text-muted text-center', spacing.pad.default)}>
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
