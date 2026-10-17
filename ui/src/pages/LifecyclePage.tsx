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
import { Upload } from '../components/ui/Icons';
import { Tooltip } from '../components/ui/Tooltip';
import { useRole } from '../contexts/RoleContext';
import { NotJsonError, useLifecycle } from '../hooks/useLifecycle';
import type {
  LifecycleComponent,
  LifecycleDevice,
} from '../types/generated/lifecycle-devices-response';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
  type RecordState,
} from '../ui/ListDetail';

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
const STATE: Partial<Record<Status, RecordState>> = {
  end_of_support: 'crit',
  end_of_security_support: 'crit',
  end_of_sw_maintenance: 'warn',
  end_of_sale: 'warn',
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

  return (
    <>
      {error ? (
        <div
          role="alert"
          className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error-strong"
        >
          {t('lifecycle.loadFailed', { reason: error })}
        </div>
      ) : null}

      <div className="flex-between">
        <p className="body-small" data-testid="lifecycle-counts">
          {data
            ? `${t('lifecycle.partCount', { count: parts.length })} · ${
                data.records > 0
                  ? t('lifecycle.recordCount', { count: data.records })
                  : t('lifecycle.noRecords')
              }`
            : t('common:status.loading')}
        </p>
        <Tooltip text={readOnlyReason}>
          <button
            type="button"
            onClick={(): void => fileInput.current?.click()}
            disabled={!canWrite || importing}
            data-testid="lifecycle-import"
            className="inline-flex items-center gap-compact rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:cursor-not-allowed disabled:control-disabled"
          >
            <Upload className="h-4 w-4" />
            {t('lifecycle.import')}
          </button>
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
        <p
          role={importResult.ok ? 'status' : 'alert'}
          data-testid="lifecycle-import-result"
          className={
            importResult.ok
              ? 'body-small text-status-success'
              : 'body-small text-status-error-strong'
          }
        >
          {importResult.text}
        </p>
      ) : null}

      <ListDetail>
        <RecordPane empty={data ? t('lifecycle.noParts') : t('common:status.loading')}>
          {parts.map((part) => (
            <RecordRow
              key={part.key}
              data-testid={`lifecycle-part-${part.key}`}
              name={part.component.productId}
              meta={[part.device.name || part.device.ip, part.component.name]
                .filter(Boolean)
                .join(' · ')}
              value={t(`lifecycle.status.${part.component.status}`)}
              state={STATE[part.component.status]}
              selected={selected?.key === part.key}
              onSelect={(): void => setSelectedKey(part.key)}
            />
          ))}
        </RecordPane>

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
              <p data-testid="lifecycle-not-listed" className="body-small">
                {data && data.records > 0 ? t('lifecycle.notListed') : t('lifecycle.importFirst')}
              </p>
            )}
            {selected.component.record?.bulletin && isHttps(selected.component.record.bulletin) ? (
              <a
                href={selected.component.record.bulletin}
                target="_blank"
                rel="noopener noreferrer"
                data-testid="lifecycle-bulletin"
                className="body-small text-status-info hover:underline"
              >
                {t('lifecycle.bulletin')}
              </a>
            ) : null}
          </DetailPane>
        ) : (
          <DetailEmpty>{t('lifecycle.emptyPrompt')}</DetailEmpty>
        )}
      </ListDetail>
    </>
  );
}
