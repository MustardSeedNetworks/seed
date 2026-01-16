/**
 * PassFailCriteriaPanel Component
 *
 * Purpose: Configure pass/fail thresholds for survey validation.
 * Allows users to enable/disable criteria and adjust threshold values.
 *
 * Key Features:
 * - Configurable thresholds per survey type
 * - Enable/disable individual criteria
 * - Reset to defaults
 * - Import criteria from AirMapper
 *
 * Usage:
 * ```typescript
 * <PassFailCriteriaPanel
 *   surveyType="passive"
 *   criteria={currentCriteria}
 *   onChange={(criteria) => setCriteria(criteria)}
 *   onValidate={() => runValidation()}
 * />
 * ```
 */

import { ChevronDown, ChevronRight, Play, RotateCcw, Settings2, Upload } from "lucide-react";
import type React from "react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ComparisonOperator, PassFailCriterion, SurveyType } from "../../hooks/useSurvey";
import {
  DEFAULT_ACTIVE_CRITERIA,
  DEFAULT_PASSIVE_CRITERIA,
  DEFAULT_THROUGHPUT_CRITERIA,
  getDefaultCriteria,
} from "../../hooks/useSurvey";
import { button, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";

interface PassFailCriteriaPanelProps {
  surveyType: SurveyType;
  criteria: PassFailCriterion[];
  onChange: (criteria: PassFailCriterion[]) => void;
  onValidate?: () => void;
  onImportFromAirMapper?: () => void;
  validating?: boolean;
  disabled?: boolean;
}

/** Group criteria by mode for display */
function groupCriteriaByMode(criteria: PassFailCriterion[]): Map<string, PassFailCriterion[]> {
  const groups = new Map<string, PassFailCriterion[]>();
  for (const criterion of criteria) {
    const mode = criterion.mode === "all" ? "all" : criterion.mode;
    if (!groups.has(mode)) {
      groups.set(mode, []);
    }
    groups.get(mode)?.push(criterion);
  }
  return groups;
}

/** Render comparison operator symbol */
function _comparisonSymbol({ comparison }: { comparison: ComparisonOperator }): React.ReactElement {
  return (
    <span class="text-text-muted font-mono">{comparison === "gte" ? "\u2265" : "\u2264"}</span>
  );
}

/**
 * A single criterion row with toggle and threshold input
 */
function _criterionRow({
  criterion,
  onChange,
  disabled,
}: {
  criterion: PassFailCriterion;
  onChange: (updated: PassFailCriterion) => void;
  disabled?: boolean;
}): React.ReactElement {
  const { t } = useTranslation("survey");

  const handleToggle = useCallback(() => {
    onChange({ ...criterion, enabled: !criterion.enabled });
  }, [criterion, onChange]);

  const handleThresholdChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = Number.parseFloat(e.target.value);
      if (!Number.isNaN(value)) {
        onChange({ ...criterion, threshold: value });
      }
    },
    [criterion, onChange],
  );

  return (
    <div
      class={cn(
        layout.inline.default,
        "justify-between py-1.5 border-b border-surface-border/50 last:border-b-0",
      )}
    >
      {/* Enable checkbox and name */}
      <label class={cn(layout.inline.tight, "cursor-pointer flex-1")}>
        <input
          type="checkbox"
          checked={criterion.enabled}
          onChange={handleToggle}
          disabled={disabled}
          class="w-4 h-4 rounded border-surface-border text-brand-primary focus:ring-brand-primary"
        />
        <span class={cn("caption", criterion.enabled ? "text-text-primary" : "text-text-muted")}>
          {t(criterion.displayKey as never)}
        </span>
      </label>

      {/* Comparison and threshold */}
      <div class={cn(layout.inline.tight)}>
        <comparisonSymbol comparison={criterion.comparison} />
        <input
          type="number"
          value={criterion.threshold}
          onChange={handleThresholdChange}
          disabled={disabled || !criterion.enabled}
          class={cn(
            "w-16 px-2 py-1 text-right caption bg-surface-default border border-surface-border disabled:opacity-50 focus:outline-none focus:ring-1 focus:ring-brand-primary",
            radius.sm,
          )}
          step={criterion.suffix === "%" || criterion.suffix === "ms" ? 1 : 0.1}
        />
        <span class="caption text-text-muted w-12">{criterion.suffix}</span>
      </div>
    </div>
  );
}

/**
 * Collapsible section for criteria group
 */
