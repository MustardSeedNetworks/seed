/**
 * SurveyConfigPanel Component
 *
 * Purpose: Provides advanced survey configuration for band selection, channel
 * filtering, and multi-adapter setup. Enables comprehensive WiFi survey planning.
 *
 * Key Features:
 * - Band selection: Choose which WiFi bands to scan (2.4/5/6 GHz)
 * - Channel selection: Filter specific channels per band
 * - Multi-adapter config: Assign different adapters to different modes
 * - Guided recommendations: Suggests optimal config based on goals
 * - Survey type settings: Configure passive/active/throughput parameters
 *
 * Usage:
 * ```typescript
 * <SurveyConfigPanel
 *   config={survey.config}
 *   surveyType={survey.surveyType}
 *   availableAdapters={wifiStatus.availableAdapters}
 *   onUpdate={(config) => handleConfigUpdate(config)}
 * />
 * ```
 */

import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { SurveyConfig, SurveyType, WiFiBand, AdapterConfig } from "../../hooks/useSurvey";
import { Radio, Settings, Wifi, Lightbulb, ChevronDown, ChevronUp } from "lucide-react";
import {
  radius,
  spacing,
  layout,
  button,
  icon as iconTokens,
  input as inputTokens,
} from "../../styles/theme";

/** Default channels for display */
const CHANNELS: Record<WiFiBand, number[]> = {
  "2.4": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11],
  "5": [
    36, 40, 44, 48, 52, 56, 60, 64, 100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144, 149,
    153, 157, 161, 165,
  ],
  "6": [
    1, 5, 9, 13, 17, 21, 25, 29, 33, 37, 41, 45, 49, 53, 57, 61, 65, 69, 73, 77, 81, 85, 89, 93,
  ],
};

/** Band display info */
const BAND_INFO: Record<WiFiBand, { label: string; color: string }> = {
  "2.4": { label: "2.4 GHz", color: "bg-blue-500" },
  "5": { label: "5 GHz", color: "bg-green-500" },
  "6": { label: "6 GHz", color: "bg-purple-500" },
};

function getBandInfo(band: WiFiBand): { label: string; color: string } {
  switch (band) {
    case "2.4":
      return BAND_INFO["2.4"];
    case "5":
      return BAND_INFO["5"];
    case "6":
      return BAND_INFO["6"];
  }
}

function getChannelsForBand(band: WiFiBand): number[] {
  switch (band) {
    case "2.4":
      return CHANNELS["2.4"];
    case "5":
      return CHANNELS["5"];
    case "6":
      return CHANNELS["6"];
  }
}

function getCustomChannelsForBand(
  customChannels: Record<WiFiBand, number[]>,
  band: WiFiBand
): number[] {
  switch (band) {
    case "2.4":
      return customChannels["2.4"];
    case "5":
      return customChannels["5"];
    case "6":
      return customChannels["6"];
  }
}

function updateCustomChannelsForBand(
  customChannels: Record<WiFiBand, number[]>,
  band: WiFiBand,
  channels: number[]
): Record<WiFiBand, number[]> {
  switch (band) {
    case "2.4":
      return { ...customChannels, "2.4": channels };
    case "5":
      return { ...customChannels, "5": channels };
    case "6":
      return { ...customChannels, "6": channels };
  }
}

/** Survey goal presets */
interface SurveyGoal {
  id: string;
  labelKey: string;
  descriptionKey: string;
  config: Partial<SurveyConfig>;
  recommendedType: SurveyType;
}

