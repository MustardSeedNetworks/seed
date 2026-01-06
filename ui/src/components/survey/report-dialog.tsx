/**
 * ReportDialog Component
 *
 * Modal dialog for generating PDF survey reports. Allows users to configure
 * what sections to include in the report and download the generated PDF.
 *
 * Features:
 * - Report options checkboxes (heatmaps, raw data, recommendations, summary)
 * - Company name input for branding
 * - Generate button with loading state
 * - Auto-download PDF on completion
 *
 * Usage:
 * ```typescript
 * <ReportDialog
 *   surveyId={survey.id}
 *   surveyName={survey.name}
 *   open={showReportDialog}
 *   onClose={() => setShowReportDialog(false)}
 * />
 * ```
 */

import { CheckCircle, Download, FileText, Loader, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { button, radius, spacing } from "../../styles/theme";

// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface ReportOptions {
  includeHeatmaps: boolean;
  includeRawData: boolean;
  includeRecommendations: boolean;
  includeExecutiveSummary: boolean;
  companyName: string;
}

interface ReportDialogProps {
  surveyId: string;
  surveyName: string;
  open: boolean;
  onClose: () => void;
}

/**
 * Modal for configuring and generating survey PDF reports.
 */
export function ReportDialog({ surveyId, surveyName, open, onClose }: ReportDialogProps) {
  const { t } = useTranslation("survey");
  const [options, setOptions] = useState<ReportOptions>({
    includeHeatmaps: true,
    includeRawData: false,
    includeRecommendations: true,
    includeExecutiveSummary: true,
    companyName: "",
  });
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  if (!open) return null;

  const handleOptionChange = (key: keyof ReportOptions, value: boolean | string) => {
    setOptions((prev) => ({ ...prev, [key]: value }));
    setError(null);
    setSuccess(false);
  };

  const handleGenerate = async () => {
    setGenerating(true);
    setError(null);
    setSuccess(false);

    try {
      const response = await fetch(`${API_BASE}/api/canopy/survey/report?id=${surveyId}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(options),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || "Failed to generate report");
      }

      // Get the PDF blob
      const blob = await response.blob();

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `survey-report-${surveyName.replace(/[^a-z0-9]/gi, "_")}.pdf`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      setSuccess(true);
    } catch (err) {
      console.error("Report generation failed:", err);
      setError(err instanceof Error ? err.message : "Failed to generate report");
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 1000,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "rgba(0, 0, 0, 0.5)",
        backdropFilter: "blur(4px)",
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      onKeyDown={(e) => {
        if (e.key === "Escape") onClose();
      }}
      role="dialog"
      aria-modal="true"
    >
      <div
        style={{
          background: "var(--color-background)",
          borderRadius: radius.lg,
          boxShadow: "var(--shadow-lg)",
          width: "100%",
          maxWidth: "480px",
          maxHeight: "90vh",
          overflow: "auto",
        }}
      >
        {/* Header */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: spacing.md,
            borderBottom: "1px solid var(--color-border)",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: spacing.sm }}>
            <FileText size={20} />
            <h2 style={{ margin: 0, fontSize: "1.125rem", fontWeight: 600 }}>
              {t("report.title", "Generate Report")}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            style={{ ...button.icon, padding: spacing.xs }}
            title={t("common.close", "Close")}
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div style={{ padding: spacing.md }}>
          <p
            style={{
              margin: `0 0 ${spacing.md} 0`,
              color: "var(--color-text-secondary)",
            }}
          >
            {t(
              "report.description",
              "Configure what sections to include in your PDF survey report.",
            )}
          </p>

          {/* Options */}
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: spacing.md,
            }}
          >
            {/* Company Name */}
            <div>
              <label
                htmlFor="companyName"
                style={{
                  display: "block",
                  marginBottom: spacing.xs,
                  fontWeight: 500,
                }}
              >
                {t("report.companyName", "Company Name")} ({t("common.optional", "optional")})
              </label>
              <input
                id="companyName"
                type="text"
                value={options.companyName}
                onChange={(e) => handleOptionChange("companyName", e.target.value)}
                placeholder={t("report.companyNamePlaceholder", "Your Company")}
                style={{
                  width: "100%",
                  padding: spacing.sm,
                  borderRadius: radius.md,
                  border: "1px solid var(--color-border)",
                  background: "var(--color-background)",
                  color: "var(--color-text)",
                }}
              />
            </div>

            {/* Checkboxes */}
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                gap: spacing.sm,
              }}
            >
              <label
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: spacing.sm,
                  cursor: "pointer",
                }}
              >
                <input
                  type="checkbox"
                  checked={options.includeExecutiveSummary}
                  onChange={(e) => handleOptionChange("includeExecutiveSummary", e.target.checked)}
                  style={{ width: "18px", height: "18px" }}
                />
                <div>
                  <div style={{ fontWeight: 500 }}>
                    {t("report.executiveSummary", "Executive Summary")}
                  </div>
                  <div
                    style={{
                      fontSize: "0.875rem",
                      color: "var(--color-text-secondary)",
                    }}
                  >
                    {t(
                      "report.executiveSummaryDesc",
                      "Coverage score, key metrics, and signal distribution",
                    )}
                  </div>
                </div>
              </label>

              <label
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: spacing.sm,
                  cursor: "pointer",
                }}
              >
                <input
                  type="checkbox"
                  checked={options.includeHeatmaps}
                  onChange={(e) => handleOptionChange("includeHeatmaps", e.target.checked)}
                  style={{ width: "18px", height: "18px" }}
                />
                <div>
                  <div style={{ fontWeight: 500 }}>
                    {t("report.heatmaps", "Heatmap References")}
                  </div>
                  <div
                    style={{
                      fontSize: "0.875rem",
                      color: "var(--color-text-secondary)",
                    }}
                  >
                    {t("report.heatmapsDesc", "Note about heatmap visualization availability")}
                  </div>
                </div>
              </label>

              <label
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: spacing.sm,
                  cursor: "pointer",
                }}
              >
                <input
                  type="checkbox"
                  checked={options.includeRecommendations}
                  onChange={(e) => handleOptionChange("includeRecommendations", e.target.checked)}
                  style={{ width: "18px", height: "18px" }}
                />
                <div>
                  <div style={{ fontWeight: 500 }}>
                    {t("report.recommendations", "Recommendations")}
                  </div>
                  <div
                    style={{
                      fontSize: "0.875rem",
                      color: "var(--color-text-secondary)",
                    }}
                  >
                    {t(
                      "report.recommendationsDesc",
                      "Prioritized improvement suggestions based on analysis",
                    )}
                  </div>
                </div>
              </label>

              <label
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: spacing.sm,
                  cursor: "pointer",
                }}
              >
                <input
                  type="checkbox"
                  checked={options.includeRawData}
                  onChange={(e) => handleOptionChange("includeRawData", e.target.checked)}
                  style={{ width: "18px", height: "18px" }}
                />
                <div>
                  <div style={{ fontWeight: 500 }}>{t("report.rawData", "Raw Sample Data")}</div>
                  <div
                    style={{
                      fontSize: "0.875rem",
                      color: "var(--color-text-secondary)",
                    }}
                  >
                    {t("report.rawDataDesc", "Appendix with individual sample measurements")}
                  </div>
                </div>
              </label>
            </div>
          </div>

          {/* Error Message */}
          {error && (
            <div
              style={{
                marginTop: spacing.md,
                padding: spacing.sm,
                background: "var(--color-error-light)",
                borderRadius: radius.md,
                color: "var(--color-error)",
                fontSize: "0.875rem",
              }}
            >
              {error}
            </div>
          )}

          {/* Success Message */}
          {success && (
            <div
              style={{
                marginTop: spacing.md,
                padding: spacing.sm,
                background: "var(--color-success-light)",
                borderRadius: radius.md,
                color: "var(--color-success)",
                fontSize: "0.875rem",
                display: "flex",
                alignItems: "center",
                gap: spacing.xs,
              }}
            >
              <CheckCircle size={16} />
              {t(
                "report.downloadStarted",
                "Report generated! Download should start automatically.",
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div
          style={{
            display: "flex",
            justifyContent: "flex-end",
            gap: spacing.sm,
            padding: spacing.md,
            borderTop: "1px solid var(--color-border)",
          }}
        >
          <button
            type="button"
            onClick={onClose}
            style={{
              ...button.secondary,
              padding: `${spacing.sm} ${spacing.md}`,
            }}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            onClick={handleGenerate}
            disabled={generating}
            style={{
              ...button.primary,
              padding: `${spacing.sm} ${spacing.md}`,
              display: "flex",
              alignItems: "center",
              gap: spacing.xs,
            }}
          >
            {generating ? (
              <>
                <Loader size={16} style={{ animation: "spin 1s linear infinite" }} />
                {t("report.generating", "Generating...")}
              </>
            ) : (
              <>
                <Download size={16} />
                {t("report.generate", "Generate PDF")}
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
