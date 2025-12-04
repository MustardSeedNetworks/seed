import { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../../hooks/useTheme';
import { getAuthHeaders } from '../../hooks/useAuth';

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
}

interface IPSettings {
  mode: 'dhcp' | 'static';
  address: string;
  netmask: string;
  gateway: string;
  dns: string[];
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
  });
  const [ipSettings, setIPSettings] = useState<IPSettings>({
    mode: 'dhcp',
    address: '',
    netmask: '24',
    gateway: '',
    dns: [],
  });
  const [dnsInput, setDnsInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [savingIP, setSavingIP] = useState(false);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [ipMessage, setIPMessage] = useState<string | null>(null);

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

  useEffect(() => {
    if (isOpen) {
      fetchThresholds();
      fetchIPSettings();
    }
  }, [isOpen, fetchThresholds, fetchIPSettings]);

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
        setSaveMessage('Settings saved');
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

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-40"
        onClick={onClose}
      />

      {/* Drawer - full width on mobile, 320px on larger screens */}
      <div className="fixed right-0 top-0 h-full w-full sm:w-80 bg-surface-raised border-l border-surface-border z-50 overflow-y-auto shadow-xl">
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

        <div className="p-4 pb-8 space-y-6">
          {/* IP Configuration Section */}
          <section>
            <h3 className="text-sm font-medium text-text-muted mb-3">IP Configuration</h3>
            <div className="p-3 bg-surface-base rounded border border-surface-border space-y-3">
              {/* Mode Toggle */}
              <div className="flex gap-2">
                <button
                  onClick={() => setIPSettings((prev) => ({ ...prev, mode: 'dhcp' }))}
                  className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                    ipSettings.mode === 'dhcp'
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-raised border border-surface-border text-text-primary hover:bg-surface-hover'
                  }`}
                >
                  DHCP
                </button>
                <button
                  onClick={() => setIPSettings((prev) => ({ ...prev, mode: 'static' }))}
                  className={`flex-1 py-2 px-3 rounded text-sm font-medium transition-colors ${
                    ipSettings.mode === 'static'
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-raised border border-surface-border text-text-primary hover:bg-surface-hover'
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
                      className={`w-full mt-1 px-2 py-1 bg-surface-raised border rounded text-sm text-text-primary ${
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
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
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
                      className={`w-full mt-1 px-2 py-1 bg-surface-raised border rounded text-sm text-text-primary ${
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
                      className="w-full mt-1 px-2 py-1 bg-surface-raised border border-surface-border rounded text-sm text-text-primary"
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
          </section>

          {/* Theme Section */}
          <section>
            <h3 className="text-sm font-medium text-text-muted mb-3">Appearance</h3>
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
          </section>

          {/* Thresholds Section */}
          <section>
            <h3 className="text-sm font-medium text-text-muted mb-3">Response Thresholds (ms)</h3>
            <div className="space-y-4">
              {/* DNS Thresholds */}
              <div className="p-3 bg-surface-base rounded border border-surface-border">
                <span className="text-sm font-medium text-text-primary block mb-2">DNS Lookup</span>
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
                <span className="text-sm font-medium text-text-primary block mb-2">Gateway Ping</span>
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
            </div>
          </section>

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
              NetScope v0.7.0
              <br />
              Network Diagnostic Tool
            </p>
          </section>
        </div>
      </div>
    </>
  );
}
