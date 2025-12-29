import type React from "react";
import { memo, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from "../../../styles/theme";
import type {
  SaveStatus,
  SnmpSettings as SnmpSettingsType,
  Snmpv3Credential,
} from "../../../types/settings";
import { generateId } from "../../../utils/id";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { Server } from "../../ui/Icons";
import { AutoSaveIndicator } from "./AutoSaveIndicator";

interface SnmpSettingsProps {
  snmpSettings: SnmpSettingsType;
  setSnmpSettings: React.Dispatch<React.SetStateAction<SnmpSettingsType>>;
  snmpStatus: SaveStatus;
}

// Protocol values - labels are translated in the component
const AUTH_PROTOCOL_VALUES = ["", "MD5", "SHA", "SHA224", "SHA256", "SHA384", "SHA512"];
const PRIV_PROTOCOL_VALUES = ["", "DES", "AES", "AES192", "AES256", "AES192C", "AES256C"];
const SECURITY_LEVEL_VALUES = ["noAuthNoPriv", "authNoPriv", "authPriv"] as const;

export const SnmpSettings = memo(function SnmpSettings({
  snmpSettings,
  setSnmpSettings,
  snmpStatus,
}: SnmpSettingsProps) {
  const { t } = useTranslation("settings");
  const [newCommunity, setNewCommunity] = useState("");
  const [expandedCredential, setExpandedCredential] = useState<string | null>(null);

  // Get translated label for auth protocol
  const getAuthProtocolLabel = (value: string) => {
    if (value === "") return t("snmp.noAuth");
    return value; // MD5, SHA, etc. don't need translation
  };

  // Get translated label for privacy protocol
  const getPrivProtocolLabel = (value: string) => {
    if (value === "") return t("snmp.noPrivacy");
    return value; // DES, AES, etc. don't need translation
  };

  // Get translated label for security level
  const getSecurityLevelLabel = (value: string) => {
    switch (value) {
      case "noAuthNoPriv":
        return t("snmp.noAuthNoPriv");
      case "authNoPriv":
        return t("snmp.authNoPriv");
      case "authPriv":
        return t("snmp.authPriv");
      default:
        return value;
    }
  };

  const addCommunity = useCallback(() => {
    if (newCommunity.trim() === "") return;
    if (snmpSettings.communities.includes(newCommunity.trim())) {
      setNewCommunity("");
      return;
    }
    setSnmpSettings((prev) => ({
      ...prev,
      communities: [...prev.communities, newCommunity.trim()],
    }));
    setNewCommunity("");
  }, [newCommunity, setSnmpSettings, snmpSettings.communities]);

  const removeCommunity = useCallback(
    (community: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        communities: prev.communities.filter((c) => c !== community),
      }));
    },
    [setSnmpSettings],
  );

  const addv3Credential = useCallback(() => {
    const newCred: Snmpv3Credential = {
      id: generateId(),
      name: t("snmp.newCredential"),
      username: "",
      authProtocol: "",
      authPassword: "",
      privProtocol: "",
      privPassword: "",
      contextName: "",
      securityLevel: "noAuthNoPriv",
    };
    setSnmpSettings((prev) => ({
      ...prev,
      v3Credentials: [...prev.v3Credentials, newCred],
    }));
    setExpandedCredential(newCred.id ?? "");
  }, [setSnmpSettings, t]);

  const removev3Credential = useCallback(
    (id: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        v3Credentials: prev.v3Credentials.filter((c) => c.id !== id),
      }));
      if (expandedCredential === id) {
        setExpandedCredential(null);
      }
    },
    [setSnmpSettings, expandedCredential],
  );

  const updatev3Credential = useCallback(
    (id: string, field: keyof Snmpv3Credential, value: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        v3Credentials: prev.v3Credentials.map((c) => (c.id === id ? { ...c, [field]: value } : c)),
      }));
    },
    [setSnmpSettings],
  );

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Server className={iconTokens.size.sm} />
          <span>{t("sections.snmp")}</span>
          <AutoSaveIndicator status={snmpStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* SNMP Port */}
        <div>
          <label className="caption text-text-muted" htmlFor="snmp-port">
            {t("snmp.port")}
          </label>
          <input
            id="snmp-port"
            type="number"
            value={snmpSettings.port}
            onChange={(e) =>
              setSnmpSettings((prev) => ({
                ...prev,
                port: Number.parseInt(e.target.value, 10) || 161,
              }))
            }
            min="1"
            max="65535"
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.md,
              spacing.margin.top.tight,
              "body-small",
            )}
          />
          <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
            {t("snmp.portDesc")}
          </p>
        </div>

        {/* Timeout */}
        <div>
          <label className="caption text-text-muted" htmlFor="snmp-timeout">
            {t("snmp.timeout")}
          </label>
          <input
            id="snmp-timeout"
            type="number"
            value={snmpSettings.timeout / 1000}
            onChange={(e) =>
              setSnmpSettings((prev) => ({
                ...prev,
                timeout: (Number.parseFloat(e.target.value) || 5) * 1000,
              }))
            }
            min="1"
            max="30"
            step="1"
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.md,
              spacing.margin.top.tight,
              "body-small",
            )}
          />
          <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
            {t("snmp.timeoutDesc")}
          </p>
        </div>

        {/* Retries */}
        <div>
          <label className="caption text-text-muted" htmlFor="snmp-retries">
            {t("snmp.retries")}
          </label>
          <input
            id="snmp-retries"
            type="number"
            value={snmpSettings.retries}
            onChange={(e) =>
              setSnmpSettings((prev) => ({
                ...prev,
                retries: Number.parseInt(e.target.value, 10) || 2,
              }))
            }
            min="0"
            max="10"
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.md,
              spacing.margin.top.tight,
              "body-small",
            )}
          />
          <p className={cn("caption text-text-muted", spacing.margin.top.tight)}>
            {t("snmp.retriesDesc")}
          </p>
        </div>

        {/* Community Strings (v1/v2c) */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <div className={cn("flex items-center justify-between", spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">
              {t("snmp.communityStrings")}
            </span>
            <button
              type="button"
              onClick={() => addCommunity()}
              className="caption text-brand-primary hover:text-brand-accent"
              aria-label="Add community string"
            >
              {t("common.add")}
            </button>
          </div>
          <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
            {t("snmp.communityDesc")}
          </p>
          <div className={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}>
            <label className="sr-only" htmlFor="snmp-community-new">
              {t("snmp.communityString")}
            </label>
            <input
              id="snmp-community-new"
              type="text"
              value={newCommunity}
              onChange={(e) => setNewCommunity(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") addCommunity();
              }}
              placeholder={t("snmp.communityString")}
              className={cn(
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.md,
                "flex-1 caption",
              )}
            />
          </div>
          {snmpSettings.communities.map((community) => (
            <div
              key={community}
              className={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}
            >
              <input
                aria-label={`Community string ${community}`}
                type="text"
                value={community}
                readOnly
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  "flex-1 bg-surface-hover caption",
                )}
              />
              <button
                type="button"
                onClick={() => removeCommunity(community)}
                className={cn("text-status-error hover:text-status-error/80", spacing.actionBtn)}
                aria-label={t("common.remove")}
              >
                {t("common.remove")}
              </button>
            </div>
          ))}
        </div>

        {/* SNMPv3 Credentials */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <div className={cn("flex items-center justify-between", spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">{t("snmp.v3Credentials")}</span>
            <button
              type="button"
              onClick={addv3Credential}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
            {t("snmp.v3CredentialsDesc")}
          </p>
          {snmpSettings.v3Credentials.map((cred) => (
            <div
              key={cred.id}
              className={cn(
                spacing.margin.bottom.inline,
                "border border-surface-border",
                radius.md,
                "overflow-hidden",
              )}
            >
              <div
                className={cn(
                  "flex items-center justify-between",
                  spacing.pad.xs,
                  "bg-surface-base cursor-pointer hover:bg-surface-hover",
                )}
                onClick={() =>
                  setExpandedCredential(expandedCredential === cred.id ? null : (cred.id ?? ""))
                }
              >
                <span className="body-small text-text-primary">
                  {cred.name || t("snmp.unnamedCredential")}
                </span>
                <div className={cn("flex items-center", spacing.gap.compact)}>
                  <span className="caption text-text-muted">
                    {cred.username || t("snmp.noUsername")}
                  </span>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      removev3Credential(cred.id ?? "");
                    }}
                    className={cn(
                      "text-status-error hover:text-status-error/80",
                      spacing.actionBtn,
                    )}
                    aria-label={t("common.remove")}
                  >
                    {t("common.remove")}
                  </button>
                </div>
              </div>
              {expandedCredential === cred.id && (
                <div className={cn(spacing.pad.sm, "bg-surface-hover stack-sm")}>
                  {/* Name */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`cred-name-${cred.id}`}>
                      {t("common.name")}
                    </label>
                    <input
                      id={`cred-name-${cred.id}`}
                      type="text"
                      value={cred.name}
                      onChange={(e) => updatev3Credential(cred.id ?? "", "name", e.target.value)}
                      placeholder={t("snmp.credentialName")}
                      className={cn(
                        inputTokens.base,
                        inputTokens.state.default,
                        inputTokens.size.sm,
                        spacing.margin.top.tight,
                        "caption",
                      )}
                    />
                  </div>

                  {/* Username */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`cred-username-${cred.id}`}>
                      {t("snmp.username")}
                    </label>
                    <input
                      id={`cred-username-${cred.id}`}
                      type="text"
                      value={cred.username}
                      onChange={(e) =>
                        updatev3Credential(cred.id ?? "", "username", e.target.value)
                      }
                      placeholder={t("snmp.snmpv3Username")}
                      className={cn(
                        inputTokens.base,
                        inputTokens.state.default,
                        inputTokens.size.sm,
                        spacing.margin.top.tight,
                        "caption",
                      )}
                    />
                  </div>

                  {/* Security Level */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`sec-level-${cred.id}`}>
                      {t("snmp.securityLevel")}
                    </label>
                    <select
                      id={`sec-level-${cred.id}`}
                      value={cred.securityLevel}
                      onChange={(e) =>
                        updatev3Credential(cred.id ?? "", "securityLevel", e.target.value)
                      }
                      className={cn(
                        inputTokens.base,
                        inputTokens.state.default,
                        inputTokens.size.sm,
                        spacing.margin.top.tight,
                        "caption",
                      )}
                    >
                      {SECURITY_LEVEL_VALUES.map((value) => (
                        <option key={value} value={value}>
                          {getSecurityLevelLabel(value)}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Authentication Protocol */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`auth-proto-${cred.id}`}>
                      {t("snmp.authProtocol")}
                    </label>
                    <select
                      id={`auth-proto-${cred.id}`}
                      value={cred.authProtocol}
                      onChange={(e) =>
                        updatev3Credential(cred.id ?? "", "authProtocol", e.target.value)
                      }
                      className={cn(
                        inputTokens.base,
                        inputTokens.state.default,
                        inputTokens.size.sm,
                        spacing.margin.top.tight,
                        "caption",
                      )}
                    >
                      {AUTH_PROTOCOL_VALUES.map((value) => (
                        <option key={value} value={value}>
                          {getAuthProtocolLabel(value)}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Authentication Password */}
                  {cred.authProtocol !== "" && (
                    <div>
                      <label className="caption text-text-muted" htmlFor={`auth-pass-${cred.id}`}>
                        {t("snmp.authPassword")}
                      </label>
                      <input
                        id={`auth-pass-${cred.id}`}
                        type="password"
                        value={cred.authPassword}
                        onChange={(e) =>
                          updatev3Credential(cred.id ?? "", "authPassword", e.target.value)
                        }
                        placeholder={t("snmp.authPasswordPlaceholder")}
                        className={cn(
                          inputTokens.base,
                          inputTokens.state.default,
                          inputTokens.size.sm,
                          spacing.margin.top.tight,
                          "caption",
                        )}
                      />
                    </div>
                  )}

                  {/* Privacy Protocol */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`priv-proto-${cred.id}`}>
                      {t("snmp.privProtocol")}
                    </label>
                    <select
                      id={`priv-proto-${cred.id}`}
                      value={cred.privProtocol}
                      onChange={(e) =>
                        updatev3Credential(cred.id ?? "", "privProtocol", e.target.value)
                      }
                      className={cn(
                        inputTokens.base,
                        inputTokens.state.default,
                        inputTokens.size.sm,
                        spacing.margin.top.tight,
                        "caption",
                      )}
                    >
                      {PRIV_PROTOCOL_VALUES.map((value) => (
                        <option key={value} value={value}>
                          {getPrivProtocolLabel(value)}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Privacy Password */}
                  {cred.privProtocol !== "" && (
                    <div>
                      <label className="caption text-text-muted" htmlFor={`priv-pass-${cred.id}`}>
                        {t("snmp.privPassword")}
                      </label>
                      <input
                        id={`priv-pass-${cred.id}`}
                        type="password"
                        value={cred.privPassword}
                        onChange={(e) =>
                          updatev3Credential(cred.id ?? "", "privPassword", e.target.value)
                        }
                        placeholder={t("snmp.privPasswordPlaceholder")}
                        className={cn(
                          inputTokens.base,
                          inputTokens.state.default,
                          inputTokens.size.sm,
                          spacing.margin.top.tight,
                          "caption",
                        )}
                      />
                    </div>
                  )}

                  {/* Context Name */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`context-name-${cred.id}`}>
                      {t("snmp.contextName")}
                    </label>
                    <input
                      id={`context-name-${cred.id}`}
                      type="text"
                      value={cred.contextName}
                      onChange={(e) =>
                        updatev3Credential(cred.id ?? "", "contextName", e.target.value)
                      }
                      placeholder={t("snmp.snmpContext")}
                      className={cn(
                        inputTokens.base,
                        inputTokens.state.default,
                        inputTokens.size.sm,
                        spacing.margin.top.tight,
                        "caption",
                      )}
                    />
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
});
