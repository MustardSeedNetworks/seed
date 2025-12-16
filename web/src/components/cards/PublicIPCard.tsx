/**
 * PublicIPCard Component
 *
 * Purpose: Displays the public IPv4 and IPv6 addresses as seen from the internet.
 * Shows when these addresses were last checked and any lookup errors.
 *
 * Key Features:
 * - Dual-stack support: shows both IPv4 and IPv6 public addresses
 * - Timestamp: displays when addresses were last verified (e.g., "5m ago", "just now")
 * - Error handling: shows error message if lookup fails
 * - Status indication: success (has IP), error (lookup failed), unknown (no data)
 *
 * Usage:
 * ```typescript
 * <PublicIPCard
 *   data={publicIPData}
 *   loading={isFetching}
 * />
 * ```
 *
 * Dependencies: BaseCard, Card UI components, Globe icon, theme utilities
 * State: Receives data from parent component via props
 */

import { memo } from "react";
import { useTranslation } from "react-i18next";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { BaseCard } from "./BaseCard";
import { Globe } from "../ui/Icons";
import { icon as iconTokens } from "../../styles/theme";

export interface PublicIPData {
  ipv4?: string;
  ipv6?: string;
  lastChecked: string;
  error?: string;
}

interface PublicIPCardProps {
  data: PublicIPData | null;
  loading?: boolean;
}

function formatLastChecked(isoDate: string): string {
  try {
    const date = new Date(isoDate);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return "just now";
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    return date.toLocaleDateString();
  } catch {
    return "unknown";
  }
}

function getStatus(data: PublicIPData): Status {
  if (data.error && !data.ipv4 && !data.ipv6) return "error";
  if (data.ipv4 || data.ipv6) return "success";
  return "unknown";
}

export const PublicIPCard = memo(function PublicIPCard({ data, loading }: PublicIPCardProps) {
  const { t } = useTranslation("cards");

  return (
    <BaseCard
      title={t("publicIp.title")}
      icon={<Globe className={iconTokens.size.md} />}
      data={data}
      loading={loading}
      getStatus={getStatus}
      loadingContent={<CardValue value={t("publicIp.checking")} size="lg" />}
      emptyMessage={t("publicIp.unableToDetect")}
    >
      {(ipData) => (
        <>
          {/* IPv4 Address */}
          {ipData.ipv4 ? (
            <>
              <p className="caption font-medium">{t("publicIp.ipv4")}</p>
              <CardValue value={ipData.ipv4} size="lg" />
            </>
          ) : (
            <>
              <p className="caption font-medium">{t("publicIp.ipv4")}</p>
              <p className="body-small text-text-muted">{t("publicIp.notAvailable")}</p>
            </>
          )}

          <CardDivider />

          {/* IPv6 Address */}
          {ipData.ipv6 ? (
            <>
              <p className="caption font-medium">{t("publicIp.ipv6")}</p>
              <p className="body-small font-mono break-all text-text-primary">{ipData.ipv6}</p>
            </>
          ) : (
            <>
              <p className="caption font-medium">{t("publicIp.ipv6")}</p>
              <p className="body-small text-text-muted">{t("publicIp.notAvailable")}</p>
            </>
          )}

          {/* Last checked */}
          {ipData.lastChecked && (
            <>
              <CardDivider />
              <CardRow
                label={t("publicIp.lastChecked")}
                value={formatLastChecked(ipData.lastChecked)}
              />
            </>
          )}

          {/* Error if any */}
          {ipData.error && (
            <>
              <CardDivider />
              <p className="caption text-status-error">{ipData.error}</p>
            </>
          )}
        </>
      )}
    </BaseCard>
  );
});