const SURVEY_GOALS: SurveyGoal[] = [
  {
    id: "coverage",
    labelKey: "config.goals.coverage",
    descriptionKey: "config.goals.coverageDesc",
    config: { bands: ["2.4", "5"] },
    recommendedType: "passive",
  },
  {
    id: "performance",
    labelKey: "config.goals.performance",
    descriptionKey: "config.goals.performanceDesc",
    config: { bands: ["5"] },
    recommendedType: "throughput",
  },
  {
    id: "comprehensive",
    labelKey: "config.goals.comprehensive",
    descriptionKey: "config.goals.comprehensiveDesc",
    config: { bands: ["2.4", "5", "6"] },
    recommendedType: "passive",
  },
  {
    id: "troubleshoot",
    labelKey: "config.goals.troubleshoot",
    descriptionKey: "config.goals.troubleshootDesc",
    config: { bands: ["2.4", "5"], minRssi: -80 },
    recommendedType: "active",
  },
];

interface SurveyConfigPanelProps {
  config?: SurveyConfig;
  surveyType: SurveyType;
  availableAdapters: string[];
  currentInterface: string;
  iperfServer?: string;
  testDuration?: number;
  onUpdate: (config: Partial<SurveyConfig>) => void;
  onSurveyTypeChange?: (type: SurveyType) => void;
  onIperfSettingsChange?: (server: string, duration: number) => void;
}

/**
 * SurveyConfigPanel provides advanced survey configuration
 * including band/channel selection and multi-adapter setup.
 */
