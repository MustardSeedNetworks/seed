/**
 * WiFiSurveyCard Component
 *
 * Purpose: Manages WiFi site surveys - allows creating floor plan-based WiFi signal mapping campaigns.
 * Shows active and completed surveys with ability to create, pause, complete, and delete surveys.
 *
 * Key Features:
 * - Survey creation: dialog to create new survey with custom floor plan image
 * - Survey types: supports different survey modes (e.g., signal mapping, interference detection)
 * - Active surveys: shows in-progress and paused surveys with controls
 * - Completed surveys: displays finished surveys with ability to view results
 * - Survey controls: start, pause, complete, delete operations
 * - Floor plan visualization: SurveyView component displays floor plan with sample points
 * - Sample tracking: displays number of signal samples collected
 * - Status: warning (active survey), success (ready/completed) - fixes #737
 *
 * Usage:
 * ```typescript
 * <WiFiSurveyCard isWifi={wifiConnected} />
 * ```
 *
 * Dependencies: useSurvey hook, SurveyView component, Card UI components, Icons
 * State: Manages surveys list, selected survey, create dialog state, fetches from API
 */

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Card, Status } from "../ui/Card";
import { useSurvey, type Survey, type SurveyType } from "../../hooks/useSurvey";
import { SurveyView } from "../survey/SurveyView";
import { Activity } from "../ui/Icons";
import { logger, LogComponents } from "../../lib/logger";
import {
  radius,
  input as inputTokens,
  icon as iconTokens,
  modal,
  button,
  spacing,
  layout,
} from "../../styles/theme";

interface WiFiSurveyCardProps {
  isWifi: boolean;
  /** Current WiFi interface name - fix #572: no hardcoded interface names */
  currentInterface?: string;
}

/**
 * Manages WiFi site surveys for signal mapping with floor plan integration.
 */
export function WiFiSurveyCard({
  isWifi,
  currentInterface = "",
}: WiFiSurveyCardProps) {
  const { t } = useTranslation("cards");
  const {
    surveys,
    loading,
    error,
    createSurvey,
    deleteSurvey,
    startSurvey,
    pauseSurvey,
    completeSurvey,
  } = useSurvey();

  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [selectedSurvey, setSelectedSurvey] = useState<Survey | null>(null);

  const activeSurveys = surveys.filter(
    (s) => s.status === "in_progress" || s.status === "paused"
  );

  // Fixes #737: Use "success" for no surveys (ready state) instead of confusing "?" badge
  const getCardStatus = (): Status => {
    if (activeSurveys.length > 0) return "warning"; // Active work needs attention
    return "success"; // Ready or completed - system is healthy
  };

  const handleCreateSurvey = async (
    name: string,
    surveyType: SurveyType,
    iface: string
  ) => {
    try {
      const newSurvey = await createSurvey({
        name,
        surveyType,
        interface: iface,
      });
      setShowCreateDialog(false);
      // Automatically open the survey view for setup
      setSelectedSurvey(newSurvey);
    } catch (err) {
      logger.error(LogComponents.SURVEY, "Failed to create survey", err);
    }
  };

  const handleDelete = async (id: string) => {
    if (confirm(t("survey.confirmDelete"))) {
      await deleteSurvey(id);
    }
  };

  const getSurveyTypeLabel = (type: SurveyType) => {
    switch (type) {
      case "passive":
        return t("survey.typePassiveLabel");
      case "active":
        return t("survey.typeActiveLabel");
      case "throughput":
        return t("survey.typeThroughputLabel");
      default:
        return type;
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case "in_progress":
        return t("survey.inProgress");
      case "paused":
        return t("survey.paused");
      case "completed":
        return t("survey.completed");
      case "created":
        return t("survey.created");
      default:
        return status
          .replace("_", " ")
          .replace(/\b\w/g, (l) => l.toUpperCase());
    }
  };

  return (
    <>
      <Card
        title={t("survey.title")}
        status={getCardStatus()}
        icon={<Activity className={iconTokens.size.md} />}
        headerAction={
          <button
            onClick={(e) => {
              e.stopPropagation();
              setShowCreateDialog(true);
            }}
            className="caption font-medium text-brand-primary hover:underline"
          >
            {t("survey.new")}
          </button>
        }
      >
        {!isWifi && (
          <div
            className={`bg-status-warning/10 border border-status-warning/20 text-status-warning ${spacing.pad.sm} ${radius.md} body-small ${spacing.margin.bottom.heading}`}
          >
            {t("survey.wifiRequired")}
          </div>
        )}

        {error && (
          <div
            className={`bg-status-error/10 border border-status-error/20 text-status-error ${spacing.pad.sm} ${radius.md} body-small ${spacing.margin.bottom.heading}`}
          >
            {error}
          </div>
        )}

        {loading && surveys.length === 0 ? (
          <div
            className={`text-center ${spacing.pad.lg} text-text-muted body-small`}
          >
            {t("survey.loading")}
          </div>
        ) : surveys.length === 0 ? (
          <div className={`text-center ${spacing.pad.lg} text-text-muted`}>
            <p className={`body-small ${spacing.margin.bottom.inline}`}>
              {t("survey.noSurveys")}
            </p>
            <button
              onClick={() => setShowCreateDialog(true)}
              className="body-small text-brand-primary hover:underline"
            >
              {t("survey.createFirst")}
            </button>
          </div>
        ) : (
          <div className="stack-sm">
            {surveys.slice(0, 3).map((survey) => (
              <div
                key={survey.id}
                className={`border border-surface-border ${radius.md} pad-sm hover:bg-surface-hover transition-colors cursor-pointer`}
                onClick={() => setSelectedSurvey(survey)}
              >
                <div className={layout.flex.between}>
                  <div className="flex-1 min-w-0">
                    <div className={layout.inline.default}>
                      <h4 className="font-medium body-small truncate">
                        {survey.name}
                      </h4>
                      <span className="caption text-text-muted">
                        {getStatusLabel(survey.status)}
                      </span>
                    </div>
                    <div
                      className={`${layout.inline.comfortable} ${spacing.margin.top.inline} caption text-text-muted`}
                    >
                      <span>{getSurveyTypeLabel(survey.surveyType)}</span>
                      <span>
                        {survey.samples?.length ?? 0}{" "}
                        {t("survey.samples").toLowerCase()}
                      </span>
                    </div>
                  </div>
                  <div
                    className={`${layout.inline.tight} ${spacing.margin.left.inline}`}
                  >
                    {survey.status === "created" && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          startSurvey(survey.id);
                        }}
                        className={`${button.size.xs} caption border border-surface-border ${radius.md} hover:bg-surface-hover`}
                        title={t("survey.start")}
                      >
                        ▶
                      </button>
                    )}
                    {survey.status === "in_progress" && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          pauseSurvey(survey.id);
                        }}
                        className={`${button.size.xs} caption border border-surface-border ${radius.md} hover:bg-surface-hover`}
                        title={t("survey.pause")}
                      >
                        ⏸
                      </button>
                    )}
                    {survey.status === "paused" && (
                      <>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            startSurvey(survey.id);
                          }}
                          className={`${button.size.xs} caption border border-surface-border ${radius.md} hover:bg-surface-hover`}
                          title={t("survey.resume")}
                        >
                          ▶
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            completeSurvey(survey.id);
                          }}
                          className={`${button.size.xs} caption border border-surface-border ${radius.md} hover:bg-surface-hover`}
                          title={t("survey.complete")}
                        >
                          ✓
                        </button>
                      </>
                    )}
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDelete(survey.id);
                      }}
                      className={`${button.size.xs} caption border border-surface-border ${radius.md} hover:bg-status-error/10 text-status-error`}
                      title={t("survey.delete")}
                    >
                      ×
                    </button>
                  </div>
                </div>
              </div>
            ))}
            {surveys.length > 3 && (
              <div
                className={`text-center caption text-text-muted ${spacing.padding.top.tight}`}
              >
                {t("survey.more", { count: surveys.length - 3 })}
              </div>
            )}
          </div>
        )}
      </Card>

      {showCreateDialog && (
        <CreateSurveyDialog
          onClose={() => setShowCreateDialog(false)}
          onCreate={handleCreateSurvey}
          t={t}
          currentInterface={currentInterface}
        />
      )}

      {selectedSurvey && (
        <SurveyView
          survey={selectedSurvey}
          onClose={() => setSelectedSurvey(null)}
          onUpdate={() => {
            // Refresh surveys list when survey is updated
            // The useSurvey hook will automatically refresh
          }}
        />
      )}
    </>
  );
}

