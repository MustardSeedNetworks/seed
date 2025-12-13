import { CardValue, CardRow, CardDivider } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";
import { Network } from "../ui/Icons";

export interface SwitchData {
  protocol: "lldp" | "cdp" | "edp" | "fdp" | "unknown";
  switchName: string | null;
  portId: string | null;
  portDescription: string | null;
  managementIp: string | null;
  systemDescription: string | null;
}

export interface VLANData {
  nativeVlan: number | null;
  taggedVlans: number[];
  voiceVlan: number | null;
  configured: {
    enabled: boolean;
    id: number;
  };
}

interface SwitchCardProps {
  data: SwitchData | null;
  vlanData?: VLANData | null;
  loading?: boolean;
}

const protocolLabels: Record<string, string> = {
  lldp: "LLDP",
  cdp: "CDP",
  edp: "EDP",
  fdp: "FDP",
  unknown: "Unknown",
};

export function SwitchCard({ data, vlanData, loading }: SwitchCardProps) {
  // Determine status based on whether we have switch name or VLAN info
  const hasSwitch = data?.switchName;
  const hasVlanInfo =
    vlanData &&
    (vlanData.nativeVlan !== null ||
      vlanData.taggedVlans.length > 0 ||
      vlanData.voiceVlan !== null);
  const status = loading
    ? "loading"
    : hasSwitch || hasVlanInfo
      ? "success"
      : "unknown";

  return (
    <SimpleBaseCard
      title="Nearest Switch"
      icon={<Network className="w-5 h-5" />}
      status={status}
      loading={loading}
      loadingContent={<CardValue value="Listening..." size="lg" />}
    >
      {/* Switch Info Section */}
      {!hasSwitch ? (
        <>
          <CardValue value="No discovery frames" size="md" />
          <p className="text-xs text-text-muted mt-2">
            Waiting for LLDP/CDP frames...
          </p>
        </>
      ) : (
        <>
          <CardValue value={data!.switchName!} size="lg" />
          <CardDivider />
          {data!.portId && <CardRow label="Port" value={data!.portId} />}
          {data!.portDescription && (
            <CardRow label="Description" value={data!.portDescription} />
          )}
          {data!.managementIp && (
            <CardRow label="Management IP" value={data!.managementIp} />
          )}
          <div className="mt-2">
            <span className="text-xs px-2 py-0.5 bg-brand-primary/20 text-brand-primary rounded">
              {protocolLabels[data!.protocol]}
            </span>
          </div>
        </>
      )}

      {/* VLAN Section */}
      {vlanData && (
        <>
          <CardDivider />
          <p className="text-xs uppercase tracking-wide text-text-muted font-semibold mb-2">
            VLANs
          </p>
          {vlanData.nativeVlan !== null ? (
            <CardRow
              label="Native VLAN"
              value={vlanData.nativeVlan.toString()}
            />
          ) : (
            <CardRow label="Native VLAN" value="Untagged" />
          )}
          {vlanData.voiceVlan !== null && (
            <CardRow label="Voice VLAN" value={vlanData.voiceVlan.toString()} />
          )}
          {vlanData.taggedVlans.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-text-muted mb-1">Tagged VLANs</p>
              <div className="flex flex-wrap gap-1">
                {vlanData.taggedVlans.map((vlan) => (
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
          {vlanData.configured.enabled && (
            <div className="mt-3 pt-2 border-t border-surface-border">
              <CardRow
                label="Configured Tag"
                value={`VLAN ${vlanData.configured.id}`}
                status="success"
              />
            </div>
          )}
        </>
      )}
    </SimpleBaseCard>
  );
}