export function SurveyConfigPanel({
  config,
  surveyType,
  availableAdapters,
  currentInterface,
  iperfServer = "",
  testDuration = 3,
  onUpdate,
  onSurveyTypeChange,
  onIperfSettingsChange,
}: SurveyConfigPanelProps) {
  const { t } = useTranslation("survey");

  // Local state for config
  const [selectedBands, setSelectedBands] = useState<WiFiBand[]>(config?.bands || ["2.4", "5"]);
  const [channelMode, setChannelMode] = useState<"all" | "custom">("all");
  const [customChannels, setCustomChannels] = useState<Record<WiFiBand, number[]>>({
    "2.4": [],
    "5": [],
    "6": [],
  });
  const [selectedGoal, setSelectedGoal] = useState<string | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [localIperfServer, setLocalIperfServer] = useState(iperfServer);
  const [localTestDuration, setLocalTestDuration] = useState(testDuration);

  // Multi-adapter state
  const [adapterConfigs, setAdapterConfigs] = useState<AdapterConfig[]>(
    config?.adapters || [{ interface: currentInterface, mode: surveyType, bands: selectedBands }]
  );

  const hasMultipleAdapters = availableAdapters.length > 1;

  // Handle band toggle
  const handleBandToggle = (band: WiFiBand) => {
    const newBands = selectedBands.includes(band)
      ? selectedBands.filter((b) => b !== band)
      : [...selectedBands, band];

    // Ensure at least one band is selected
    if (newBands.length === 0) return;

    setSelectedBands(newBands);
    setSelectedGoal(null); // Clear goal when manually changing
    onUpdate({ bands: newBands });
  };

  // Handle channel selection for a band
  const handleChannelToggle = (band: WiFiBand, channel: number) => {
    const currentChannels = getCustomChannelsForBand(customChannels, band);
    const newChannels = currentChannels.includes(channel)
      ? currentChannels.filter((c) => c !== channel)
      : [...currentChannels, channel].sort((a, b) => a - b);

    const updatedCustomChannels = updateCustomChannelsForBand(customChannels, band, newChannels);
    setCustomChannels(updatedCustomChannels);

    // Update config based on band
    switch (band) {
      case "2.4":
        onUpdate({ channels2_4: newChannels.length > 0 ? newChannels : undefined });
        break;
      case "5":
        onUpdate({ channels5: newChannels.length > 0 ? newChannels : undefined });
        break;
      case "6":
        onUpdate({ channels6: newChannels.length > 0 ? newChannels : undefined });
        break;
    }
  };

  // Handle goal selection
  const handleGoalSelect = (goal: SurveyGoal) => {
    setSelectedGoal(goal.id);
    setSelectedBands(goal.config.bands as WiFiBand[]);
    onUpdate(goal.config);
    if (onSurveyTypeChange && goal.recommendedType !== surveyType) {
      onSurveyTypeChange(goal.recommendedType);
    }
  };

  // Handle adapter mode change
  const handleAdapterModeChange = (adapterIndex: number, mode: SurveyType) => {
    const newConfigs = adapterConfigs.map((config, index) =>
      index === adapterIndex ? { ...config, mode } : config
    );
    setAdapterConfigs(newConfigs);
    onUpdate({ adapters: newConfigs });
  };

  // Handle adapter band assignment
  const handleAdapterBandChange = (adapterIndex: number, bands: WiFiBand[]) => {
    const newConfigs = adapterConfigs.map((config, index) =>
      index === adapterIndex ? { ...config, bands } : config
    );
    setAdapterConfigs(newConfigs);
    onUpdate({ adapters: newConfigs });
  };

  // Add second adapter config
  const handleAddAdapter = () => {
    if (availableAdapters.length > adapterConfigs.length) {
      const unusedAdapter = availableAdapters.find(
        (a) => !adapterConfigs.some((c) => c.interface === a)
      );
      if (unusedAdapter) {
        const newConfig: AdapterConfig = {
          interface: unusedAdapter,
          mode: surveyType === "passive" ? "active" : "passive",
          bands: selectedBands,
        };
        const newConfigs = [...adapterConfigs, newConfig];
        setAdapterConfigs(newConfigs);
        onUpdate({ adapters: newConfigs });
      }
    }
  };

  // Remove adapter config
  const handleRemoveAdapter = (index: number) => {
    if (adapterConfigs.length > 1) {
      const newConfigs = adapterConfigs.filter((_, i) => i !== index);
      setAdapterConfigs(newConfigs);
      onUpdate({ adapters: newConfigs });
    }
  };

  // Handle iperf settings
  const handleIperfSave = () => {
    if (onIperfSettingsChange) {
      onIperfSettingsChange(localIperfServer, localTestDuration);
    }
  };

  return (
    <div
      className={`bg-surface-raised ${radius.md} border border-surface-border ${spacing.pad.default}`}
    >
      <h3 className={`heading-3 ${spacing.margin.bottom.content}`}>{t("config.title")}</h3>

      {/* Survey Goal Selection */}
      <div className={`${spacing.margin.bottom.content}`}>
        <div className={`${layout.inline.default} ${spacing.margin.bottom.inline}`}>
          <Lightbulb className={iconTokens.size.sm} />
          <span className="body-small font-medium">{t("config.whatGoal")}</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {SURVEY_GOALS.map((goal) => (
            <button
              key={goal.id}
              onClick={() => handleGoalSelect(goal)}
              className={`${button.size.sm} ${radius.md} border ${
                selectedGoal === goal.id
                  ? "bg-brand-primary text-text-inverse border-brand-primary"
                  : "border-surface-border hover:bg-surface-hover"
              }`}
            >
              {t(goal.labelKey as never)}
            </button>
          ))}
        </div>
        {selectedGoal && (
          <p className={`caption text-text-muted ${spacing.margin.top.tight}`}>
            {t((SURVEY_GOALS.find((g) => g.id === selectedGoal)?.descriptionKey || "") as never)}
          </p>
        )}
      </div>

      {/* Band Selection */}
      <div
        className={`border border-surface-border ${radius.md} ${spacing.pad.sm} ${spacing.margin.bottom.content}`}
      >
        <div className={`${layout.inline.default} ${spacing.margin.bottom.inline}`}>
          <Radio className={iconTokens.size.sm} />
          <span className="body-small font-medium">{t("config.bandsToScan")}</span>
        </div>
        <div className="flex flex-wrap gap-3">
          {(["2.4", "5", "6"] as WiFiBand[]).map((band) => {
            const bandInfo = getBandInfo(band);
            return (
              <label key={band} className={`${layout.inline.default} cursor-pointer`}>
                <input
                  type="checkbox"
                  checked={selectedBands.includes(band)}
                  onChange={() => handleBandToggle(band)}
                  className="w-4 h-4 accent-brand-primary"
                />
                <span className={`${layout.inline.default}`}>
                  <span className={`w-2 h-2 ${bandInfo.color} ${radius.full}`} />
                  <span className="body-small">{bandInfo.label}</span>
                </span>
              </label>
            );
          })}
        </div>
      </div>

      {/* Channel Selection (Collapsible) */}
      <div
        className={`border border-surface-border ${radius.md} ${spacing.pad.sm} ${spacing.margin.bottom.content}`}
      >
        <button
          onClick={() => setShowAdvanced(!showAdvanced)}
          className={`w-full ${layout.flex.between} body-small font-medium`}
        >
          <div className={layout.inline.default}>
            <Settings className={iconTokens.size.sm} />
            <span>{t("config.channelSelection")}</span>
          </div>
          {showAdvanced ? (
            <ChevronUp className={iconTokens.size.sm} />
          ) : (
            <ChevronDown className={iconTokens.size.sm} />
          )}
        </button>

        {showAdvanced && (
          <div className={`${spacing.margin.top.content}`}>
            {/* Channel mode toggle */}
            <div className={`${layout.inline.default} ${spacing.margin.bottom.content}`}>
              <label className={`${layout.inline.default} cursor-pointer`}>
                <input
                  type="radio"
                  name="channelMode"
                  checked={channelMode === "all"}
                  onChange={() => setChannelMode("all")}
                  className="w-4 h-4 accent-brand-primary"
                />
                <span className="body-small">{t("config.allChannels")}</span>
              </label>
              <label className={`${layout.inline.default} cursor-pointer`}>
                <input
                  type="radio"
                  name="channelMode"
                  checked={channelMode === "custom"}
                  onChange={() => setChannelMode("custom")}
                  className="w-4 h-4 accent-brand-primary"
                />
                <span className="body-small">{t("config.customChannels")}</span>
              </label>
            </div>

            {/* Per-band channel selection */}
            {channelMode === "custom" &&
              selectedBands.map((band) => {
                const selectedChannels = getCustomChannelsForBand(customChannels, band);
                return (
                  <div key={band} className={`${spacing.margin.bottom.content}`}>
                    <label
                      className={`caption text-text-muted block ${spacing.margin.bottom.tight}`}
                    >
                      {getBandInfo(band).label} {t("config.channels")}
                    </label>
                    <div className="flex flex-wrap gap-1">
                      {getChannelsForBand(band).map((channel) => (
                        <button
                          key={channel}
                          onClick={() => handleChannelToggle(band, channel)}
                          className={`${button.size.xs} ${radius.sm} min-w-[2.5rem] ${
                            selectedChannels.includes(channel)
                              ? "bg-brand-primary text-text-inverse"
                              : "bg-surface-base border border-surface-border hover:bg-surface-hover"
                          }`}
                        >
                          {channel}
                        </button>
                      ))}
                    </div>
                  </div>
                );
              })}
          </div>
        )}
      </div>

      {/* Multi-Adapter Configuration */}
      {hasMultipleAdapters && (
        <div
          className={`border border-surface-border ${radius.md} ${spacing.pad.sm} ${spacing.margin.bottom.content}`}
        >
          <div className={`${layout.inline.default} ${spacing.margin.bottom.inline}`}>
            <Wifi className={iconTokens.size.sm} />
            <span className="body-small font-medium">{t("config.adapterConfig")}</span>
            <span className={`caption text-text-muted`}>
              ({availableAdapters.length} {t("config.detected")})
            </span>
          </div>

          {/* Adapter list */}
          {adapterConfigs.map((adapter, index) => (
            <div
              key={adapter.interface}
              className={`bg-surface-base ${radius.md} ${spacing.pad.sm} ${
                index > 0 ? spacing.margin.top.content : ""
              }`}
            >
              <div className={`${layout.flex.between} ${spacing.margin.bottom.inline}`}>
                <span className="body-small font-medium">{adapter.interface}</span>
                {index > 0 && (
                  <button
                    onClick={() => handleRemoveAdapter(index)}
                    className="caption text-status-error hover:underline"
                  >
                    {t("config.remove")}
                  </button>
                )}
              </div>
              <div className={`${layout.inline.default}`}>
                <div>
                  <label className="caption text-text-muted">{t("config.mode")}</label>
                  <select
                    value={adapter.mode}
                    onChange={(e) => handleAdapterModeChange(index, e.target.value as SurveyType)}
                    className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm}`}
                  >
                    <option value="passive">{t("settings.types.passive")}</option>
                    <option value="active">{t("settings.types.active")}</option>
                    <option value="throughput">{t("settings.types.throughput")}</option>
                  </select>
                </div>
                <div>
                  <label className="caption text-text-muted">{t("config.bands")}</label>
                  <div className="flex gap-2">
                    {(["2.4", "5", "6"] as WiFiBand[]).map((band) => (
                      <label key={band} className={`${layout.inline.default} cursor-pointer`}>
                        <input
                          type="checkbox"
                          checked={adapter.bands.includes(band)}
                          onChange={() => {
                            const newBands = adapter.bands.includes(band)
                              ? adapter.bands.filter((b) => b !== band)
                              : [...adapter.bands, band];
                            if (newBands.length > 0) {
                              handleAdapterBandChange(index, newBands);
                            }
                          }}
                          className="w-3 h-3 accent-brand-primary"
                        />
                        <span className="caption">{band}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}

          {/* Add adapter button */}
          {adapterConfigs.length < availableAdapters.length && (
            <button
              onClick={handleAddAdapter}
              className={`${button.size.sm} border border-dashed border-surface-border ${radius.md} hover:bg-surface-hover w-full ${spacing.margin.top.content}`}
            >
              + {t("config.addAdapter")}
            </button>
          )}

          {/* Multi-adapter recommendation */}
          {adapterConfigs.length > 1 && (
            <div
              className={`bg-status-info/10 border border-status-info/20 ${radius.md} ${spacing.pad.sm} ${spacing.margin.top.content}`}
            >
              <div className="body-small text-status-info">{t("config.multiAdapterTip")}</div>
            </div>
          )}
        </div>
      )}

      {/* Throughput Settings (if applicable) */}
      {(surveyType === "throughput" || adapterConfigs.some((a) => a.mode === "throughput")) && (
        <div className={`border border-surface-border ${radius.md} ${spacing.pad.sm}`}>
          <div className={`${layout.inline.default} ${spacing.margin.bottom.inline}`}>
            <Settings className={iconTokens.size.sm} />
            <span className="body-small font-medium">{t("config.throughputSettings")}</span>
          </div>
          <div className={`${layout.stack.default}`}>
            <div>
              <label className={`caption text-text-muted block ${spacing.margin.bottom.tight}`}>
                {t("settings.iperfServer")}
              </label>
              <input
                type="text"
                value={localIperfServer}
                onChange={(e) => setLocalIperfServer(e.target.value)}
                onBlur={handleIperfSave}
                placeholder="192.168.1.100:5201"
                className={`w-full ${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm}`}
              />
              <p className={`caption text-text-muted ${spacing.margin.top.tight}`}>
                {t("settings.iperfServerHint")}
              </p>
            </div>
            <div>
              <label className={`caption text-text-muted block ${spacing.margin.bottom.tight}`}>
                {t("settings.testDuration")}
              </label>
              <input
                type="number"
                min={1}
                max={30}
                value={localTestDuration}
                onChange={(e) => setLocalTestDuration(parseInt(e.target.value) || 3)}
                onBlur={handleIperfSave}
                className={`w-24 ${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm}`}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
