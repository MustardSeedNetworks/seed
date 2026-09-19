import { ReportsCard } from '../components/cards/ReportsCard';
import { SLADashboardCard } from '../components/cards/SlaDashboardCard';
import { ReportsPreview } from '../components/previews/ReportsPreview';
import { GatedPreview } from '../components/ui/GatedPreview';
import { useRole } from '../contexts/RoleContext';
import { useReports } from '../hooks/useReports';
import { CardGrid } from '../ui/CardGrid';

/**
 * Reports — Card grid.
 *
 * One card today, and still a grid rather than a bare div: the page's shape is
 * "facets of the system's reporting", and the second one should land next to
 * the first without a layout rewrite.
 *
 * No rollup — the licence gate below is the page's state: on a tier without
 * the feature it shows a sample of the reports plus the pitch.
 */
function ReportsCardContainer() {
  const { canWrite } = useRole();
  const { reports, loading, error, generating, generate, remove } = useReports();

  // ReportsCard already documents that an absent handler means the action is
  // unavailable; nothing implemented it. POST /reports/generate and
  // DELETE /reports/{id} are both minRole: op, so a viewer's click could only
  // 403 (#1254).
  return (
    <ReportsCard
      reports={reports}
      loading={loading}
      error={error}
      generating={generating}
      onGenerate={canWrite ? (): void => void generate('executive', 'pdf') : undefined}
      onDelete={canWrite ? (id): void => void remove(id) : undefined}
    />
  );
}

export function ReportsPage() {
  return (
    <GatedPreview feature="export_csv_json" preview={<ReportsPreview />}>
      <CardGrid>
        <SLADashboardCard />
        <ReportsCardContainer />
      </CardGrid>
    </GatedPreview>
  );
}
