import { memo, useCallback } from "react";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import {
  TestsSettings,
  SaveStatus,
  PingTarget,
  TCPPort,
  UDPPort,
  HTTPEndpoint,
} from "../../../types/settings";
import { generateId } from "../../../utils/id";

interface HealthChecksSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
}

export const HealthChecksSettings = memo(function HealthChecksSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
}: HealthChecksSettingsProps) {
  // Ping target helpers
  const addPingTarget = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: [
        ...prev.pingTargets,
        { id: generateId(), name: "", host: "", enabled: true, count: 3 },
      ],
    }));
  }, [setTestsSettings]);

  const removePingTarget = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        pingTargets: prev.pingTargets.filter((t) => t.id !== id),
      }));
    },
    [setTestsSettings],
  );

  const updatePingTarget = useCallback(
    (id: string, field: keyof PingTarget, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        pingTargets: prev.pingTargets.map((t) =>
          t.id === id ? { ...t, [field]: value } : t,
        ),
      }));
    },
    [setTestsSettings],
  );

  // TCP port helpers
  const addTCPPort = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: [
        ...prev.tcpPorts,
        { id: generateId(), name: "", host: "", port: 80, enabled: true },
      ],
    }));
  }, [setTestsSettings]);

  const removeTCPPort = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        tcpPorts: prev.tcpPorts.filter((p) => p.id !== id),
      }));
    },
    [setTestsSettings],
  );

  const updateTCPPort = useCallback(
    (id: string, field: keyof TCPPort, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        tcpPorts: prev.tcpPorts.map((p) =>
          p.id === id ? { ...p, [field]: value } : p,
        ),
      }));
    },
    [setTestsSettings],
  );

  // UDP port helpers
  const addUDPPort = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: [
        ...prev.udpPorts,
        { id: generateId(), name: "", host: "", port: 53, enabled: true },
      ],
    }));
  }, [setTestsSettings]);

  const removeUDPPort = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        udpPorts: prev.udpPorts.filter((p) => p.id !== id),
      }));
    },
    [setTestsSettings],
  );

  const updateUDPPort = useCallback(
    (id: string, field: keyof UDPPort, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        udpPorts: prev.udpPorts.map((p) =>
          p.id === id ? { ...p, [field]: value } : p,
        ),
      }));
    },
    [setTestsSettings],
  );

  // HTTP endpoint helpers
  const addHTTPEndpoint = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: [
        ...prev.httpEndpoints,
        {
          id: generateId(),
          name: "",
          url: "",
          expectedStatus: 200,
          enabled: true,
        },
      ],
    }));
  }, [setTestsSettings]);

  const removeHTTPEndpoint = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        httpEndpoints: prev.httpEndpoints.filter((e) => e.id !== id),
      }));
    },
    [setTestsSettings],
  );

  const updateHTTPEndpoint = useCallback(
    (
      id: string,
      field: keyof HTTPEndpoint,
      value: string | boolean | number,
    ) => {
      setTestsSettings((prev) => ({
        ...prev,
        httpEndpoints: prev.httpEndpoints.map((e) =>
          e.id === id ? { ...e, [field]: value } : e,
        ),
      }));
    },
    [setTestsSettings],
  );

  return (
    <CollapsibleSection
      title={
        <>
          Health Checks
          <AutoSaveIndicator status={testsStatus} />
        </>
      }
    >
      <div className="space-y-4">
        {/* Ping Targets */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-text-muted font-medium">
              Ping Targets
            </span>
            <button
              onClick={addPingTarget}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="text-xs text-text-muted mb-2">
            Default: 3 pings per target
          </p>
          {testsSettings.pingTargets.map((target) => (
            <div key={target.id || target.host} className="flex gap-2 mb-2">
              <input
                type="text"
                value={target.name}
                onChange={(e) =>
                  updatePingTarget(target.id!, "name", e.target.value)
                }
                placeholder="Name"
                className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="text"
                value={target.host}
                onChange={(e) =>
                  updatePingTarget(target.id!, "host", e.target.value)
                }
                placeholder="Host/IP"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="number"
                value={target.count || 3}
                onChange={(e) =>
                  updatePingTarget(
                    target.id!,
                    "count",
                    parseInt(e.target.value) || 3,
                  )
                }
                min={1}
                max={10}
                title="Number of pings"
                className="w-14 px-2 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary text-center"
              />
              <button
                onClick={() => removePingTarget(target.id!)}
                className="text-status-error hover:text-status-error/80 px-1"
              >
                x
              </button>
            </div>
          ))}
        </div>

        {/* TCP Ports */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-text-muted font-medium">
              TCP Port Tests
            </span>
            <button
              onClick={addTCPPort}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          {testsSettings.tcpPorts.map((port) => (
            <div
              key={port.id || `${port.host}:${port.port}`}
              className="flex gap-2 mb-2"
            >
              <input
                type="text"
                value={port.name}
                onChange={(e) =>
                  updateTCPPort(port.id!, "name", e.target.value)
                }
                placeholder="Name"
                className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="text"
                value={port.host}
                onChange={(e) =>
                  updateTCPPort(port.id!, "host", e.target.value)
                }
                placeholder="Host"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="number"
                value={port.port}
                onChange={(e) =>
                  updateTCPPort(
                    port.id!,
                    "port",
                    parseInt(e.target.value) || 80,
                  )
                }
                placeholder="Port"
                className="w-20 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <button
                onClick={() => removeTCPPort(port.id!)}
                className="text-status-error hover:text-status-error/80 px-1"
              >
                x
              </button>
            </div>
          ))}
        </div>

        {/* UDP Ports */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-text-muted font-medium">
              UDP Port Tests
            </span>
            <button
              onClick={addUDPPort}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="text-xs text-text-muted mb-2">
            Test UDP services (DNS:53, NTP:123, etc.)
          </p>
          {testsSettings.udpPorts.map((port) => (
            <div
              key={port.id || `${port.host}:${port.port}`}
              className="flex gap-2 mb-2"
            >
              <input
                type="text"
                value={port.name}
                onChange={(e) =>
                  updateUDPPort(port.id!, "name", e.target.value)
                }
                placeholder="Name"
                className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="text"
                value={port.host}
                onChange={(e) =>
                  updateUDPPort(port.id!, "host", e.target.value)
                }
                placeholder="Host"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="number"
                value={port.port}
                onChange={(e) =>
                  updateUDPPort(
                    port.id!,
                    "port",
                    parseInt(e.target.value) || 53,
                  )
                }
                placeholder="Port"
                className="w-20 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <button
                onClick={() => removeUDPPort(port.id!)}
                className="text-status-error hover:text-status-error/80 px-1"
              >
                x
              </button>
            </div>
          ))}
        </div>

        {/* HTTP Endpoints */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-text-muted font-medium">
              HTTP Endpoints
            </span>
            <button
              onClick={addHTTPEndpoint}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          {testsSettings.httpEndpoints.map((endpoint) => (
            <div
              key={endpoint.id || endpoint.url}
              className="space-y-1 mb-3 p-2 bg-surface-base rounded border border-surface-border"
            >
              <div className="flex gap-2">
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e) =>
                    updateHTTPEndpoint(endpoint.id!, "name", e.target.value)
                  }
                  placeholder="Name"
                  className="flex-1 px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                />
                <input
                  type="number"
                  value={endpoint.expectedStatus}
                  onChange={(e) =>
                    updateHTTPEndpoint(
                      endpoint.id!,
                      "expectedStatus",
                      parseInt(e.target.value) || 200,
                    )
                  }
                  placeholder="Status"
                  className="w-20 px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                />
                <button
                  onClick={() => removeHTTPEndpoint(endpoint.id!)}
                  className="text-status-error hover:text-status-error/80 px-1"
                >
                  x
                </button>
              </div>
              <input
                type="text"
                value={endpoint.url}
                onChange={(e) =>
                  updateHTTPEndpoint(endpoint.id!, "url", e.target.value)
                }
                placeholder="https://example.com/health"
                className="w-full px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
              />
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
});
