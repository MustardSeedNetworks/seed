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
        <div className="flex items-center gap-2">
          <Server className="w-4 h-4" />
          <span>SNMP</span>
          <AutoSaveIndicator status={snmpStatus} />
        </div>
      }
    >
      <div className="space-y-4">
        {/* SNMP Port */}
        <div>
          <label className="text-xs text-text-muted">SNMP Port</label>
          <input
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
            className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          />
          <p className="text-xs text-text-muted mt-1">
            UDP port for SNMP queries (default: 161)
          </p>
        </div>

        {/* Timeout */}
        <div>
          <label className="text-xs text-text-muted">Timeout (seconds)</label>
          <input
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
            className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          />
          <p className="text-xs text-text-muted mt-1">
            SNMP query timeout (default: 5 seconds)
          </p>
        </div>

        {/* Retries */}
        <div>
          <label className="text-xs text-text-muted">Retries</label>
          <input
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
            className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          />
          <p className="text-xs text-text-muted mt-1">
            Number of retries on timeout (default: 2)
          </p>
        </div>

        {/* Community Strings (v1/v2c) */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-text-muted font-medium">
              Community Strings (v1/v2c)
            </span>
            <button
              onClick={() => addCommunity()}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="text-xs text-text-muted mb-2">
            SNMP v1/v2c community strings (e.g., "public", "private")
          </p>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newCommunity}
              onChange={(e) => setNewCommunity(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") addCommunity();
              }}
              placeholder="Community string"
              className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
            />
          </div>
          {snmpSettings.communities.map((community, index) => (
            <div key={`${community}-${index}`} className="flex gap-2 mb-2">
              <input
                type="text"
                value={community}
                readOnly
                className="flex-1 px-2.5 py-2 bg-surface-hover border border-surface-border rounded text-xs text-text-primary"
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
            <span className="text-xs text-text-muted font-medium">
              SNMPv3 Credentials
            </span>
            <button
              onClick={addV3Credential}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="text-xs text-text-muted mb-2">
            SNMPv3 user credentials with authentication and privacy
          </p>
          {snmpSettings.v3Credentials.map((cred) => (
            <div
              key={cred.id}
              className="mb-2 border border-surface-border rounded overflow-hidden"
            >
              <div
                className="flex items-center justify-between p-2 bg-surface-base cursor-pointer hover:bg-surface-hover"
                onClick={() =>
                  setExpandedCredential(
                    expandedCredential === cred.id ? null : cred.id!,
                  )
                }
              >
                <span className="text-sm text-text-primary">
                  {cred.name || "Unnamed Credential"}
                </span>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-text-muted">
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
                <div className="p-3 bg-surface-hover space-y-3">
                  {/* Name */}
                  <div>
                    <label className="text-xs text-text-muted">Name</label>
                    <input
                      type="text"
                      value={cred.name}
                      onChange={(e) =>
                        updateV3Credential(cred.id!, "name", e.target.value)
                      }
                      placeholder="Credential name"
                      className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                  </div>

                  {/* Username */}
                  <div>
                    <label className="text-xs text-text-muted">Username</label>
                    <input
                      type="text"
                      value={cred.username}
                      onChange={(e) =>
                        updateV3Credential(cred.id!, "username", e.target.value)
                      }
                      placeholder="SNMPv3 username"
                      className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                  </div>

                  {/* Security Level */}
                  <div>
                    <label className="text-xs text-text-muted">
                      Security Level
                    </label>
                    <select
                      value={cred.securityLevel}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "securityLevel",
                          e.target.value,
                        )
                      }
                      className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
                    <label className="text-xs text-text-muted">
                      Auth Protocol
                    </label>
                    <select
                      value={cred.authProtocol}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "authProtocol",
                          e.target.value,
                        )
                      }
                      className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
                      <label className="text-xs text-text-muted">
                        Auth Password
                      </label>
                      <input
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
                        className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                      />
                    </div>
                  )}

                  {/* Privacy Protocol */}
                  <div>
                    <label className="text-xs text-text-muted">
                      Privacy Protocol
                    </label>
                    <select
                      value={cred.privProtocol}
                      onChange={(e) =>
                        updateV3Credential(
                          cred.id!,
                          "privProtocol",
                          e.target.value,
                        )
                      }
                      className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
                      <label className="text-xs text-text-muted">
                        Privacy Password
                      </label>
                      <input
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
                        className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                      />
                    </div>
                  )}

                  {/* Context Name */}
                  <div>
                    <label className="text-xs text-text-muted">
                      Context Name (optional)
                    </label>
                    <input
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
                      className="w-full mt-1 px-2.5 py-1.5 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
