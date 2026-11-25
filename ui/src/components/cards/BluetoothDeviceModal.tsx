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
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useTableLabels } from '../../hooks/useTableLabels';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { Modal } from '../../shared/Modal';
import { SearchInput } from '../../shared/SearchInput';
import { cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import type {
  BluetoothDevice,
  BluetoothDiscoveryStats,
} from '../../types/generated/bluetooth-scan-response';
import { BluetoothConnected, Bluetooth as BluetoothIcon } from '../ui/Icons';

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

function deviceKey(d: BluetoothDevice): string {
  return d.id || d.address;
}

function expandedDetail(
  d: BluetoothDevice,
  t: TFunction<readonly ['cards', 'common']>,
): React.ReactElement {
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

export function BluetoothDeviceModal({
  isOpen,
  onClose,
  devices,
  stats,
}: BluetoothDeviceModalProps): React.ReactElement {
  const { t } = useTranslation(['cards', 'common']);
  const [filter, setFilter] = useState('');
  const [openKey, setOpenKey] = useState<string | null>(null);
  const labels = useTableLabels<BluetoothDevice>({ loading: t('bluetooth.modalTitle') });

  const needle = filter.trim().toLowerCase();
  const visible = devices.filter((d) =>
    [d.name, d.alias, d.address, d.companyName, d.vendor].some((value) =>
      value?.toLowerCase().includes(needle),
    ),
  );
  const openDevice = devices.find((d) => deviceKey(d) === openKey);

  const columns: DataTableColumn<BluetoothDevice>[] = [
    {
      key: 'name',
      header: t('bluetooth.colName'),
      sortValue: (d) => displayName(d),
      cell: (d) => (
        <span className="flex items-center gap-1">
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
      sortValue: (d) => d.address,
      cell: (d) => d.address,
    },
    {
      key: 'type',
      header: t('bluetooth.colType'),
      sortValue: (d) => d.type,
      cell: (d) => d.type,
    },
    {
      key: 'company',
      header: t('bluetooth.colManufacturer'),
      sortValue: (d) => d.companyName || d.vendor || '',
      cell: (d) => d.companyName || d.vendor || '',
    },
    {
      key: 'rssi',
      header: t('bluetooth.colSignal'),
      sortValue: (d) => d.rssi,
      cell: (d) => <span className="text-text-secondary">{rssiLabel(d.rssi)}</span>,
    },
    {
      key: 'distance',
      header: t('bluetooth.colDistance'),
      sortValue: (d) => d.estDistanceM,
      cell: (d) => <span className="text-text-secondary">{distanceLabel(d.estDistanceM)}</span>,
    },
  ];

  return (
    <Modal
      open={isOpen}
      onClose={onClose}
      size="full"
      title={t('bluetooth.modalTitle')}
      closeLabel={t('common:buttons.close')}
    >
      <div className="stack-md" data-testid="bluetooth-modal">
        {stats ? (
          <div className={cn('flex flex-wrap gap-default', spacing.margin.bottom.inline)}>
            <StatChip label={t('bluetooth.statTotal')} value={stats.totalDevices} />
            <StatChip label="BLE" value={stats.bleDevices} />
            <StatChip label={t('bluetooth.statClassic')} value={stats.classicDevices} />
            <StatChip label={t('bluetooth.statConnected')} value={stats.connectedDevices} />
          </div>
        ) : null}
        <SearchInput
          value={filter}
          onValueChange={setFilter}
          label={t('bluetooth.searchPlaceholder')}
          clearLabel={t('common:dataTable.clearFilter')}
          placeholder={t('bluetooth.searchPlaceholder')}
        />
        <DataTable
          rows={visible}
          columns={columns}
          getRowKey={deviceKey}
          caption={t('bluetooth.modalTitle')}
          empty={
            <EmptyState
              title={devices.length === 0 ? t('bluetooth.empty') : t('common:dataTable.noMatches')}
            />
          }
          labels={labels}
          maxHeight={480}
          onRowClick={(d) => setOpenKey(openKey === deviceKey(d) ? null : deviceKey(d))}
          selectedRowKey={openKey ?? undefined}
          data-testid="bluetooth-device-table"
        />
        {openDevice ? expandedDetail(openDevice, t) : null}
      </div>
    </Modal>
  );
}

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
