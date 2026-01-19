/**
 * Link Status Card Component
 *
 * Displays physical link layer (Layer 2) and network layer (Layer 3) status.
 *
 * Features:
 * - Link state detection (up/down)
 * - Carrier signal detection (physical link present)
 * - IP configuration status
 * - Connection speed and duplex mode
 * - Negotiated speeds (from auto-negotiation)
 * - MTU and auto-negotiation settings
 * - Link flap counting (24-hour window)
 * - Uptime tracking
 * - Link state history
 *
 * Status Indicators:
 * - **Error (Red)**: No physical carrier detected (L2 down)
 * - **Warning (Yellow)**: Carrier present but no IP address (L3 down)
 * - **Success (Green)**: Both L2 and L3 up, fully connected
 *
 * The card is the primary indicator of network interface health.
 */

import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { CardDivider, CardRow, CardValue, type Status } from '../ui/Card';
import { Cable } from '../ui/Icons';
import { Skeleton } from '../ui/Skeleton';
import { BaseCard } from './BaseCard';

/**
 * Historical link state event
 */
interface LinkHistoryEvent {
  state: string; // State change ("up", "down", "flap", etc.)
  timestamp: string; // ISO 8601 timestamp
}

/**
 * PoE (Power over Ethernet) status
 */
interface PoeInfo {
  detected: boolean;
  standard?: string; // 802.3af, 802.3at, 802.3bt
  class?: number;
  powerMw?: number;
  voltage?: number;
}

/**
 * SFP DDM (Digital Diagnostics Monitoring) readings
 */
interface SfpDdmInfo {
  temperature: number; // Celsius
  voltage: number; // Volts
  txPowerDbm: number;
  txPowerMw: number;
  rxPowerDbm: number;
  rxPowerMw: number;
  laserBiasMa: number;
  alarms?: string[];
  warnings?: string[];
}

/**
 * SFP module information
 */
interface SfpInfo {
  present: boolean;
  vendor?: string;
  partNumber?: string;
  serial?: string;
  type?: string; // SR, LR, ER
  wavelength?: number; // nm
  distance?: number; // meters
  connector?: string; // LC, SC
  ddmSupport: boolean;
  ddm?: SfpDdmInfo;
}

/**
 * Link layer and network layer status data
 */
export interface LinkData {
  linkUp: boolean; // Link is administratively up
  carrier: boolean; // Physical carrier/link detected (L2)
  hasIp: boolean; // Has routable IP address (L3)
  speed: string; // Current connection speed (e.g., "1000Mb/s")
  duplex: string; // Duplex mode ("full" or "half")
  advertisedSpeeds: string[]; // Speeds supported by auto-negotiation
  mtu?: number; // Maximum transmission unit
  autoNeg?: boolean; // Auto-negotiation enabled
  flapCount24h?: number; // Number of link state changes in last 24h
  history?: LinkHistoryEvent[]; // Recent link state events
  uptimeMs?: number; // Time since last state change (ms)
  poe?: PoeInfo; // Power over Ethernet status
  sfp?: SfpInfo; // SFP module and DDM info
}

/**
 * Props for Link Card
 */
interface LinkCardProps {
  data: LinkData | null; // Link status data
  loading?: boolean; // True while loading
}

/**
 * Determines card status based on link and IP state.
 * Uses both L2 (carrier) and L3 (IP) information.
 *
 * @param data - Link status data
 * @returns Status indicator ('success', 'warning', 'error')
 */
function getStatus(data: LinkData): Status {
  if (!data.carrier) {
    return 'error'; // No physical link
  }
  if (!data.hasIp) {
    return 'warning'; // Carrier but no IP
  }
  return 'success'; // Fully connected
}

function _linkLoadingSkeleton(): JSX.Element {
  return (
    <>
      <Skeleton class={cn('h-8 w-32', spacing.margin.bottom.heading)} />
      <div class={cn('stack-sm', spacing.margin.top.content)}>
        <div class={layout.flex.between}>
          <Skeleton class="h-3 w-16" />
          <Skeleton class="h-3 w-20" />
        </div>
        <div class={layout.flex.between}>
          <Skeleton class="h-3 w-12" />
          <Skeleton class="h-3 w-8" />
        </div>
      </div>
    </>
  );
}

