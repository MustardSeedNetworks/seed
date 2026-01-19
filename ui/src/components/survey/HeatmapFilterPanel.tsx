// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * HeatmapFilterPanel Component
 *
 * Purpose: Provide controls for selecting heatmap visualization type and applying filters.
 * Allows filtering by SSID, BSSID, band, channel, and AP.
 *
 * Key Features:
 * - Heatmap metric selection (RSSI, SNR, throughput, latency, interference)
 * - SSID filter with autocomplete from survey data
 * - BSSID/AP filter with autocomplete
 * - Band and channel filtering
 * - Minimum RSSI threshold
 *
 * Usage:
 * ```typescript
 * <HeatmapFilterPanel
 *   metric={heatmapMetric}
 *   onMetricChange={setHeatmapMetric}
 *   filter={heatmapFilter}
 *   onFilterChange={setHeatmapFilter}
 *   availableNetworks={uniqueNetworks}
 *   surveyType="passive"
 * />
 * ```
 */

import {
  Activity,
  ChevronDown,
  ChevronUp,
  Clock,
  Filter,
  Gauge,
  Radio,
  Waves,
  Wifi,
  X,
} from 'lucide-react';
import type React from 'react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type {
  ApLocation,
  ChannelWidth,
  HeatmapFilter,
  HeatmapMetric,
  PhyType,
  SamplePoint,
  ScannedNetwork,
  SecurityType,
  SurveyViewMode,
  WiFiBand,
} from '../../hooks/useSurvey';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';

interface HeatmapFilterPanelProps {
  metric: HeatmapMetric;
  onMetricChange: (metric: HeatmapMetric) => void;
  filter?: HeatmapFilter;
  onFilterChange: (filter: HeatmapFilter | undefined) => void;
  samples: SamplePoint[];
  surveyType: 'passive' | 'active' | 'throughput';
  apLocations?: ApLocation[];
}

/** Heatmap metric option */
interface MetricOption {
  id: HeatmapMetric;
  labelKey: string;
  icon: React.ComponentType<{ className?: string }>;
  availableFor: Array<'passive' | 'active' | 'throughput'>;
  category: 'signal' | 'performance' | 'interference';
}

/** Available heatmap metrics */
const METRIC_OPTIONS: MetricOption[] = [
  {
    id: 'rssi',
    labelKey: 'heatmaps.rssi',
    icon: Wifi,
    availableFor: ['passive', 'active', 'throughput'],
    category: 'signal',
  },
  {
    id: 'snr',
    labelKey: 'heatmaps.snr',
    icon: Activity,
    availableFor: ['passive', 'active', 'throughput'],
    category: 'signal',
  },
  {
    id: 'noise',
    labelKey: 'heatmaps.noise',
    icon: Waves,
    availableFor: ['passive'],
    category: 'signal',
  },
  {
    id: 'throughput',
    labelKey: 'heatmaps.throughput',
    icon: Gauge,
    availableFor: ['throughput'],
    category: 'performance',
  },
  {
    id: 'latency',
    labelKey: 'heatmaps.latency',
    icon: Clock,
    availableFor: ['throughput'],
    category: 'performance',
  },
  {
    id: 'cochannel',
    labelKey: 'heatmaps.cochannel',
    icon: Radio,
    availableFor: ['passive'],
    category: 'interference',
  },
  {
    id: 'adjacent',
    labelKey: 'heatmaps.adjacent',
    icon: Radio,
    availableFor: ['passive'],
    category: 'interference',
  },
];

/**
 * HeatmapFilterPanel provides controls for selecting and filtering heatmap visualization
 */
