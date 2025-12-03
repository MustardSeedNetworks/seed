/**
 * NetScope - Main Application Component
 *
 * TODO: Implement dashboard with diagnostic cards
 */

function App() {
  return (
    <div className="min-h-screen bg-surface-base text-text-primary">
      {/* Header */}
      <header className="border-b border-surface-border bg-surface-raised px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold text-brand-primary">◉</span>
            <h1 className="text-lg font-semibold">NetScope</h1>
          </div>
          <div className="flex items-center gap-2">
            {/* Interface selector placeholder */}
            <select className="rounded border border-surface-border bg-surface-base px-2 py-1 text-sm">
              <option>eth0</option>
              <option>wlan0</option>
            </select>
            {/* Theme toggle placeholder */}
            <button className="rounded p-2 hover:bg-surface-hover">🌙</button>
            {/* Settings placeholder */}
            <button className="rounded p-2 hover:bg-surface-hover">⚙️</button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="p-4">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {/* Placeholder cards */}
          <PlaceholderCard title="Link" status="success" value="1 Gbps" />
          <PlaceholderCard title="Cable Test" status="success" value="OK" />
          <PlaceholderCard title="VLAN" status="success" value="ID: 10" />
          <PlaceholderCard title="Switch" status="success" value="SW-CORE-01" />
          <PlaceholderCard title="DHCP" status="warning" value="1.2s" />
          <PlaceholderCard title="DNS" status="success" value="45ms" />
          <PlaceholderCard title="Gateway" status="success" value="12ms" />
        </div>

        <div className="mt-8 rounded-lg border border-surface-border bg-surface-raised p-6 text-center">
          <h2 className="text-lg font-semibold text-text-muted">
            NetScope is under development
          </h2>
          <p className="mt-2 text-sm text-text-muted">
            See PROJECT_PLAN.md for the roadmap
          </p>
        </div>
      </main>
    </div>
  );
}

interface PlaceholderCardProps {
  title: string;
  status: 'success' | 'warning' | 'error';
  value: string;
}

function PlaceholderCard({ title, status, value }: PlaceholderCardProps) {
  const statusColors = {
    success: 'text-status-success',
    warning: 'text-status-warning',
    error: 'text-status-error',
  };

  const statusIcons = {
    success: '🟢',
    warning: '🟡',
    error: '🔴',
  };

  return (
    <div className="rounded-lg border border-surface-border bg-surface-raised p-4">
      <div className="flex items-center justify-between">
        <h3 className="font-medium">{title}</h3>
        <span className={statusColors[status]}>{statusIcons[status]}</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

export default App;
