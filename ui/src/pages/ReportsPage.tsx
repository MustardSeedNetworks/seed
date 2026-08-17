import { SLADashboardCard } from '../components/cards/SlaDashboardCard';
import { RequireFeature } from '../components/ui/RequireFeature';
import { layout } from '../styles/theme';

export function ReportsPage() {
  return (
    <RequireFeature
      feature="export_csv_json"
      fallback={
        <div className="rounded-lg border border-status-warning/30 bg-status-warning/5 pad text-sm text-status-warning">
          Reports require the Starter tier or higher. Start a 14-day Pro trial with
          <code className="mx-1 px-1 rounded bg-surface-raised">seed license trial</code>
          or activate a Starter / Pro key with
          <code className="ml-tight px-1 rounded bg-surface-raised">
            seed license activate -k &lt;KEY&gt;
          </code>
          .
        </div>
      }
    >
      <div className={layout.grid.cards}>
        <SLADashboardCard />
      </div>
    </RequireFeature>
  );
}
