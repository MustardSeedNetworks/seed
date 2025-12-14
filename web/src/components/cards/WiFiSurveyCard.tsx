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
 * - Status: warning (active survey), success (completed), unknown (none)
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
import { Card, Status } from "../ui/Card";
import { useSurvey, type Survey, type SurveyType } from "../../hooks/useSurvey";
import { SurveyView } from "../survey/SurveyView";
import { Activity } from "../ui/Icons";
import { radius, input as inputTokens } from "../../styles/theme";

interface WiFiSurveyCardProps {
  isWifi: boolean;
}

export function WiFiSurveyCard({ isWifi }: WiFiSurveyCardProps) {
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
    (s) => s.status === "in_progress" || s.status === "paused",
  );
  const completedSurveys = surveys.filter((s) => s.status === "completed");

  const getCardStatus = (): Status => {
    if (activeSurveys.length > 0) return "warning";
    if (completedSurveys.length > 0) return "success";
    return "unknown";
  };

  const handleCreateSurvey = async (
    name: string,
    surveyType: SurveyType,
    iface: string,
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
      console.error("Failed to create survey:", err);
    }
  };

  const handleDelete = async (id: string) => {
    if (confirm("Are you sure you want to delete this survey?")) {
      await deleteSurvey(id);
    }
  };

  const getSurveyTypeLabel = (type: SurveyType) => {
    switch (type) {
      case "passive":
        return "Passive Scan";
      case "active":
        return "Active";
      case "throughput":
        return "Throughput";
      default:
        return type;
    }
  };

  const getStatusLabel = (status: string) => {
    return status.replace("_", " ").replace(/\b\w/g, (l) => l.toUpperCase());
  };

  return (
    <>
      <Card
        title="WiFi Site Survey"
        status={getCardStatus()}
        icon={<Activity className="h-5 w-5" />}
        headerAction={
          <button
            onClick={(e) => {
              e.stopPropagation();
              setShowCreateDialog(true);
            }}
            className="caption font-medium text-brand-primary hover:underline"
          >
            + New
          </button>
        }
      >
        {!isWifi && (
          <div
            className={`bg-status-warning/10 border border-status-warning/20 text-status-warning px-3 py-2 ${radius.md} body-small mb-3`}
          >
            WiFi interface required for site surveys. Switch to a WiFi interface
            to create surveys.
          </div>
        )}

        {error && (
          <div
            className={`bg-status-error/10 border border-status-error/20 text-status-error px-3 py-2 ${radius.md} body-small mb-3`}
          >
            {error}
          </div>
        )}

        {loading && surveys.length === 0 ? (
          <div className="text-center py-6 text-text-muted body-small">
            Loading...
          </div>
        ) : surveys.length === 0 ? (
          <div className="text-center py-6 text-text-muted">
            <p className="body-small mb-2">No surveys yet</p>
            <button
              onClick={() => setShowCreateDialog(true)}
              className="body-small text-brand-primary hover:underline"
            >
              Create your first survey
            </button>
          </div>
        ) : (
          <div className="stack-sm">
            {surveys.slice(0, 3).map((survey) => (
              <div
                key={survey.id}
                className={`border ${radius.md} p-2 hover:bg-surface-hover transition-colors cursor-pointer`}
                onClick={() => setSelectedSurvey(survey)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h4 className="font-medium body-small truncate">
                        {survey.name}
                      </h4>
                      <span className="caption text-text-muted">
                        {getStatusLabel(survey.status)}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 mt-1 caption text-text-muted">
                      <span>{getSurveyTypeLabel(survey.surveyType)}</span>
                      <span>{survey.samples.length} samples</span>
                    </div>
                  </div>
                  <div className="flex gap-1 ml-2">
                    {survey.status === "created" && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          startSurvey(survey.id);
                        }}
                        className={`px-2 py-1 caption border ${radius.md} hover:bg-surface-hover`}
                        title="Start"
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
                        className={`px-2 py-1 caption border ${radius.md} hover:bg-surface-hover`}
                        title="Pause"
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
                          className={`px-2 py-1 caption border ${radius.md} hover:bg-surface-hover`}
                          title="Resume"
                        >
                          ▶
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            completeSurvey(survey.id);
                          }}
                          className={`px-2 py-1 caption border ${radius.md} hover:bg-surface-hover`}
                          title="Complete"
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
                      className={`px-2 py-1 caption border ${radius.md} hover:bg-status-error/10 text-status-error`}
                      title="Delete"
                    >
                      ×
                    </button>
                  </div>
                </div>
              </div>
            ))}
            {surveys.length > 3 && (
              <div className="text-center caption text-text-muted pt-1">
                +{surveys.length - 3} more
              </div>
            )}
          </div>
        )}
      </Card>

      {showCreateDialog && (
        <CreateSurveyDialog
          onClose={() => setShowCreateDialog(false)}
          onCreate={handleCreateSurvey}
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
}

function CreateSurveyDialog({ onClose, onCreate }: CreateSurveyDialogProps) {
  const [name, setName] = useState("");
  const [surveyType, setSurveyType] = useState<SurveyType>("passive");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim()) {
      onCreate(name.trim(), surveyType, "wlan0");
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div
        className={`bg-surface-raised ${radius.md} p-6 max-w-md w-full mx-4`}
      >
        <h2 className="heading-2 mb-4">Create New Survey</h2>
        <form onSubmit={handleSubmit}>
          <div className="stack">
            <div>
              <label className="label block mb-1">Survey Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md}`}
                placeholder="e.g., Office Floor 1"
                required
              />
            </div>
            <div>
              <label className="label block mb-1" htmlFor="survey-type">
                Survey Type
              </label>
              <select
                id="survey-type"
                value={surveyType}
                onChange={(e) => setSurveyType(e.target.value as SurveyType)}
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md}`}
              >
                <option value="passive">Passive Scan (All Networks)</option>
                <option value="active">Active Monitoring (Connection)</option>
                <option value="throughput">Throughput Testing (iperf3)</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2 mt-6">
            <button
              type="button"
              onClick={onClose}
              className={`flex-1 px-4 py-2 border border-surface-border ${radius.md} hover:bg-surface-hover`}
            >
              Cancel
            </button>
            <button
              type="submit"
              className={`flex-1 px-4 py-2 bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90`}
            >
              Create
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
