import { memo, useCallback, useState } from "react";
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
} from "../../../styles/theme";

interface SNMPSettingsProps {
  snmpSettings: SNMPSettingsType;
  setSnmpSettings: React.Dispatch<React.SetStateAction<SNMPSettingsType>>;
  snmpStatus: SaveStatus;
}

const AUTH_PROTOCOLS = [
  { value: "", label: "No Auth" },
  { value: "MD5", label: "MD5" },
  { value: "SHA", label: "SHA" },
  { value: "SHA224", label: "SHA224" },
  { value: "SHA256", label: "SHA256" },
  { value: "SHA384", label: "SHA384" },
  { value: "SHA512", label: "SHA512" },
];

const PRIV_PROTOCOLS = [
  { value: "", label: "No Privacy" },
  { value: "DES", label: "DES" },
  { value: "AES", label: "AES" },
  { value: "AES192", label: "AES192" },
  { value: "AES256", label: "AES256" },
  { value: "AES192C", label: "AES192C" },
  { value: "AES256C", label: "AES256C" },
];

const SECURITY_LEVELS = [
  { value: "noAuthNoPriv", label: "No Auth / No Privacy" },
  { value: "authNoPriv", label: "Auth / No Privacy" },
  { value: "authPriv", label: "Auth / Privacy" },
];

