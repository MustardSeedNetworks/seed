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

import { useTranslation } from "react-i18next";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";
import { Cable } from "../ui/Icons";
import { icon as iconTokens, spacing } from "../../styles/theme";

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

// Status mapping - labels are resolved dynamically using i18n
const statusMap: Record<string, Status> = {
  ok: "success",
  open: "error",
  short: "error",
  impedance_mismatch: "warning",
  unknown: "unknown",
};

function getCardStatus(data: CableData | null): Status {
  if (!data || !data.supported) return "unknown";
  return statusMap[data.status] || "unknown";
}

/**
 * Displays TDR cable diagnostic results with fault distance indication.
 */
export function CableCard({ data, loading }: CableCardProps) {
  const { t } = useTranslation("cards");

  const getStatusLabel = (status: string): string => {
    switch (status) {
      case "ok":
        return t("cable.statusOk");
      case "open":
        return t("cable.statusOpen");
      case "short":
        return t("cable.statusShort");
      case "impedance_mismatch":
        return t("cable.statusImpedanceMismatch");
      default:
        return t("cable.statusUnknown");
    }
  };

  return (
    <SimpleBaseCard
      title={t("cable.title")}
      icon={<Cable className={iconTokens.size.md} />}
      status={loading ? "loading" : getCardStatus(data)}
      loading={loading}
      loadingContent={<CardValue value={t("cable.testing")} size="lg" />}
    >
      {!data ? (
        <CardValue value={t("cable.noData")} size="md" />
      ) : !data.supported ? (
        <>
          <CardValue value={t("cable.notSupported")} size="md" />
          <p className={`caption ${spacing.margin.top.inline}`}>{t("cable.tdrNotSupported")}</p>
        </>
      ) : (
        <>
          <CardValue
            value={getStatusLabel(data.status)}
            size="lg"
            status={statusMap[data.status] || "unknown"}
          />
          {data.length !== null && (
            <>
              <CardDivider />
              <CardRow label={t("cable.length")} value={`${data.length}m`} />
            </>
          )}
          {data.faults.length > 0 && (
            <>
              <CardDivider />
              <p className={`caption ${spacing.margin.bottom.tight}`}>{t("cable.faults")}</p>
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
