/**
 * SwitchCard Component
 *
 * Purpose: Displays switch/network device information learned via Layer 2 discovery protocols
 * (LLDP, CDP, EDP, FDP). Shows switch name, connected port, VLAN configuration, and management IP.
 *
 * Key Features:
 * - Protocol detection: identifies which discovery protocol learned the info (LLDP/CDP/EDP/FDP)
 * - Switch identification: switch name, management IP, system description
 * - Port information: port ID and description from switch
 * - VLAN support: native VLAN, tagged VLANs, voice VLAN detection and configuration
 * - Dual data: combines switch info and VLAN data into single view
 * - Status determination: success if switch/VLAN info available, unknown if none
 * - Compact layout: horizontal cards showing each section
 *
 * Usage:
 * ```typescript
 * <SwitchCard
 *   data={switchInfo}
 *   vlanData={vlanInfo}
 *   loading={isScanning}
 * />
 * ```
 *
 * Dependencies: BaseCard (SimpleBaseCard), Card UI components, Icons, theme utilities
 * State: Receives data from parent component via props
 */

import type React from "react";
import { useTranslation } from "react-i18next";
import { border, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";
import { CardDivider, CardRow, CardValue } from "../ui/Card";
import { Network } from "../ui/Icons";
import { SimpleBaseCard } from "./BaseCard";

export interface SwitchData {
  protocol: "lldp" | "cdp" | "edp" | "fdp" | "unknown";
  switchName: string | null;
  portId: string | null;
  portDescription: string | null;
  managementIp: string | null;
  systemDescription: string | null;
}

export interface VlanData {
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
  vlanData?: VlanData | null;
  loading?: boolean;
}

const protocolLabels: Record<string, string> = {
  lldp: "LLDP",
  cdp: "CDP",
  edp: "EDP",
  fdp: "FDP",
  unknown: "Unknown",
};

/**
 * Displays detected switch information from LLDP/CDP discovery protocols.
 */
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Switch card requires multiple conditional VLAN sections
export function SwitchCard({ data, vlanData, loading }: SwitchCardProps): React.ReactElement {
  const { t } = useTranslation("cards");

  // Determine status based on whether we have switch name or VLAN info
  const hasSwitch = data?.switchName;
  const hasVlanInfo =
    vlanData &&
    (vlanData.nativeVlan !== null ||
      vlanData.taggedVlans.length > 0 ||
      vlanData.voiceVlan !== null);
  const getStatus = (): "loading" | "success" | "unknown" => {
    if (loading) {
      return "loading";
    }
    if (hasSwitch || hasVlanInfo) {
      return "success";
    }
    return "unknown";
  };
  const status = getStatus();

  return (
    <SimpleBaseCard
      title={t("switch.title")}
      icon={<Network class={iconTokens.size.md} />}
      status={status}
      loading={loading}
      loadingContent={<CardValue value={t("switch.listening")} size="lg" />}
    >
      {/* Switch Info Section */}
      {hasSwitch ? (
        <>
          <CardValue value={data?.switchName ?? ""} size="lg" />
          <CardDivider />
          {data?.portId ? <CardRow label={t("switch.port")} value={data?.portId} /> : null}
          {data?.portDescription ? (
            <CardRow label={t("switch.description")} value={data?.portDescription} />
          ) : null}
          {data?.managementIp ? (
            <CardRow label={t("switch.managementIp")} value={data?.managementIp} />
          ) : null}
          <div class={spacing.margin.top.inline}>
            <span
              class={cn(
                "caption",
                spacing.chip.sm,
                "bg-brand-primary/20",
                "text-brand-primary",
                radius.default,
              )}
            >
              {protocolLabels[data?.protocol]}
            </span>
          </div>
        </>
      ) : (
        <>
          <CardValue value={t("switch.noDiscoveryFrames")} size="md" />
          <p class={cn("caption", spacing.margin.top.inline)}>{t("switch.waitingFrames")}</p>
        </>
      )}

      {/* VLAN Section */}
      {vlanData ? (
        <>
          <CardDivider />
          <p class={cn("section-title", spacing.margin.bottom.inline)}>{t("switch.vlans")}</p>
          {vlanData.nativeVlan !== null ? (
            <CardRow label={t("switch.nativeVlan")} value={vlanData.nativeVlan.toString()} />
          ) : (
            <CardRow label={t("switch.nativeVlan")} value={t("switch.untagged")} />
          )}
          {vlanData.voiceVlan !== null ? (
            <CardRow label={t("switch.voiceVlan")} value={vlanData.voiceVlan.toString()} />
          ) : null}
          {vlanData.taggedVlans.length > 0 ? (
            <div class={spacing.margin.top.inline}>
              <p class={cn("caption", spacing.margin.bottom.inline)}>{t("switch.taggedVlans")}</p>
              <div class={layout.inline.wrap}>
                {vlanData.taggedVlans.map((vlan) => (
                  <span
                    key={vlan}
                    class={cn("caption", spacing.chip.sm, "bg-surface-hover", radius.default)}
                  >
                    {vlan}
                  </span>
                ))}
              </div>
            </div>
          ) : null}
          {vlanData.configured.enabled ? (
            <div
              class={cn(spacing.margin.top.heading, spacing.padding.top.heading, border.divider)}
            >
              <CardRow
                label={t("switch.configuredTag")}
                value={`VLAN ${vlanData.configured.id}`}
                status="success"
              />
            </div>
          ) : null}
        </>
      ) : null}
    </SimpleBaseCard>
  );
}
