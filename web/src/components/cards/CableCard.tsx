/**
 * CableCard Component
 *
 * Purpose: Displays Ethernet cable test results using Time Domain Reflectometry (TDR).
 * Shows cable condition (OK, Open, Short, Impedance Mismatch) and length measurement.
 *
 * Key Features:
 * - Detects cable status: ok, open circuit, short circuit, impedance mismatch, unknown
 * - Per-pair TDR results (Pairs A-D / pins 1-2, 3-6, 4-5, 7-8)
 * - 568A/568B wiring standard color mapping display
 * - Displays cable length in both meters and feet
 * - Shows crossover cable detection
 * - Shows list of detected faults (if any)
 * - Gracefully handles unsupported NICs (displays "Not Supported" message)
 * - Status color-coding: green (ok), red (open/short), yellow (impedance), gray (unknown)
 *
 * Usage:
 * ```typescript
 * <CableCard
 *   data={cableTestData}
 *   loading={isTesting}
 *   wiringStandard="568B"
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
import {
  cn,
  icon as iconTokens,
  layout,
  radius,
  spacing,
} from "../../styles/theme";

/** Per-pair TDR test result */
interface CablePairResult {
  pair: string; // "1-2", "3-6", "4-5", "7-8"
  pairLetter: string; // "A", "B", "C", "D"
  status: string; // ok, open, short, etc.
  lengthM?: number | null;
  lengthFt?: number | null;
}

/** Pin-to-color mapping for wiring standard display */
interface CablePinout {
  pin: number;
  color: string;
  pair: string;
}

export interface CableData {
  supported: boolean;
  status:
    | "ok"
    | "open"
    | "short"
    | "impedance_mismatch"
    | "crosstalk"
    | "split_pair"
    | "unknown";
  length?: number | null; // meters
  lengthFt?: number | null; // feet
  pairs?: CablePairResult[];
  faults: string[];
  wiringStandard?: string; // "568A" or "568B"
  pinout?: CablePinout[];
  isCrossover?: boolean;
  driverName?: string;
}

interface CableCardProps {
  data: CableData | null;
  loading?: boolean;
  showPinout?: boolean; // Whether to show pinout color mapping
}

// Status mapping - labels are resolved dynamically using i18n
const statusMap: Record<string, Status> = {
  ok: "success",
  open: "error",
  short: "error",
  impedance_mismatch: "warning",
  crosstalk: "warning",
  split_pair: "warning",
  unknown: "unknown",
};

// Wire color to CSS class mapping for visual display
const wireColorMap: Record<string, string> = {
  "White/Orange": "bg-orange-100 border-orange-400",
  Orange: "bg-orange-500",
  "White/Green": "bg-green-100 border-green-400",
  Green: "bg-green-500",
  "White/Blue": "bg-blue-100 border-blue-400",
  Blue: "bg-blue-500",
  "White/Brown": "bg-amber-100 border-amber-600",
  Brown: "bg-amber-700",
};

function getCardStatus(data: CableData | null): Status {
  if (!data || !data.supported) return "unknown";
  return statusMap[data.status] || "unknown";
}

/**
 * Displays TDR cable diagnostic results with fault distance indication.
 */
export function CableCard({
  data,
  loading,
  showPinout = true,
}: CableCardProps) {
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
      case "crosstalk":
        return t("cable.statusCrosstalk", "Crosstalk");
      case "split_pair":
        return t("cable.statusSplitPair", "Split Pair");
      default:
        return t("cable.statusUnknown");
    }
  };

  const formatLength = (
    meters?: number | null,
    feet?: number | null
  ): string => {
    if (meters === null || meters === undefined) return "-";
    const ft = feet ?? meters * 3.28084;
    return `${meters.toFixed(1)}m / ${ft.toFixed(1)}ft`;
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
          <p className={cn("caption", spacing.margin.top.inline)}>
            {t("cable.tdrNotSupported")}
          </p>
          {data.driverName && (
            <p
              className={cn(
                "caption text-text-muted",
                spacing.margin.top.tight
              )}
            >
              {t("cable.driver", "Driver")}: {data.driverName}
            </p>
          )}
        </>
      ) : (
        <>
          <CardValue
            value={getStatusLabel(data.status)}
            size="lg"
            status={statusMap[data.status] || "unknown"}
          />

          {/* Crossover indicator */}
          {data.isCrossover && (
            <p
              className={cn(
                "caption text-status-warning font-medium",
                spacing.margin.top.tight
              )}
            >
              {t("cable.crossover", "Crossover Cable Detected")}
            </p>
          )}

          {/* Overall length */}
          {data.length !== null && data.length !== undefined && (
            <>
              <CardDivider />
              <CardRow
                label={t("cable.length")}
                value={formatLength(data.length, data.lengthFt)}
              />
            </>
          )}

          {/* Per-pair results */}
          {data.pairs && data.pairs.length > 0 && (
            <>
              <CardDivider />
              <p
                className={cn(
                  "caption font-medium text-text-muted",
                  spacing.margin.bottom.tight
                )}
              >
                {t("cable.pairResults", "Pair Results")}
              </p>
              <div className={cn("stack-sm", spacing.margin.top.tight)}>
                {data.pairs.map((pair) => (
                  <div
                    key={pair.pair}
                    className={cn(layout.flex.between, "body-small")}
                  >
                    <span className="text-text-muted">
                      {t("cable.pair", "Pair")} {pair.pairLetter} ({pair.pair})
                    </span>
                    <span
                      className={cn(
                        pair.status === "ok"
                          ? "text-status-success"
                          : pair.status === "unknown"
                            ? "text-text-muted"
                            : "text-status-error"
                      )}
                    >
                      {getStatusLabel(pair.status)}
                      {pair.lengthM !== null &&
                        pair.lengthM !== undefined &&
                        ` (${pair.lengthM.toFixed(1)}m)`}
                    </span>
                  </div>
                ))}
              </div>
            </>
          )}

          {/* Wiring standard pinout */}
          {showPinout && data.pinout && data.pinout.length > 0 && (
            <>
              <CardDivider />
              <p
                className={cn(
                  "caption font-medium text-text-muted",
                  spacing.margin.bottom.tight
                )}
              >
                {t("cable.wiringStandard", "Wiring Standard")}:{" "}
                {data.wiringStandard || "568B"}
              </p>
              <div
                className={cn(
                  "grid grid-cols-8",
                  spacing.gap.tight,
                  spacing.margin.top.tight
                )}
              >
                {data.pinout.map((pin) => (
                  <div key={pin.pin} className="text-center">
                    <div
                      className={cn(
                        "w-4 h-6 mx-auto border",
                        radius.sm,
                        wireColorMap[pin.color] || "bg-gray-400"
                      )}
                      title={pin.color}
                    />
                    <span className="caption text-text-muted">{pin.pin}</span>
                  </div>
                ))}
              </div>
            </>
          )}

          {/* Faults */}
          {data.faults.length > 0 && (
            <>
              <CardDivider />
              <p className={cn("caption", spacing.margin.bottom.tight)}>
                {t("cable.faults")}
              </p>
              <ul className="body-small text-status-error">
                {data.faults.map((fault, index) => (
                  <li key={index}>• {fault}</li>
                ))}
              </ul>
            </>
          )}

          {/* Driver info */}
          {data.driverName && (
            <p
              className={cn(
                "caption text-text-muted",
                spacing.margin.top.inline
              )}
            >
              {t("cable.driver", "Driver")}: {data.driverName}
            </p>
          )}
        </>
      )}
    </SimpleBaseCard>
  );
}
