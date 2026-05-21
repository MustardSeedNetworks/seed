import { ScrollText } from 'lucide-react';
import { LogViewerCard } from '../components/cards/LogViewerCard';
import { SystemHealthCard } from '../components/cards/SystemHealthCard';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function LogsPage() {
  return (
    <section className="space-y-6">
      <Breadcrumbs />
      <PageHeader
        icon={ScrollText}
        title="Logs"
        description="Live log stream and system health for the daemon."
        iconColorClass="text-module-harvest"
      />
      <div className={layout.grid.cards}>
        <SystemHealthCard />
      </div>
      <LogViewerCard />
    </section>
  );
}
