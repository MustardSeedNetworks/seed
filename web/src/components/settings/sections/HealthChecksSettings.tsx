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

interface HealthChecksSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
}

export function HealthChecksSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
}: HealthChecksSettingsProps) {
  // Ping target helpers
  const addPingTarget = () => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: [
        ...prev.pingTargets,
        { name: "", host: "", enabled: true, count: 3 },
      ],
    }));
  };

  const removePingTarget = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: prev.pingTargets.filter((_, i) => i !== index),
    }));
  };

  const updatePingTarget = (
    index: number,
    field: keyof PingTarget,
    value: string | boolean | number,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: prev.pingTargets.map((t, i) =>
        i === index ? { ...t, [field]: value } : t,
      ),
    }));
  };

  // TCP port helpers
  const addTCPPort = () => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: [
        ...prev.tcpPorts,
        { name: "", host: "", port: 80, enabled: true },
      ],
    }));
  };

  const removeTCPPort = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: prev.tcpPorts.filter((_, i) => i !== index),
    }));
  };

  const updateTCPPort = (
    index: number,
    field: keyof TCPPort,
    value: string | boolean | number,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: prev.tcpPorts.map((p, i) =>
        i === index ? { ...p, [field]: value } : p,
      ),
    }));
  };

  // UDP port helpers
  const addUDPPort = () => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: [
        ...prev.udpPorts,
        { name: "", host: "", port: 53, enabled: true },
      ],
    }));
  };

  const removeUDPPort = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: prev.udpPorts.filter((_, i) => i !== index),
    }));
  };

  const updateUDPPort = (
    index: number,
    field: keyof UDPPort,
    value: string | boolean | number,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: prev.udpPorts.map((p, i) =>
        i === index ? { ...p, [field]: value } : p,
      ),
    }));
  };

  // HTTP endpoint helpers
  const addHTTPEndpoint = () => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: [
        ...prev.httpEndpoints,
        { name: "", url: "", expectedStatus: 200, enabled: true },
      ],
    }));
  };

  const removeHTTPEndpoint = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: prev.httpEndpoints.filter((_, i) => i !== index),
    }));
  };

  const updateHTTPEndpoint = (
    index: number,
    field: keyof HTTPEndpoint,
    value: string | boolean | number,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: prev.httpEndpoints.map((e, i) =>
        i === index ? { ...e, [field]: value } : e,
      ),
    }));
  };

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
          {testsSettings.pingTargets.map((target, idx) => (
            <div key={idx} className="flex gap-2 mb-2">
              <input
                type="text"
                value={target.name}
                onChange={(e) => updatePingTarget(idx, "name", e.target.value)}
                placeholder="Name"
                className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="text"
                value={target.host}
                onChange={(e) => updatePingTarget(idx, "host", e.target.value)}
                placeholder="Host/IP"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="number"
                value={target.count || 3}
                onChange={(e) =>
                  updatePingTarget(idx, "count", parseInt(e.target.value) || 3)
                }
                min={1}
                max={10}
                title="Number of pings"
                className="w-14 px-2 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary text-center"
              />
              <button
                onClick={() => removePingTarget(idx)}
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
          {testsSettings.tcpPorts.map((port, idx) => (
            <div key={idx} className="flex gap-2 mb-2">
              <input
                type="text"
                value={port.name}
                onChange={(e) => updateTCPPort(idx, "name", e.target.value)}
                placeholder="Name"
                className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="text"
                value={port.host}
                onChange={(e) => updateTCPPort(idx, "host", e.target.value)}
                placeholder="Host"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="number"
                value={port.port}
                onChange={(e) =>
                  updateTCPPort(idx, "port", parseInt(e.target.value) || 80)
                }
                placeholder="Port"
                className="w-20 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <button
                onClick={() => removeTCPPort(idx)}
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
          {testsSettings.udpPorts.map((port, idx) => (
            <div key={idx} className="flex gap-2 mb-2">
              <input
                type="text"
                value={port.name}
                onChange={(e) => updateUDPPort(idx, "name", e.target.value)}
                placeholder="Name"
                className="w-24 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="text"
                value={port.host}
                onChange={(e) => updateUDPPort(idx, "host", e.target.value)}
                placeholder="Host"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <input
                type="number"
                value={port.port}
                onChange={(e) =>
                  updateUDPPort(idx, "port", parseInt(e.target.value) || 53)
                }
                placeholder="Port"
                className="w-20 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <button
                onClick={() => removeUDPPort(idx)}
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
          {testsSettings.httpEndpoints.map((endpoint, idx) => (
            <div
              key={idx}
              className="space-y-1 mb-3 p-2 bg-surface-base rounded border border-surface-border"
            >
              <div className="flex gap-2">
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e) =>
                    updateHTTPEndpoint(idx, "name", e.target.value)
                  }
                  placeholder="Name"
                  className="flex-1 px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                />
                <input
                  type="number"
                  value={endpoint.expectedStatus}
                  onChange={(e) =>
                    updateHTTPEndpoint(
                      idx,
                      "expectedStatus",
                      parseInt(e.target.value) || 200,
                    )
                  }
                  placeholder="Status"
                  className="w-20 px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                />
                <button
                  onClick={() => removeHTTPEndpoint(idx)}
                  className="text-status-error hover:text-status-error/80 px-1"
                >
                  x
                </button>
              </div>
              <input
                type="text"
                value={endpoint.url}
                onChange={(e) => updateHTTPEndpoint(idx, "url", e.target.value)}
                placeholder="https://example.com/health"
                className="w-full px-2.5 py-2 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
              />
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
}
