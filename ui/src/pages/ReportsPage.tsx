import { ReportsCard } from '../components/cards/ReportsCard';
import { ScheduledReportsCard } from '../components/cards/ScheduledReportsCard';
import { SLADashboardCard } from '../components/cards/SlaDashboardCard';
import { ReportsPreview } from '../components/previews/ReportsPreview';
import { GatedPreview } from '../components/ui/GatedPreview';
import { RequireFeature } from '../components/ui/RequireFeature';
import { useRole } from '../contexts/RoleContext';
import { useReportSchedules } from '../hooks/useReportSchedules';
import { useReports } from '../hooks/useReports';
import { CardGrid } from '../ui/CardGrid';

/**
 * Reports — Card grid.
 *
 * A grid of the facets of the system's reporting: SLA, the generated reports,
 * and on Pro the schedules that generate them.
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

function ScheduledReportsCardContainer() {
  const { canWrite } = useRole();
  const { schedules, loading, error, save, remove } = useReportSchedules();

  // Every schedule write is minRole: op; a viewer reads the list only.
  return (
    <ScheduledReportsCard
      schedules={schedules}
      loading={loading}
      error={error}
      onSave={canWrite ? save : undefined}
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
        {/* Schedules are Pro; below it the route answers 402, so the card is absent. */}
        <RequireFeature feature="scheduled_reports">
          <ScheduledReportsCardContainer />
        </RequireFeature>
      </CardGrid>
    </GatedPreview>
  );
}
