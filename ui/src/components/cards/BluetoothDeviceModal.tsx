/**
 * BluetoothDeviceModal — full-screen view of every discovered Bluetooth device.
 *
 * Consumes the decoded fields the backend now provides (companyName,
 * serviceNames, appearanceLabel — BT.1) so the operator sees vendor, GATT
 * service, and device-type context rather than raw IDs. Each row expands to the
 * full decoded detail. Bluetooth / BLE / RSSI / GATT / UUID / MAC are protocol
 * nouns (Do-Not-Translate per the language memo).
 */

import type { TFunction } from 'i18next';
import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import type {
  BluetoothDevice,
  BluetoothDiscoveryStats,
} from '../../types/generated/bluetooth-scan-response';
import { type Column, DataTable } from '../ui/DataTable';
import { BluetoothConnected, Bluetooth as BluetoothIcon } from '../ui/icons';
import { Modal } from '../ui/Modal';

interface BluetoothDeviceModalProps {
  isOpen: boolean;
  onClose: () => void;
  devices: BluetoothDevice[];
  stats: BluetoothDiscoveryStats | null;
}

/** displayName falls back through name → alias → address so a row is never blank. */
function displayName(d: BluetoothDevice): string {
  return d.name || d.alias || d.address || '(unknown)';
}

/** rssiLabel renders the raw RSSI in dBm (RSSI is a protocol noun, kept verbatim). */
function rssiLabel(rssi: number): string {
  return rssi === 0 ? '—' : `${rssi} dBm`;
}

/** distanceLabel renders the estimated distance to one decimal, or a dash. */
function distanceLabel(m: number): string {
  return m > 0 ? `~${m.toFixed(1)} m` : '—';
}

/** A small label/value pair used in the expanded row detail. */
function DetailRow({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}): React.ReactElement {
  return (
    <div className="flex gap-compact">
      <span className="caption text-text-muted min-w-32">{label}</span>
      <span className="caption text-text-secondary break-all">{value}</span>
    </div>
  );
}

function expandedDetail(d: BluetoothDevice, t: TFunction<'cards'>): React.ReactElement {
  const flags = [
    d.isConnected ? t('bluetooth.flagConnected') : null,
    d.isPaired ? t('bluetooth.flagPaired') : null,
    d.isTrusted ? t('bluetooth.flagTrusted') : null,
    d.isConnectable ? t('bluetooth.flagConnectable') : null,
  ].filter(Boolean) as string[];

  return (
    <div className={cn('stack-xs', spacing.pad.sm)} data-testid="bluetooth-device-detail">
      {d.appearanceLabel ? (
        <DetailRow label={t('bluetooth.appearance')} value={d.appearanceLabel} />
      ) : null}
      {d.serviceNames && d.serviceNames.length > 0 ? (
        <DetailRow label={t('bluetooth.services')} value={d.serviceNames.join(', ')} />
      ) : null}
      {d.manufacturerId ? (
        <DetailRow
          label={t('bluetooth.manufacturerId')}
          value={`0x${d.manufacturerId.toString(16).toUpperCase().padStart(4, '0')}`}
        />
      ) : null}
      {d.deviceClass ? (
        <DetailRow label={t('bluetooth.deviceClass')} value={d.deviceClass} />
      ) : null}
      {d.txPower !== 0 ? (
        <DetailRow label={t('bluetooth.txPower')} value={`${d.txPower} dBm`} />
      ) : null}
      {flags.length > 0 ? (
        <DetailRow label={t('bluetooth.flags')} value={flags.join(' · ')} />
      ) : null}
      <DetailRow label={t('bluetooth.lastSeen')} value={d.lastSeen} />
    </div>
  );
}

export const BluetoothDeviceModal: React.NamedExoticComponent<BluetoothDeviceModalProps> = memo(
  function bluetoothDeviceModal({
    isOpen,
    onClose,
    devices,
    stats,
  }: BluetoothDeviceModalProps): React.ReactElement {
    const { t } = useTranslation('cards');

    const columns: Column<BluetoothDevice>[] = [
      {
        key: 'name',
        header: t('bluetooth.colName'),
        accessor: (d) => displayName(d),
        sortable: true,
        render: (d) => (
          <span className="flex items-center gap-tight">
            {d.isConnected ? (
              <BluetoothConnected className={cn(iconTokens.size.xs, 'text-brand-primary')} />
            ) : (
              <BluetoothIcon className={cn(iconTokens.size.xs, 'text-text-muted')} />
            )}
            <span className="text-text-primary">{displayName(d)}</span>
          </span>
        ),
      },
      {
        key: 'address',
        header: t('bluetooth.colAddress'),
        accessor: (d) => d.address,
        sortable: true,
      },
      {
        key: 'type',
        header: t('bluetooth.colType'),
        accessor: (d) => d.type,
        sortable: true,
        hiddenOnMobile: true,
      },
      {
        key: 'company',
        header: t('bluetooth.colManufacturer'),
        accessor: (d) => d.companyName || d.vendor || '',
        sortable: true,
        hiddenOnMobile: true,
      },
      {
        key: 'rssi',
        header: t('bluetooth.colSignal'),
        accessor: (d) => d.rssi,
        sortable: true,
        render: (d) => <span className="text-text-secondary">{rssiLabel(d.rssi)}</span>,
      },
      {
        key: 'distance',
        header: t('bluetooth.colDistance'),
        accessor: (d) => d.estDistanceM,
        sortable: true,
        hiddenOnMobile: true,
        render: (d) => <span className="text-text-secondary">{distanceLabel(d.estDistanceM)}</span>,
      },
    ];

    return (
      <Modal isOpen={isOpen} onClose={onClose} size="full" title={t('bluetooth.modalTitle')}>
        <div className="stack-md" data-testid="bluetooth-modal">
          {stats ? (
            <div className={cn('flex flex-wrap gap-default', spacing.margin.bottom.inline)}>
              <StatChip label={t('bluetooth.statTotal')} value={stats.totalDevices} />
              <StatChip label="BLE" value={stats.bleDevices} />
              <StatChip label={t('bluetooth.statClassic')} value={stats.classicDevices} />
              <StatChip label={t('bluetooth.statConnected')} value={stats.connectedDevices} />
            </div>
          ) : null}
          <div data-testid="bluetooth-device-table">
            <DataTable<BluetoothDevice>
              data={devices}
              columns={columns}
              keyExtractor={(d) => d.id || d.address}
              searchKeys={['name', 'alias', 'address', 'companyName', 'vendor']}
              searchPlaceholder={t('bluetooth.searchPlaceholder')}
              emptyMessage={t('bluetooth.empty')}
              expandedContent={(d) => expandedDetail(d, t)}
              maxHeight="max-h-[60vh]"
            />
          </div>
        </div>
      </Modal>
    );
  },
);

function StatChip({ label, value }: { label: string; value: number }): React.ReactElement {
  return (
    <div
      className={cn(
        'flex items-baseline gap-tight',
        spacing.pad.sm,
        radius.md,
        'bg-surface-sunken',
      )}
    >
      <span className="heading-3 text-text-primary">{value}</span>
      <span className="caption text-text-muted">{label}</span>
    </div>
  );
}
