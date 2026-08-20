import { SLADashboardCard } from '../components/cards/SlaDashboardCard';
import { RequireFeature } from '../components/ui/RequireFeature';
import { CardGrid } from '../ui/CardGrid';

/**
 * Reports — Card grid.
 *
 * One card today, and still a grid rather than a bare div: the page's shape is
 * "facets of the system's reporting", and the second one should land next to
 * the first without a layout rewrite.
 *
 * No rollup — the licence gate below is the page's state, and it already says
 * what is missing and how to fix it.
 */
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
      <CardGrid>
        <SLADashboardCard />
      </CardGrid>
    </RequireFeature>
  );
}
