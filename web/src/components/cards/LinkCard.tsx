/**
 * Link Status Card Component
 *
 * Displays physical link layer (Layer 2) and network layer (Layer 3) status.
 *
 * Features:
 * - Link state detection (up/down)
 * - Carrier signal detection (physical link present)
 * - IP configuration status
 * - Connection speed and duplex mode
 * - Negotiated speeds (from auto-negotiation)
 * - MTU and auto-negotiation settings
 * - Link flap counting (24-hour window)
 * - Uptime tracking
 * - Link state history
 *
 * Status Indicators:
 * - **Error (Red)**: No physical carrier detected (L2 down)
 * - **Warning (Yellow)**: Carrier present but no IP address (L3 down)
 * - **Success (Green)**: Both L2 and L3 up, fully connected
 *
 * The card is the primary indicator of network interface health.
 */

import { memo } from "react";
import { useTranslation } from "react-i18next";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";
import { BaseCard } from "./BaseCard";
import { Cable } from "../ui/Icons";
import { layout, radius, spacing, icon as iconTokens } from "../../styles/theme";

/**
 * Historical link state event
 */
interface LinkHistoryEvent {
  state: string; // State change ("up", "down", "flap", etc.)
  timestamp: string; // ISO 8601 timestamp
}

/**
 * Link layer and network layer status data
 */
export interface LinkData {
  linkUp: boolean; // Link is administratively up
  carrier: boolean; // Physical carrier/link detected (L2)
  hasIP: boolean; // Has routable IP address (L3)
  speed: string; // Current connection speed (e.g., "1000Mb/s")
  duplex: string; // Duplex mode ("full" or "half")
  advertisedSpeeds: string[]; // Speeds supported by auto-negotiation
  mtu?: number; // Maximum transmission unit
  autoNeg?: boolean; // Auto-negotiation enabled
  flapCount24h?: number; // Number of link state changes in last 24h
  history?: LinkHistoryEvent[]; // Recent link state events
  uptimeMs?: number; // Time since last state change (ms)
}

/**
 * Props for Link Card
 */
interface LinkCardProps {
  data: LinkData | null; // Link status data
  loading?: boolean; // True while loading
}

/**
 * Determines card status based on link and IP state.
 * Uses both L2 (carrier) and L3 (IP) information.
 *
 * @param data - Link status data
 * @returns Status indicator ('success', 'warning', 'error')
 */
function getStatus(data: LinkData): Status {
  if (!data.carrier) return "error"; // No physical link
  if (!data.hasIP) return "warning"; // Carrier but no IP
  return "success"; // Fully connected
}

function LinkLoadingSkeleton() {
  return (
    <>
      <Skeleton className="h-8 w-32 mb-3" />
      <div className={`stack-sm ${spacing.margin.top.content}`}>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-8" />
        </div>
      </div>
    </>
  );
}

export const LinkCard = memo(function LinkCard({ data, loading }: LinkCardProps) {
  const { t } = useTranslation("cards");
  const { t: tc } = useTranslation("common");

  const getLocalizedStatusText = (linkData: LinkData): string => {
    if (!linkData.carrier) return tc("status.noCarrier");
    if (!linkData.hasIP) return tc("status.noIP");
    return linkData.speed || tc("status.connected");
  };

  return (
    <BaseCard
      title={t("link.title")}
      icon={<Cable className={iconTokens.size.md} />}
      data={data}
      loading={loading}
      getStatus={getStatus}
      loadingContent={<LinkLoadingSkeleton />}
      emptyMessage={tc("status.noData")}
    >
      {(linkData) => {
        const status = getStatus(linkData);
        return (
          <>
            <CardValue value={getLocalizedStatusText(linkData)} size="lg" status={status} />
            <CardDivider />
            <CardRow
              label={t("link.carrier")}
              value={linkData.carrier ? tc("status.connected") : tc("status.noSignal")}
            />
            {linkData.carrier && (
              <>
                <CardRow label={t("link.duplex")} value={linkData.duplex || tc("status.unknown")} />
                {linkData.mtu && <CardRow label={t("link.mtu")} value={linkData.mtu.toString()} />}
                {linkData.autoNeg !== undefined && (
                  <CardRow
                    label={t("link.autoNeg")}
                    value={linkData.autoNeg ? tc("status.on") : tc("status.off")}
                  />
                )}
                {linkData.flapCount24h !== undefined && (
                  <CardRow label={t("link.flaps24h")} value={linkData.flapCount24h.toString()} />
                )}
                {linkData.advertisedSpeeds && linkData.advertisedSpeeds.length > 0 && (
                  <div className="mt-2">
                    <p className="caption mb-1">{t("link.advertisedSpeeds")}</p>
                    <div className={layout.inline.wrap}>
                      {linkData.advertisedSpeeds.map((speed) => (
                        <span
                          key={speed}
                          className={`caption px-2 py-0.5 bg-surface-hover ${radius.default}`}
                        >
                          {speed}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </>
            )}
          </>
        );
      }}
    </BaseCard>
  );
});
