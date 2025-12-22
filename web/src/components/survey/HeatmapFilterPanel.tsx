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

import { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import {
  Wifi,
  Activity,
  Gauge,
  Clock,
  Radio,
  Waves,
  Filter,
  X,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import {
  cn,
  radius,
  spacing,
  layout,
  button,
  icon as iconTokens,
} from "../../styles/theme";
import type {
  HeatmapMetric,
  HeatmapFilter,
  SamplePoint,
  WiFiBand,
  APLocation,
  ChannelWidth,
  PhyType,
  SecurityType,
  SurveyViewMode,
  ScannedNetwork,
} from "../../hooks/useSurvey";

interface HeatmapFilterPanelProps {
  metric: HeatmapMetric;
  onMetricChange: (metric: HeatmapMetric) => void;
  filter?: HeatmapFilter;
  onFilterChange: (filter: HeatmapFilter | undefined) => void;
  samples: SamplePoint[];
  surveyType: "passive" | "active" | "throughput";
  apLocations?: APLocation[];
}

/** Heatmap metric option */
interface MetricOption {
  id: HeatmapMetric;
  labelKey: string;
  icon: React.ComponentType<{ className?: string }>;
  availableFor: Array<"passive" | "active" | "throughput">;
  category: "signal" | "performance" | "interference";
}

/** Available heatmap metrics */
const METRIC_OPTIONS: MetricOption[] = [
  {
    id: "rssi",
    labelKey: "heatmaps.rssi",
    icon: Wifi,
    availableFor: ["passive", "active", "throughput"],
    category: "signal",
  },
  {
    id: "snr",
    labelKey: "heatmaps.snr",
    icon: Activity,
    availableFor: ["passive", "active", "throughput"],
    category: "signal",
  },
  {
    id: "noise",
    labelKey: "heatmaps.noise",
    icon: Waves,
    availableFor: ["passive"],
    category: "signal",
  },
  {
    id: "throughput",
    labelKey: "heatmaps.throughput",
    icon: Gauge,
    availableFor: ["throughput"],
    category: "performance",
  },
  {
    id: "latency",
    labelKey: "heatmaps.latency",
    icon: Clock,
    availableFor: ["throughput"],
    category: "performance",
  },
  {
    id: "cochannel",
    labelKey: "heatmaps.cochannel",
    icon: Radio,
    availableFor: ["passive"],
    category: "interference",
  },
  {
    id: "adjacent",
    labelKey: "heatmaps.adjacent",
    icon: Radio,
    availableFor: ["passive"],
    category: "interference",
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
}: HeatmapFilterPanelProps) {
  const { t } = useTranslation("survey");

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

    samples.forEach((sample) => {
      const data = sample.sampleData;

      // Handle passive survey (multiple networks)
      if ("networks" in data && Array.isArray(data.networks)) {
        (data.networks as ScannedNetwork[]).forEach((n) => {
          if (n.ssid) ssidSet.add(n.ssid);
          if (n.bssid) bssidSet.add(n.bssid);
          if (n.channel) channelSet.add(n.channel);
          if (n.channelWidth) channelWidthSet.add(n.channelWidth);
          if (n.phyType) phyTypeSet.add(n.phyType);
          if (n.security) securityTypeSet.add(n.security);
          if (n.vendor) vendorSet.add(n.vendor);
        });
      }

      // Handle active/throughput survey (single network)
      if ("ssid" in data && data.ssid) {
        ssidSet.add(data.ssid as string);
      }
      if ("bssid" in data && data.bssid) {
        bssidSet.add(data.bssid as string);
      }
    });

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
  const availableMetrics = METRIC_OPTIONS.filter((m) =>
    m.availableFor.includes(surveyType)
  );

  // Group metrics by category
  const metricsByCategory = {
    signal: availableMetrics.filter((m) => m.category === "signal"),
    performance: availableMetrics.filter((m) => m.category === "performance"),
    interference: availableMetrics.filter((m) => m.category === "interference"),
  };

  // Handle metric change
  const handleMetricSelect = (newMetric: HeatmapMetric) => {
    onMetricChange(newMetric === metric ? null : newMetric);
  };

  // Handle filter changes
  const updateFilter = (key: keyof HeatmapFilter, value: unknown) => {
    if (!value || (typeof value === "string" && value === "")) {
      // Remove the key from filter
      if (!filter) return;
      const { [key]: _removed, ...rest } = filter;
      void _removed; // Consume the removed value
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
  const clearFilters = () => {
    onFilterChange(undefined);
  };

  // Check if any filters are active
  const hasActiveFilters = filter && Object.keys(filter).length > 0;

  return (
    <div
      className={cn(
        "bg-surface-raised",
        radius.md,
        "border border-surface-border",
        spacing.pad.sm
      )}
    >
      {/* Metric Selection */}
      <div className={spacing.margin.bottom.content}>
        <h4
          className={cn(
            "caption font-medium text-text-muted",
            spacing.margin.bottom.tight
          )}
        >
          {t("heatmaps.title")}
        </h4>

        {/* Signal metrics */}
        {metricsByCategory.signal.length > 0 && (
          <div
            className={cn(
              layout.inline.default,
              "flex-wrap",
              spacing.margin.bottom.tight
            )}
          >
            {metricsByCategory.signal.map((option) => {
              const Icon = option.icon;
              const isSelected = metric === option.id;
              return (
                <button
                  key={option.id}
                  onClick={() => handleMetricSelect(option.id)}
                  className={cn(
                    button.size.xs,
                    radius.md,
                    layout.inline.default,
                    "transition-colors",
                    isSelected
                      ? "bg-brand-primary text-text-inverse"
                      : "bg-surface-base border border-surface-border hover:bg-surface-hover"
                  )}
                >
                  <Icon className={iconTokens.size.xs} />
                  <span>{t(option.labelKey as never)}</span>
                </button>
              );
            })}
          </div>
        )}

        {/* Performance metrics */}
        {metricsByCategory.performance.length > 0 && (
          <div
            className={cn(
              layout.inline.default,
              "flex-wrap",
              spacing.margin.bottom.tight
            )}
          >
            {metricsByCategory.performance.map((option) => {
              const Icon = option.icon;
              const isSelected = metric === option.id;
              return (
                <button
                  key={option.id}
                  onClick={() => handleMetricSelect(option.id)}
                  className={cn(
                    button.size.xs,
                    radius.md,
                    layout.inline.default,
                    "transition-colors",
                    isSelected
                      ? "bg-green-600 text-text-inverse"
                      : "bg-surface-base border border-surface-border hover:bg-surface-hover"
                  )}
                >
                  <Icon className={iconTokens.size.xs} />
                  <span>{t(option.labelKey as never)}</span>
                </button>
              );
            })}
          </div>
        )}

        {/* Interference metrics */}
        {metricsByCategory.interference.length > 0 && (
          <div className={cn(layout.inline.default, "flex-wrap")}>
            {metricsByCategory.interference.map((option) => {
              const Icon = option.icon;
              const isSelected = metric === option.id;
              return (
                <button
                  key={option.id}
                  onClick={() => handleMetricSelect(option.id)}
                  className={cn(
                    button.size.xs,
                    radius.md,
                    layout.inline.default,
                    "transition-colors",
                    isSelected
                      ? "bg-purple-600 text-text-inverse"
                      : "bg-surface-base border border-surface-border hover:bg-surface-hover"
                  )}
                >
                  <Icon className={iconTokens.size.xs} />
                  <span>{t(option.labelKey as never)}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {/* Filter Toggle */}
      <button
        onClick={() => setShowFilters(!showFilters)}
        className={cn(
          layout.inline.default,
          "w-full justify-between",
          spacing.pad.sm,
          radius.md,
          "bg-surface-base border border-surface-border hover:bg-surface-hover"
        )}
      >
        <div className={cn(layout.inline.default)}>
          <Filter className={iconTokens.size.sm} />
          <span className="body-small">{t("heatmaps.filters")}</span>
          {hasActiveFilters && (
            <span className="px-1.5 py-0.5 text-xs bg-brand-primary text-text-inverse rounded-full">
              {Object.keys(filter!).length}
            </span>
          )}
        </div>
        {showFilters ? (
          <ChevronUp className={iconTokens.size.sm} />
        ) : (
          <ChevronDown className={iconTokens.size.sm} />
        )}
      </button>

      {/* Filter Options */}
      {showFilters && (
        <div className={cn(layout.stack.default, spacing.margin.top.content)}>
          {/* SSID Filter */}
          <div>
            <label className="caption text-text-muted">
              {t("heatmaps.filterSsid")}
            </label>
            <select
              value={filter?.ssid || ""}
              onChange={(e) => updateFilter("ssid", e.target.value)}
              className={cn(
                "w-full",
                spacing.pad.sm,
                radius.md,
                "border border-surface-border bg-surface-base body-small"
              )}
            >
              <option value="">{t("heatmaps.allSsids")}</option>
              {uniqueSsids.map((ssid) => (
                <option key={ssid} value={ssid}>
                  {ssid || t("heatmaps.hiddenSsid")}
                </option>
              ))}
            </select>
          </div>

          {/* BSSID Filter */}
          <div>
            <label className="caption text-text-muted">
              {t("heatmaps.filterBssid")}
            </label>
            <select
              value={filter?.bssid || ""}
              onChange={(e) => updateFilter("bssid", e.target.value)}
              className={cn(
                "w-full",
                spacing.pad.sm,
                radius.md,
                "border border-surface-border bg-surface-base body-small"
              )}
            >
              <option value="">{t("heatmaps.allAps")}</option>
              {uniqueBssids.map((bssid) => (
                <option key={bssid} value={bssid}>
                  {bssid}
                </option>
              ))}
            </select>
          </div>

          {/* AP Location Filter */}
          {apLocations.length > 0 && (
            <div>
              <label className="caption text-text-muted">
                {t("heatmaps.filterAp")}
              </label>
              <select
                value={filter?.apId || ""}
                onChange={(e) => updateFilter("apId", e.target.value)}
                className={cn(
                  "w-full",
                  spacing.pad.sm,
                  radius.md,
                  "border border-surface-border bg-surface-base body-small"
                )}
              >
                <option value="">{t("heatmaps.allApLocations")}</option>
                {apLocations.map((ap) => (
                  <option key={ap.id} value={ap.id}>
                    {ap.label} {ap.bssid ? `(${ap.bssid})` : ""}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Band Filter */}
          <div>
            <label className="caption text-text-muted">
              {t("heatmaps.filterBand")}
            </label>
            <div className={cn(layout.inline.default)}>
              {(["2.4", "5", "6"] as WiFiBand[]).map((band) => (
                <button
                  key={band}
                  onClick={() =>
                    updateFilter(
                      "band",
                      filter?.band === band ? undefined : band
                    )
                  }
                  className={cn(
                    button.size.xs,
                    radius.md,
                    "transition-colors",
                    filter?.band === band
                      ? band === "2.4"
                        ? "bg-blue-500 text-text-inverse"
                        : band === "5"
                          ? "bg-green-500 text-text-inverse"
                          : "bg-purple-500 text-text-inverse"
                      : "bg-surface-base border border-surface-border hover:bg-surface-hover"
                  )}
                >
                  {band} GHz
                </button>
              ))}
            </div>
          </div>

          {/* Channel Filter */}
          {uniqueChannels.length > 0 && (
            <div>
              <label className="caption text-text-muted">
                {t("heatmaps.filterChannel")}
              </label>
              <select
                value={filter?.channel || ""}
                onChange={(e) =>
                  updateFilter(
                    "channel",
                    e.target.value ? parseInt(e.target.value, 10) : undefined
                  )
                }
                className={cn(
                  "w-full",
                  spacing.pad.sm,
                  radius.md,
                  "border border-surface-border bg-surface-base body-small"
                )}
              >
                <option value="">{t("heatmaps.allChannels")}</option>
                {uniqueChannels.map((ch) => (
                  <option key={ch} value={ch}>
                    {t("heatmaps.channel")} {ch}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Channel Width Filter */}
          {uniqueChannelWidths.length > 0 && (
            <div>
              <label className="caption text-text-muted">
                {t("heatmaps.filterChannelWidth")}
              </label>
              <div className={cn(layout.inline.default, "flex-wrap")}>
                {([20, 40, 80, 160, 320] as ChannelWidth[]).map((width) => (
                  <button
                    key={width}
                    onClick={() =>
                      updateFilter(
                        "channelWidth",
                        filter?.channelWidth === width ? undefined : width
                      )
                    }
                    disabled={!uniqueChannelWidths.includes(width)}
                    className={cn(
                      button.size.xs,
                      radius.md,
                      "transition-colors",
                      filter?.channelWidth === width
                        ? "bg-cyan-500 text-text-inverse"
                        : uniqueChannelWidths.includes(width)
                          ? "bg-surface-base border border-surface-border hover:bg-surface-hover"
                          : "bg-surface-base border border-surface-border opacity-40"
                    )}
                  >
                    {width} MHz
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* PHY Type / 802.11 Standard Filter */}
          {uniquePhyTypes.length > 0 && (
            <div>
              <label className="caption text-text-muted">
                {t("heatmaps.filterPhyType")}
              </label>
              <div className={cn(layout.inline.default, "flex-wrap")}>
                {(["a", "b", "g", "n", "ac", "ax", "be"] as PhyType[]).map(
                  (phy) => (
                    <button
                      key={phy}
                      onClick={() =>
                        updateFilter(
                          "phyType",
                          filter?.phyType === phy ? undefined : phy
                        )
                      }
                      disabled={!uniquePhyTypes.includes(phy)}
                      className={cn(
                        button.size.xs,
                        radius.md,
                        "transition-colors",
                        filter?.phyType === phy
                          ? "bg-indigo-500 text-text-inverse"
                          : uniquePhyTypes.includes(phy)
                            ? "bg-surface-base border border-surface-border hover:bg-surface-hover"
                            : "bg-surface-base border border-surface-border opacity-40"
                      )}
                    >
                      802.11{phy}
                    </button>
                  )
                )}
              </div>
            </div>
          )}

          {/* Security Type Filter */}
          {uniqueSecurityTypes.length > 0 && (
            <div>
              <label className="caption text-text-muted">
                {t("heatmaps.filterSecurity")}
              </label>
              <select
                value={filter?.security || ""}
                onChange={(e) =>
                  updateFilter("security", e.target.value || undefined)
                }
                className={cn(
                  "w-full",
                  spacing.pad.sm,
                  radius.md,
                  "border border-surface-border bg-surface-base body-small"
                )}
              >
                <option value="">{t("heatmaps.allSecurity")}</option>
                {uniqueSecurityTypes.map((sec) => (
                  <option key={sec} value={sec}>
                    {t(`heatmaps.security.${sec}` as never)}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Vendor Filter */}
          {uniqueVendors.length > 0 && (
            <div>
              <label className="caption text-text-muted">
                {t("heatmaps.filterVendor")}
              </label>
              <select
                value={filter?.vendor || ""}
                onChange={(e) =>
                  updateFilter("vendor", e.target.value || undefined)
                }
                className={cn(
                  "w-full",
                  spacing.pad.sm,
                  radius.md,
                  "border border-surface-border bg-surface-base body-small"
                )}
              >
                <option value="">{t("heatmaps.allVendors")}</option>
                {uniqueVendors.map((vendor) => (
                  <option key={vendor} value={vendor}>
                    {vendor}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Survey View Mode (for imported surveys with multiple modes) */}
          <div>
            <label className="caption text-text-muted">
              {t("heatmaps.viewMode")}
            </label>
            <div className={cn(layout.inline.default, "flex-wrap")}>
              {(
                [
                  "all",
                  "passive",
                  "active",
                  "client",
                  "probingClient",
                ] as SurveyViewMode[]
              ).map((mode) => (
                <button
                  key={mode}
                  onClick={() =>
                    updateFilter(
                      "viewMode",
                      filter?.viewMode === mode ? undefined : mode
                    )
                  }
                  className={cn(
                    button.size.xs,
                    radius.md,
                    "transition-colors",
                    filter?.viewMode === mode
                      ? "bg-amber-500 text-text-inverse"
                      : "bg-surface-base border border-surface-border hover:bg-surface-hover"
                  )}
                >
                  {t(`heatmaps.viewModes.${mode}` as never)}
                </button>
              ))}
            </div>
          </div>

          {/* Min RSSI Filter */}
          <div>
            <label className="caption text-text-muted">
              {t("heatmaps.minRssi")}
            </label>
            <div className={cn(layout.inline.default)}>
              <input
                type="range"
                min="-100"
                max="-30"
                value={filter?.minRssi || -100}
                onChange={(e) =>
                  updateFilter(
                    "minRssi",
                    parseInt(e.target.value, 10) === -100
                      ? undefined
                      : parseInt(e.target.value, 10)
                  )
                }
                className="flex-1"
              />
              <span className="body-small w-16 text-right">
                {filter?.minRssi || -100} dBm
              </span>
            </div>
          </div>

          {/* Clear Filters */}
          {hasActiveFilters && (
            <button
              onClick={clearFilters}
              className={cn(
                button.size.sm,
                radius.md,
                "border border-surface-border hover:bg-surface-hover",
                layout.inline.default,
                "justify-center"
              )}
            >
              <X className={iconTokens.size.sm} />
              <span>{t("heatmaps.clearFilters")}</span>
            </button>
          )}
        </div>
      )}
    </div>
  );
}
