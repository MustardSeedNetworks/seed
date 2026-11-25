/**
 * LifecyclePage — hardware end-of-life over /api/v1/lifecycle (P-D9,
 * UI-SEED-47).
 *
 * Lists every part discovery inventoried over ENTITY-MIB with where its
 * product stands in Cisco's EoX data, parts past the furthest milestone
 * first. The server assesses each status against today; the page only ranks
 * and words it. An operator imports a downloaded EoX API response file here.
 * The feed that fetches the same data is an admin setting, in Settings.
 */

import { type ChangeEvent, type JSX, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError } from '../api/client';
import { Cpu, Upload } from '../components/ui/Icons';
import { useRole } from '../contexts/RoleContext';
import { NotJsonError, useLifecycle } from '../hooks/useLifecycle';
import { useTableLabels } from '../hooks/useTableLabels';
import { Alert } from '../shared/Alert';
import { Button } from '../shared/Button';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { StatusPill, type StatusTone } from '../shared/StatusPill';
import { Tooltip } from '../shared/Tooltip';
import type {
  LifecycleComponent,
  LifecycleDevice,
} from '../types/generated/lifecycle-devices-response';
import { DetailEmpty, DetailFacts, DetailPane } from '../ui/ListDetail';

type Status = LifecycleComponent['status'];

/** How far through its end-of-life a part is; higher sorts first. */
const RANK: Record<Status, number> = {
  end_of_support: 5,
  end_of_security_support: 4,
  end_of_sw_maintenance: 3,
  end_of_sale: 2,
  announced: 1,
  not_listed: 0,
};

/**
 * No security fixes is the line that makes a part a risk. An announced or
 * unlisted part claims no state: neither is evidence it is healthy.
 */
const TONE: Record<Status, StatusTone> = {
  end_of_support: 'error',
  end_of_security_support: 'error',
  end_of_sw_maintenance: 'warning',
  end_of_sale: 'warning',
  announced: 'unknown',
  not_listed: 'unknown',
};

interface Part {
  key: string;
  device: LifecycleDevice;
  component: LifecycleComponent;
}

function partsOf(devices: LifecycleDevice[]): Part[] {
  const parts = devices.flatMap((device) =>
    device.components.map((component, index) => ({
      key: `${device.ip}#${index}`,
      device,
      component,
    })),
  );
  // Stable, so equal ranks keep the server's address and entity order.
  return parts.sort((a, b) => RANK[b.component.status] - RANK[a.component.status]);
}

/** Only an https bulletin is linked; the imported file is operator data. */
function isHttps(url: string): boolean {
  return URL.canParse(url) && new URL(url).protocol === 'https:';
}

