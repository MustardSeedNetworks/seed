import { memo } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  layout,
  spacing,
  radius,
  icon as iconTokens,
} from "../../../../styles/theme";
import { AutoSaveIndicator } from "../AutoSaveIndicator";
import type { SubnetConfig, SaveStatus } from "../../../../types/settings";

interface SubnetManagerProps {
  subnets: SubnetConfig[];
  subnetsStatus: SaveStatus;
  newSubnetCidr: string;
  setNewSubnetCidr: React.Dispatch<React.SetStateAction<string>>;
  newSubnetName: string;
  setNewSubnetName: React.Dispatch<React.SetStateAction<string>>;
  subnetError: string | null;
  setSubnetError: React.Dispatch<React.SetStateAction<string | null>>;
  addSubnet: () => void;
  toggleSubnet: (cidr: string, enabled: boolean) => void;
  deleteSubnet: (cidr: string) => void;
}

/**
 * Manages target networks (subnets) for discovery.
 * Only shown when full_scan or custom profile is selected.
 */
export const SubnetManager = memo(function SubnetManager({
  subnets,
  subnetsStatus,
  newSubnetCidr,
  setNewSubnetCidr,
  newSubnetName,
  setNewSubnetName,
  subnetError,
  setSubnetError,
  addSubnet,
  toggleSubnet,
  deleteSubnet,
}: SubnetManagerProps) {
  const { t } = useTranslation("settings");

  return (
    <div className={cn("border-t border-surface-border", spacing.pad.sm)}>
      <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
        <span className="caption text-text-muted font-medium">
          {t("discovery.targetNetworks")}{" "}
          <AutoSaveIndicator status={subnetsStatus} />
        </span>
      </div>
      <p
        className={cn("caption text-text-muted", spacing.margin.bottom.inline)}
      >
        {t("discovery.targetNetworksDesc")}
      </p>

      {/* List of configured subnets */}
      {subnets.length > 0 && (
        <div className={cn("stack-sm", spacing.margin.bottom.heading)}>
          {subnets.map((subnet) => (
            <div
              key={subnet.cidr}
              className={cn(
                layout.flex.between,
                spacing.pad.xs,
                "bg-surface-base",
                radius.default,
                "border border-surface-border"
              )}
            >
              <div className="flex-1 min-w-0">
                <div className="body-small text-text-primary truncate">
                  {subnet.name || subnet.cidr}
                </div>
                <div className="caption text-text-muted">{subnet.cidr}</div>
              </div>
              <div
                className={cn(
                  layout.inline.default,
                  spacing.margin.left.inline
                )}
              >
                <input
                  type="checkbox"
                  checked={subnet.enabled}
                  onChange={(e) => toggleSubnet(subnet.cidr, e.target.checked)}
                  className={iconTokens.size.sm}
                  title={
                    subnet.enabled
                      ? t("discovery.disableSubnet")
                      : t("discovery.enableSubnet")
                  }
                />
                <button
                  onClick={() => deleteSubnet(subnet.cidr)}
                  className="text-status-error hover:text-status-error/70 body-small"
                  title={t("discovery.removeSubnet")}
                >
                  X
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add new subnet form */}
      <div className="stack-sm">
        <input
          type="text"
          value={newSubnetCidr}
          onChange={(e) => {
            setNewSubnetCidr(e.target.value);
            setSubnetError(null);
          }}
          placeholder={t("discovery.cidrPlaceholder")}
          className={cn(
            "w-full",
            spacing.chip.lg,
            "bg-surface-base border border-surface-border",
            radius.default,
            "body-small text-text-primary"
          )}
        />
        <input
          type="text"
          value={newSubnetName}
          onChange={(e) => setNewSubnetName(e.target.value)}
          placeholder={t("discovery.namePlaceholder")}
          className={cn(
            "w-full",
            spacing.chip.lg,
            "bg-surface-base border border-surface-border",
            radius.default,
            "body-small text-text-primary"
          )}
        />
        {subnetError && (
          <p className="caption text-status-error">{subnetError}</p>
        )}
        <button
          onClick={addSubnet}
          className={cn(
            "w-full",
            spacing.pad.sm,
            "bg-brand-primary hover:bg-brand-accent text-text-inverse",
            radius.default,
            "body-small"
          )}
        >
          {t("discovery.addSubnet")}
        </button>
      </div>
    </div>
  );
});
