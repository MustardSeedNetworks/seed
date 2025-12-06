import { Card, CardValue, CardRow, CardDivider, Status } from '../ui/Card';
import { Skeleton } from '../ui/Skeleton';

export interface LinkData {
  linkUp: boolean;
  carrier: boolean;    // Physical link/carrier detected (Layer 2)
  hasIP: boolean;      // Has routable IP address (Layer 3)
  speed: string;
  duplex: string;
  advertisedSpeeds: string[];
  mtu?: number;
  autoNeg?: boolean;
}

interface LinkCardProps {
  data: LinkData | null;
  loading?: boolean;
}

export function LinkCard({ data, loading }: LinkCardProps) {
  if (loading || !data) {
    return (
      <Card title="Link" status="loading">
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
      </Card>
    );
  }

  // Determine status based on carrier (L2) and IP (L3)
  const getStatus = (): Status => {
    if (!data.carrier) return 'error';      // No physical link
    if (!data.hasIP) return 'warning';      // Carrier but no IP
    return 'success';                        // Fully connected
  };

  const getStatusText = (): string => {
    if (!data.carrier) return 'No Carrier';
    if (!data.hasIP) return 'No IP';
    return data.speed || 'Connected';
  };

  const status = getStatus();

  return (
    <Card title="Link" status={status}>
      <CardValue
        value={getStatusText()}
        size="lg"
        status={status}
      />
      <CardDivider />
      <CardRow label="Carrier" value={data.carrier ? 'Connected' : 'No Signal'} />
      <CardRow label="IP Status" value={data.hasIP ? 'Assigned' : 'None'} />
      {data.carrier && (
        <>
          <CardRow label="Duplex" value={data.duplex || 'Unknown'} />
          {data.mtu && <CardRow label="MTU" value={data.mtu.toString()} />}
          {data.autoNeg !== undefined && (
            <CardRow label="Auto-Neg" value={data.autoNeg ? 'On' : 'Off'} />
          )}
          {data.advertisedSpeeds && data.advertisedSpeeds.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-text-muted mb-1">Advertised Speeds</p>
              <div className="flex flex-wrap gap-1">
                {data.advertisedSpeeds.map((speed) => (
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
    </Card>
  );
}
