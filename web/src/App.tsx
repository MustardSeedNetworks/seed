import { useCallback, useEffect, useState } from 'react';
import { useWebSocket, Message, CardUpdate } from './hooks/useWebSocket';
import { useAuth, getAuthHeaders } from './hooks/useAuth';

const API_BASE = import.meta.env.VITE_API_BASE || '';
import {
  LinkCard,
  LinkData,
  SwitchCard,
  SwitchData,
  DHCPCard,
  DHCPData,
  DNSCard,
  DNSData,
  GatewayCard,
  GatewayData,
  VLANCard,
  VLANData,
  WiFiCard,
  WiFiData,
  CableCard,
  CableData,
} from './components/cards';

interface CardState {
  link: LinkData | null;
  cable: CableData | null;
  vlan: VLANData | null;
  switch: SwitchData | null;
  wifi: WiFiData | null;
  dhcp: DHCPData | null;
  dns: DNSData | null;
  gateway: GatewayData | null;
}

function App() {
  const { isAuthenticated, login, logout, isLoading, error } = useAuth();
  const [cards, setCards] = useState<CardState>({
    link: null,
    cable: null,
    vlan: null,
    switch: null,
    wifi: null,
    dhcp: null,
    dns: null,
    gateway: null,
  });
  const [loading, setLoading] = useState(true);
  const [currentInterface, setCurrentInterface] = useState('eth0');
  const [isWifi, setIsWifi] = useState(false);
  const [interfaces, setInterfaces] = useState<Array<{ name: string; type: string; up: boolean }>>([]);

  const handleMessage = useCallback((message: Message) => {
    if (message.type === 'initial_state') {
      setLoading(false);
      const payload = message.payload as { interface?: string; cards?: CardState };
      if (payload.interface) {
        setCurrentInterface(payload.interface);
        setIsWifi(payload.interface.startsWith('wl'));
      }
    }
  }, []);

  const handleCardUpdate = useCallback((update: CardUpdate) => {
    setCards((prev) => ({
      ...prev,
      [update.cardId]: update.data,
    }));
  }, []);

  // Fetch link data
  const fetchLinkData = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/link`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setCards((prev) => ({
          ...prev,
          link: {
            linkUp: data.linkUp,
            speed: data.speed || '',
            duplex: data.duplex || '',
            advertisedSpeeds: data.advertisedSpeeds || [],
            mac: data.mac || '',
            mtu: data.mtu || 0,
            addresses: data.addresses || [],
          },
        }));
        setCurrentInterface(data.interface || 'unknown');
        setIsWifi(data.interface?.startsWith('wl') || false);
      }
    } catch (err) {
      console.error('Failed to fetch link data:', err);
    }
  }, []);

  // Fetch interfaces
  const fetchInterfaces = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/interfaces`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setInterfaces(data);
      }
    } catch (err) {
      console.error('Failed to fetch interfaces:', err);
    }
  }, []);

  // Fetch data on mount and periodically
  useEffect(() => {
    if (!isAuthenticated) return;

    fetchLinkData();
    fetchInterfaces();
    setLoading(false);

    const interval = setInterval(() => {
      fetchLinkData();
    }, 5000); // Refresh every 5 seconds

    return () => clearInterval(interval);
  }, [isAuthenticated, fetchLinkData, fetchInterfaces]);

  const { status, reconnect } = useWebSocket({
    url: '/ws',
    onMessage: handleMessage,
    onCardUpdate: handleCardUpdate,
  });

  // Login form
  if (!isAuthenticated) {
    return <LoginForm onLogin={login} isLoading={isLoading} error={error} />;
  }

  return (
    <div className="min-h-screen bg-surface-base text-text-primary">
      {/* Header */}
      <header className="border-b border-surface-border bg-surface-raised px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold text-brand-primary">◉</span>
            <h1 className="text-lg font-semibold">NetScope</h1>
            <ConnectionStatus status={status} onReconnect={reconnect} />
          </div>
          <div className="flex items-center gap-2">
            <select
              className="rounded border border-surface-border bg-surface-base px-2 py-1 text-sm"
              value={currentInterface}
              onChange={(e) => setCurrentInterface(e.target.value)}
            >
              {interfaces.length > 0 ? (
                interfaces
                  .filter((iface) => iface.type === 'ethernet' || iface.type === 'wifi')
                  .map((iface) => (
                    <option key={iface.name} value={iface.name}>
                      {iface.name} {!iface.up && '(down)'}
                    </option>
                  ))
              ) : (
                <option value={currentInterface}>{currentInterface}</option>
              )}
            </select>
            <button
              className="rounded p-2 hover:bg-surface-hover"
              title="Toggle theme"
            >
              🌙
            </button>
            <button
              className="rounded p-2 hover:bg-surface-hover"
              title="Settings"
            >
              ⚙️
            </button>
            <button
              className="rounded p-2 hover:bg-surface-hover text-sm"
              onClick={logout}
              title="Logout"
            >
              Logout
            </button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="p-4">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <LinkCard data={cards.link} loading={loading} />
          <CableCard data={cards.cable} loading={loading} />
          <VLANCard data={cards.vlan} loading={loading} />
          <SwitchCard data={cards.switch} loading={loading} />
          <WiFiCard data={cards.wifi} loading={loading} visible={isWifi} />
          <DHCPCard data={cards.dhcp} loading={loading} />
          <DNSCard data={cards.dns} loading={loading} />
          <GatewayCard data={cards.gateway} loading={loading} />
        </div>

        {/* Development notice */}
        <div className="mt-8 rounded-lg border border-surface-border bg-surface-raised p-6 text-center">
          <h2 className="text-lg font-semibold text-text-muted">
            NetScope v0.1.0 - Core Infrastructure Complete
          </h2>
          <p className="mt-2 text-sm text-text-muted">
            Backend server, WebSocket, and authentication are ready.
            <br />
            Card data collection coming in next milestone.
          </p>
        </div>
      </main>
    </div>
  );
}