export function HeatmapFilterPanel({
  metric,
  onMetricChange,
  filter,
  onFilterChange,
  samples,
  surveyType,
  apLocations = [],
}: HeatmapFilterPanelProps): React.ReactElement {
  const { t } = useTranslation('survey');

  // State for expanded sections
  const [showFilters, setShowFilters] = useState(false);

  // Extract unique values from samples for filter dropdowns
  const {
    uniqueSsids,
    uniqueBssids,
    uniqueChannels,
    uniqueChannelWidths,
    uniquePhyTypes,
    uniqueSecurityTypes,
    uniqueVendors,
  } = useMemo(() => {
    const ssidSet = new Set<string>();
    const bssidSet = new Set<string>();
    const channelSet = new Set<number>();
    const channelWidthSet = new Set<ChannelWidth>();
    const phyTypeSet = new Set<PhyType>();
    const securityTypeSet = new Set<SecurityType>();
    const vendorSet = new Set<string>();

    for (const sample of samples) {
      const data = sample.sampleData;

      // Handle passive survey (multiple networks)
      if ('networks' in data && Array.isArray(data.networks)) {
        for (const n of data.networks as ScannedNetwork[]) {
          if (n.ssid) {
            ssidSet.add(n.ssid);
          }
          if (n.bssid) {
            bssidSet.add(n.bssid);
          }
          if (n.channel) {
            channelSet.add(n.channel);
          }
          if (n.channelWidth) {
            channelWidthSet.add(n.channelWidth);
          }
          if (n.phyType) {
            phyTypeSet.add(n.phyType);
          }
          if (n.security) {
            securityTypeSet.add(n.security);
          }
          if (n.vendor) {
            vendorSet.add(n.vendor);
          }
        }
      }

      // Handle active/throughput survey (single network)
      if ('ssid' in data && data.ssid) {
        ssidSet.add(data.ssid as string);
      }
      if ('bssid' in data && data.bssid) {
        bssidSet.add(data.bssid as string);
      }
    }

    return {
      uniqueSsids: Array.from(ssidSet).sort(),
      uniqueBssids: Array.from(bssidSet).sort(),
      uniqueChannels: Array.from(channelSet).sort((a, b) => a - b),
      uniqueChannelWidths: Array.from(channelWidthSet).sort((a, b) => a - b),
      uniquePhyTypes: Array.from(phyTypeSet),
      uniqueSecurityTypes: Array.from(securityTypeSet),
      uniqueVendors: Array.from(vendorSet).sort(),
    };
  }, [samples]);

  // Available metrics for current survey type
  const availableMetrics = METRIC_OPTIONS.filter((m) => m.availableFor.includes(surveyType));

  // Group metrics by category
  const metricsByCategory = {
    signal: availableMetrics.filter((m) => m.category === 'signal'),
    performance: availableMetrics.filter((m) => m.category === 'performance'),
    interference: availableMetrics.filter((m) => m.category === 'interference'),
  };

  // Handle metric change
  const handleMetricSelect = (newMetric: HeatmapMetric): void => {
    onMetricChange(newMetric === metric ? null : newMetric);
  };

  // Handle filter changes
  const updateFilter = (key: keyof HeatmapFilter, value: unknown): void => {
    if (!value || (typeof value === 'string' && value === '')) {
      // Remove the key from filter
      if (!filter) {
        return;
      }
      const { [key]: _removed, ...rest } = filter;
      const hasRemainingFilters = Object.keys(rest).length > 0;
      onFilterChange(hasRemainingFilters ? rest : undefined);
    } else {
      onFilterChange({
        ...filter,
        [key]: value,
      });
    }
  };

  // Clear all filters
  const clearFilters = (): void => {
    onFilterChange(undefined);
  };

  // Check if any filters are active
  const hasActiveFilters = filter && Object.keys(filter).length > 0;

  return (
    <div class={cn('bg-surface-raised', radius.md, 'border border-surface-border', spacing.pad.sm)}>
      {/* Metric Selection */}
      <div class={spacing.margin.bottom.content}>
        <h4 class={cn('caption font-medium text-text-muted', spacing.margin.bottom.tight)}>
          {t('heatmaps.title')}
        </h4>

        {/* Signal metrics */}
        {metricsByCategory.signal.length > 0 ? (
          <div class={cn(layout.inline.default, 'flex-wrap', spacing.margin.bottom.tight)}>
            {metricsByCategory.signal.map((option) => {
              const ICON = option.icon;
              const isSelected = metric === option.id;
              return (
                <button
                  type="button"
                  key={option.id}
                  onClick={(): void => handleMetricSelect(option.id)}
                  class={cn(
                    button.size.xs,
                    radius.md,
                    layout.inline.default,
                    'transition-colors',
                    isSelected
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-base border border-surface-border hover:bg-surface-hover',
                  )}
                >
                  <ICON class={iconTokens.size.xs} />
                  <span>{t(option.labelKey as never)}</span>
                </button>
              );
            })}
          </div>
        ) : null}

        {/* Performance metrics */}
        {metricsByCategory.performance.length > 0 ? (
          <div class={cn(layout.inline.default, 'flex-wrap', spacing.margin.bottom.tight)}>
            {metricsByCategory.performance.map((option) => {
              const ICON = option.icon;
              const isSelected = metric === option.id;
              return (
                <button
                  type="button"
                  key={option.id}
                  onClick={(): void => handleMetricSelect(option.id)}
                  class={cn(
                    button.size.xs,
                    radius.md,
                    layout.inline.default,
                    'transition-colors',
                    isSelected
                      ? 'bg-green-600 text-text-inverse'
                      : 'bg-surface-base border border-surface-border hover:bg-surface-hover',
                  )}
                >
                  <ICON class={iconTokens.size.xs} />
                  <span>{t(option.labelKey as never)}</span>
                </button>
              );
            })}
          </div>
        ) : null}

        {/* Interference metrics */}
        {metricsByCategory.interference.length > 0 ? (
          <div class={cn(layout.inline.default, 'flex-wrap')}>
            {metricsByCategory.interference.map((option) => {
              const ICON = option.icon;
              const isSelected = metric === option.id;
              return (
                <button
                  type="button"
                  key={option.id}
                  onClick={(): void => handleMetricSelect(option.id)}
                  class={cn(
                    button.size.xs,
                    radius.md,
                    layout.inline.default,
                    'transition-colors',
                    isSelected
                      ? 'bg-purple-600 text-text-inverse'
                      : 'bg-surface-base border border-surface-border hover:bg-surface-hover',
                  )}
                >
                  <ICON class={iconTokens.size.xs} />
                  <span>{t(option.labelKey as never)}</span>
                </button>
              );
            })}
          </div>
        ) : null}
      </div>

      {/* Filter Toggle */}
      <button
        type="button"
        onClick={(): void => setShowFilters(!showFilters)}
        class={cn(
          layout.inline.default,
          'w-full justify-between',
          spacing.pad.sm,
          radius.md,
          'bg-surface-base border border-surface-border hover:bg-surface-hover',
        )}
      >
        <div class={cn(layout.inline.default)}>
          <Filter class={iconTokens.size.sm} />
          <span class="body-small">{t('heatmaps.filters')}</span>
          {hasActiveFilters ? (
            <span class="px-1.5 py-0.5 text-xs bg-brand-primary text-text-inverse rounded-full">
              {filter ? Object.keys(filter).length : 0}
            </span>
          ) : null}
        </div>
        {showFilters ? (
          <ChevronUp class={iconTokens.size.sm} />
        ) : (
          <ChevronDown class={iconTokens.size.sm} />
        )}
      </button>

      {/* Filter Options */}
      {showFilters ? (
        <div class={cn(layout.stack.default, spacing.margin.top.content)}>
          {/* SSID Filter */}
          <div>
            <label for="filter-ssid" class="caption text-text-muted">
              {t('heatmaps.filterSsid')}
            </label>
            <select
              id="filter-ssid"
              value={filter?.ssid || ''}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFilter('ssid', e.target.value)
              }
              class={cn(
                'w-full',
                spacing.pad.sm,
                radius.md,
                'border border-surface-border bg-surface-base body-small',
              )}
            >
              <option value="">{t('heatmaps.allSsids')}</option>
              {uniqueSsids.map((ssid) => (
                <option key={ssid} value={ssid}>
                  {ssid || t('heatmaps.hiddenSsid')}
                </option>
              ))}
            </select>
          </div>

          {/* BSSID Filter */}
          <div>
            <label for="filter-bssid" class="caption text-text-muted">
              {t('heatmaps.filterBssid')}
            </label>
            <select
              id="filter-bssid"
              value={filter?.bssid || ''}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                updateFilter('bssid', e.target.value)
              }
              class={cn(
                'w-full',
                spacing.pad.sm,
                radius.md,
                'border border-surface-border bg-surface-base body-small',
              )}
            >
              <option value="">{t('heatmaps.allAps')}</option>
              {uniqueBssids.map((bssid) => (
                <option key={bssid} value={bssid}>
                  {bssid}
                </option>
              ))}
            </select>
          </div>

          {/* AP Location Filter */}
          {apLocations.length > 0 ? (
            <div>
              <label for="filter-ap-location" class="caption text-text-muted">
                {t('heatmaps.filterAp')}
              </label>
              <select
                id="filter-ap-location"
                value={filter?.apId || ''}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateFilter('apId', e.target.value)
                }
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base body-small',
                )}
              >
                <option value="">{t('heatmaps.allApLocations')}</option>
                {apLocations.map((ap) => (
                  <option key={ap.id} value={ap.id}>
                    {ap.label} {ap.bssid ? `(${ap.bssid})` : ''}
                  </option>
                ))}
              </select>
            </div>
          ) : null}

          {/* Band Filter */}
          <div>
            <span class="caption text-text-muted">{t('heatmaps.filterBand')}</span>
            <div class={cn(layout.inline.default)}>
              {(['2.4', '5', '6'] as WiFiBand[]).map((band) => {
                const getBandColorClass = (): string => {
                  if (filter?.band !== band) {
                    return 'bg-surface-base border border-surface-border hover:bg-surface-hover';
                  }
                  if (band === '2.4') {
                    return 'bg-blue-500 text-text-inverse';
                  }
                  if (band === '5') {
                    return 'bg-green-500 text-text-inverse';
                  }
                  return 'bg-purple-500 text-text-inverse';
                };
                return (
                  <button
                    type="button"
                    key={band}
                    onClick={(): void =>
                      updateFilter('band', filter?.band === band ? undefined : band)
                    }
                    class={cn(button.size.xs, radius.md, 'transition-colors', getBandColorClass())}
                  >
                    {band} GHz
                  </button>
                );
              })}
            </div>
          </div>

          {/* Channel Filter */}
          {uniqueChannels.length > 0 ? (
            <div>
              <label for="filter-channel" class="caption text-text-muted">
                {t('heatmaps.filterChannel')}
              </label>
              <select
                id="filter-channel"
                value={filter?.channel || ''}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateFilter(
                    'channel',
                    e.target.value ? Number.parseInt(e.target.value, 10) : undefined,
                  )
                }
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base body-small',
                )}
              >
                <option value="">{t('heatmaps.allChannels')}</option>
                {uniqueChannels.map((ch) => (
                  <option key={ch} value={ch}>
                    {t('heatmaps.channel')} {ch}
                  </option>
                ))}
              </select>
            </div>
          ) : null}

          {/* Channel Width Filter */}
          {uniqueChannelWidths.length > 0 ? (
            <div>
              <span class="caption text-text-muted">{t('heatmaps.filterChannelWidth')}</span>
              <div class={cn(layout.inline.default, 'flex-wrap')}>
                {([20, 40, 80, 160, 320] as ChannelWidth[]).map((width) => {
                  const getWidthColorClass = (): string => {
                    if (filter?.channelWidth === width) {
                      return 'bg-cyan-500 text-text-inverse';
                    }
                    if (uniqueChannelWidths.includes(width)) {
                      return 'bg-surface-base border border-surface-border hover:bg-surface-hover';
                    }
                    return 'bg-surface-base border border-surface-border opacity-40';
                  };
                  return (
                    <button
                      type="button"
                      key={width}
                      onClick={(): void =>
                        updateFilter(
                          'channelWidth',
                          filter?.channelWidth === width ? undefined : width,
                        )
                      }
                      disabled={!uniqueChannelWidths.includes(width)}
                      class={cn(
                        button.size.xs,
                        radius.md,
                        'transition-colors',
                        getWidthColorClass(),
                      )}
                    >
                      {width} MHz
                    </button>
                  );
                })}
              </div>
            </div>
          ) : null}

          {/* PHY Type / 802.11 Standard Filter */}
          {uniquePhyTypes.length > 0 ? (
            <div>
              <span class="caption text-text-muted">{t('heatmaps.filterPhyType')}</span>
              <div class={cn(layout.inline.default, 'flex-wrap')}>
                {(['a', 'b', 'g', 'n', 'ac', 'ax', 'be'] as PhyType[]).map((phy) => {
                  const getPhyColorClass = (): string => {
                    if (filter?.phyType === phy) {
                      return 'bg-indigo-500 text-text-inverse';
                    }
                    if (uniquePhyTypes.includes(phy)) {
                      return 'bg-surface-base border border-surface-border hover:bg-surface-hover';
                    }
                    return 'bg-surface-base border border-surface-border opacity-40';
                  };
                  return (
                    <button
                      type="button"
                      key={phy}
                      onClick={(): void =>
                        updateFilter('phyType', filter?.phyType === phy ? undefined : phy)
                      }
                      disabled={!uniquePhyTypes.includes(phy)}
                      class={cn(button.size.xs, radius.md, 'transition-colors', getPhyColorClass())}
                    >
                      802.11{phy}
                    </button>
                  );
                })}
              </div>
            </div>
          ) : null}

          {/* Security Type Filter */}
          {uniqueSecurityTypes.length > 0 ? (
            <div>
              <label for="filter-security" class="caption text-text-muted">
                {t('heatmaps.filterSecurity')}
              </label>
              <select
                id="filter-security"
                value={filter?.security || ''}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateFilter('security', e.target.value || undefined)
                }
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base body-small',
                )}
              >
                <option value="">{t('heatmaps.allSecurity')}</option>
                {uniqueSecurityTypes.map((sec) => (
                  <option key={sec} value={sec}>
                    {t(`heatmaps.security.${sec}` as never)}
                  </option>
                ))}
              </select>
            </div>
          ) : null}

          {/* Vendor Filter */}
          {uniqueVendors.length > 0 ? (
            <div>
              <label for="filter-vendor" class="caption text-text-muted">
                {t('heatmaps.filterVendor')}
              </label>
              <select
                id="filter-vendor"
                value={filter?.vendor || ''}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateFilter('vendor', e.target.value || undefined)
                }
                class={cn(
                  'w-full',
                  spacing.pad.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base body-small',
                )}
              >
                <option value="">{t('heatmaps.allVendors')}</option>
                {uniqueVendors.map((vendor) => (
                  <option key={vendor} value={vendor}>
                    {vendor}
                  </option>
                ))}
              </select>
            </div>
          ) : null}

          {/* Survey View Mode (for imported surveys with multiple modes) */}
          <div>
            <span class="caption text-text-muted">{t('heatmaps.viewMode')}</span>
            <div class={cn(layout.inline.default, 'flex-wrap')}>
              {(['all', 'passive', 'active', 'client', 'probingClient'] as SurveyViewMode[]).map(
                (mode) => (
                  <button
                    type="button"
                    key={mode}
                    onClick={(): void =>
                      updateFilter('viewMode', filter?.viewMode === mode ? undefined : mode)
                    }
                    class={cn(
                      button.size.xs,
                      radius.md,
                      'transition-colors',
                      filter?.viewMode === mode
                        ? 'bg-amber-500 text-text-inverse'
                        : 'bg-surface-base border border-surface-border hover:bg-surface-hover',
                    )}
                  >
                    {t(`heatmaps.viewModes.${mode}` as never)}
                  </button>
                ),
              )}
            </div>
          </div>

          {/* Min RSSI Filter */}
          <div>
            <label for="filter-min-rssi" class="caption text-text-muted">
              {t('heatmaps.minRssi')}
            </label>
            <div class={cn(layout.inline.default)}>
              <input
                id="filter-min-rssi"
                type="range"
                min="-100"
                max="-30"
                value={filter?.minRssi || -100}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateFilter(
                    'minRssi',
                    Number.parseInt(e.target.value, 10) === -100
                      ? undefined
                      : Number.parseInt(e.target.value, 10),
                  )
                }
                class="flex-1"
              />
              <span class="body-small w-16 text-right">{filter?.minRssi || -100} dBm</span>
            </div>
          </div>

          {/* Clear Filters */}
          {hasActiveFilters ? (
            <button
              type="button"
              onClick={clearFilters}
              class={cn(
                button.size.sm,
                radius.md,
                'border border-surface-border hover:bg-surface-hover',
                layout.inline.default,
                'justify-center',
              )}
            >
              <X class={iconTokens.size.sm} />
              <span>{t('heatmaps.clearFilters')}</span>
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
