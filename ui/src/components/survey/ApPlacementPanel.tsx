// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * ApPlacementPanel Component
 *
 * Purpose: Manage AP location markers on the floor plan.
 * Allows adding, editing, and deleting AP locations with metadata.
 *
 * Key Features:
 * - Add AP markers by clicking on floor plan
 * - Edit AP details (label, Bssid, Ssids, band, channel, model)
 * - Delete AP markers
 * - Select AP to highlight on floor plan
 * - Import AP locations from AirMapper
 *
 * Usage:
 * ```typescript
 * <ApPlacementPanel
 *   apLocations={apLocations}
 *   onApLocationsChange={setApLocations}
 *   selectedApId={selectedApId}
 *   onApSelect={setSelectedApId}
 *   placementMode={placementMode}
 *   onPlacementModeChange={setPlacementMode}
 * />
 * ```
 */

import { Check, Edit2, MapPin, Plus, Radio, Trash2, X } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { ApLocation, WiFiBand } from "../../hooks/useSurvey";
import { button, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";

interface ApPlacementPanelProps {
  apLocations: ApLocation[];
  onApLocationsChange: (apLocations: ApLocation[]) => void;
  selectedApId?: string;
  onApSelect: (apId: string | undefined) => void;
  placementMode: boolean;
  onPlacementModeChange: (enabled: boolean) => void;
}

/**
 * ApPlacementPanel manages AP location markers on the survey floor plan
 */
export function ApPlacementPanel({
  apLocations,
  onApLocationsChange,
  selectedApId,
  onApSelect,
  placementMode,
  onPlacementModeChange,
}: ApPlacementPanelProps): React.ReactElement {
  const { t } = useTranslation("survey");

  // State for editing
  const [editingApId, setEditingApId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<Partial<ApLocation>>({});

  // Start editing an AP
  const handleEdit = (ap: ApLocation): void => {
    setEditingApId(ap.id);
    setEditForm({ ...ap });
  };

  // Save edits
  const handleSave = (): void => {
    if (!(editingApId && editForm.label)) {
      return;
    }

    const updatedLocations = apLocations.map((ap) =>
      ap.id === editingApId ? { ...ap, ...editForm } : ap,
    );
    onApLocationsChange(updatedLocations);
    setEditingApId(null);
    setEditForm({});
  };

  // Cancel editing
  const handleCancel = (): void => {
    setEditingApId(null);
    setEditForm({});
  };

  // Delete an AP
  const handleDelete = (apId: string): void => {
    const updatedLocations = apLocations.filter((ap) => ap.id !== apId);
    onApLocationsChange(updatedLocations);
    if (selectedApId === apId) {
      onApSelect(undefined);
    }
  };

  // Toggle placement mode
  const togglePlacementMode = (): void => {
    onPlacementModeChange(!placementMode);
  };

  return (
    <div class={cn("bg-surface-raised", radius.md, "border border-surface-border", spacing.pad.sm)}>
      {/* Header */}
      <div class={cn(layout.inline.default, "justify-between", spacing.margin.bottom.content)}>
        <div class={cn(layout.inline.default)}>
          <Radio class={iconTokens.size.sm} />
          <h4 class="body-small font-medium">{t("apPlacement.title")}</h4>
        </div>
        <button
          type="button"
          onClick={togglePlacementMode}
          class={cn(
            button.size.xs,
            radius.md,
            layout.inline.default,
            "transition-colors",
            placementMode
              ? "bg-purple-500 text-text-inverse"
              : "bg-surface-base border border-surface-border hover:bg-surface-hover",
          )}
        >
          <Plus class={iconTokens.size.xs} />
          <span>{t("apPlacement.addAp")}</span>
        </button>
      </div>

      {/* Placement mode instructions */}
      {placementMode ? (
        <div
          class={cn(
            "bg-purple-500/10 border border-purple-500/20",
            radius.md,
            spacing.pad.sm,
            spacing.margin.bottom.content,
          )}
        >
          <p class="caption text-purple-600 dark:text-purple-400">
            <MapPin class="w-3 h-3 inline mr-1" />
            {t("apPlacement.placementMode")}
          </p>
        </div>
      ) : null}

      {/* AP List */}
      {apLocations.length === 0 ? (
        <p class="caption text-text-muted text-center py-4">{t("apPlacement.noAps")}</p>
      ) : (
        <div class={cn(layout.stack.tight, "max-h-64 overflow-y-auto")}>
          {apLocations.map((ap) => (
            // biome-ignore lint/a11y/useSemanticElements: Complex card with nested interactive elements
            <div
              key={ap.id}
              class={cn(
                spacing.pad.sm,
                radius.md,
                "border transition-colors cursor-pointer",
                selectedApId === ap.id
                  ? "border-purple-500 bg-purple-500/5"
                  : "border-surface-border hover:bg-surface-hover",
              )}
              onClick={(): void => {
                if (editingApId !== ap.id) {
                  onApSelect(ap.id);
                }
              }}
              onKeyDown={(e: React.KeyboardEvent): void => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  if (editingApId !== ap.id) {
                    onApSelect(ap.id);
                  }
                }
              }}
              role="button"
              tabIndex={0}
            >
              {editingApId === ap.id ? (
                <div
                  class={cn(layout.stack.tight)}
                  onClick={(e: React.MouseEvent): void => e.stopPropagation()}
                  aria-hidden="true"
                >
                  {/* Label */}
                  <input
                    type="text"
                    value={editForm.label || ""}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setEditForm({ ...editForm, label: e.target.value })
                    }
                    placeholder={t("apPlacement.label")}
                    class={cn(
                      "w-full",
                      spacing.pad.sm,
                      radius.md,
                      "border border-surface-border bg-surface-base body-small",
                    )}
                  />

                  {/* BSSID */}
                  <input
                    type="text"
                    value={editForm.bssid || ""}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setEditForm({ ...editForm, bssid: e.target.value })
                    }
                    placeholder={t("apPlacement.bssid")}
                    class={cn(
                      "w-full",
                      spacing.pad.sm,
                      radius.md,
                      "border border-surface-border bg-surface-base body-small font-mono",
                    )}
                  />

                  {/* Band and Channel */}
                  <div class={cn(layout.inline.default)}>
                    <select
                      value={editForm.band || ""}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void =>
                        setEditForm({
                          ...editForm,
                          band: e.target.value as WiFiBand | undefined,
                        })
                      }
                      class={cn(
                        "flex-1",
                        spacing.pad.sm,
                        radius.md,
                        "border border-surface-border bg-surface-base body-small",
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
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void =>
                        setEditForm({
                          ...editForm,
                          channel: e.target.value ? Number.parseInt(e.target.value, 10) : undefined,
                        })
                      }
                      placeholder={t("apPlacement.channel")}
                      class={cn(
                        "w-20",
                        spacing.pad.sm,
                        radius.md,
                        "border border-surface-border bg-surface-base body-small",
                      )}
                    />
                  </div>

                  {/* Model */}
                  <input
                    type="text"
                    value={editForm.model || ""}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setEditForm({ ...editForm, model: e.target.value })
                    }
                    placeholder={t("apPlacement.model")}
                    class={cn(
                      "w-full",
                      spacing.pad.sm,
                      radius.md,
                      "border border-surface-border bg-surface-base body-small",
                    )}
                  />

                  {/* Actions */}
                  <div class={cn(layout.inline.default, "justify-end")}>
                    <button
                      type="button"
                      onClick={handleCancel}
                      class={cn(
                        button.size.xs,
                        radius.md,
                        "border border-surface-border hover:bg-surface-hover",
                        layout.inline.default,
                      )}
                    >
                      <X class={iconTokens.size.xs} />
                    </button>
                    <button
                      type="button"
                      onClick={handleSave}
                      class={cn(
                        button.size.xs,
                        radius.md,
                        "bg-brand-primary text-text-inverse",
                        layout.inline.default,
                      )}
                    >
                      <Check class={iconTokens.size.xs} />
                    </button>
                  </div>
                </div>
              ) : (
                // Display mode
                <div class={cn(layout.inline.default, "justify-between")}>
                  <div>
                    <div class={cn(layout.inline.default)}>
                      <Radio
                        class={cn(
                          "w-3 h-3",
                          selectedApId === ap.id ? "text-purple-500" : "text-text-muted",
                        )}
                      />
                      <span class="body-small font-medium">{ap.label}</span>
                    </div>
                    {ap.bssid ? <p class="caption text-text-muted font-mono">{ap.bssid}</p> : null}
                    {ap.band || ap.channel ? (
                      <p class="caption text-text-muted">
                        {ap.band ? `${ap.band} GHz` : null}
                        {ap.band && ap.channel ? " · " : null}
                        {ap.channel ? `Ch ${ap.channel}` : null}
                      </p>
                    ) : null}
                  </div>
                  {/* biome-ignore lint/a11y/noStaticElementInteractions: Used to stop propagation to parent button */}
                  <div
                    class={cn(layout.inline.tight)}
                    onClick={(e: React.MouseEvent): void => e.stopPropagation()}
                    onKeyDown={(e: React.KeyboardEvent): void => e.stopPropagation()}
                  >
                    <button
                      type="button"
                      onClick={(): void => handleEdit(ap)}
                      class={cn(button.size.xs, radius.md, "hover:bg-surface-hover")}
                    >
                      <Edit2 class={iconTokens.size.xs} />
                    </button>
                    <button
                      type="button"
                      onClick={(): void => handleDelete(ap.id)}
                      class={cn(
                        button.size.xs,
                        radius.md,
                        "hover:bg-status-error/10 text-status-error",
                      )}
                    >
                      <Trash2 class={iconTokens.size.xs} />
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
