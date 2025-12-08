import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";

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

export function PublicIPCard({ data, loading }: PublicIPCardProps) {
  if (loading) {
    return (
      <Card title="Public IP" status="loading">
        <CardValue value="Checking..." size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title="Public IP" status="unknown">
        <CardValue value="No data" size="md" />
        <p className="text-xs text-text-muted mt-1">
          Unable to detect public IP
        </p>
      </Card>
    );
  }

  // Determine status
  let status: Status = "unknown";
  if (data.error && !data.ipv4 && !data.ipv6) {
    status = "error";
  } else if (data.ipv4 || data.ipv6) {
    status = "success";
  }

  return (
    <Card title="Public IP" status={status}>
      {/* IPv4 Address */}
      {data.ipv4 ? (
        <>
          <p className="text-xs text-text-muted font-medium">IPv4</p>
          <CardValue value={data.ipv4} size="lg" />
        </>
      ) : (
        <>
          <p className="text-xs text-text-muted font-medium">IPv4</p>
          <p className="text-sm text-text-muted">Not available</p>
        </>
      )}

      <CardDivider />

      {/* IPv6 Address */}
      {data.ipv6 ? (
        <>
          <p className="text-xs text-text-muted font-medium">IPv6</p>
          <p className="text-sm font-mono break-all text-text-primary">
            {data.ipv6}
          </p>
        </>
      ) : (
        <>
          <p className="text-xs text-text-muted font-medium">IPv6</p>
          <p className="text-sm text-text-muted">Not available</p>
        </>
      )}

      {/* Last checked */}
      {data.lastChecked && (
        <>
          <CardDivider />
          <CardRow
            label="Last checked"
            value={formatLastChecked(data.lastChecked)}
          />
        </>
      )}

      {/* Error if any */}
      {data.error && (
        <>
          <CardDivider />
          <p className="text-xs text-status-error">{data.error}</p>
        </>
      )}
    </Card>
  );
}
