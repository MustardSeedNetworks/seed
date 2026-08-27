/**
 * ReportsCard lists generated reports and offers generate / download / delete.
 *
 * Presentational only: the data and the actions come in as props, which is what
 * lets the stories assert on real rendered output rather than a mock server.
 * The fetching lives in useReports.
 */
import { useTranslation } from 'react-i18next';
import { cn, spacing } from '../../styles/theme';
import type { ReportInfo } from '../../types/generated/reports-response';
import { Button } from '../ui/Button';
import { Card, CardDivider, CardRow } from '../ui/card';
import type { Status } from '../ui/StatusBadge';

export interface ReportsCardProps {
  reports: ReportInfo[];
  loading?: boolean;
  error?: string | null;
  /** Undefined for a viewer: generate and delete are operator-gated. */
  onGenerate?: () => void;
  onDelete?: (id: string) => void;
  generating?: boolean;
}

/** Report statuses the backend can report. */
const terminalStatuses = new Set(['complete', 'failed']);

function cardStatus(reports: ReportInfo[], loading: boolean, error: string | null): Status {
  if (loading) {
    return 'loading';
  }
  if (error) {
    return 'error';
  }
  if (reports.some((r) => r.status === 'failed')) {
    return 'warning';
  }
  return reports.length > 0 ? 'success' : 'unknown';
}

export function ReportsCard({
  reports,
  loading = false,
  error = null,
  onGenerate,
  onDelete,
  generating = false,
}: ReportsCardProps): React.ReactElement {
  const { t } = useTranslation('cards');

  return (
    <Card
      title={t('reports.title')}
      status={cardStatus(reports, loading, error)}
      headerAction={
        onGenerate ? (
          <Button
            size="sm"
            onClick={onGenerate}
            loading={generating}
            data-testid="reports-generate"
          >
            {t('reports.generate')}
          </Button>
        ) : undefined
      }
    >
      {error ? <p className="text-status-error text-sm">{error}</p> : null}

      {!error && reports.length === 0 && !loading ? (
        <p className={cn('caption', spacing.margin.top.inline)}>{t('reports.empty')}</p>
      ) : null}

      {reports.map((report, index) => (
        <div key={report.id} data-testid="report-row">
          {index > 0 ? <CardDivider /> : null}
          <CardRow label={report.name} value={report.format.toUpperCase()} />
          <CardRow label={t('reports.status')} value={report.status} />
          {report.error ? <CardRow label={t('reports.error')} value={report.error} /> : null}
          <div className={cn('flex gap-2', spacing.margin.top.inline)}>
            {/* A report that has not finished has no file to fetch yet. */}
            {report.status === 'complete' ? (
              <a
                className="caption underline"
                href={`/api/v1/reports/${report.id}/download`}
                data-testid={`report-download-${report.id}`}
              >
                {t('reports.download')}
              </a>
            ) : null}
            {onDelete && terminalStatuses.has(report.status) ? (
              <button
                type="button"
                className="caption underline text-status-error"
                onClick={() => onDelete(report.id)}
                data-testid={`report-delete-${report.id}`}
              >
                {t('reports.delete')}
              </button>
            ) : null}
          </div>
        </div>
      ))}
    </Card>
  );
}
