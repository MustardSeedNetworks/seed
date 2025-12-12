import { memo } from "react";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { BaseCard } from "./BaseCard";

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

export const PublicIPCard = memo(function PublicIPCard({
  data,
  loading,
}: PublicIPCardProps) {
  return (
    <BaseCard
      title="Public IP"
      data={data}
      loading={loading}
      getStatus={getStatus}
      loadingContent={<CardValue value="Checking..." size="lg" />}
      emptyMessage="Unable to detect public IP"
    >
      {(ipData) => (
        <>
          {/* IPv4 Address */}
          {ipData.ipv4 ? (
            <>
              <p className="text-xs text-text-muted font-medium">IPv4</p>
              <CardValue value={ipData.ipv4} size="lg" />
            </>
          ) : (
            <>
              <p className="text-xs text-text-muted font-medium">IPv4</p>
              <p className="text-sm text-text-muted">Not available</p>
            </>
          )}

          <CardDivider />

          {/* IPv6 Address */}
          {ipData.ipv6 ? (
            <>
              <p className="text-xs text-text-muted font-medium">IPv6</p>
              <p className="text-sm font-mono break-all text-text-primary">
                {ipData.ipv6}
              </p>
            </>
          ) : (
            <>
              <p className="text-xs text-text-muted font-medium">IPv6</p>
              <p className="text-sm text-text-muted">Not available</p>
            </>
          )}

          {/* Last checked */}
          {ipData.lastChecked && (
            <>
              <CardDivider />
              <CardRow
                label="Last checked"
                value={formatLastChecked(ipData.lastChecked)}
              />
            </>
          )}

          {/* Error if any */}
          {ipData.error && (
            <>
              <CardDivider />
              <p className="text-xs text-status-error">{ipData.error}</p>
            </>
          )}
        </>
      )}
    </BaseCard>
  );
});
