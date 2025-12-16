import { memo, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Server } from "../../ui/Icons";
import {
  SNMPSettings as SNMPSettingsType,
  SNMPv3Credential,
  SaveStatus,
} from "../../../types/settings";
import { generateId } from "../../../utils/id";
import {
  layout,
  icon as iconTokens,
  radius,
  input as inputTokens,
  spacing,
} from "../../../styles/theme";

interface SNMPSettingsProps {
  snmpSettings: SNMPSettingsType;
  setSnmpSettings: React.Dispatch<React.SetStateAction<SNMPSettingsType>>;
  snmpStatus: SaveStatus;
}

// Protocol values - labels are translated in the component
const AUTH_PROTOCOL_VALUES = ["", "MD5", "SHA", "SHA224", "SHA256", "SHA384", "SHA512"];
const PRIV_PROTOCOL_VALUES = ["", "DES", "AES", "AES192", "AES256", "AES192C", "AES256C"];
const SECURITY_LEVEL_VALUES = ["noAuthNoPriv", "authNoPriv", "authPriv"] as const;

export const SNMPSettings = memo(function SNMPSettings({
  snmpSettings,
  setSnmpSettings,
  snmpStatus,
}: SNMPSettingsProps) {
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
    [setSnmpSettings]
  );

  const addV3Credential = useCallback(() => {
    const newCred: SNMPv3Credential = {
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
    setExpandedCredential(newCred.id!);
  }, [setSnmpSettings, t]);

  const removeV3Credential = useCallback(
    (id: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        v3Credentials: prev.v3Credentials.filter((c) => c.id !== id),
      }));
      if (expandedCredential === id) {
        setExpandedCredential(null);
      }
    },
    [setSnmpSettings, expandedCredential]
  );

  const updateV3Credential = useCallback(
    (id: string, field: keyof SNMPv3Credential, value: string) => {
      setSnmpSettings((prev) => ({
        ...prev,
        v3Credentials: prev.v3Credentials.map((c) => (c.id === id ? { ...c, [field]: value } : c)),
      }));
    },
    [setSnmpSettings]
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
                port: parseInt(e.target.value, 10) || 161,
              }))
            }
            min="1"
            max="65535"
            className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} mt-1 body-small`}
          />
          <p className="caption text-text-muted mt-1">{t("snmp.portDesc")}</p>
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
                timeout: (parseFloat(e.target.value) || 5) * 1000,
              }))
            }
            min="1"
            max="30"
            step="1"
            className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} mt-1 body-small`}
          />
          <p className="caption text-text-muted mt-1">{t("snmp.timeoutDesc")}</p>
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
                retries: parseInt(e.target.value, 10) || 2,
              }))
            }
            min="0"
            max="10"
            className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} mt-1 body-small`}
          />
          <p className="caption text-text-muted mt-1">{t("snmp.retriesDesc")}</p>
        </div>

        {/* Community Strings (v1/v2c) */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="caption text-text-muted font-medium">
              {t("snmp.communityStrings")}
            </span>
            <button
              onClick={() => addCommunity()}
              className="caption text-brand-primary hover:text-brand-accent"
              aria-label="Add community string"
            >
              {t("common.add")}
            </button>
          </div>
          <p className="caption text-text-muted mb-2">{t("snmp.communityDesc")}</p>
          <div className="flex gap-2 mb-2">
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
              className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} flex-1 caption`}
            />
          </div>
          {snmpSettings.communities.map((community, index) => (
            <div key={`${community}-${index}`} className="flex gap-2 mb-2">
              <input
                aria-label={`Community string ${community}`}
                type="text"
                value={community}
                readOnly
                className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.md} flex-1 bg-surface-hover caption`}
              />
              <button
                onClick={() => removeCommunity(community)}
                className="text-status-error hover:text-status-error/80 px-1"
              >
                {t("common.remove")}
              </button>
            </div>
          ))}
        </div>

        {/* SNMPv3 Credentials */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="caption text-text-muted font-medium">{t("snmp.v3Credentials")}</span>
            <button
              onClick={addV3Credential}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          <p className="caption text-text-muted mb-2">{t("snmp.v3CredentialsDesc")}</p>
          {snmpSettings.v3Credentials.map((cred) => (
            <div
              key={cred.id}
              className={`mb-2 border border-surface-border ${radius.md} overflow-hidden`}
            >
              <div
                className="flex items-center justify-between p-2 bg-surface-base cursor-pointer hover:bg-surface-hover"
                onClick={() =>
                  setExpandedCredential(expandedCredential === cred.id ? null : cred.id!)
                }
              >
                <span className="body-small text-text-primary">
                  {cred.name || t("snmp.unnamedCredential")}
                </span>
                <div className="flex items-center gap-2">
                  <span className="caption text-text-muted">
                    {cred.username || t("snmp.noUsername")}
                  </span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      removeV3Credential(cred.id!);
                    }}
                    className="text-status-error hover:text-status-error/80 px-1"
                  >
                    {t("common.remove")}
                  </button>
                </div>
              </div>
              {expandedCredential === cred.id && (
                <div className={`${spacing.pad.sm} bg-surface-hover stack-sm`}>
                  {/* Name */}
                  <div>
                    <label className="caption text-text-muted" htmlFor={`cred-name-${cred.id}`}>
                      {t("common.name")}
                    </label>
                    <input
                      id={`cred-name-${cred.id}`}
                      type="text"
                      value={cred.name}
                      onChange={(e) => updateV3Credential(cred.id!, "name", e.target.value)}
                      placeholder={t("snmp.credentialName")}
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                      onChange={(e) => updateV3Credential(cred.id!, "username", e.target.value)}
                      placeholder={t("snmp.snmpv3Username")}
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                        updateV3Credential(cred.id!, "securityLevel", e.target.value)
                      }
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                      onChange={(e) => updateV3Credential(cred.id!, "authProtocol", e.target.value)}
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                          updateV3Credential(cred.id!, "authPassword", e.target.value)
                        }
                        placeholder={t("snmp.authPasswordPlaceholder")}
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                      onChange={(e) => updateV3Credential(cred.id!, "privProtocol", e.target.value)}
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                          updateV3Credential(cred.id!, "privPassword", e.target.value)
                        }
                        placeholder={t("snmp.privPasswordPlaceholder")}
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
                      onChange={(e) => updateV3Credential(cred.id!, "contextName", e.target.value)}
                      placeholder={t("snmp.snmpContext")}
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
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
