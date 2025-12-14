/**
 * CableCard Component
 *
 * Purpose: Displays Ethernet cable test results using Time Domain Reflectometry (TDR).
 * Shows cable condition (OK, Open, Short, Impedance Mismatch) and length measurement in meters.
 *
 * Key Features:
 * - Detects cable status: ok, open circuit, short circuit, impedance mismatch, unknown
 * - Displays cable length measurement (in meters)
 * - Shows list of detected faults (if any)
 * - Gracefully handles unsupported NICs (displays "Not Supported" message)
 * - Status color-coding: green (ok), red (open/short), yellow (impedance), gray (unknown)
 *
 * Usage:
 * ```typescript
 * <CableCard
 *   data={cableTestData}
 *   loading={isTesting}
 * />
 * ```
 *
 * Dependencies: BaseCard (SimpleBaseCard), Card UI components, Icons, theme utilities
 * State: Receives data from parent component via props
 */

import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";
import { Cable } from "../ui/Icons";
import { icon as iconTokens } from "../../styles/theme";

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
      icon={<Cable className={iconTokens.size.md} />}
      status={loading ? "loading" : getCardStatus(data)}
      loading={loading}
      loadingContent={<CardValue value="Testing..." size="lg" />}
    >
      {!data ? (
        <CardValue value="No data" size="md" />
      ) : !data.supported ? (
        <>
          <CardValue value="Not Supported" size="md" />
          <p className="caption mt-2">
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
              <p className="caption mb-1">Faults</p>
              <ul className="body-small text-status-error">
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