interface LoginFormProps {
  onLogin: (username: string, password: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

function LoginForm({ onLogin, isLoading, error }: LoginFormProps) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onLogin(username, password);
  };

  return (
    <div className="min-h-screen bg-surface-base flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <span className="text-4xl font-bold text-brand-primary">◉</span>
          <h1 className="text-2xl font-bold text-text-primary mt-2">NetScope</h1>
          <p className="text-text-muted mt-1">Network Diagnostic Tool</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-surface-raised rounded-lg border border-surface-border p-6"
        >
          <div className="mb-4">
            <label className="block text-sm font-medium text-text-primary mb-1">
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="admin"
              required
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium text-text-primary mb-1">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 rounded border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:border-brand-primary"
              placeholder="••••••••"
              required
            />
          </div>

          {error && (
            <div className="mb-4 p-3 bg-status-error/10 border border-status-error/20 rounded text-status-error text-sm">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-2 px-4 bg-brand-primary text-text-inverse rounded font-medium hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base disabled:opacity-50"
          >
            {isLoading ? 'Logging in...' : 'Login'}
          </button>

          <p className="mt-4 text-xs text-text-muted text-center">
            Default: admin / netscope
          </p>
        </form>
      </div>
    </div>
  );
}

interface ConnectionStatusProps {
  status: 'connecting' | 'connected' | 'disconnected' | 'error';
  onReconnect: () => void;
}

function ConnectionStatus({ status, onReconnect }: ConnectionStatusProps) {
  const statusConfig = {
    connecting: { color: 'text-status-warning', label: 'Connecting...' },
    connected: { color: 'text-status-success', label: 'Connected' },
    disconnected: { color: 'text-status-error', label: 'Disconnected' },
    error: { color: 'text-status-error', label: 'Error' },
  };

  const config = statusConfig[status];

  return (
    <div className="flex items-center gap-2 ml-4">
      <span className={`text-xs ${config.color}`}>● {config.label}</span>
      {(status === 'disconnected' || status === 'error') && (
        <button
          onClick={onReconnect}
          className="text-xs text-brand-primary hover:underline"
        >
          Reconnect
        </button>
      )}
    </div>
  );
}

export default App;
