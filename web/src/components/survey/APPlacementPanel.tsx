/**
 * APPlacementPanel Component
 *
 * Purpose: Manage AP location markers on the floor plan.
 * Allows adding, editing, and deleting AP locations with metadata.
 *
 * Key Features:
 * - Add AP markers by clicking on floor plan
 * - Edit AP details (label, BSSID, SSIDs, band, channel, model)
 * - Delete AP markers
 * - Select AP to highlight on floor plan
 * - Import AP locations from AirMapper
 *
 * Usage:
 * ```typescript
 * <APPlacementPanel
 *   apLocations={apLocations}
 *   onApLocationsChange={setApLocations}
 *   selectedApId={selectedApId}
 *   onApSelect={setSelectedApId}
 *   placementMode={placementMode}
 *   onPlacementModeChange={setPlacementMode}
 * />
 * ```
 */

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Radio, Plus, Trash2, Edit2, Check, X, MapPin } from "lucide-react";
import {
  cn,
  radius,
  spacing,
  layout,
  button,
  icon as iconTokens,
} from "../../styles/theme";
import type { APLocation, WiFiBand } from "../../hooks/useSurvey";

interface APPlacementPanelProps {
  apLocations: APLocation[];
  onApLocationsChange: (apLocations: APLocation[]) => void;
  selectedApId?: string;
  onApSelect: (apId: string | undefined) => void;
  placementMode: boolean;
  onPlacementModeChange: (enabled: boolean) => void;
}

/**
 * APPlacementPanel manages AP location markers on the survey floor plan
 */
