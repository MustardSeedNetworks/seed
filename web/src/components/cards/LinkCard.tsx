import { memo } from "react";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";
import { BaseCard } from "./BaseCard";

interface LinkHistoryEvent {
  state: string;
  timestamp: string;
}

export interface LinkData {
  linkUp: boolean;
  carrier: boolean; // Physical link/carrier detected (Layer 2)
  hasIP: boolean; // Has routable IP address (Layer 3)
  speed: string;
  duplex: string;
  advertisedSpeeds: string[];
  mtu?: number;
  autoNeg?: boolean;
  flapCount24h?: number;
  history?: LinkHistoryEvent[];
  uptimeMs?: number;
}

interface LinkCardProps {
  data: LinkData | null;
  loading?: boolean;
}

// Determine status based on carrier (L2) and IP (L3)
function getStatus(data: LinkData): Status {
  if (!data.carrier) return "error"; // No physical link
  if (!data.hasIP) return "warning"; // Carrier but no IP
  return "success"; // Fully connected
}

function getStatusText(data: LinkData): string {
  if (!data.carrier) return "No Carrier";
  if (!data.hasIP) return "No IP";
  return data.speed || "Connected";
}

function LinkLoadingSkeleton() {
  return (
    <>
      <Skeleton className="h-8 w-32 mb-3" />
      <div className="space-y-2 mt-4">
        <div className="flex justify-between">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-8" />
        </div>
      </div>
    </>
  );
}

export const LinkCard = memo(function LinkCard({
  data,
  loading,
}: LinkCardProps) {
  return (
    <BaseCard
      title="Link"
      data={data}
      loading={loading}
      getStatus={getStatus}
      loadingContent={<LinkLoadingSkeleton />}
      emptyMessage="No link data"
    >
      {(linkData) => {
        const status = getStatus(linkData);
        return (
          <>
            <CardValue
              value={getStatusText(linkData)}
              size="lg"
              status={status}
            />
            <CardDivider />
            <CardRow
              label="Carrier"
              value={linkData.carrier ? "Connected" : "No Signal"}
            />
            {linkData.carrier && (
              <>
                <CardRow label="Duplex" value={linkData.duplex || "Unknown"} />
                {linkData.mtu && (
                  <CardRow label="MTU" value={linkData.mtu.toString()} />
                )}
                {linkData.autoNeg !== undefined && (
                  <CardRow
                    label="Auto-Neg"
                    value={linkData.autoNeg ? "On" : "Off"}
                  />
                )}
                {linkData.flapCount24h !== undefined && (
                  <CardRow
                    label="Flaps (24h)"
                    value={linkData.flapCount24h.toString()}
                  />
                )}
                {linkData.advertisedSpeeds &&
                  linkData.advertisedSpeeds.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs text-text-muted mb-1">
                        Advertised Speeds
                      </p>
                      <div className="flex flex-wrap gap-1">
                        {linkData.advertisedSpeeds.map((speed) => (
                          <span
                            key={speed}
                            className="text-xs px-2 py-0.5 bg-surface-hover rounded"
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
