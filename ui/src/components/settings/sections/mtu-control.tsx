/**
 * MtuControl Component
 *
 * Purpose: Provides controls for setting interface MTU (Maximum Transmission Unit).
 * Split from SettingsDrawer.tsx for code organization (Plan F).
 *
 * Key Features:
 * - Set MTU with validation (68-9000 bytes)
 * - Visual feedback for success/error states
 * - Loading state during API calls
 * - Common presets: Standard (1500), Jumbo frames (9000)
 *
 * Usage:
 * ```typescript
 * <MtuControl />
 * ```
 */

import { memo, useState } from "react";
import { useTranslation } from "react-i18next";
import { button, cn, input, layout, radius } from "../../../styles/theme";

const API_BASE = import.meta.env.VITE_API_BASE || "";

export const MtuControl = memo(function MtuControl() {
  const { t } = useTranslation("settings");
  const [mtu, setMtu] = useState("1500");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{
    text: string;
    isError: boolean;
  } | null>(null);

  const handleApply = async () => {
    const mtuVal = Number.parseInt(mtu, 10);
    if (Number.isNaN(mtuVal) || mtuVal < 68 || mtuVal > 9000) {
      setMessage({ text: t("network.mtuControl.invalidRange"), isError: true });
      return;
    }
    setLoading(true);
    setMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/network/mtu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ mtu: mtuVal }),
      });
      if (response.ok) {
        setMessage({
          text: t("network.mtuControl.setSuccess", { value: mtuVal }),
          isError: false,
        });
      } else {
        const text = await response.text();
        setMessage({
          text: text || t("network.mtuControl.setFailed"),
          isError: true,
        });
      }
    } catch {
      setMessage({ text: t("network.mtuControl.networkError"), isError: true });
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
          min="68"
          max="9000"
          value={mtu}
          onChange={(e) => setMtu(e.target.value)}
          placeholder={t("network.mtuControl.placeholder")}
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
          onClick={handleApply}
          disabled={loading}
          className={cn(
            button.size.md,
            "bg-brand-primary text-text-inverse",
            radius.md,
            "body-small font-medium hover:bg-brand-accent disabled:opacity-50",
          )}
        >
          {loading ? t("network.applying") : t("network.mtuControl.apply")}
        </button>
      </div>
      {message && (
        <p className={cn("caption", message.isError ? "text-status-error" : "text-status-success")}>
          {message.text}
        </p>
      )}
      <p className="caption">{t("network.mtuControl.description")}</p>
    </div>
  );
});
