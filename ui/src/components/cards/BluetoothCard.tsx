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
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useBluetoothScan } from '../../hooks/useBluetoothScan';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card, type CardStatus } from '../../shared/Card';
import { cn, icon as iconTokens } from '../../styles/theme';
import { Bluetooth, Loader, Maximize2, X } from '../ui/Icons';
import { BluetoothDeviceModal } from './BluetoothDeviceModal';
import { useCardStatusLabel } from './CardStatusLabel';

export function BluetoothCard(): React.ReactElement {
  const { t } = useTranslation('cards');
  const labelFor = useCardStatusLabel();
  const { running, status: scanStatus, devices, stats, startScan, cancelScan } = useBluetoothScan();
  const [isModalOpen, setIsModalOpen] = useState(false);

  const hasResult = scanStatus.state === 'complete' || devices.length > 0;
  const cardStatus: CardStatus = (() => {
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
      title={t('bluetooth.title')}
      icon={<Bluetooth className={iconTokens.size.md} />}
      status={cardStatus}
      statusLabel={labelFor(cardStatus)}
      data-testid="card"
      headerAction={
        <Button
          variant="ghost"
          size="xs"
          iconOnly
          icon={<Maximize2 className={iconTokens.size.sm} />}
          onClick={() => setIsModalOpen(true)}
          disabled={devices.length === 0}
          data-testid="bluetooth-card-maximize"
          aria-label={t('bluetooth.viewAll')}
        />
      }
    >
      <div className="stack-sm">
        <p className="body-small text-text-muted">{t('bluetooth.description')}</p>

        {hasResult ? (
          <div className="flex items-baseline gap-tight" data-testid="bluetooth-device-count">
            <span className="heading-2 text-text-primary">{devices.length}</span>
            <span className="body-small text-text-muted">
              {t('bluetooth.devicesFound', { count: devices.length })}
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
            <span className="body-small text-text-secondary">{t('bluetooth.scanning')}</span>
            <Button
              variant="secondary"
              size="xs"
              icon={<X className={iconTokens.size.xs} />}
              onClick={() => cancelScan().catch(() => undefined)}
              data-testid="bluetooth-cancel-button"
              aria-label={t('bluetooth.cancel')}
              className="ml-auto"
            >
              <span className="hidden sm:inline">{t('bluetooth.cancel')}</span>
            </Button>
          </div>
        ) : (
          <Button
            variant="primary"
            onClick={() => startScan().catch(() => undefined)}
            data-testid="bluetooth-scan-button"
          >
            {t('bluetooth.scanButton')}
          </Button>
        )}

        {scanStatus.error ? (
          <Alert tone="error" size="sm">
            <span data-testid="bluetooth-error">{scanStatus.error}</span>
          </Alert>
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
}