interface CreateSurveyDialogProps {
  onClose: () => void;
  onCreate: (name: string, type: SurveyType, iface: string) => void;
  t: ReturnType<typeof useTranslation<"cards">>["t"];
  /** Current WiFi interface name - fix #572: no hardcoded interface names */
  currentInterface?: string;
}

function CreateSurveyDialog({
  onClose,
  onCreate,
  t,
  currentInterface = "",
}: CreateSurveyDialogProps) {
  const [name, setName] = useState("");
  const [surveyType, setSurveyType] = useState<SurveyType>("passive");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim()) {
      // Fix #572: Use interface from props instead of hardcoded "wlan0"
      onCreate(name.trim(), surveyType, currentInterface);
    }
  };

  return (
    <div className={modal.overlay}>
      <div
        className={`bg-surface-raised ${radius.md} ${spacing.pad.lg} max-w-md w-full ${spacing.pad.default}`}
      >
        <h2 className={`heading-2 ${spacing.margin.bottom.content}`}>
          {t("survey.createNewSurvey")}
        </h2>
        <form onSubmit={handleSubmit}>
          <div className="stack">
            <div>
              <label className={`label block ${spacing.margin.bottom.tight}`}>
                {t("survey.surveyName")}
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md}`}
                placeholder={t("survey.namePlaceholder")}
                required
              />
            </div>
            <div>
              <label
                className={`label block ${spacing.margin.bottom.tight}`}
                htmlFor="survey-type"
              >
                {t("survey.surveyType")}
              </label>
              <select
                id="survey-type"
                value={surveyType}
                onChange={(e) => setSurveyType(e.target.value as SurveyType)}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md}`}
              >
                <option value="passive">{t("survey.typePassive")}</option>
                <option value="active">{t("survey.typeActive")}</option>
                <option value="throughput">{t("survey.typeThroughput")}</option>
              </select>
            </div>
          </div>
          <div
            className={`${layout.inline.default} ${spacing.margin.top.section}`}
          >
            <button
              type="button"
              onClick={onClose}
              className={`flex-1 ${button.size.md} border border-surface-border ${radius.md} hover:bg-surface-hover`}
            >
              {t("survey.cancel")}
            </button>
            <button
              type="submit"
              className={`flex-1 ${button.size.md} bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90`}
            >
              {t("survey.create")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
