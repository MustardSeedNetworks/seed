import { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../../hooks/useTheme';
import { getAuthHeaders } from '../../hooks/useAuth';
import { CollapsibleSection } from '../ui/CollapsibleSection';

const API_BASE = import.meta.env.VITE_API_BASE || '';

interface Thresholds {
  dns: {
    good: number;
    warning: number;
  };
  gateway: {
    good: number;
    warning: number;
  };
  wifi: {
    good: number;
    warning: number;
  };
  customPing: {
    good: number;
    warning: number;
  };
  customTcp: {
    good: number;
    warning: number;
  };
  customHttp: {
    good: number;
    warning: number;
  };
}

interface WiFiSettings {
  interface: string;
  availableWifi: string[];
  isWireless: boolean;
}

interface IPSettings {
  mode: 'dhcp' | 'static';
  address: string;
  netmask: string;
  gateway: string;
  dns: string[];
}

interface PingTarget {
  name: string;
  host: string;
  enabled: boolean;
}

interface TCPPort {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

interface UDPPort {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

interface HTTPEndpoint {
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
}

interface TestsSettings {
  dnsHostname: string;
  pingTargets: PingTarget[];
  tcpPorts: TCPPort[];
  udpPorts: UDPPort[];
  httpEndpoints: HTTPEndpoint[];
  speedtest: {
    serverId: string;
    autoRunOnLink: boolean;
  };
}

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsDrawer({ isOpen, onClose }: SettingsDrawerProps) {
  const { theme, setTheme, isDark } = useTheme();
  const [thresholds, setThresholds] = useState<Thresholds>({
    dns: { good: 50, warning: 100 },
    gateway: { good: 20, warning: 50 },
    wifi: { good: -50, warning: -70 },
    customPing: { good: 50, warning: 100 },
    customTcp: { good: 100, warning: 500 },
    customHttp: { good: 500, warning: 2000 },
  });
  const [ipSettings, setIPSettings] = useState<IPSettings>({
    mode: 'dhcp',
    address: '',
    netmask: '24',
    gateway: '',
    dns: [],
  });
  const [testsSettings, setTestsSettings] = useState<TestsSettings>({
    dnsHostname: 'google.com',
    pingTargets: [],
    tcpPorts: [],
    udpPorts: [],
    httpEndpoints: [],
    speedtest: {
      serverId: '',
      autoRunOnLink: false,
    },
  });
  const [wifiSettings, setWifiSettings] = useState<WiFiSettings>({
    interface: '',
    availableWifi: [],
    isWireless: false,
  });
  const [dnsInput, setDnsInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [savingIP, setSavingIP] = useState(false);
  const [savingTests, setSavingTests] = useState(false);
  const [savingWifi, setSavingWifi] = useState(false);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [ipMessage, setIPMessage] = useState<string | null>(null);
  const [testsMessage, setTestsMessage] = useState<string | null>(null);
  const [wifiMessage, setWifiMessage] = useState<string | null>(null);

  // Fetch current thresholds
  const fetchThresholds = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        if (data.thresholds) {
          setThresholds((prev) => ({
            ...prev,
            ...data.thresholds,
          }));
        }
      }
    } catch (err) {
      console.error('Failed to fetch thresholds:', err);
    }
  }, []);

  // Fetch current IP settings
  const fetchIPSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setIPSettings({
          mode: data.mode || 'dhcp',
          address: data.address || '',
          netmask: data.netmask || '24',
          gateway: data.gateway || '',
          dns: data.dns || [],
        });
        setDnsInput((data.dns || []).join(', '));
      }
    } catch (err) {
      console.error('Failed to fetch IP settings:', err);
    }
  }, []);

  // Fetch current tests settings
  const fetchTestsSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/tests/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setTestsSettings({
          dnsHostname: data.dnsHostname || 'google.com',
          pingTargets: data.pingTargets || [],
          tcpPorts: data.tcpPorts || [],
          udpPorts: data.udpPorts || [],
          httpEndpoints: data.httpEndpoints || [],
          speedtest: {
            serverId: data.speedtest?.serverId || '',
            autoRunOnLink: data.speedtest?.autoRunOnLink || false,
          },
        });
      }
    } catch (err) {
      console.error('Failed to fetch tests settings:', err);
    }
  }, []);

  // Fetch WiFi settings
  const fetchWifiSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/wifi/settings`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setWifiSettings({
          interface: data.interface || '',
          availableWifi: data.availableWifi || [],
          isWireless: data.isWireless || false,
        });
      }
    } catch (err) {
      console.error('Failed to fetch WiFi settings:', err);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      fetchThresholds();
      fetchIPSettings();
      fetchTestsSettings();
      fetchWifiSettings();
    }
  }, [isOpen, fetchThresholds, fetchIPSettings, fetchTestsSettings, fetchWifiSettings]);

  const saveThresholds = async () => {
    setSaving(true);
    setSaveMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ thresholds }),
      });
      if (response.ok) {
        setSaveMessage('Thresholds saved');
        setTimeout(() => setSaveMessage(null), 2000);
      } else {
        setSaveMessage('Failed to save');
      }
    } catch (err) {
      setSaveMessage('Error saving settings');
    } finally {
      setSaving(false);
    }
  };

  const updateThreshold = (
    category: keyof Thresholds,
    level: 'good' | 'warning',
    value: number
  ) => {
    setThresholds((prev) => ({
      ...prev,
      [category]: {
        ...prev[category],
        [level]: value,
      },
    }));
  };

  const saveIPSettings = async () => {
    setSavingIP(true);
    setIPMessage(null);
    try {
      // Parse DNS from input
      const dns = dnsInput
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s.length > 0);

      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          mode: ipSettings.mode,
          address: ipSettings.address,
          netmask: ipSettings.netmask,
          gateway: ipSettings.gateway,
          dns,
        }),
      });
      if (response.ok) {
        setIPMessage('IP settings applied');
        setTimeout(() => setIPMessage(null), 3000);
      } else {
        const error = await response.text();
        setIPMessage(`Failed: ${error}`);
      }
    } catch (err) {
      setIPMessage('Error applying IP settings');
    } finally {
      setSavingIP(false);
    }
  };

  const saveTestsSettings = async () => {
    setSavingTests(true);
    setTestsMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/tests/settings`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(testsSettings),
      });
      if (response.ok) {
        setTestsMessage('Tests settings saved');
        setTimeout(() => setTestsMessage(null), 2000);
        // Dispatch event to notify Health Checks card to refresh
        window.dispatchEvent(new CustomEvent('healthChecksUpdated'));
      } else {
        const error = await response.text();
        setTestsMessage(`Failed: ${error}`);
      }
    } catch (err) {
      setTestsMessage('Error saving tests settings');
    } finally {
      setSavingTests(false);
    }
  };

  const saveWifiSettings = async () => {
    setSavingWifi(true);
    setWifiMessage(null);
    try {
      const response = await fetch(`${API_BASE}/api/wifi/settings`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ interface: wifiSettings.interface }),
      });
      if (response.ok) {
        setWifiMessage('WiFi settings saved');
        setTimeout(() => setWifiMessage(null), 2000);
      } else {
        const error = await response.text();
        setWifiMessage(`Failed: ${error}`);
      }
    } catch (err) {
      setWifiMessage('Error saving WiFi settings');
    } finally {
      setSavingWifi(false);
    }
  };

  // Validate IP address format
  const isValidIP = (ip: string): boolean => {
    if (!ip) return true; // Empty is OK for optional fields
    const parts = ip.split('.');
    if (parts.length !== 4) return false;
    return parts.every((p) => {
      const n = parseInt(p, 10);
      return !isNaN(n) && n >= 0 && n <= 255 && p === String(n);
    });
  };

  // Add/remove ping target
  const addPingTarget = () => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: [...prev.pingTargets, { name: '', host: '', enabled: true }],
    }));
  };

  const removePingTarget = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: prev.pingTargets.filter((_, i) => i !== index),
    }));
  };

  const updatePingTarget = (index: number, field: keyof PingTarget, value: string | boolean) => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: prev.pingTargets.map((t, i) =>
        i === index ? { ...t, [field]: value } : t
      ),
    }));
  };

  // Add/remove TCP port
  const addTCPPort = () => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: [...prev.tcpPorts, { name: '', host: '', port: 80, enabled: true }],
    }));
  };

  const removeTCPPort = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: prev.tcpPorts.filter((_, i) => i !== index),
    }));
  };

  const updateTCPPort = (index: number, field: keyof TCPPort, value: string | number | boolean) => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: prev.tcpPorts.map((t, i) =>
        i === index ? { ...t, [field]: value } : t
      ),
    }));
  };

  // Add/remove UDP port
  const addUDPPort = () => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: [...prev.udpPorts, { name: '', host: '', port: 53, enabled: true }],
    }));
  };

  const removeUDPPort = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: prev.udpPorts.filter((_, i) => i !== index),
    }));
  };

  const updateUDPPort = (index: number, field: keyof UDPPort, value: string | number | boolean) => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: prev.udpPorts.map((u, i) =>
        i === index ? { ...u, [field]: value } : u
      ),
    }));
  };

  // Add/remove HTTP endpoint
  const addHTTPEndpoint = () => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: [...prev.httpEndpoints, { name: '', url: '', expectedStatus: 200, enabled: true }],
    }));
  };

  const removeHTTPEndpoint = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: prev.httpEndpoints.filter((_, i) => i !== index),
    }));
  };

  const updateHTTPEndpoint = (index: number, field: keyof HTTPEndpoint, value: string | number | boolean) => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: prev.httpEndpoints.map((t, i) =>
        i === index ? { ...t, [field]: value } : t
      ),
    }));
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-40"
        onClick={onClose}
      />

      {/* Drawer - full width on mobile, 384px on larger screens */}
      <div className="fixed right-0 top-0 h-full w-full sm:w-96 bg-surface-raised border-l border-surface-border z-50 overflow-y-auto shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-surface-border sticky top-0 bg-surface-raised z-10">
          <h2 className="text-lg font-semibold text-text-primary">Settings</h2>
          <button
            onClick={onClose}
            className="p-2.5 rounded hover:bg-surface-hover active:bg-surface-hover text-text-muted touch-manipulation"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-4 pb-8 space-y-4">
          {/* Network Section */}
          <CollapsibleSection title="Network" defaultOpen>
            {/* IP Configuration */}
            <div className="space-y-3">
              <p className="text-xs text-text-muted font-medium">IP Configuration</p>
              {/* Mode Toggle */}
              <div className="flex gap-2">
                <button
                  onClick={() => setIPSettings((prev) => ({ ...prev, mode: 'dhcp' }))}
                  className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                    ipSettings.mode === 'dhcp'
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover'
                  }`}
                >
                  DHCP
                </button>
                <button
                  onClick={() => setIPSettings((prev) => ({ ...prev, mode: 'static' }))}
                  className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                    ipSettings.mode === 'static'
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover'
                  }`}
                >
                  Static
                </button>
              </div>

              {/* Static IP Fields */}
              {ipSettings.mode === 'static' && (
                <div className="space-y-2 pt-2 border-t border-surface-border">
                  <div>
                    <label className="text-xs text-text-muted">IP Address *</label>
                    <input
                      type="text"
                      value={ipSettings.address}
                      onChange={(e) =>
                        setIPSettings((prev) => ({ ...prev, address: e.target.value }))
                      }
                      placeholder="192.168.1.100"
                      className={`w-full mt-1 px-2 py-1 bg-surface-base border rounded text-sm text-text-primary ${
                        ipSettings.address && !isValidIP(ipSettings.address)
                          ? 'border-status-error'
                          : 'border-surface-border'
                      }`}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Subnet Mask *</label>
                    <input
                      type="text"
                      value={ipSettings.netmask}
                      onChange={(e) =>
                        setIPSettings((prev) => ({ ...prev, netmask: e.target.value }))
                      }
                      placeholder="24 or 255.255.255.0"
                      className="w-full mt-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Gateway</label>
                    <input
                      type="text"
                      value={ipSettings.gateway}
                      onChange={(e) =>
                        setIPSettings((prev) => ({ ...prev, gateway: e.target.value }))
                      }
                      placeholder="192.168.1.1"
                      className={`w-full mt-1 px-2 py-1 bg-surface-base border rounded text-sm text-text-primary ${
                        ipSettings.gateway && !isValidIP(ipSettings.gateway)
                          ? 'border-status-error'
                          : 'border-surface-border'
                      }`}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">DNS Servers (comma-separated)</label>
                    <input
                      type="text"
                      value={dnsInput}
                      onChange={(e) => setDnsInput(e.target.value)}
                      placeholder="8.8.8.8, 8.8.4.4"
                      className="w-full mt-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              )}

              {/* Apply Button */}
              <button
                onClick={saveIPSettings}
                disabled={savingIP || (ipSettings.mode === 'static' && !ipSettings.address)}
                className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors"
              >
                {savingIP ? 'Applying...' : 'Apply IP Settings'}
              </button>

              {ipMessage && (
                <p
                  className={`text-xs text-center ${
                    ipMessage.includes('Failed') || ipMessage.includes('Error')
                      ? 'text-status-error'
                      : 'text-status-success'
                  }`}
                >
                  {ipMessage}
                </p>
              )}

              <p className="text-xs text-text-muted">
                Note: Requires root/admin privileges to apply
              </p>
            </div>
          </CollapsibleSection>

          {/* WiFi Section */}
          <CollapsibleSection title="WiFi">
            <div className="space-y-3">
              <div>
                <label className="text-xs text-text-muted">WiFi Interface</label>
                {wifiSettings.availableWifi.length > 0 ? (
                  <select
                    value={wifiSettings.interface}
                    onChange={(e) =>
                      setWifiSettings((prev) => ({ ...prev, interface: e.target.value }))
                    }
                    className="w-full mt-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                  >
                    {wifiSettings.availableWifi.map((iface) => (
                      <option key={iface} value={iface}>
                        {iface}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    value={wifiSettings.interface}
                    onChange={(e) =>
                      setWifiSettings((prev) => ({ ...prev, interface: e.target.value }))
                    }
                    placeholder="wlan0 or en0"
                    className="w-full mt-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                  />
                )}
                <p className="text-xs text-text-muted mt-1">
                  {wifiSettings.isWireless
                    ? 'Currently monitoring a wireless interface'
                    : 'No wireless interface detected'}
                </p>
              </div>

              <button
                onClick={saveWifiSettings}
                disabled={savingWifi}
                className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors"
              >
                {savingWifi ? 'Saving...' : 'Save WiFi Settings'}
              </button>

              {wifiMessage && (
                <p
                  className={`text-xs text-center ${
                    wifiMessage.includes('Failed') || wifiMessage.includes('Error')
                      ? 'text-status-error'
                      : 'text-status-success'
                  }`}
                >
                  {wifiMessage}
                </p>
              )}
            </div>
          </CollapsibleSection>

          {/* Health Checks Section */}
          <CollapsibleSection title="Health Checks">
            <div className="space-y-4">
              {/* DNS Hostname */}
              <div>
                <label className="text-xs text-text-muted">DNS Test Hostname</label>
                <input
                  type="text"
                  value={testsSettings.dnsHostname}
                  onChange={(e) =>
                    setTestsSettings((prev) => ({ ...prev, dnsHostname: e.target.value }))
                  }
                  placeholder="google.com"
                  className="w-full mt-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  Hostname used for DNS forward/reverse lookups
                </p>
              </div>

              {/* Ping Targets */}
              <div className="border-t border-surface-border pt-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-text-muted font-medium">Ping Targets</span>
                  <button
                    onClick={addPingTarget}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                {testsSettings.pingTargets.map((target, idx) => (
                  <div key={idx} className="flex gap-2 mb-2">
                    <input
                      type="text"
                      value={target.name}
                      onChange={(e) => updatePingTarget(idx, 'name', e.target.value)}
                      placeholder="Name"
                      className="flex-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="text"
                      value={target.host}
                      onChange={(e) => updatePingTarget(idx, 'host', e.target.value)}
                      placeholder="Host/IP"
                      className="flex-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
                  <span className="text-xs text-text-muted font-medium">TCP Port Tests</span>
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
                      onChange={(e) => updateTCPPort(idx, 'name', e.target.value)}
                      placeholder="Name"
                      className="w-20 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="text"
                      value={port.host}
                      onChange={(e) => updateTCPPort(idx, 'host', e.target.value)}
                      placeholder="Host"
                      className="flex-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="number"
                      value={port.port}
                      onChange={(e) => updateTCPPort(idx, 'port', parseInt(e.target.value) || 80)}
                      placeholder="Port"
                      className="w-16 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
                  <span className="text-xs text-text-muted font-medium">UDP Port Tests</span>
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
                      onChange={(e) => updateUDPPort(idx, 'name', e.target.value)}
                      placeholder="Name"
                      className="w-20 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="text"
                      value={port.host}
                      onChange={(e) => updateUDPPort(idx, 'host', e.target.value)}
                      placeholder="Host"
                      className="flex-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
                    />
                    <input
                      type="number"
                      value={port.port}
                      onChange={(e) => updateUDPPort(idx, 'port', parseInt(e.target.value) || 53)}
                      placeholder="Port"
                      className="w-16 px-2 py-1 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
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
                  <span className="text-xs text-text-muted font-medium">HTTP Endpoints</span>
                  <button
                    onClick={addHTTPEndpoint}
                    className="text-xs text-brand-primary hover:text-brand-accent"
                  >
                    + Add
                  </button>
                </div>
                {testsSettings.httpEndpoints.map((endpoint, idx) => (
                  <div key={idx} className="space-y-1 mb-3 p-2 bg-surface-base rounded border border-surface-border">
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={endpoint.name}
                        onChange={(e) => updateHTTPEndpoint(idx, 'name', e.target.value)}
                        placeholder="Name"
                        className="flex-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                      />
                      <input
                        type="number"
                        value={endpoint.expectedStatus}
                        onChange={(e) => updateHTTPEndpoint(idx, 'expectedStatus', parseInt(e.target.value) || 200)}
                        placeholder="Status"
                        className="w-16 px-2 py-1 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
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
                      onChange={(e) => updateHTTPEndpoint(idx, 'url', e.target.value)}
                      placeholder="https://example.com/health"
                      className="w-full px-2 py-1 bg-surface-raised border border-surface-border rounded text-xs text-text-primary"
                    />
                  </div>
                ))}
              </div>

              {/* Save Health Checks Button */}
              <button
                onClick={saveTestsSettings}
                disabled={savingTests}
                className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors"
              >
                {savingTests ? 'Saving...' : 'Save Health Checks'}
              </button>

              {testsMessage && (
                <p
                  className={`text-xs text-center ${
                    testsMessage.includes('Failed') || testsMessage.includes('Error')
                      ? 'text-status-error'
                      : 'text-status-success'
                  }`}
                >
                  {testsMessage}
                </p>
              )}
            </div>
          </CollapsibleSection>

          {/* Speedtest Section */}
          <CollapsibleSection title="Speedtest">
            <div className="space-y-3">
              <div>
                <label className="text-xs text-text-muted">Server ID (optional)</label>
                <input
                  type="text"
                  value={testsSettings.speedtest.serverId}
                  onChange={(e) =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      speedtest: { ...prev.speedtest, serverId: e.target.value },
                    }))
                  }
                  placeholder="Auto (closest server)"
                  className="w-full mt-1 px-2 py-1 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
                />
                <p className="text-xs text-text-muted mt-1">
                  Leave empty for auto-selection
                </p>
              </div>

              <label className="flex items-center justify-between p-2 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">Auto-run on link up</span>
                <input
                  type="checkbox"
                  checked={testsSettings.speedtest.autoRunOnLink}
                  onChange={(e) =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      speedtest: { ...prev.speedtest, autoRunOnLink: e.target.checked },
                    }))
                  }
                  className="w-4 h-4"
                />
              </label>

              <button
                onClick={saveTestsSettings}
                disabled={savingTests}
                className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors"
              >
                {savingTests ? 'Saving...' : 'Save Speedtest Settings'}
              </button>
            </div>
          </CollapsibleSection>

          {/* Thresholds Section */}
          <CollapsibleSection title="Thresholds">
            <div className="space-y-3">
              {/* DNS Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">DNS Lookup (ms)</span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">Good (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.dns.good}
                      onChange={(e) => updateThreshold('dns', 'good', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Warning (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.dns.warning}
                      onChange={(e) => updateThreshold('dns', 'warning', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Gateway Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">Gateway Ping (ms)</span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">Good (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.gateway.good}
                      onChange={(e) => updateThreshold('gateway', 'good', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Warning (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.gateway.warning}
                      onChange={(e) => updateThreshold('gateway', 'warning', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Wi-Fi Signal Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">Wi-Fi Signal (dBm)</span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">Good (&gt;)</label>
                    <input
                      type="number"
                      value={thresholds.wifi.good}
                      onChange={(e) => updateThreshold('wifi', 'good', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Warning (&gt;)</label>
                    <input
                      type="number"
                      value={thresholds.wifi.warning}
                      onChange={(e) => updateThreshold('wifi', 'warning', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Health Check Ping Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">Health Check: Ping (ms)</span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">Good (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.customPing.good}
                      onChange={(e) => updateThreshold('customPing', 'good', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Warning (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.customPing.warning}
                      onChange={(e) => updateThreshold('customPing', 'warning', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Health Check TCP Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">Health Check: TCP (ms)</span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">Good (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.customTcp.good}
                      onChange={(e) => updateThreshold('customTcp', 'good', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Warning (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.customTcp.warning}
                      onChange={(e) => updateThreshold('customTcp', 'warning', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Health Check HTTP Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">Health Check: HTTP (ms)</span>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs text-text-muted">Good (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.customHttp.good}
                      onChange={(e) => updateThreshold('customHttp', 'good', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-text-muted">Warning (&lt;)</label>
                    <input
                      type="number"
                      value={thresholds.customHttp.warning}
                      onChange={(e) => updateThreshold('customHttp', 'warning', Number(e.target.value))}
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
                    />
                  </div>
                </div>
              </div>

              {/* Save Button */}
              <button
                onClick={saveThresholds}
                disabled={saving}
                className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving...' : 'Save Thresholds'}
              </button>

              {saveMessage && (
                <p className={`text-sm text-center ${saveMessage.includes('Error') || saveMessage.includes('Failed') ? 'text-status-error' : 'text-status-success'}`}>
                  {saveMessage}
                </p>
              )}
            </div>
          </CollapsibleSection>

          {/* Appearance Section */}
          <CollapsibleSection title="Appearance">
            <div className="space-y-2">
              <label className="flex items-center justify-between p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm text-text-primary">Theme</span>
                <select
                  value={theme}
                  onChange={(e) => setTheme(e.target.value as 'light' | 'dark' | 'system')}
                  className="bg-surface-raised border border-surface-border rounded px-2 py-1 text-sm text-text-primary"
                >
                  <option value="light">Light</option>
                  <option value="dark">Dark</option>
                  <option value="system">System</option>
                </select>
              </label>

              <button
                onClick={() => setTheme(isDark ? 'light' : 'dark')}
                className="w-full flex items-center justify-between p-3 bg-surface-base rounded border border-surface-border hover:bg-surface-hover transition-colors"
              >
                <span className="text-sm text-text-primary">Quick Toggle</span>
                <span className="text-xl">{isDark ? '🌙' : '☀️'}</span>
              </button>
            </div>
          </CollapsibleSection>

          {/* Export Section */}
          <section className="pt-4 border-t border-surface-border">
            <h3 className="text-sm font-medium text-text-muted mb-3">Export</h3>
            <a
              href={`${API_BASE}/api/export`}
              download="netscope-export.json"
              className="w-full py-2 px-4 bg-surface-base border border-surface-border text-text-primary rounded font-medium hover:bg-surface-hover transition-colors flex items-center justify-center gap-2 touch-manipulation"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              Download JSON Export
            </a>
            <p className="text-xs text-text-muted mt-2">
              Export all diagnostic data as JSON for documentation or analysis.
            </p>
          </section>

          {/* About Section */}
          <section className="pt-4 border-t border-surface-border">
            <h3 className="text-sm font-medium text-text-muted mb-2">About</h3>
            <p className="text-xs text-text-muted">
              NetScope v0.7.3
              <br />
              Network Diagnostic Tool
            </p>
          </section>
        </div>
      </div>
    </>
  );
}
