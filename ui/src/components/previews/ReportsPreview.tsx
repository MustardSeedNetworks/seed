/**
 * ReportsPreview — the sample body <GatedPreview> shows for /reports.
 *
 * Renders the real ReportsCard, which is presentational, with a fixture typed
 * by the wire type: a schema change breaks the build here rather than shipping
 * a sample that no longer resembles the feature. No handlers are passed, so the
 * card renders in its read-only shape even before GatedPreview makes it inert.
 */

import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';
import type { ReportInfo } from '../../types/generated/reports-response';
import { ReportsCard } from '../cards/ReportsCard';

function sampleReports(t: TFunction<'errors'>): ReportInfo[] {
  return [
    {
      id: 'sample-executive',
      name: t('license.gated.samples.executiveSummary'),
      type: 'executive',
      format: 'pdf',
      status: 'complete',
      fileSize: 284_160,
      createdAt: '2026-03-04T09:12:00Z',
      completedAt: '2026-03-04T09:12:41Z',
    },
    {
      id: 'sample-inventory',
      name: t('license.gated.samples.deviceInventory'),
      type: 'inventory',
      format: 'csv',
      status: 'complete',
      fileSize: 48_720,
      createdAt: '2026-03-04T08:00:00Z',
      completedAt: '2026-03-04T08:00:09Z',
    },
  ];
}

export function ReportsPreview(): React.ReactElement {
  const { t } = useTranslation('errors');

  return <ReportsCard reports={sampleReports(t)} />;
}
