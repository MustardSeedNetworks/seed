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
const AUTH_PROTOCOL_VALUES: string[] = ["", "MD5", "SHA", "SHA224", "SHA256", "SHA384", "SHA512"];
const PRIV_PROTOCOL_VALUES: string[] = ["", "DES", "AES", "AES192", "AES256", "AES192C", "AES256C"];
const SECURITY_LEVEL_VALUES: readonly ["noAuthNoPriv", "authNoPriv", "authPriv"] = [
  "noAuthNoPriv",
  "authNoPriv",
  "authPriv",
] as const;

export const SnmpSettings: React.NamedExoticComponent<SnmpSettingsProps> = memo(
  function snmpSettingsComponent({ snmpSettings, setSnmpSettings, snmpStatus }: SnmpSettingsProps) {
    const { t } = useTranslation("settings");
    const [newCommunity, setNewCommunity] = useState("");
    const [expandedCredential, setExpandedCredential] = useState<string | null>(null);

    // Get translated label for auth protocol
    const getAuthProtocolLabel = (value: string): string => {
      if (value === "") {
        return t("snmp.noAuth");
      }
      return value; // MD5, SHA, etc. don't need translation
    };

    // Get translated label for privacy protocol
    const getPrivProtocolLabel = (value: string): string => {
      if (value === "") {
        return t("snmp.noPrivacy");
      }
      return value; // DES, AES, etc. don't need translation
    };

    // Get translated label for security level
    const getSecurityLevelLabel = (value: string): string => {
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

    const addCommunity = useCallback((): void => {
      if (newCommunity.trim() === "") {
        return;
      }
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
      (community: string): void => {
        setSnmpSettings((prev) => ({
          ...prev,
          communities: prev.communities.filter((c) => c !== community),
        }));
      },
      [setSnmpSettings],
    );

    const addv3Credential = useCallback((): void => {
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
      (id: string): void => {
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
      (id: string, field: keyof Snmpv3Credential, value: string): void => {
        setSnmpSettings((prev) => ({
          ...prev,
          v3Credentials: prev.v3Credentials.map((c) =>
            c.id === id ? { ...c, [field]: value } : c,
          ),
        }));
      },
      [setSnmpSettings],
    );

    return (
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <Server class={iconTokens.size.sm} />
            <span>{t("sections.snmp")}</span>
            <AutoSaveIndicator status={snmpStatus} />
          </div>
        }
      >
        <div class="stack">
          {/* SNMP Port */}
          <div>
            <label class="caption text-text-muted" for="snmp-port">
              {t("snmp.port")}
            </label>
            <input
              id="snmp-port"
              type="number"
              value={snmpSettings.port}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setSnmpSettings((prev) => ({
                  ...prev,
                  port: Number.parseInt(e.target.value, 10) || 161,
                }))
              }
              min="1"
              max="65535"
              class={cn(
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.md,
                spacing.margin.top.tight,
                "body-small",
              )}
            />
            <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
              {t("snmp.portDesc")}
            </p>
          </div>

          {/* Timeout */}
          <div>
            <label class="caption text-text-muted" for="snmp-timeout">
              {t("snmp.timeout")}
            </label>
            <input
              id="snmp-timeout"
              type="number"
              value={snmpSettings.timeout / 1000}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setSnmpSettings((prev) => ({
                  ...prev,
                  timeout: (Number.parseFloat(e.target.value) || 5) * 1000,
                }))
              }
              min="1"
              max="30"
              step="1"
              class={cn(
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.md,
                spacing.margin.top.tight,
                "body-small",
              )}
            />
            <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
              {t("snmp.timeoutDesc")}
            </p>
          </div>

          {/* Retries */}
          <div>
            <label class="caption text-text-muted" for="snmp-retries">
              {t("snmp.retries")}
            </label>
            <input
              id="snmp-retries"
              type="number"
              value={snmpSettings.retries}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setSnmpSettings((prev) => ({
                  ...prev,
                  retries: Number.parseInt(e.target.value, 10) || 2,
                }))
              }
              min="0"
              max="10"
              class={cn(
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.md,
                spacing.margin.top.tight,
                "body-small",
              )}
            />
            <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
              {t("snmp.retriesDesc")}
            </p>
          </div>

          {/* Community Strings (v1/v2c) */}
          <div class={cn("border-t border-surface-border", spacing.padding.top.heading)}>
            <div class={cn("flex items-center justify-between", spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t("snmp.communityStrings")}</span>
              <button
                type="button"
                onClick={(): void => addCommunity()}
                class="caption text-brand-primary hover:text-brand-accent"
                aria-label="Add community string"
              >
                {t("common.add")}
              </button>
            </div>
            <p class={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
              {t("snmp.communityDesc")}
            </p>
            <div class={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}>
              <label class="sr-only" for="snmp-community-new">
                {t("snmp.communityString")}
              </label>
              <input
                id="snmp-community-new"
                type="text"
                value={newCommunity}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setNewCommunity(e.target.value)
                }
                onKeyDown={(e: React.KeyboardEvent): void => {
                  if (e.key === "Enter") {
                    addCommunity();
                  }
                }}
                placeholder={t("snmp.communityString")}
                class={cn(
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
                class={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}
              >
                <input
                  aria-label={`Community string ${community}`}
                  type="text"
                  value={community}
                  readOnly={true}
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.md,
                    "flex-1 bg-surface-hover caption",
                  )}
                />
                <button
                  type="button"
                  onClick={(): void => removeCommunity(community)}
                  class={cn("text-status-error hover:text-status-error/80", spacing.actionBtn)}
                  aria-label={t("common.remove")}
                >
                  {t("common.remove")}
                </button>
              </div>
            ))}
          </div>

          {/* SNMPv3 Credentials */}
          <div class={cn("border-t border-surface-border", spacing.padding.top.heading)}>
            <div class={cn("flex items-center justify-between", spacing.margin.bottom.inline)}>
              <span class="caption text-text-muted font-medium">{t("snmp.v3Credentials")}</span>
              <button
                type="button"
                onClick={addv3Credential}
                class="caption text-brand-primary hover:text-brand-accent"
              >
                {t("common.add")}
              </button>
            </div>
            <p class={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
              {t("snmp.v3CredentialsDesc")}
            </p>
            {snmpSettings.v3Credentials.map((cred) => (
              <div
                key={cred.id}
                class={cn(
                  spacing.margin.bottom.inline,
                  "border border-surface-border",
                  radius.md,
                  "overflow-hidden",
                )}
              >
                {/* biome-ignore lint/a11y/useSemanticElements: Accordion header pattern with nested interactive elements */}
                <div
                  class={cn(
                    "flex items-center justify-between",
                    spacing.pad.xs,
                    "bg-surface-base cursor-pointer hover:bg-surface-hover",
                  )}
                  onClick={(): void =>
                    setExpandedCredential(expandedCredential === cred.id ? null : (cred.id ?? ""))
                  }
                  onKeyDown={(e: React.KeyboardEvent): void => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      setExpandedCredential(
                        expandedCredential === cred.id ? null : (cred.id ?? ""),
                      );
                    }
                  }}
                  role="button"
                  tabIndex={0}
                >
                  <span class="body-small text-text-primary">
                    {cred.name || t("snmp.unnamedCredential")}
                  </span>
                  <div class={cn("flex items-center", spacing.gap.compact)}>
                    <span class="caption text-text-muted">
                      {cred.username || t("snmp.noUsername")}
                    </span>
                    <button
                      type="button"
                      onClick={(e: React.MouseEvent): void => {
                        e.stopPropagation();
                        removev3Credential(cred.id ?? "");
                      }}
                      class={cn("text-status-error hover:text-status-error/80", spacing.actionBtn)}
                      aria-label={t("common.remove")}
                    >
                      {t("common.remove")}
                    </button>
                  </div>
                </div>
                {expandedCredential === cred.id && (
                  <div class={cn(spacing.pad.sm, "bg-surface-hover stack-sm")}>
                    {/* Name */}
                    <div>
                      <label class="caption text-text-muted" for={`cred-name-${cred.id}`}>
                        {t("common.name")}
                      </label>
                      <input
                        id={`cred-name-${cred.id}`}
                        type="text"
                        value={cred.name}
                        onChange={(
                          e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                        ): void => updatev3Credential(cred.id ?? "", "name", e.target.value)}
                        placeholder={t("snmp.credentialName")}
                        class={cn(
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
                      <label class="caption text-text-muted" for={`cred-username-${cred.id}`}>
                        {t("snmp.username")}
                      </label>
                      <input
                        id={`cred-username-${cred.id}`}
                        type="text"
                        value={cred.username}
                        onChange={(
                          e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                        ): void => updatev3Credential(cred.id ?? "", "username", e.target.value)}
                        placeholder={t("snmp.snmpv3Username")}
                        class={cn(
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
                      <label class="caption text-text-muted" for={`sec-level-${cred.id}`}>
                        {t("snmp.securityLevel")}
                      </label>
                      <select
                        id={`sec-level-${cred.id}`}
                        value={cred.securityLevel}
                        onChange={(
                          e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                        ): void =>
                          updatev3Credential(cred.id ?? "", "securityLevel", e.target.value)
                        }
                        class={cn(
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
                      <label class="caption text-text-muted" for={`auth-proto-${cred.id}`}>
                        {t("snmp.authProtocol")}
                      </label>
                      <select
                        id={`auth-proto-${cred.id}`}
                        value={cred.authProtocol}
                        onChange={(
                          e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                        ): void =>
                          updatev3Credential(cred.id ?? "", "authProtocol", e.target.value)
                        }
                        class={cn(
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
                    {cred.authProtocol !== "" ? (
                      <div>
                        <label class="caption text-text-muted" for={`auth-pass-${cred.id}`}>
                          {t("snmp.authPassword")}
                        </label>
                        <input
                          id={`auth-pass-${cred.id}`}
                          type="password"
                          value={cred.authPassword}
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                          ): void =>
                            updatev3Credential(cred.id ?? "", "authPassword", e.target.value)
                          }
                          placeholder={t("snmp.authPasswordPlaceholder")}
                          class={cn(
                            inputTokens.base,
                            inputTokens.state.default,
                            inputTokens.size.sm,
                            spacing.margin.top.tight,
                            "caption",
                          )}
                        />
                      </div>
                    ) : null}

                    {/* Privacy Protocol */}
                    <div>
                      <label class="caption text-text-muted" for={`priv-proto-${cred.id}`}>
                        {t("snmp.privProtocol")}
                      </label>
                      <select
                        id={`priv-proto-${cred.id}`}
                        value={cred.privProtocol}
                        onChange={(
                          e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                        ): void =>
                          updatev3Credential(cred.id ?? "", "privProtocol", e.target.value)
                        }
                        class={cn(
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
                    {cred.privProtocol !== "" ? (
                      <div>
                        <label class="caption text-text-muted" for={`priv-pass-${cred.id}`}>
                          {t("snmp.privPassword")}
                        </label>
                        <input
                          id={`priv-pass-${cred.id}`}
                          type="password"
                          value={cred.privPassword}
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                          ): void =>
                            updatev3Credential(cred.id ?? "", "privPassword", e.target.value)
                          }
                          placeholder={t("snmp.privPasswordPlaceholder")}
                          class={cn(
                            inputTokens.base,
                            inputTokens.state.default,
                            inputTokens.size.sm,
                            spacing.margin.top.tight,
                            "caption",
                          )}
                        />
                      </div>
                    ) : null}

                    {/* Context Name */}
                    <div>
                      <label class="caption text-text-muted" for={`context-name-${cred.id}`}>
                        {t("snmp.contextName")}
                      </label>
                      <input
                        id={`context-name-${cred.id}`}
                        type="text"
                        value={cred.contextName}
                        onChange={(
                          e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                        ): void => updatev3Credential(cred.id ?? "", "contextName", e.target.value)}
                        placeholder={t("snmp.snmpContext")}
                        class={cn(
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
  },
);