export const LinkCard: React.MemoExoticComponent<(props: LinkCardProps) => JSX.Element> = memo(
  function linkCard({ data, loading }: LinkCardProps): JSX.Element {
    const { t } = useTranslation('cards');
    const { t: tc } = useTranslation('common');

    const getLocalizedStatusText = (linkData: LinkData): string => {
      if (!linkData.carrier) {
        return tc('status.noCarrier');
      }
      if (!linkData.hasIp) {
        return tc('status.noIP');
      }
      return linkData.speed || tc('status.connected');
    };

    return (
      <BaseCard
        title={t('link.title')}
        icon={<Cable class={iconTokens.size.md} />}
        data={data}
        loading={loading}
        getStatus={getStatus}
        loadingContent={<linkLoadingSkeleton />}
        emptyMessage={tc('status.noData')}
      >
        {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex link card with multiple status displays */}
        {(linkData: LinkData): JSX.Element => {
          const status = getStatus(linkData);
          return (
            <>
              <CardValue value={getLocalizedStatusText(linkData)} size="lg" status={status} />
              <CardDivider />
              <CardRow
                label={t('link.carrier')}
                value={linkData.carrier ? tc('status.connected') : tc('status.noSignal')}
              />
              {linkData.carrier ? (
                <>
                  <CardRow
                    label={t('link.duplex')}
                    value={linkData.duplex || tc('status.unknown')}
                  />
                  {linkData.mtu ? (
                    <CardRow label={t('link.mtu')} value={linkData.mtu.toString()} />
                  ) : null}
                  {linkData.autoNeg !== undefined && (
                    <CardRow
                      label={t('link.autoNeg')}
                      value={linkData.autoNeg ? tc('status.on') : tc('status.off')}
                    />
                  )}
                  {linkData.flapCount24h !== undefined && (
                    <CardRow label={t('link.flaps24h')} value={linkData.flapCount24h.toString()} />
                  )}
                  {linkData.advertisedSpeeds && linkData.advertisedSpeeds.length > 0 && (
                    <div class={spacing.margin.top.inline}>
                      <p class={cn('caption', spacing.margin.bottom.inline)}>
                        {t('link.advertisedSpeeds')}
                      </p>
                      <div class={layout.inline.wrap}>
                        {linkData.advertisedSpeeds.map((speed) => (
                          <span
                            key={speed}
                            class={cn('caption bg-surface-hover', spacing.chip.sm, radius.default)}
                          >
                            {speed}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* PoE Status */}
                  {linkData.poe?.detected ? (
                    <>
                      <CardDivider />
                      <p
                        class={cn(
                          'caption font-medium text-text-muted',
                          spacing.margin.bottom.tight,
                        )}
                      >
                        {t('link.poe', 'Power over Ethernet')}
                      </p>
                      <CardRow
                        label={t('link.poeStandard', 'Standard')}
                        value={linkData.poe.standard || 'Unknown'}
                      />
                      {linkData.poe.class !== undefined && (
                        <CardRow
                          label={t('link.poeClass', 'Class')}
                          value={linkData.poe.class.toString()}
                        />
                      )}
                      {linkData.poe.powerMw !== undefined && (
                        <CardRow
                          label={t('link.poePower', 'Power')}
                          value={`${(linkData.poe.powerMw / 1000).toFixed(1)} W`}
                        />
                      )}
                      {linkData.poe.voltage !== undefined && (
                        <CardRow
                          label={t('link.poeVoltage', 'Voltage')}
                          value={`${linkData.poe.voltage.toFixed(1)} V`}
                        />
                      )}
                    </>
                  ) : null}

                  {/* SFP Module Info */}
                  {linkData.sfp?.present ? (
                    <>
                      <CardDivider />
                      <p
                        class={cn(
                          'caption font-medium text-text-muted',
                          spacing.margin.bottom.tight,
                        )}
                      >
                        {t('link.sfp', 'SFP Module')}
                      </p>
                      {linkData.sfp.vendor ? (
                        <CardRow
                          label={t('link.sfpVendor', 'Vendor')}
                          value={linkData.sfp.vendor}
                        />
                      ) : null}
                      {linkData.sfp.type ? (
                        <CardRow label={t('link.sfpType', 'Type')} value={linkData.sfp.type} />
                      ) : null}
                      {linkData.sfp.wavelength ? (
                        <CardRow
                          label={t('link.sfpWavelength', 'Wavelength')}
                          value={`${linkData.sfp.wavelength} nm`}
                        />
                      ) : null}
                      {linkData.sfp.distance ? (
                        <CardRow
                          label={t('link.sfpDistance', 'Max Distance')}
                          value={`${linkData.sfp.distance} m`}
                        />
                      ) : null}

                      {/* SFP DDM Readings */}
                      {linkData.sfp.ddmSupport && linkData.sfp.ddm ? (
                        <div class={spacing.margin.top.inline}>
                          <p
                            class={cn(
                              'caption font-medium text-text-muted',
                              spacing.margin.bottom.tight,
                            )}
                          >
                            {t('link.ddm', 'DDM Readings')}
                          </p>
                          <CardRow
                            label={t('link.ddmTemp', 'Temperature')}
                            value={`${linkData.sfp.ddm.temperature.toFixed(1)}°C`}
                          />
                          <CardRow
                            label={t('link.ddmVoltage', 'Voltage')}
                            value={`${linkData.sfp.ddm.voltage.toFixed(2)} V`}
                          />
                          <CardRow
                            label={t('link.ddmTxPower', 'TX Power')}
                            value={`${linkData.sfp.ddm.txPowerDbm.toFixed(1)} dBm`}
                          />
                          <CardRow
                            label={t('link.ddmRxPower', 'RX Power')}
                            value={`${linkData.sfp.ddm.rxPowerDbm.toFixed(1)} dBm`}
                          />
                          {linkData.sfp.ddm.alarms && linkData.sfp.ddm.alarms.length > 0 && (
                            <div class={cn('caption text-status-error', spacing.margin.top.tight)}>
                              {linkData.sfp.ddm.alarms.map((alarm) => (
                                <p key={alarm}>{alarm}</p>
                              ))}
                            </div>
                          )}
                          {linkData.sfp.ddm.warnings && linkData.sfp.ddm.warnings.length > 0 && (
                            <div
                              class={cn('caption text-status-warning', spacing.margin.top.tight)}
                            >
                              {linkData.sfp.ddm.warnings.map((warning) => (
                                <p key={warning}>{warning}</p>
                              ))}
                            </div>
                          )}
                        </div>
                      ) : null}
                    </>
                  ) : null}
                </>
              ) : null}
            </>
          );
        }}
      </BaseCard>
    );
  },
);
