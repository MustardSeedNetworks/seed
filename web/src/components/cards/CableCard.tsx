import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";

export interface CableData {
  supported: boolean;
  length: number | null; // meters
  status: "ok" | "open" | "short" | "impedance_mismatch" | "unknown";
  faults: string[];
}

interface CableCardProps {
  data: CableData | null;
  loading?: boolean;
}

const statusLabels: Record<string, { label: string; status: Status }> = {
  ok: { label: "OK", status: "success" },
  open: { label: "Open", status: "error" },
  short: { label: "Short", status: "error" },
  impedance_mismatch: { label: "Impedance Mismatch", status: "warning" },
  unknown: { label: "Unknown", status: "unknown" },
};

export function CableCard({ data, loading }: CableCardProps) {
  if (loading) {
    return (
      <Card title="Cable Test" status="loading">
        <CardValue value="Testing..." size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title="Cable Test" status="unknown">
        <CardValue value="No data" size="md" />
      </Card>
    );
  }

  if (!data.supported) {
    return (
      <Card title="Cable Test" status="unknown">
        <CardValue value="Not Supported" size="md" />
        <p className="text-xs text-text-muted mt-2">
          This NIC does not support TDR cable testing.
        </p>
      </Card>
    );
  }

  const statusInfo = statusLabels[data.status] || statusLabels.unknown;

  return (
    <Card title="Cable Test" status={statusInfo.status}>
      <CardValue
        value={statusInfo.label}
        size="lg"
        status={statusInfo.status}
      />
      {data.length !== null && (
        <>
          <CardDivider />
          <CardRow label="Length" value={`${data.length}m`} />
        </>
      )}
      {data.faults.length > 0 && (
        <>
          <CardDivider />
          <p className="text-xs text-text-muted mb-1">Faults</p>
          <ul className="text-sm text-status-error">
            {data.faults.map((fault, index) => (
              <li key={index}>• {fault}</li>
            ))}
          </ul>
        </>
      )}
    </Card>
  );
}
