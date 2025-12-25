/**
 * VLANControl Component
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
 * <VLANControl />
 * ```
 */

import { useState, memo } from "react";
import { useTranslation } from "react-i18next";
import {
  radius,
  layout,
  button,
  input,
  cn,
} from "../../../styles/theme";

const API_BASE = import.meta.env.VITE_API_BASE || "";

export const VLANControl = memo(function VLANControl() {
  const { t } = useTranslation("settings");
  const [vlanId, setVlanId] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleCreate = async () => {
    const id = parseInt(vlanId, 10);
    if (isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: t("network.vlan.invalidId"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/vlan/interface`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ vlanId: id }),
      });
      if (response.ok) {
        setMessage({ text: t("network.vlan.created", { id }), isError: false });
        setVlanId("");
      } else {
        const text = await response.text();
        setMessage({
          text: text || t("network.vlan.createFailed"),
          isError: true,
        });
      }
    } catch {
      setMessage({ text: t("network.vlan.networkError"), isError: true });
    } finally {
      setLoading(false);
      setTimeout(() => setMessage(null), 3000);
    }
  };

  const handleDelete = async () => {
    const id = parseInt(vlanId, 10);
    if (isNaN(id) || id < 1 || id > 4094) {
      setMessage({ text: t("network.vlan.invalidId"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/vlan/interface`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ vlanId: id }),
      });
      if (response.ok) {
        setMessage({ text: t("network.vlan.deleted", { id }), isError: false });
        setVlanId("");
      } else {
        const text = await response.text();
        setMessage({
          text: text || t("network.vlan.deleteFailed"),
          isError: true,
        });
      }
    } catch {
      setMessage({ text: t("network.vlan.networkError"), isError: true });
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
            "body-small text-text-primary"
          )}
          disabled={loading}
        />
        <button
          onClick={handleCreate}
          disabled={loading || !vlanId}
          className={cn(
            button.size.sm,
            "bg-brand-primary text-text-inverse",
            radius.md,
            "body-small font-medium hover:bg-brand-accent disabled:opacity-50"
          )}
        >
          {t("network.vlan.add")}
        </button>
        <button
          onClick={handleDelete}
          disabled={loading || !vlanId}
          className={cn(
            button.size.sm,
            "bg-status-error text-text-inverse",
            radius.md,
            "body-small font-medium hover:opacity-80 disabled:opacity-50"
          )}
        >
          {t("network.vlan.remove")}
        </button>
      </div>
      {message && (
        <p
          className={cn(
            "caption",
            message.isError ? "text-status-error" : "text-status-success"
          )}
        >
          {message.text}
        </p>
      )}
      <p className="caption">{t("network.vlan.description")}</p>
    </div>
  );
});
