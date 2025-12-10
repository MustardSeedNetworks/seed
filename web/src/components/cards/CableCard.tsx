import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";

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

function getCardStatus(data: CableData | null): Status {
  if (!data || !data.supported) return "unknown";
  return statusLabels[data.status]?.status || "unknown";
}

export function CableCard({ data, loading }: CableCardProps) {
  return (
    <SimpleBaseCard
      title="Cable Test"
      status={loading ? "loading" : getCardStatus(data)}
      loading={loading}
      loadingContent={<CardValue value="Testing..." size="lg" />}
    >
      {!data ? (
        <CardValue value="No data" size="md" />
      ) : !data.supported ? (
        <>
          <CardValue value="Not Supported" size="md" />
          <p className="text-xs text-text-muted mt-2">
            This NIC does not support TDR cable testing.
          </p>
        </>
      ) : (
        <>
          <CardValue
            value={statusLabels[data.status]?.label || "Unknown"}
            size="lg"
            status={statusLabels[data.status]?.status || "unknown"}
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
        </>
      )}
    </SimpleBaseCard>
  );
}
