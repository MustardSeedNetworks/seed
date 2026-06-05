/**
 * BluetoothCard — discover and surface nearby Bluetooth devices.
 *
 * Runs a scan through the unified jobs spine (useBluetoothScan → bluetooth-scan
 * job) and shows a compact summary: how many devices were found, how many are
 * connected, and a button to open the full-screen device table (decoded vendor
 * / GATT service / appearance per device, BT.1). Self-contained — it owns its
 * own scan state, like the other security cards.
 *
 * Bluetooth / BLE / RSSI are protocol nouns (Do-Not-Translate).
 */

import type React from 'react';
import { memo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useBluetoothScan } from '../../hooks/useBluetoothScan';
import {
  button,
  cn,
  icon as iconTokens,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';
import { Card, type Status } from '../ui/card';
import { Bluetooth, Loader, Maximize2, X } from '../ui/icons';
import { BluetoothDeviceModal } from './BluetoothDeviceModal';

export const BluetoothCard: React.NamedExoticComponent = memo(
  function bluetoothCard(): React.ReactElement {
    const { t } = useTranslation('cards');
    const {
      running,
      status: scanStatus,
      devices,
      stats,
      startScan,
      cancelScan,
    } = useBluetoothScan();
    const [isModalOpen, setIsModalOpen] = useState(false);

    const hasResult = scanStatus.state === 'complete' || devices.length > 0;
    const cardStatus: Status = (() => {
      if (running) {
        return 'loading';
      }
      if (scanStatus.error) {
        return 'error';
      }
      return hasResult ? 'success' : 'unknown';
    })();

    return (
      <Card
        title={t('bluetooth.title', { defaultValue: 'Bluetooth' })}
        icon={<Bluetooth className={iconTokens.size.md} />}
        status={cardStatus}
        headerAction={
          <button
            type="button"
            onClick={() => setIsModalOpen(true)}
            disabled={devices.length === 0}
            data-testid="bluetooth-card-maximize"
            className={cn(
              'p-1 text-text-muted hover:text-text-primary transition-colors',
              radius.md,
              'hover:bg-surface-hover disabled:opacity-40 disabled:cursor-not-allowed',
            )}
            aria-label={t('bluetooth.viewAll', { defaultValue: 'View all devices' })}
          >
            <Maximize2 className={iconTokens.size.sm} />
          </button>
        }
      >
        <div className="stack-sm">
          <p className="body-small text-text-muted">
            {t('bluetooth.description', {
              defaultValue:
                'Scan for nearby Bluetooth and BLE devices and decode what they advertise.',
            })}
          </p>

          {hasResult ? (
            <div className="flex items-baseline gap-tight" data-testid="bluetooth-device-count">
              <span className="heading-2 text-text-primary">{devices.length}</span>
              <span className="body-small text-text-muted">
                {t('bluetooth.devicesFound', {
                  count: devices.length,
                  defaultValue: 'devices found',
                })}
                {stats && stats.connectedDevices > 0
                  ? ` · ${t('bluetooth.connectedCount', {
                      count: stats.connectedDevices,
                      defaultValue: `${stats.connectedDevices} connected`,
                    })}`
                  : ''}
              </span>
            </div>
          ) : null}

          {running ? (
            <div className="flex items-center gap-compact" data-testid="bluetooth-scanning">
              <Loader className={cn(iconTokens.size.sm, 'text-brand-primary animate-spin')} />
              <span className="body-small text-text-secondary">
                {t('bluetooth.scanning', { defaultValue: 'Scanning…' })}
              </span>
              <button
                type="button"
                onClick={() => cancelScan().catch(() => undefined)}
                data-testid="bluetooth-cancel-button"
                className={cn(
                  button.base,
                  button.size.sm,
                  button.variant.secondary,
                  'ml-auto flex items-center gap-tight',
                )}
                aria-label={t('bluetooth.cancel', { defaultValue: 'Cancel' })}
              >
                <X className={iconTokens.size.xs} />
                <span className="hidden sm:inline">
                  {t('bluetooth.cancel', { defaultValue: 'Cancel' })}
                </span>
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => startScan().catch(() => undefined)}
              data-testid="bluetooth-scan-button"
              className={cn(
                button.size.md,
                'bg-brand-primary text-on-brand',
                radius.md,
                'font-medium hover:bg-brand-primary/90 disabled:opacity-50',
              )}
            >
              {t('bluetooth.scanButton', { defaultValue: 'Scan for devices' })}
            </button>
          )}

          {scanStatus.error ? (
            <div className={cn(spacing.pad.sm, statusColor.bg.errorSoft, radius.md)} role="alert">
              <span className="body-small text-status-error" data-testid="bluetooth-error">
                {scanStatus.error}
              </span>
            </div>
          ) : null}
        </div>

        <BluetoothDeviceModal
          isOpen={isModalOpen}
          onClose={() => setIsModalOpen(false)}
          devices={devices}
          stats={stats}
        />
      </Card>
    );
  },
);
