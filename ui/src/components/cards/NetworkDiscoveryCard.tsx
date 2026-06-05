// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
import type React from 'react';
import { memo, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useEnginePhase } from '../../hooks/useEnginePhase';
import { useEngineScan } from '../../hooks/useEngineScan';
import { useNetworkDiscoveryAutoScan } from '../../hooks/useNetworkDiscoveryAutoScan';
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

    // Discovery scan via the unified jobs spine: an engine-scan job
    // (useEngineScan) drives lifecycle + progress %, and the engine event bus
    // (useEnginePhase) supplies the current phase name.
    const { running, status: scanStatus, startScan, cancelScan } = useEngineScan();
    const { phase: scanPhase } = useEnginePhase();

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
                'bg-brand-primary text-on-brand',
                radius.md,
                'hover:bg-brand-primary/90 transition-colors font-medium body-small',
              )}
              aria-label="Start network discovery scan"
              data-testid="discovery-scan-button"
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
      if (status.scanning || running) {
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
          <div className="flex items-center gap-compact">
            {/* Full Screen button */}
            <button
              type="button"
              onClick={(): void => setIsModalOpen(true)}
              data-testid="discovery-card-maximize"
              className={cn(
                'p-1.5',
                'bg-surface-hover text-text-secondary',
                radius.md,
                'hover:bg-surface-border hover:text-text-primary transition-colors flex-center cursor-pointer',
              )}
              aria-label="Open full screen view"
              title={t('discovery.fullScreen', 'Full Screen')}
            >
              <Maximize2 className={iconTokens.size.sm} aria-hidden="true" />
            </button>

            {/* Scan button */}
            <button
              type="button"
              onClick={(): void => {
                // Submit an engine-scan job (name resolution + service
                // discovery at quick intensity) via the unified jobs spine.
                // The hook's default params encode the prior pipeline config.
                startScan().catch(() => {
                  // Errors surfaced via useEngineScan status.
                });
                // Also call onScan for backwards compatibility.
                onScan?.();
              }}
              disabled={status.scanning || running}
              className={cn(
                spacing.chip.sm,
                'bg-brand-primary text-on-brand',
                radius.md,
                'hover:bg-brand-primary/90 transition-colors font-medium caption disabled:opacity-50 disabled:cursor-not-allowed flex items-center',
                spacing.inline.sm,
              )}
              aria-label={status.scanning || running ? 'Scanning network' : 'Start network scan'}
              data-testid="discovery-scan-button"
            >
              {status.scanning || running ? (
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
          </div>
        }
      >
        {/* Discovery Summary - Minimal view showing status, subnet, device count, and categories */}
        <DiscoverySummary
          status={status}
          deviceCount={deviceCount}
          categories={categories}
          scanRunning={running}
          scanPercent={scanStatus.percentComplete}
          scanPhase={scanPhase}
          onCancelScan={cancelScan}
          t={t}
        />
        {deviceCount === 0 && !status.scanning && !running ? (
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
