import { BarChart3 } from 'lucide-react';
import { SLADashboardCard } from '../components/cards/SlaDashboardCard';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function ReportsPage() {
  return (
    <section className="space-y-6">
      <Breadcrumbs />
      <PageHeader
        icon={BarChart3}
        title="Reports"
        description="Aggregated SLA dashboard, compliance tracking, and historical reporting."
        iconColorClass="text-module-harvest"
      />
      <div className={layout.grid.cards}>
        <SLADashboardCard />
      </div>
    </section>
  );
}