export function APPlacementPanel({
  apLocations,
  onApLocationsChange,
  selectedApId,
  onApSelect,
  placementMode,
  onPlacementModeChange,
}: APPlacementPanelProps) {
  const { t } = useTranslation("survey");

  // State for editing
  const [editingApId, setEditingApId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<Partial<APLocation>>({});

  // Start editing an AP
  const handleEdit = (ap: APLocation) => {
    setEditingApId(ap.id);
    setEditForm({ ...ap });
  };

  // Save edits
  const handleSave = () => {
    if (!editingApId || !editForm.label) return;

    const updatedLocations = apLocations.map((ap) =>
      ap.id === editingApId ? { ...ap, ...editForm } : ap
    );
    onApLocationsChange(updatedLocations);
    setEditingApId(null);
    setEditForm({});
  };

  // Cancel editing
  const handleCancel = () => {
    setEditingApId(null);
    setEditForm({});
  };

  // Delete an AP
  const handleDelete = (apId: string) => {
    const updatedLocations = apLocations.filter((ap) => ap.id !== apId);
    onApLocationsChange(updatedLocations);
    if (selectedApId === apId) {
      onApSelect(undefined);
    }
  };

  // Toggle placement mode
  const togglePlacementMode = () => {
    onPlacementModeChange(!placementMode);
  };

  return (
    <div
      className={cn(
        "bg-surface-raised",
        radius.md,
        "border border-surface-border",
        spacing.pad.sm
      )}
    >
      {/* Header */}
      <div
        className={cn(
          layout.inline.default,
          "justify-between",
          spacing.margin.bottom.content
        )}
      >
        <div className={cn(layout.inline.default)}>
          <Radio className={iconTokens.size.sm} />
          <h4 className="body-small font-medium">{t("apPlacement.title")}</h4>
        </div>
        <button
          onClick={togglePlacementMode}
          className={cn(
            button.size.xs,
            radius.md,
            layout.inline.default,
            "transition-colors",
            placementMode
              ? "bg-purple-500 text-text-inverse"
              : "bg-surface-base border border-surface-border hover:bg-surface-hover"
          )}
        >
          <Plus className={iconTokens.size.xs} />
          <span>{t("apPlacement.addAp")}</span>
        </button>
      </div>

      {/* Placement mode instructions */}
      {placementMode && (
        <div
          className={cn(
            "bg-purple-500/10 border border-purple-500/20",
            radius.md,
            spacing.pad.sm,
            spacing.margin.bottom.content
          )}
        >
          <p className="caption text-purple-600 dark:text-purple-400">
            <MapPin className="w-3 h-3 inline mr-1" />
            {t("apPlacement.placementMode")}
          </p>
        </div>
      )}

      {/* AP List */}
      {apLocations.length === 0 ? (
        <p className="caption text-text-muted text-center py-4">
          {t("apPlacement.noAps")}
        </p>
      ) : (
        <div className={cn(layout.stack.tight, "max-h-64 overflow-y-auto")}>
          {apLocations.map((ap) => (
            <div
              key={ap.id}
              className={cn(
                spacing.pad.sm,
                radius.md,
                "border transition-colors cursor-pointer",
                selectedApId === ap.id
                  ? "border-purple-500 bg-purple-500/5"
                  : "border-surface-border hover:bg-surface-hover"
              )}
              onClick={() => editingApId !== ap.id && onApSelect(ap.id)}
            >
              {editingApId === ap.id ? (
                // Edit mode
                <div
                  className={cn(layout.stack.tight)}
                  onClick={(e) => e.stopPropagation()}
                >
                  {/* Label */}
                  <input
                    type="text"
                    value={editForm.label || ""}
                    onChange={(e) =>
                      setEditForm({ ...editForm, label: e.target.value })
                    }
                    placeholder={t("apPlacement.label")}
                    className={cn(
                      "w-full",
                      spacing.pad.sm,
                      radius.md,
                      "border border-surface-border bg-surface-base body-small"
                    )}
                  />

                  {/* BSSID */}
                  <input
                    type="text"
                    value={editForm.bssid || ""}
                    onChange={(e) =>
                      setEditForm({ ...editForm, bssid: e.target.value })
                    }
                    placeholder={t("apPlacement.bssid")}
                    className={cn(
                      "w-full",
                      spacing.pad.sm,
                      radius.md,
                      "border border-surface-border bg-surface-base body-small font-mono"
                    )}
                  />

                  {/* Band and Channel */}
                  <div className={cn(layout.inline.default)}>
                    <select
                      value={editForm.band || ""}
                      onChange={(e) =>
                        setEditForm({
                          ...editForm,
                          band: e.target.value as WiFiBand | undefined,
                        })
                      }
                      className={cn(
                        "flex-1",
                        spacing.pad.sm,
                        radius.md,
                        "border border-surface-border bg-surface-base body-small"
                      )}
                    >
                      <option value="">{t("apPlacement.band")}</option>
                      <option value="2.4">2.4 GHz</option>
                      <option value="5">5 GHz</option>
                      <option value="6">6 GHz</option>
                    </select>
                    <input
                      type="number"
                      value={editForm.channel || ""}
                      onChange={(e) =>
                        setEditForm({
                          ...editForm,
                          channel: e.target.value
                            ? parseInt(e.target.value, 10)
                            : undefined,
                        })
                      }
                      placeholder={t("apPlacement.channel")}
                      className={cn(
                        "w-20",
                        spacing.pad.sm,
                        radius.md,
                        "border border-surface-border bg-surface-base body-small"
                      )}
                    />
                  </div>

                  {/* Model */}
                  <input
                    type="text"
                    value={editForm.model || ""}
                    onChange={(e) =>
                      setEditForm({ ...editForm, model: e.target.value })
                    }
                    placeholder={t("apPlacement.model")}
                    className={cn(
                      "w-full",
                      spacing.pad.sm,
                      radius.md,
                      "border border-surface-border bg-surface-base body-small"
                    )}
                  />

                  {/* Actions */}
                  <div className={cn(layout.inline.default, "justify-end")}>
                    <button
                      onClick={handleCancel}
                      className={cn(
                        button.size.xs,
                        radius.md,
                        "border border-surface-border hover:bg-surface-hover",
                        layout.inline.default
                      )}
                    >
                      <X className={iconTokens.size.xs} />
                    </button>
                    <button
                      onClick={handleSave}
                      className={cn(
                        button.size.xs,
                        radius.md,
                        "bg-brand-primary text-text-inverse",
                        layout.inline.default
                      )}
                    >
                      <Check className={iconTokens.size.xs} />
                    </button>
                  </div>
                </div>
              ) : (
                // Display mode
                <div className={cn(layout.inline.default, "justify-between")}>
                  <div>
                    <div className={cn(layout.inline.default)}>
                      <Radio
                        className={cn(
                          "w-3 h-3",
                          selectedApId === ap.id
                            ? "text-purple-500"
                            : "text-text-muted"
                        )}
                      />
                      <span className="body-small font-medium">{ap.label}</span>
                    </div>
                    {ap.bssid && (
                      <p className="caption text-text-muted font-mono">
                        {ap.bssid}
                      </p>
                    )}
                    {(ap.band || ap.channel) && (
                      <p className="caption text-text-muted">
                        {ap.band && `${ap.band} GHz`}
                        {ap.band && ap.channel && " · "}
                        {ap.channel && `Ch ${ap.channel}`}
                      </p>
                    )}
                  </div>
                  <div
                    className={cn(layout.inline.tight)}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <button
                      onClick={() => handleEdit(ap)}
                      className={cn(
                        button.size.xs,
                        radius.md,
                        "hover:bg-surface-hover"
                      )}
                    >
                      <Edit2 className={iconTokens.size.xs} />
                    </button>
                    <button
                      onClick={() => handleDelete(ap.id)}
                      className={cn(
                        button.size.xs,
                        radius.md,
                        "hover:bg-status-error/10 text-status-error"
                      )}
                    >
                      <Trash2 className={iconTokens.size.xs} />
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
