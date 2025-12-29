/**
 * FloorSelector Component
 *
 * Provides a vertical tab list for navigating between floors in a multi-floor survey.
 * Allows users to add, delete, and rename floors.
 *
 * Features:
 * - Vertical floor list with level indicators
 * - Add/delete floor buttons
 * - Active floor highlighting
 * - Floor plan status indicators (uploaded vs pending)
 *
 * Usage:
 * ```typescript
 * <FloorSelector
 *   floors={survey.floors}
 *   activeFloorId={survey.activeFloorId}
 *   onSelectFloor={(floorId) => handleSelectFloor(floorId)}
 *   onAddFloor={(name, level) => handleAddFloor(name, level)}
 *   onDeleteFloor={(floorId) => handleDeleteFloor(floorId)}
 *   onRenameFloor={(floorId, name) => handleRenameFloor(floorId, name)}
 * />
 * ```
 */

import { AlertCircle, Check, Edit2, Image, Layers, Plus, Trash2, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Floor } from "../../hooks/useSurvey";
import { button, radius, spacing } from "../../styles/theme";

interface FloorSelectorProps {
  floors: Floor[];
  activeFloorId?: string;
  onSelectFloor: (floorId: string) => void;
  onAddFloor: (name: string, level: number) => Promise<void>;
  onDeleteFloor: (floorId: string) => Promise<void>;
  onRenameFloor?: (floorId: string, name: string, level: number) => Promise<void>;
  disabled?: boolean;
}

/**
 * Vertical floor navigation with add/edit/delete controls.
 */
