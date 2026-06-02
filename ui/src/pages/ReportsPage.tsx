import { BarChart3 } from 'lucide-react';
import { SLADashboardCard } from '../components/cards/SlaDashboardCard';
import { RequireFeature } from '../components/ui/RequireFeature';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function ReportsPage() {
  return (
    <RequireFeature
      feature="export_csv_json"
      fallback={
        <section className="stack-xl">
          <Breadcrumbs />
          <PageHeader
            icon={BarChart3}
            title="Reports"
            description="Aggregated SLA dashboard, compliance tracking, and historical reporting."
            iconColorClass="text-module-reporting"
          />
          <div className="rounded-lg border border-status-warning/30 bg-status-warning/5 pad text-sm text-status-warning">
            Reports require the Starter tier or higher. Start a 14-day Pro trial with
            <code className="mx-1 px-1 rounded bg-surface-raised">seed license trial</code>
            or activate a Starter / Pro key with
            <code className="ml-tight px-1 rounded bg-surface-raised">
              seed license activate -k &lt;KEY&gt;
            </code>
            .
          </div>
        </section>
      }
    >
      <section className="stack-xl">
        <Breadcrumbs />
        <PageHeader
          icon={BarChart3}
          title="Reports"
          description="Aggregated SLA dashboard, compliance tracking, and historical reporting."
          iconColorClass="text-module-reporting"
        />
        <div className={layout.grid.cards}>
          <SLADashboardCard />
        </div>
      </section>
    </RequireFeature>
  );
}