function _criteriaSection({
  title,
  criteria,
  expanded,
  onToggle,
  onChange,
  disabled,
}: {
  title: string;
  criteria: PassFailCriterion[];
  expanded: boolean;
  onToggle: () => void;
  onChange: (id: string, updated: PassFailCriterion) => void;
  disabled?: boolean;
}): React.ReactElement {
  return (
    <div class="mb-2">
      {/* Section header */}
      <button
        type="button"
        onClick={onToggle}
        class={cn(
          "w-full justify-between py-1.5 hover:bg-surface-hover transition-colors",
          layout.inline.default,
          radius.sm,
        )}
      >
        <span class="caption font-medium text-text-secondary">{title}</span>
        {expanded ? (
          <ChevronDown class={iconTokens.size.xs} />
        ) : (
          <ChevronRight class={iconTokens.size.xs} />
        )}
      </button>

      {/* Criteria list */}
      {expanded ? (
        <div class={cn("pl-2", spacing.margin.top.tight)}>
          {criteria.map((criterion) => (
            <criterionRow
              key={criterion.id}
              criterion={criterion}
              onChange={(updated: PassFailCriterion): void => onChange(criterion.id, updated)}
              disabled={disabled}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

/**
 * PassFailCriteriaPanel for configuring validation thresholds
 */
export function PassFailCriteriaPanel({
  surveyType,
  criteria,
  onChange,
  onValidate,
  onImportFromAirMapper,
  validating = false,
  disabled = false,
}: PassFailCriteriaPanelProps): React.ReactElement {
  const { t } = useTranslation("survey");

  // Track which sections are expanded
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set([surveyType]));

  // Toggle section expansion
  const toggleSection = useCallback((mode: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(mode)) {
        next.delete(mode);
      } else {
        next.add(mode);
      }
      return next;
    });
  }, []);

  // Update a single criterion
  const handleCriterionChange = useCallback(
    (id: string, updated: PassFailCriterion) => {
      onChange(criteria.map((c) => (c.id === id ? updated : c)));
    },
    [criteria, onChange],
  );

  // Reset to defaults
  const handleReset = useCallback(() => {
    onChange(getDefaultCriteria(surveyType));
  }, [surveyType, onChange]);

  // Load all criteria for comprehensive testing
  const handleLoadAll = useCallback(() => {
    onChange([
      ...DEFAULT_PASSIVE_CRITERIA,
      ...DEFAULT_ACTIVE_CRITERIA,
      ...DEFAULT_THROUGHPUT_CRITERIA,
    ]);
    setExpandedSections(new Set(["passive", "active", "throughput"]));
  }, [onChange]);

  // Group criteria by mode
  const groupedCriteria = groupCriteriaByMode(criteria);

  const getSectionTitle = (mode: string): string => {
    switch (mode) {
      case "passive":
        return t("criteria.passiveSection");
      case "active":
        return t("criteria.activeSection");
      case "throughput":
        return t("criteria.throughputSection");
      case "all":
        return t("criteria.allSection");
      default:
        return mode;
    }
  };

  return (
    <div class={cn("bg-surface-raised border border-surface-border", radius.md, spacing.pad.sm)}>
      {/* Header */}
      <div class={cn("justify-between", layout.inline.default, spacing.margin.bottom.content)}>
        <div class={cn(layout.inline.default)}>
          <Settings2 class={iconTokens.size.sm} />
          <h4 class="body-small font-medium">{t("criteria.title")}</h4>
        </div>
        {onValidate ? (
          <button
            type="button"
            onClick={onValidate}
            disabled={disabled || validating}
            class={cn(
              "bg-brand-primary text-text-inverse hover:opacity-90 disabled:opacity-50",
              button.size.sm,
              radius.md,
              layout.inline.tight,
            )}
          >
            <Play class="w-3 h-3" />
            <span>{validating ? t("criteria.validating") : t("criteria.runTest")}</span>
          </button>
        ) : null}
      </div>

      {/* Criteria sections */}
      <div class={spacing.margin.bottom.content}>
        {Array.from(groupedCriteria.entries()).map(([mode, modeCriteria]) => (
          <criteriaSection
            key={mode}
            title={getSectionTitle(mode)}
            criteria={modeCriteria}
            expanded={expandedSections.has(mode)}
            onToggle={(): void => toggleSection(mode)}
            onChange={handleCriterionChange}
            disabled={disabled}
          />
        ))}
      </div>

      {/* Actions */}
      <div class={cn("justify-between pt-2 border-t border-surface-border", layout.inline.default)}>
        <div class={cn(layout.inline.tight)}>
          <button
            type="button"
            onClick={handleReset}
            disabled={disabled}
            class={cn(
              "bg-surface-default border border-surface-border hover:bg-surface-hover disabled:opacity-50",
              button.size.sm,
              radius.md,
              layout.inline.tight,
            )}
          >
            <RotateCcw class="w-3 h-3" />
            <span>{t("criteria.resetDefaults")}</span>
          </button>
          <button
            type="button"
            onClick={handleLoadAll}
            disabled={disabled}
            class={cn(
              "bg-surface-default border border-surface-border hover:bg-surface-hover disabled:opacity-50",
              button.size.sm,
              radius.md,
            )}
          >
            {t("criteria.loadAll")}
          </button>
        </div>
        {onImportFromAirMapper ? (
          <button
            type="button"
            onClick={onImportFromAirMapper}
            disabled={disabled}
            class={cn(
              "bg-surface-default border border-surface-border hover:bg-surface-hover disabled:opacity-50",
              button.size.sm,
              radius.md,
              layout.inline.tight,
            )}
          >
            <Upload class="w-3 h-3" />
            <span>{t("criteria.importAirMapper")}</span>
          </button>
        ) : null}
      </div>
    </div>
  );
}