export function FloorSelector({
  floors,
  activeFloorId,
  onSelectFloor,
  onAddFloor,
  onDeleteFloor,
  onRenameFloor,
  disabled = false,
}: FloorSelectorProps) {
  /**
   * Renders the vertical floor navigation and inline CRUD controls.
   */
  const { t } = useTranslation("survey");
  const [isAdding, setIsAdding] = useState(false);
  const [editingFloorId, setEditingFloorId] = useState<string | null>(null);
  const [newFloorName, setNewFloorName] = useState("");
  const [newFloorLevel, setNewFloorLevel] = useState(1);
  const [editName, setEditName] = useState("");
  const [editLevel, setEditLevel] = useState(0);
  const [loading, setLoading] = useState(false);

  // Sort floors by level
  const sortedFloors = [...floors].sort((a, b) => a.level - b.level);

  // Suggest next level based on existing floors
  const suggestNextLevel = (): number => {
    if (floors.length === 0) return 1;
    const maxLevel = Math.max(...floors.map((f) => f.level));
    return maxLevel + 1;
  };

  const handleStartAdd = () => {
    const nextLevel = suggestNextLevel();
    setNewFloorLevel(nextLevel);
    setNewFloorName(`Floor ${nextLevel}`);
    setIsAdding(true);
  };

  const handleCancelAdd = () => {
    setIsAdding(false);
    setNewFloorName("");
    setNewFloorLevel(1);
  };

  const handleConfirmAdd = async () => {
    if (!newFloorName.trim()) return;

    setLoading(true);
    try {
      await onAddFloor(newFloorName.trim(), newFloorLevel);
      handleCancelAdd();
    } catch (err) {
      console.error("Failed to add floor:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleStartEdit = (floor: Floor) => {
    setEditingFloorId(floor.id);
    setEditName(floor.name);
    setEditLevel(floor.level);
  };

  const handleCancelEdit = () => {
    setEditingFloorId(null);
    setEditName("");
    setEditLevel(0);
  };

  const handleConfirmEdit = async () => {
    if (!editingFloorId || !editName.trim() || !onRenameFloor) return;

    setLoading(true);
    try {
      await onRenameFloor(editingFloorId, editName.trim(), editLevel);
      handleCancelEdit();
    } catch (err) {
      console.error("Failed to rename floor:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (floorId: string) => {
    if (floors.length <= 1) {
      alert(t("floorSelector.cannotDeleteLastFloor", "Cannot delete the last floor"));
      return;
    }

    if (!confirm(t("floorSelector.confirmDelete", "Are you sure you want to delete this floor?"))) {
      return;
    }

    setLoading(true);
    try {
      await onDeleteFloor(floorId);
    } catch (err) {
      console.error("Failed to delete floor:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: spacing.sm,
        padding: spacing.md,
        background: "var(--color-background-secondary)",
        borderRadius: radius.lg,
        minWidth: "200px",
      }}
    >
      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: spacing.xs,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: spacing.xs }}>
          <Layers size={16} />
          <span style={{ fontWeight: 600 }}>{t("floorSelector.floors", "Floors")}</span>
        </div>
        <button
          type="button"
          onClick={handleStartAdd}
          disabled={disabled || loading || isAdding}
          style={{
            ...button.icon,
            padding: "4px",
            borderRadius: radius.sm,
          }}
          title={t("floorSelector.addFloor", "Add Floor")}
        >
          <Plus size={16} />
        </button>
      </div>

      {/* Add Floor Form */}
      {isAdding && (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: spacing.xs,
            padding: spacing.sm,
            background: "var(--color-background)",
            borderRadius: radius.md,
            border: "1px dashed var(--color-border)",
          }}
        >
          <input
            type="text"
            value={newFloorName}
            onChange={(e) => setNewFloorName(e.target.value)}
            placeholder={t("floorSelector.floorName", "Floor name")}
            style={{
              padding: spacing.xs,
              borderRadius: radius.sm,
              border: "1px solid var(--color-border)",
              background: "var(--color-background)",
              color: "var(--color-text)",
            }}
          />
          <div style={{ display: "flex", alignItems: "center", gap: spacing.xs }}>
            <span style={{ fontSize: "0.875rem" }}>{t("floorSelector.level", "Level")}:</span>
            <input
              type="number"
              value={newFloorLevel}
              onChange={(e) => setNewFloorLevel(Number.parseInt(e.target.value, 10) || 0)}
              style={{
                width: "60px",
                padding: spacing.xs,
                borderRadius: radius.sm,
                border: "1px solid var(--color-border)",
                background: "var(--color-background)",
                color: "var(--color-text)",
              }}
            />
          </div>
          <div style={{ display: "flex", gap: spacing.xs }}>
            <button
              type="button"
              onClick={handleConfirmAdd}
              disabled={loading || !newFloorName.trim()}
              style={{
                ...button.primary,
                flex: 1,
                padding: spacing.xs,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: spacing.xs,
              }}
            >
              <Check size={14} />
              {t("common.add", "Add")}
            </button>
            <button
              type="button"
              onClick={handleCancelAdd}
              disabled={loading}
              style={{
                ...button.secondary,
                flex: 1,
                padding: spacing.xs,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: spacing.xs,
              }}
            >
              <X size={14} />
              {t("common.cancel", "Cancel")}
            </button>
          </div>
        </div>
      )}

      {/* Floor List */}
      <div style={{ display: "flex", flexDirection: "column", gap: spacing.xs }}>
        {sortedFloors.map((floor) => {
          const isActive = floor.id === activeFloorId;
          const isEditing = floor.id === editingFloorId;
          const hasFloorPlan = !!floor.floorPlan;
          const sampleCount = floor.samples?.length || 0;

          if (isEditing && onRenameFloor) {
            return (
              <div
                key={floor.id}
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: spacing.xs,
                  padding: spacing.sm,
                  background: "var(--color-background)",
                  borderRadius: radius.md,
                  border: "1px solid var(--color-primary)",
                }}
              >
                <input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  style={{
                    padding: spacing.xs,
                    borderRadius: radius.sm,
                    border: "1px solid var(--color-border)",
                    background: "var(--color-background)",
                    color: "var(--color-text)",
                  }}
                />
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: spacing.xs,
                  }}
                >
                  <span style={{ fontSize: "0.875rem" }}>{t("floorSelector.level", "Level")}:</span>
                  <input
                    type="number"
                    value={editLevel}
                    onChange={(e) => setEditLevel(Number.parseInt(e.target.value, 10) || 0)}
                    style={{
                      width: "60px",
                      padding: spacing.xs,
                      borderRadius: radius.sm,
                      border: "1px solid var(--color-border)",
                      background: "var(--color-background)",
                      color: "var(--color-text)",
                    }}
                  />
                </div>
                <div style={{ display: "flex", gap: spacing.xs }}>
                  <button
                    type="button"
                    onClick={handleConfirmEdit}
                    disabled={loading || !editName.trim()}
                    style={{
                      ...button.primary,
                      flex: 1,
                      padding: spacing.xs,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      gap: spacing.xs,
                    }}
                  >
                    <Check size={14} />
                  </button>
                  <button
                    type="button"
                    onClick={handleCancelEdit}
                    disabled={loading}
                    style={{
                      ...button.secondary,
                      flex: 1,
                      padding: spacing.xs,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      gap: spacing.xs,
                    }}
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>
            );
          }

          return (
            <div
              key={floor.id}
              onClick={() => !disabled && onSelectFloor(floor.id)}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                padding: spacing.sm,
                background: isActive ? "var(--color-primary-light)" : "var(--color-background)",
                borderRadius: radius.md,
                border: isActive
                  ? "1px solid var(--color-primary)"
                  : "1px solid var(--color-border)",
                cursor: disabled ? "default" : "pointer",
                opacity: disabled ? 0.6 : 1,
                transition: "all 0.15s ease",
              }}
            >
              {/* Floor Info */}
              <div style={{ display: "flex", flexDirection: "column", gap: "2px" }}>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: spacing.xs,
                  }}
                >
                  <span
                    style={{
                      fontSize: "0.75rem",
                      fontWeight: 600,
                      background: "var(--color-background-secondary)",
                      padding: "2px 6px",
                      borderRadius: radius.sm,
                    }}
                  >
                    L{floor.level}
                  </span>
                  <span style={{ fontWeight: isActive ? 600 : 400 }}>{floor.name}</span>
                </div>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: spacing.sm,
                    fontSize: "0.75rem",
                    color: "var(--color-text-secondary)",
                  }}
                >
                  {hasFloorPlan ? (
                    <span
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: "2px",
                      }}
                    >
                      <Image size={12} />
                      {t("floorSelector.hasFloorPlan", "Plan")}
                    </span>
                  ) : (
                    <span
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: "2px",
                        color: "var(--color-warning)",
                      }}
                    >
                      <AlertCircle size={12} />
                      {t("floorSelector.noFloorPlan", "No plan")}
                    </span>
                  )}
                  <span>
                    {sampleCount} {t("floorSelector.samples", "samples")}
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div
                style={{
                  display: "flex",
                  gap: "4px",
                }}
                onClick={(e) => e.stopPropagation()}
              >
                {onRenameFloor && (
                  <button
                    type="button"
                    onClick={() => handleStartEdit(floor)}
                    disabled={disabled || loading}
                    style={{
                      ...button.icon,
                      padding: "4px",
                      borderRadius: radius.sm,
                    }}
                    title={t("floorSelector.editFloor", "Edit Floor")}
                  >
                    <Edit2 size={14} />
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => handleDelete(floor.id)}
                  disabled={disabled || loading || floors.length <= 1}
                  style={{
                    ...button.icon,
                    padding: "4px",
                    borderRadius: radius.sm,
                    color: floors.length <= 1 ? "var(--color-text-disabled)" : "var(--color-error)",
                  }}
                  title={
                    floors.length <= 1
                      ? t("floorSelector.cannotDeleteLastFloor", "Cannot delete the last floor")
                      : t("floorSelector.deleteFloor", "Delete Floor")
                  }
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Empty State */}
      {floors.length === 0 && !isAdding && (
        <div
          style={{
            textAlign: "center",
            padding: spacing.md,
            color: "var(--color-text-secondary)",
            fontSize: "0.875rem",
          }}
        >
          {t("floorSelector.noFloors", "No floors yet. Click + to add one.")}
        </div>
      )}
    </div>
  );
}
