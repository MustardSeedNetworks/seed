import { CardValue, CardRow, CardDivider } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";

export interface SwitchData {
  protocol: "lldp" | "cdp" | "edp" | "fdp" | "unknown";
  switchName: string | null;
  portId: string | null;
  portDescription: string | null;
  managementIp: string | null;
  systemDescription: string | null;
}

interface SwitchCardProps {
  data: SwitchData | null;
  loading?: boolean;
}

const protocolLabels: Record<string, string> = {
  lldp: "LLDP",
  cdp: "CDP",
  edp: "EDP",
  fdp: "FDP",
  unknown: "Unknown",
};

export function SwitchCard({ data, loading }: SwitchCardProps) {
  // Determine status based on whether we have switch name
  const hasSwitch = data?.switchName;
  const status = loading ? "loading" : hasSwitch ? "success" : "unknown";

  return (
    <SimpleBaseCard
      title="Nearest Switch"
      status={status}
      loading={loading}
      loadingContent={<CardValue value="Listening..." size="lg" />}
    >
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
    </SimpleBaseCard>
  );
}
