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

import { ChevronDown, ChevronUp, Info, Radio, Settings, Wifi } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { AdapterConfig, SurveyConfig, SurveyType, WiFiBand } from "../../hooks/useSurvey";
import {
  button,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
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
  band: WiFiBand,
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
  channels: number[],
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

/** Survey goal presets with multi-radio support */
interface SurveyGoal {
  id: string;
  titleKey: string;
  descriptionKey: string;
  surveyTypes: SurveyType[];
  radiosRequired: 1 | 2;
  config: Partial<SurveyConfig>;
  recommended?: boolean;
}

const SURVEY_GOALS: SurveyGoal[] = [
  {
    id: "coverage",
    titleKey: "config.goals.coverage",
    descriptionKey: "config.goals.coverageDesc",
    surveyTypes: ["passive"],
    radiosRequired: 1,
    config: { bands: ["2.4", "5"] },
  },
  {
    id: "connection",
    titleKey: "config.goals.connection",
    descriptionKey: "config.goals.connectionDesc",
    surveyTypes: ["active"],
    radiosRequired: 1,
    config: { bands: ["2.4", "5"] },
  },
  {
    id: "throughput",
    titleKey: "config.goals.throughput",
    descriptionKey: "config.goals.throughputDesc",
    surveyTypes: ["throughput"],
    radiosRequired: 1,
    config: { bands: ["5"] },
  },
  {
    id: "complete",
    titleKey: "config.goals.complete",
    descriptionKey: "config.goals.completeDesc",
    surveyTypes: ["passive", "active"],
    radiosRequired: 2,
    config: { bands: ["2.4", "5", "6"] },
    recommended: true,
  },
  {
    id: "validation",
    titleKey: "config.goals.validation",
    descriptionKey: "config.goals.validationDesc",
    surveyTypes: ["passive", "throughput"],
    radiosRequired: 2,
    config: { bands: ["2.4", "5", "6"] },
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
    config?.adapters || [{ interface: currentInterface, mode: surveyType, bands: selectedBands }],
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
        onUpdate({
          channels24Ghz: newChannels.length > 0 ? newChannels : undefined,
        });
        break;
      case "5":
        onUpdate({
          channels5Ghz: newChannels.length > 0 ? newChannels : undefined,
        });
        break;
      case "6":
        onUpdate({
          channels6Ghz: newChannels.length > 0 ? newChannels : undefined,
        });
        break;
    }
  };

  // Handle goal selection
  const handleGoalSelect = (goal: SurveyGoal) => {
    setSelectedGoal(goal.id);
    setSelectedBands(goal.config.bands as WiFiBand[]);
    onUpdate(goal.config);

    // Set primary survey type to first in the array
    const primaryType = goal.surveyTypes[0];
    if (onSurveyTypeChange && primaryType !== surveyType) {
      onSurveyTypeChange(primaryType);
    }

    // If 2 radios required and we have multiple adapters, set up dual config
    if (goal.radiosRequired === 2 && availableAdapters.length >= 2) {
      const newConfigs: AdapterConfig[] = [
        {
          interface: availableAdapters[0],
          mode: goal.surveyTypes[0],
          bands: goal.config.bands as WiFiBand[],
        },
        {
          interface: availableAdapters[1],
          mode: goal.surveyTypes[1],
          bands: goal.config.bands as WiFiBand[],
        },
      ];
      setAdapterConfigs(newConfigs);
      onUpdate({ ...goal.config, adapters: newConfigs });
    }
  };

  // Helper to format survey type for display
  const formatSurveyTypes = (types: SurveyType[]): string => {
    const typeLabels: string[] = [];
    for (const type of types) {
      switch (type) {
        case "passive":
          typeLabels.push(t("settings.types.passive"));
          break;
        case "active":
          typeLabels.push(t("settings.types.active"));
          break;
        case "throughput":
          typeLabels.push(t("settings.types.throughput"));
          break;
      }
    }
    return typeLabels.join(" + ");
  };

  // Handle adapter mode change
  const handleAdapterModeChange = (adapterIndex: number, mode: SurveyType) => {
    const newConfigs = adapterConfigs.map((config, index) =>
      index === adapterIndex ? { ...config, mode } : config,
    );
    setAdapterConfigs(newConfigs);
    onUpdate({ adapters: newConfigs });
  };

  // Handle adapter band assignment
  const handleAdapterBandChange = (adapterIndex: number, bands: WiFiBand[]) => {
    const newConfigs = adapterConfigs.map((config, index) =>
      index === adapterIndex ? { ...config, bands } : config,
    );
    setAdapterConfigs(newConfigs);
    onUpdate({ adapters: newConfigs });
  };

  // Add second adapter config
  const handleAddAdapter = () => {
    if (availableAdapters.length > adapterConfigs.length) {
      const unusedAdapter = availableAdapters.find(
        (a) => !adapterConfigs.some((c) => c.interface === a),
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
      className={cn(
        "bg-surface-raised",
        radius.md,
        "border border-surface-border",
        spacing.pad.default,
      )}
    >
      <h3 className={cn("heading-3", spacing.margin.bottom.content)}>{t("config.title")}</h3>

      {/* Survey Goal Selection - Goal-First Approach */}
      <div className={cn(spacing.margin.bottom.content)}>
        <h4 className={cn("body-small font-medium", spacing.margin.bottom.content)}>
          {t("config.whatGoal")}
        </h4>
        <div className={cn(layout.stack.default)}>
          {SURVEY_GOALS.map((goal) => {
            const isSelected = selectedGoal === goal.id;
            return (
              <label
                key={goal.id}
                className={cn(
                  "flex items-start gap-3",
                  spacing.pad.sm,
                  radius.md,
                  "border cursor-pointer transition-colors",
                  isSelected
                    ? "bg-brand-primary/5 border-brand-primary"
                    : "border-surface-border hover:bg-surface-hover",
                )}
              >
                <input
                  type="radio"
                  name="surveyGoal"
                  checked={isSelected}
                  onChange={() => handleGoalSelect(goal)}
                  className="w-4 h-4 mt-0.5 accent-brand-primary"
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="body-small font-medium">{t(goal.titleKey as never)}</span>
                    {goal.recommended && (
                      <span className="caption px-1.5 py-0.5 bg-status-success/10 text-status-success rounded">
                        {t("config.recommended")}
                      </span>
                    )}
                  </div>
                  <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
                    {t(goal.descriptionKey as never)}
                  </p>
                  <div
                    className={cn(
                      "caption text-text-muted",
                      spacing.margin.top.tight,
                      "flex items-center gap-3",
                    )}
                  >
                    <span>{formatSurveyTypes(goal.surveyTypes)}</span>
                    <span className="text-surface-border">|</span>
                    <span>
                      {goal.radiosRequired === 1 ? t("config.oneRadio") : t("config.twoRadios")}
                    </span>
                  </div>
                </div>
              </label>
            );
          })}
        </div>

        {/* Multi-Radio Notice */}
        {selectedGoal && SURVEY_GOALS.find((g) => g.id === selectedGoal)?.radiosRequired === 2 && (
          <div
            className={cn(
              layout.inline.default,
              "bg-status-info/10 border border-status-info/20",
              radius.md,
              spacing.pad.sm,
              spacing.margin.top.content,
            )}
          >
            <Info className={cn(iconTokens.size.sm, "text-status-info flex-shrink-0")} />
            <div>
              <div className="body-small text-status-info font-medium">
                {t("config.twoRadiosRequired")}
              </div>
              <p className="caption text-text-muted">{t("config.twoRadiosDesc")}</p>
            </div>
          </div>
        )}
      </div>

      {/* Band Selection */}
      <div
        className={cn(
          "border border-surface-border",
          radius.md,
          spacing.pad.sm,
          spacing.margin.bottom.content,
        )}
      >
        <div className={cn(layout.inline.default, spacing.margin.bottom.inline)}>
          <Radio className={iconTokens.size.sm} />
          <span className="body-small font-medium">{t("config.bandsToScan")}</span>
        </div>
        <div className="flex flex-wrap gap-3">
          {(["2.4", "5", "6"] as WiFiBand[]).map((band) => {
            const bandInfo = getBandInfo(band);
            return (
              <label key={band} className={cn(layout.inline.default, "cursor-pointer")}>
                <input
                  type="checkbox"
                  checked={selectedBands.includes(band)}
                  onChange={() => handleBandToggle(band)}
                  className="w-4 h-4 accent-brand-primary"
                />
                <span className={cn(layout.inline.default)}>
                  <span className={cn("w-2 h-2", bandInfo.color, radius.full)} />
                  <span className="body-small">{bandInfo.label}</span>
                </span>
              </label>
            );
          })}
        </div>
      </div>

      {/* Channel Selection (Collapsible) */}
      <div
        className={cn(
          "border border-surface-border",
          radius.md,
          spacing.pad.sm,
          spacing.margin.bottom.content,
        )}
      >
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className={cn("w-full", layout.flex.between, "body-small font-medium")}
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
          <div className={cn(spacing.margin.top.content)}>
            {/* Channel mode toggle */}
            <div className={cn(layout.inline.default, spacing.margin.bottom.content)}>
              <label className={cn(layout.inline.default, "cursor-pointer")}>
                <input
                  type="radio"
                  name="channelMode"
                  checked={channelMode === "all"}
                  onChange={() => setChannelMode("all")}
                  className="w-4 h-4 accent-brand-primary"
                />
                <span className="body-small">{t("config.allChannels")}</span>
              </label>
              <label className={cn(layout.inline.default, "cursor-pointer")}>
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
                  <div key={band} className={cn(spacing.margin.bottom.content)}>
                    <span
                      className={cn("caption text-text-muted block", spacing.margin.bottom.tight)}
                    >
                      {getBandInfo(band).label} {t("config.channels")}
                    </span>
                    <div className="flex flex-wrap gap-1">
                      {getChannelsForBand(band).map((channel) => (
                        <button
                          type="button"
                          key={channel}
                          onClick={() => handleChannelToggle(band, channel)}
                          className={cn(
                            button.size.xs,
                            radius.sm,
                            "min-w-[2.5rem]",
                            selectedChannels.includes(channel)
                              ? "bg-brand-primary text-text-inverse"
                              : "bg-surface-base border border-surface-border hover:bg-surface-hover",
                          )}
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
          className={cn(
            "border border-surface-border",
            radius.md,
            spacing.pad.sm,
            spacing.margin.bottom.content,
          )}
        >
          <div className={cn(layout.inline.default, spacing.margin.bottom.inline)}>
            <Wifi className={iconTokens.size.sm} />
            <span className="body-small font-medium">{t("config.adapterConfig")}</span>
            <span className={cn("caption text-text-muted")}>
              ({availableAdapters.length} {t("config.detected")})
            </span>
          </div>

          {/* Adapter list */}
          {adapterConfigs.map((adapter, index) => (
            <div
              key={adapter.interface}
              className={cn(
                "bg-surface-base",
                radius.md,
                spacing.pad.sm,
                index > 0 ? spacing.margin.top.content : "",
              )}
            >
              <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
                <span className="body-small font-medium">{adapter.interface}</span>
                {index > 0 && (
                  <button
                    type="button"
                    onClick={() => handleRemoveAdapter(index)}
                    className="caption text-status-error hover:underline"
                  >
                    {t("config.remove")}
                  </button>
                )}
              </div>
              <div className={cn(layout.inline.default)}>
                <div>
                  <label
                    htmlFor={`adapter-mode-${adapter.interface}`}
                    className="caption text-text-muted"
                  >
                    {t("config.mode")}
                  </label>
                  <select
                    id={`adapter-mode-${adapter.interface}`}
                    value={adapter.mode}
                    onChange={(e) => handleAdapterModeChange(index, e.target.value as SurveyType)}
                    className={cn(inputTokens.base, inputTokens.state.default, inputTokens.size.sm)}
                  >
                    <option value="passive">{t("settings.types.passive")}</option>
                    <option value="active">{t("settings.types.active")}</option>
                    <option value="throughput">{t("settings.types.throughput")}</option>
                  </select>
                </div>
                <div>
                  <span className="caption text-text-muted">{t("config.bands")}</span>
                  <div className="flex gap-2">
                    {(["2.4", "5", "6"] as WiFiBand[]).map((band) => (
                      <label key={band} className={cn(layout.inline.default, "cursor-pointer")}>
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
              type="button"
              onClick={handleAddAdapter}
              className={cn(
                button.size.sm,
                "border border-dashed border-surface-border",
                radius.md,
                "hover:bg-surface-hover w-full",
                spacing.margin.top.content,
              )}
            >
              + {t("config.addAdapter")}
            </button>
          )}

          {/* Multi-adapter recommendation */}
          {adapterConfigs.length > 1 && (
            <div
              className={cn(
                "bg-status-info/10 border border-status-info/20",
                radius.md,
                spacing.pad.sm,
                spacing.margin.top.content,
              )}
            >
              <div className="body-small text-status-info">{t("config.multiAdapterTip")}</div>
            </div>
          )}
        </div>
      )}

      {/* Throughput Settings (if applicable) */}
      {(surveyType === "throughput" || adapterConfigs.some((a) => a.mode === "throughput")) && (
        <div className={cn("border border-surface-border", radius.md, spacing.pad.sm)}>
          <div className={cn(layout.inline.default, spacing.margin.bottom.inline)}>
            <Settings className={iconTokens.size.sm} />
            <span className="body-small font-medium">{t("config.throughputSettings")}</span>
          </div>
          <div className={cn(layout.stack.default)}>
            <div>
              <label
                htmlFor="config-iperf-server"
                className={cn("caption text-text-muted block", spacing.margin.bottom.tight)}
              >
                {t("settings.iperfServer")}
              </label>
              <input
                id="config-iperf-server"
                type="text"
                value={localIperfServer}
                onChange={(e) => setLocalIperfServer(e.target.value)}
                onBlur={handleIperfSave}
                placeholder="192.168.1.100:5201"
                className={cn(
                  "w-full",
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                )}
              />
              <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
                {t("settings.iperfServerHint")}
              </p>
            </div>
            <div>
              <label
                htmlFor="config-test-duration"
                className={cn("caption text-text-muted block", spacing.margin.bottom.tight)}
              >
                {t("settings.testDuration")}
              </label>
              <input
                id="config-test-duration"
                type="number"
                min={1}
                max={30}
                value={localTestDuration}
                onChange={(e) => setLocalTestDuration(Number.parseInt(e.target.value, 10) || 3)}
                onBlur={handleIperfSave}
                className={cn(
                  "w-24",
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                )}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
