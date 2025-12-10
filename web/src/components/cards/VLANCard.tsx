import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";

export interface VLANData {
  nativeVlan: number | null;
  taggedVlans: number[];
  voiceVlan: number | null;
  configured: {
    enabled: boolean;
    id: number;
  };
}

interface VLANCardProps {
  data: VLANData | null;
  loading?: boolean;
}

function getStatus(data: VLANData | null): Status {
  if (!data) return "unknown";
  const hasVlanInfo =
    data.nativeVlan !== null ||
    data.taggedVlans.length > 0 ||
    data.voiceVlan !== null;
  return hasVlanInfo ? "success" : "unknown";
}

export function VLANCard({ data, loading }: VLANCardProps) {
  return (
    <SimpleBaseCard
      title="VLAN"
      status={loading ? "loading" : getStatus(data)}
      loading={loading}
      loadingContent={<CardValue value="Detecting..." size="lg" />}
    >
      {!data ? (
        <CardValue value="No VLAN info" size="md" />
      ) : (
        <>
          {data.nativeVlan !== null ? (
            <CardValue label="Native VLAN" value={data.nativeVlan} size="lg" />
          ) : (
            <CardValue value="Untagged" size="lg" />
          )}
          <CardDivider />
          {data.voiceVlan !== null && (
            <CardRow label="Voice VLAN" value={data.voiceVlan.toString()} />
          )}
          {data.taggedVlans.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-text-muted mb-1">Tagged VLANs</p>
              <div className="flex flex-wrap gap-1">
                {data.taggedVlans.map((vlan) => (
                  <span
                    key={vlan}
                    className="text-xs px-2 py-0.5 bg-surface-hover rounded"
                  >
                    {vlan}
                  </span>
                ))}
              </div>
            </div>
          )}
          {data.configured.enabled && (
            <div className="mt-3 pt-2 border-t border-surface-border">
              <CardRow
                label="Configured Tag"
                value={`VLAN ${data.configured.id}`}
                status="success"
              />
            </div>
          )}
        </>
      )}
    </SimpleBaseCard>
  );
}