export const SNMPSettings = memo(function SNMPSettings({
  snmpSettings,
  setSnmpSettings,
  snmpStatus,
}: SNMPSettingsProps) {
  const [newCommunity, setNewCommunity] = useState("");
  const [expandedCredential, setExpandedCredential] = useState<string | null>(
    null,
  );

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

  const addV3Credential = useCallback(() => {
    const newCred: SNMPv3Credential = {
      id: generateId(),
      name: "New Credential",
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
  }, [setSnmpSettings]);

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
    [setSnmpSettings, expandedCredential],
  );

  const updateV3Credential = useCallback(
    (id: string, field: keyof SNMPv3Credential, value: string) => {
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
        <div className={layout.inline.default}>
          <Server className={iconTokens.size.sm} />
          <span>SNMP</span>
          <AutoSaveIndicator status={snmpStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* SNMP Port */}
        <div>
          <label className="caption text-text-muted" htmlFor="snmp-port">
            SNMP Port
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
          <p className="caption text-text-muted mt-1">
            UDP port for SNMP queries (default: 161)
          </p>
        </div>

        {/* Timeout */}
        <div>
          <label className="caption text-text-muted" htmlFor="snmp-timeout">
            Timeout (seconds)
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
          <p className="caption text-text-muted mt-1">
            SNMP query timeout (default: 5 seconds)
          </p>
        </div>

        {/* Retries */}
        <div>
          <label className="caption text-text-muted" htmlFor="snmp-retries">
            Retries
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
          <p className="caption text-text-muted mt-1">
            Number of retries on timeout (default: 2)
          </p>
        </div>

        {/* Community Strings (v1/v2c) */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="caption text-text-muted font-medium">
              Community Strings (v1/v2c)
            </span>
            <button
              onClick={() => addCommunity()}
              className="caption text-brand-primary hover:text-brand-accent"
              aria-label="Add community string"
            >
              + Add
            </button>
          </div>
          <p className="caption text-text-muted mb-2">
            SNMP v1/v2c community strings (e.g., "public", "private")
          </p>
          <div className="flex gap-2 mb-2">
            <label className="sr-only" htmlFor="snmp-community-new">
              New community string
            </label>
            <input
              id="snmp-community-new"
              type="text"
              value={newCommunity}
              onChange={(e) => setNewCommunity(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") addCommunity();
              }}
              placeholder="Community string"
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
                x
              </button>
            </div>
          ))}
        </div>

        {/* SNMPv3 Credentials */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="caption text-text-muted font-medium">
              SNMPv3 Credentials
            </span>
            <button
              onClick={addV3Credential}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="caption text-text-muted mb-2">
            SNMPv3 user credentials with authentication and privacy
          </p>
          {snmpSettings.v3Credentials.map((cred) => (
            <div
              key={cred.id}
              className={`mb-2 border border-surface-border ${radius.md} overflow-hidden`}
            >
              <div
                className="flex items-center justify-between p-2 bg-surface-base cursor-pointer hover:bg-surface-hover"
                onClick={() =>
                  setExpandedCredential(
                    expandedCredential === cred.id ? null : cred.id!,
                  )
                }
              >
                <span className="body-small text-text-primary">
                  {cred.name || "Unnamed Credential"}
                </span>
                <div className="flex items-center gap-2">
                  <span className="caption text-text-muted">
                    {cred.username || "(no username)"}
                  </span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      removeV3Credential(cred.id!);
                    }}
                    className="text-status-error hover:text-status-error/80 px-1"
                  >
                    x
                  </button>
                </div>
              </div>
              {expandedCredential === cred.id && (
                <div className="p-3 bg-surface-hover stack-sm">
                  {/* Name */}
                  <div>
                    <label
                      className="caption text-text-muted"
                      htmlFor={`cred-name-${cred.id}`}
                    >
                      Name
                    </label>
                    <input
                      id={`cred-name-${cred.id}`}
                      type="text"
                      value={cred.name}
                      onChange={(e) =>
                        updateV3Credential(cred.id!, "name", e.target.value)
                      }
                      placeholder="Credential name"
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                    />
                  </div>

                  {/* Username */}
                  <div>
                    <label
                      className="caption text-text-muted"
                      htmlFor={`cred-username-${cred.id}`}
                    >
                      Username
                    </label>
                    <input
                      id={`cred-username-${cred.id}`}
                      type="text"
                      value={cred.username}
                      onChange={(e) =>
                        updateV3Credential(cred.id!, "username", e.target.value)
                      }
                      placeholder="SNMPv3 username"
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                    />
                  </div>

                  {/* Security Level */}
                  <div>
                    <label
                      className="caption text-text-muted"
                      htmlFor={`sec-level-${cred.id}`}
                    >
                      Security Level
                    </label>
                    <select
                      id={`sec-level-${cred.id}`}
                      value={cred.securityLevel}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "securityLevel",
                          e.target.value,
                        )
                      }
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                    >
                      {SECURITY_LEVELS.map((level) => (
                        <option key={level.value} value={level.value}>
                          {level.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Authentication Protocol */}
                  <div>
                    <label
                      className="caption text-text-muted"
                      htmlFor={`auth-proto-${cred.id}`}
                    >
                      Auth Protocol
                    </label>
                    <select
                      id={`auth-proto-${cred.id}`}
                      value={cred.authProtocol}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "authProtocol",
                          e.target.value,
                        )
                      }
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                    >
                      {AUTH_PROTOCOLS.map((proto) => (
                        <option key={proto.value} value={proto.value}>
                          {proto.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Authentication Password */}
                  {cred.authProtocol !== "" && (
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`auth-pass-${cred.id}`}
                      >
                        Auth Password
                      </label>
                      <input
                        id={`auth-pass-${cred.id}`}
                        type="password"
                        value={cred.authPassword}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "authPassword",
                            e.target.value,
                          )
                        }
                        placeholder="Authentication password"
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                      />
                    </div>
                  )}

                  {/* Privacy Protocol */}
                  <div>
                    <label
                      className="caption text-text-muted"
                      htmlFor={`priv-proto-${cred.id}`}
                    >
                      Privacy Protocol
                    </label>
                    <select
                      id={`priv-proto-${cred.id}`}
                      value={cred.privProtocol}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "privProtocol",
                          e.target.value,
                        )
                      }
                      className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                    >
                      {PRIV_PROTOCOLS.map((proto) => (
                        <option key={proto.value} value={proto.value}>
                          {proto.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Privacy Password */}
                  {cred.privProtocol !== "" && (
                    <div>
                      <label
                        className="caption text-text-muted"
                        htmlFor={`priv-pass-${cred.id}`}
                      >
                        Privacy Password
                      </label>
                      <input
                        id={`priv-pass-${cred.id}`}
                        type="password"
                        value={cred.privPassword}
                        onChange={(e) =>
                          updateV3Credential(
                            cred.id!,
                            "privPassword",
                            e.target.value,
                          )
                        }
                        placeholder="Privacy password"
                        className={`${inputTokens.base} ${inputTokens.state.default} ${inputTokens.size.sm} mt-1 caption`}
                      />
                    </div>
                  )}

                  {/* Context Name */}
                  <div>
                    <label
                      className="caption text-text-muted"
                      htmlFor={`context-name-${cred.id}`}
                    >
                      Context Name (optional)
                    </label>
                    <input
                      id={`context-name-${cred.id}`}
                      type="text"
                      value={cred.contextName}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "contextName",
                          e.target.value,
                        )
                      }
                      placeholder="SNMP context"
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
