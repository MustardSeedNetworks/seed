/**
 * VlanControl Component
 *
 * Purpose: Provides controls for creating and deleting 802.1Q VLAN subinterfaces.
 * Split from SettingsDrawer.tsx for code organization (Plan F).
 *
 * Key Features:
 * - Create VLAN subinterfaces with validation (1-4094)
 * - Delete VLAN subinterfaces
 * - Visual feedback for success/error states
 * - Loading state during API calls
 *
 * Usage:
 * ```typescript
 * <VlanControl />
 * ```
 */

import { memo, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../../lib/api";
import { button, cn, input, layout, radius } from "../../../styles/theme";

export const VlanControl = memo(function VlanControl() {
  const { t } = useTranslation("settings");
  const [vlanId, setVlanId] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleCreate = async () => {
    const id = Number.parseInt(vlanId, 10);
    if (Number.isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: t("network.vlan.invalidId"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      await api.post("/api/vlan/interface", { vlanId: id });
      setMessage({ text: t("network.vlan.created", { id }), isError: false });
      setVlanId("");
    } catch {
      setMessage({ text: t("network.vlan.createFailed"), isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  const handleDelete = async () => {
    const id = Number.parseInt(vlanId, 10);
    if (Number.isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: t("network.vlan.invalidId"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      await api.delete("/api/vlan/interface", {
        body: JSON.stringify({ vlanId: id }),
        headers: { "Content-Type": "application/json" },
      });
      setMessage({ text: t("network.vlan.deleted", { id }), isError: false });
      setVlanId("");
    } catch {
      setMessage({ text: t("network.vlan.deleteFailed"), isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  return (
    <div className="stack-sm">
      <div className={layout.inline.default}>
        <input
          type="number"
          min="1"
          max="4094"
          value={vlanId}
          onChange={(e) => setVlanId(e.target.value)}
          placeholder={t("network.vlan.placeholder")}
          className={cn(
            "flex-1",
            input.size.sm,
            "bg-surface-base border border-surface-border",
            radius.md,
            "body-small text-text-primary",
          )}
          disabled={loading}
        />
        <button
          type="button"
          onClick={handleCreate}
          disabled={loading || !vlanId}
          className={cn(
            button.size.sm,
            "bg-brand-primary text-text-inverse",
            radius.md,
            "body-small font-medium hover:bg-brand-accent disabled:opacity-50",
          )}
        >
          {t("network.vlan.add")}
        </button>
        <button
          type="button"
          onClick={handleDelete}
          disabled={loading || !vlanId}
          className={cn(
            button.size.sm,
            "bg-status-error text-text-inverse",
            radius.md,
            "body-small font-medium hover:opacity-80 disabled:opacity-50",
          )}
        >
          {t("network.vlan.remove")}
        </button>
      </div>
      {message && (
        <p className={cn("caption", message.isError ? "text-status-error" : "text-status-success")}>
          {message.text}
        </p>
      )}
      <p className="caption">{t("network.vlan.description")}</p>
    </div>
  );
});
