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

import { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
  Settings2,
  RotateCcw,
  Play,
  ChevronDown,
  ChevronRight,
  Upload,
} from "lucide-react";
import {
  cn,
  radius,
  spacing,
  layout,
  icon as iconTokens,
  button,
} from "../../styles/theme";
import type {
  PassFailCriterion,
  SurveyType,
  ComparisonOperator,
} from "../../hooks/useSurvey";
import {
  getDefaultCriteria,
  DEFAULT_PASSIVE_CRITERIA,
  DEFAULT_ACTIVE_CRITERIA,
  DEFAULT_THROUGHPUT_CRITERIA,
} from "../../hooks/useSurvey";

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
function groupCriteriaByMode(
  criteria: PassFailCriterion[]
): Map<string, PassFailCriterion[]> {
  const groups = new Map<string, PassFailCriterion[]>();
  for (const criterion of criteria) {
    const mode = criterion.mode === "all" ? "all" : criterion.mode;
    if (!groups.has(mode)) {
      groups.set(mode, []);
    }
    groups.get(mode)!.push(criterion);
  }
  return groups;
}

/** Render comparison operator symbol */
function ComparisonSymbol({ comparison }: { comparison: ComparisonOperator }) {
  return (
    <span className="text-text-muted font-mono">
      {comparison === "gte" ? "\u2265" : "\u2264"}
    </span>
  );
}

/**
 * A single criterion row with toggle and threshold input
 */
function CriterionRow({
  criterion,
  onChange,
  disabled,
}: {
  criterion: PassFailCriterion;
  onChange: (updated: PassFailCriterion) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation("survey");

  const handleToggle = useCallback(() => {
    onChange({ ...criterion, enabled: !criterion.enabled });
  }, [criterion, onChange]);

  const handleThresholdChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = parseFloat(e.target.value);
      if (!isNaN(value)) {
        onChange({ ...criterion, threshold: value });
      }
    },
    [criterion, onChange]
  );

  return (
    <div
      className={cn(
        layout.inline.default,
        "justify-between py-1.5 border-b border-surface-border/50 last:border-b-0"
      )}
    >
      {/* Enable checkbox and name */}
      <label className={cn(layout.inline.tight, "cursor-pointer flex-1")}>
        <input
          type="checkbox"
          checked={criterion.enabled}
          onChange={handleToggle}
          disabled={disabled}
          className="w-4 h-4 rounded border-surface-border text-brand-primary focus:ring-brand-primary"
        />
        <span
          className={cn(
            "caption",
            criterion.enabled ? "text-text-primary" : "text-text-muted"
          )}
        >
          {t(criterion.displayKey as never)}
        </span>
      </label>

      {/* Comparison and threshold */}
      <div className={cn(layout.inline.tight)}>
        <ComparisonSymbol comparison={criterion.comparison} />
        <input
          type="number"
          value={criterion.threshold}
          onChange={handleThresholdChange}
          disabled={disabled || !criterion.enabled}
          className={cn(
            "w-16 px-2 py-1 text-right caption bg-surface-default border border-surface-border disabled:opacity-50 focus:outline-none focus:ring-1 focus:ring-brand-primary",
            radius.sm
          )}
          step={criterion.suffix === "%" || criterion.suffix === "ms" ? 1 : 0.1}
        />
        <span className="caption text-text-muted w-12">{criterion.suffix}</span>
      </div>
    </div>
  );
}

/**
 * Collapsible section for criteria group
 */
function CriteriaSection({
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
}) {
  return (
    <div className="mb-2">
      {/* Section header */}
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "w-full justify-between py-1.5 hover:bg-surface-hover transition-colors",
          layout.inline.default,
          radius.sm
        )}
      >
        <span className="caption font-medium text-text-secondary">{title}</span>
        {expanded ? (
          <ChevronDown className={iconTokens.size.xs} />
        ) : (
          <ChevronRight className={iconTokens.size.xs} />
        )}
      </button>

      {/* Criteria list */}
      {expanded && (
        <div className={cn("pl-2", spacing.margin.top.tight)}>
          {criteria.map((criterion) => (
            <CriterionRow
              key={criterion.id}
              criterion={criterion}
              onChange={(updated) => onChange(criterion.id, updated)}
              disabled={disabled}
            />
          ))}
        </div>
      )}
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
}: PassFailCriteriaPanelProps) {
  const { t } = useTranslation("survey");

  // Track which sections are expanded
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set([surveyType])
  );

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
    [criteria, onChange]
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

  const getSectionTitle = (mode: string) => {
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
    <div
      className={cn(
        "bg-surface-raised border border-surface-border",
        radius.md,
        spacing.pad.sm
      )}
    >
      {/* Header */}
      <div
        className={cn(
          "justify-between",
          layout.inline.default,
          spacing.margin.bottom.content
        )}
      >
        <div className={cn(layout.inline.default)}>
          <Settings2 className={iconTokens.size.sm} />
          <h4 className="body-small font-medium">{t("criteria.title")}</h4>
        </div>
        {onValidate && (
          <button
            type="button"
            onClick={onValidate}
            disabled={disabled || validating}
            className={cn(
              "bg-brand-primary text-text-inverse hover:opacity-90 disabled:opacity-50",
              button.size.sm,
              radius.md,
              layout.inline.tight
            )}
          >
            <Play className="w-3 h-3" />
            <span>
              {validating ? t("criteria.validating") : t("criteria.runTest")}
            </span>
          </button>
        )}
      </div>

      {/* Criteria sections */}
      <div className={spacing.margin.bottom.content}>
        {Array.from(groupedCriteria.entries()).map(([mode, modeCriteria]) => (
          <CriteriaSection
            key={mode}
            title={getSectionTitle(mode)}
            criteria={modeCriteria}
            expanded={expandedSections.has(mode)}
            onToggle={() => toggleSection(mode)}
            onChange={handleCriterionChange}
            disabled={disabled}
          />
        ))}
      </div>

      {/* Actions */}
      <div
        className={cn(
          "justify-between pt-2 border-t border-surface-border",
          layout.inline.default
        )}
      >
        <div className={cn(layout.inline.tight)}>
          <button
            type="button"
            onClick={handleReset}
            disabled={disabled}
            className={cn(
              "bg-surface-default border border-surface-border hover:bg-surface-hover disabled:opacity-50",
              button.size.sm,
              radius.md,
              layout.inline.tight
            )}
          >
            <RotateCcw className="w-3 h-3" />
            <span>{t("criteria.resetDefaults")}</span>
          </button>
          <button
            type="button"
            onClick={handleLoadAll}
            disabled={disabled}
            className={cn(
              "bg-surface-default border border-surface-border hover:bg-surface-hover disabled:opacity-50",
              button.size.sm,
              radius.md
            )}
          >
            {t("criteria.loadAll")}
          </button>
        </div>
        {onImportFromAirMapper && (
          <button
            type="button"
            onClick={onImportFromAirMapper}
            disabled={disabled}
            className={cn(
              "bg-surface-default border border-surface-border hover:bg-surface-hover disabled:opacity-50",
              button.size.sm,
              radius.md,
              layout.inline.tight
            )}
          >
            <Upload className="w-3 h-3" />
            <span>{t("criteria.importAirMapper")}</span>
          </button>
        )}
      </div>
    </div>
  );
}