export function LifecyclePage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { canWrite } = useRole();
  const { data, error, importEoX } = useLifecycle();
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<{ ok: boolean; text: string } | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  const parts = partsOf(data?.devices ?? []);
  const selected = parts.find((part) => part.key === selectedKey) ?? parts[0] ?? null;
  const readOnlyReason = canWrite ? undefined : t('lifecycle.readOnly');

  const onFile = async (event: ChangeEvent<HTMLInputElement>): Promise<void> => {
    const file = event.target.files?.[0];
    // Cleared so choosing the same file again still fires a change.
    event.target.value = '';
    if (!file) {
      return;
    }
    setImporting(true);
    setImportResult(null);
    await importEoX(file)
      .then((count) => setImportResult({ ok: true, text: t('lifecycle.imported', { count }) }))
      .catch((err: unknown) => {
        let text = t('lifecycle.importFailed', {
          reason: err instanceof Error ? err.message : String(err),
        });
        if (err instanceof NotJsonError || (err instanceof ApiError && err.status === 400)) {
          text = t('lifecycle.notEoX');
        } else if (err instanceof ApiError && err.status === 413) {
          text = t('lifecycle.tooLarge');
        }
        setImportResult({ ok: false, text });
      })
      .finally(() => setImporting(false));
  };

  const fmtDate = (value: string | undefined): string => value || t('lifecycle.notAnnounced');

  const labels = useTableLabels<Part>({ loading: t('common:status.loading') });
  const columns: DataTableColumn<Part>[] = [
    {
      key: 'part',
      header: t('lifecycle.colPart'),
      sortValue: (part) => part.component.productId,
      cell: (part) => (
        <div className="flex min-w-0 flex-col gap-0.5 py-1">
          <span className="figure text-text-primary">{part.component.productId}</span>
          <span className="text-xs text-text-muted">
            {[part.device.name || part.device.ip, part.component.name].filter(Boolean).join(' · ')}
          </span>
        </div>
      ),
    },
    {
      key: 'status',
      header: t('lifecycle.colStatus'),
      sortValue: (part) => RANK[part.component.status],
      cell: (part) => (
        <StatusPill
          tone={TONE[part.component.status]}
          label={t(`lifecycle.status.${part.component.status}`)}
        />
      ),
    },
  ];

  return (
    <>
      {error ? <Alert tone="error">{t('lifecycle.loadFailed', { reason: error })}</Alert> : null}

      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-text-muted" data-testid="lifecycle-counts">
          {data
            ? `${t('lifecycle.partCount', { count: parts.length })} · ${
                data.records > 0
                  ? t('lifecycle.recordCount', { count: data.records })
                  : t('lifecycle.noRecords')
              }`
            : t('common:status.loading')}
        </p>
        <Tooltip text={readOnlyReason}>
          {(description) => (
            <Button
              variant="primary"
              icon={<Upload className="h-4 w-4" />}
              loading={importing}
              // aria-disabled, not disabled: a viewer can still focus the
              // button and read why it does nothing.
              aria-disabled={canWrite ? undefined : true}
              onClick={(): void => {
                description.onClick();
                if (canWrite) {
                  fileInput.current?.click();
                }
              }}
              aria-describedby={description['aria-describedby']}
              data-testid="lifecycle-import"
            >
              {t('lifecycle.import')}
            </Button>
          )}
        </Tooltip>
        <input
          ref={fileInput}
          type="file"
          accept="application/json,.json"
          hidden
          data-testid="lifecycle-import-file"
          onChange={(event): void => void onFile(event)}
        />
      </div>

      {importResult ? (
        <div data-testid="lifecycle-import-result">
          <Alert tone={importResult.ok ? 'success' : 'error'}>{importResult.text}</Alert>
        </div>
      ) : null}

      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
        <DataTable
          rows={parts}
          columns={columns}
          getRowKey={(part) => part.key}
          caption={t('lifecycle.tableCaption')}
          labels={labels}
          loading={data === null && !error}
          pageSize={25}
          defaultSort={{ key: 'status', direction: 'desc' }}
          onRowClick={(part): void => setSelectedKey(part.key)}
          selectedRowKey={selected?.key}
          empty={
            <EmptyState
              icon={<Cpu className="h-6 w-6" aria-hidden="true" />}
              title={t('lifecycle.noPartsTitle')}
              hint={t('lifecycle.noPartsHint')}
            />
          }
        />

        {selected ? (
          <DetailPane
            eyebrow={t(`lifecycle.status.${selected.component.status}`)}
            title={selected.component.productId}
            meta={selected.component.record?.description || undefined}
          >
            <DetailFacts
              items={[
                {
                  label: t('lifecycle.labelDevice'),
                  testId: 'lifecycle-device',
                  value: selected.device.name
                    ? `${selected.device.name} (${selected.device.ip})`
                    : selected.device.ip,
                },
                {
                  label: t('lifecycle.labelPart'),
                  value: selected.component.name || '—',
                  prose: true,
                },
                { label: t('lifecycle.labelSerial'), value: selected.component.serialNum || '—' },
              ]}
            />
            {selected.component.record ? (
              <DetailFacts
                items={[
                  {
                    label: t('lifecycle.labelEndOfSale'),
                    testId: 'lifecycle-end-of-sale',
                    value: fmtDate(selected.component.record.endOfSale),
                  },
                  {
                    label: t('lifecycle.labelEndOfSwMaintenance'),
                    value: fmtDate(selected.component.record.endOfSwMaintenance),
                  },
                  {
                    label: t('lifecycle.labelEndOfSecuritySupport'),
                    value: fmtDate(selected.component.record.endOfSecuritySupport),
                  },
                  {
                    label: t('lifecycle.labelLastDateOfSupport'),
                    testId: 'lifecycle-last-support',
                    value: fmtDate(selected.component.record.lastDateOfSupport),
                  },
                  {
                    label: t('lifecycle.labelReplacement'),
                    value: selected.component.record.migrationProductId || '—',
                  },
                ]}
              />
            ) : (
              <p data-testid="lifecycle-not-listed" className="text-sm text-text-muted">
                {data && data.records > 0 ? t('lifecycle.notListed') : t('lifecycle.importFirst')}
              </p>
            )}
            {selected.component.record?.bulletin && isHttps(selected.component.record.bulletin) ? (
              <a
                href={selected.component.record.bulletin}
                target="_blank"
                rel="noopener noreferrer"
                data-testid="lifecycle-bulletin"
                className="text-sm text-status-info-strong hover:underline"
              >
                {t('lifecycle.bulletin')}
              </a>
            ) : null}
          </DetailPane>
        ) : (
          <DetailEmpty>{t('lifecycle.emptyPrompt')}</DetailEmpty>
        )}
      </div>
    </>
  );
}
