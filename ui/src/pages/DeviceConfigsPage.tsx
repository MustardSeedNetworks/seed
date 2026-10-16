/**
 * DeviceConfigsPage — saved device configurations and what changed (UI-SEED-25).
 *
 * The list is the devices backed up over SSH or NETCONF (P-D1); the detail is
 * the selected device's backup attempts, newest first, and the selected
 * attempt's change from the configuration before it (P-D2), with added and
 * removed lines highlighted. Read-only: targets and runs are API-driven until
 * the backup management UI (#3121) lands.
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { DeviceConfigsPreview } from '../components/previews/DeviceConfigsPreview';
import { GatedPreview } from '../components/ui/GatedPreview';
import { useRole } from '../contexts/RoleContext';
import {
  useDeviceConfigBackups,
  useDeviceConfigDiff,
  useDeviceConfigTargets,
} from '../hooks/useDeviceConfigs';
import { cn } from '../styles/theme';
import type { DeviceConfigBackup, DeviceConfigTarget } from '../types/deviceConfigs';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
} from '../ui/ListDetail';
import { DiffView } from './deviceConfigs/DiffView';

export function DeviceConfigsPage(): JSX.Element {
  return (
    <GatedPreview feature="compliance_advanced" preview={<DeviceConfigsPreview />}>
      <DeviceConfigsBody />
    </GatedPreview>
  );
}

function DeviceConfigsBody(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  // A saved configuration carries the device's secrets, so every read is
  // operator+; a viewer is told so instead of being shown a 403.
  const { canWrite } = useRole();
  const { targets, loading, error } = useDeviceConfigTargets(canWrite);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [query, setQuery] = useState('');

  if (!canWrite) {
    return <DetailEmpty>{t('deviceConfigs.operatorOnly')}</DetailEmpty>;
  }

  const needle = query.trim().toLowerCase();
  const shown = targets.filter(
    (target) =>
      !needle ||
      target.name.toLowerCase().includes(needle) ||
      target.host.toLowerCase().includes(needle),
  );
  const selected = shown.find((target) => target.id === selectedId) ?? shown[0] ?? null;

  return (
    <>
      {error ? (
        <div
          role="alert"
          className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error-strong"
        >
          {error}
        </div>
      ) : null}

      <p className="body-small">
        {loading
          ? t('common:status.loading')
          : t('deviceConfigs.deviceCount', { count: targets.length })}
      </p>

      <ListDetail>
        <RecordPane
          filter={
            <input
              type="search"
              value={query}
              onChange={(e): void => setQuery(e.target.value)}
              placeholder={t('deviceConfigs.filterPlaceholder', { count: targets.length })}
              aria-label={t('deviceConfigs.filterLabel')}
              className="w-full rounded-lg border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none"
            />
          }
          empty={targets.length === 0 ? t('deviceConfigs.noTargets') : t('deviceConfigs.noMatch')}
        >
          {shown.map((target) => (
            <RecordRow
              key={target.id}
              data-testid={`config-target-${target.id}`}
              name={target.name}
              nameKind="prose"
              meta={`${target.host}:${target.port}`}
              value={t(`deviceConfigs.platform.${target.platform}`)}
              selected={selected?.id === target.id}
              onSelect={(): void => setSelectedId(target.id)}
            />
          ))}
        </RecordPane>

        {selected ? (
          <TargetDetail key={selected.id} target={selected} />
        ) : (
          <DetailEmpty>
            {targets.length === 0
              ? t('deviceConfigs.emptyAddPrompt')
              : t('deviceConfigs.emptySelectPrompt')}
          </DetailEmpty>
        )}
      </ListDetail>
    </>
  );
}

/** Keyed by target, so a new device starts on its own newest backup. */
function TargetDetail({ target }: { target: DeviceConfigTarget }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { backups, error } = useDeviceConfigBackups(target.id);
  const [pickedId, setPickedId] = useState<string | null>(null);

  // The newest saved configuration is what an operator opens the page for.
  const picked =
    backups?.find((backup) => backup.id === pickedId) ??
    backups?.find((backup) => backup.status === 'ok') ??
    backups?.[0] ??
    null;

  return (
    <DetailPane
      eyebrow={t('deviceConfigs.selectedDevice')}
      title={target.name}
      meta={`${target.host}:${target.port} · ${t(`deviceConfigs.platform.${target.platform}`)}`}
    >
      <BackupsBody
        backups={backups}
        error={error}
        pickedId={picked?.id ?? null}
        onPick={setPickedId}
      />
      {picked ? <BackupChange backup={picked} /> : null}
    </DetailPane>
  );
}

function BackupsBody({
  backups,
  error,
  pickedId,
  onPick,
}: {
  backups: DeviceConfigBackup[] | null;
  error: string | null;
  pickedId: string | null;
  onPick: (id: string) => void;
}): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  if (error) {
    return (
      <p role="alert" className="body-small text-status-error-strong">
        {t('deviceConfigs.backupsFailed', { reason: error })}
      </p>
    );
  }
  if (backups === null) {
    return <p className="body-small">{t('common:status.loading')}</p>;
  }
  if (backups.length === 0) {
    return (
      <p data-testid="config-backups-empty" className="body-small">
        {t('deviceConfigs.noBackups')}
      </p>
    );
  }
  return <BackupList backups={backups} pickedId={pickedId} onPick={onPick} />;
}

const STATUS_DOT: Record<'ok' | 'failed', string> = {
  ok: 'bg-status-success',
  failed: 'bg-status-error',
};

function BackupList({
  backups,
  pickedId,
  onPick,
}: {
  backups: DeviceConfigBackup[];
  pickedId: string | null;
  onPick: (id: string) => void;
}): JSX.Element {
  const { t } = useTranslation(['pages']);
  return (
    <div className="stack-xs">
      <p className="caption">{t('deviceConfigs.backupsHeading')}</p>
      <ul className="max-h-60 divide-y divide-surface-border overflow-auto rounded-lg border border-surface-border">
        {backups.map((backup) => (
          <li key={backup.id}>
            <button
              type="button"
              data-testid={`config-backup-${backup.id}`}
              aria-pressed={backup.id === pickedId}
              onClick={(): void => onPick(backup.id)}
              className={cn(
                'flex w-full flex-wrap items-center gap-x-default gap-y-1 px-cell py-2 text-left hover:bg-surface-hover',
                backup.id === pickedId && 'bg-surface-hover',
              )}
            >
              <span
                aria-hidden="true"
                className={`h-2 w-2 shrink-0 rounded-full ${STATUS_DOT[backup.status === 'ok' ? 'ok' : 'failed']}`}
              />
              <span className="figure min-w-0 flex-1 text-sm text-text-primary">
                {new Date(backup.takenAt).toLocaleString()}
              </span>
              <span className="text-xs text-text-secondary">
                {t(`deviceConfigs.status.${backup.status}`)}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** A failed attempt holds no configuration, so it shows why instead of a diff. */
function BackupChange({ backup }: { backup: DeviceConfigBackup }): JSX.Element {
  const { t } = useTranslation(['pages']);
  if (backup.status !== 'ok') {
    return (
      <p data-testid="config-backup-failed" className="body-small text-status-error-strong">
        {t('deviceConfigs.attemptFailed', {
          status: t(`deviceConfigs.status.${backup.status}`),
          reason: backup.error || '—',
        })}
      </p>
    );
  }
  return <SavedChange backupId={backup.id} />;
}

function SavedChange({ backupId }: { backupId: string }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const state = useDeviceConfigDiff(backupId);

  switch (state.kind) {
    case 'loading':
      return <p className="body-small">{t('common:status.loading')}</p>;
    case 'first':
      return (
        <p data-testid="config-diff-first" className="body-small">
          {t('deviceConfigs.firstBackup')}
        </p>
      );
    case 'error':
      return (
        <p role="alert" className="body-small text-status-error-strong">
          {t('deviceConfigs.diffFailed', { reason: state.message })}
        </p>
      );
    case 'ready': {
      const { diff } = state;
      return (
        <div className="stack-sm">
          <DetailFacts
            items={[
              {
                label: t('deviceConfigs.labelComparedWith'),
                value: new Date(diff.previousTakenAt).toLocaleString(),
              },
              {
                label: t('deviceConfigs.labelChanges'),
                testId: 'config-diff-counts',
                value: t('deviceConfigs.changeCounts', {
                  added: diff.added,
                  removed: diff.removed,
                }),
              },
            ]}
          />
          {diff.unified ? (
            <DiffView unified={diff.unified} />
          ) : (
            <p data-testid="config-diff-unchanged" className="body-small">
              {t('deviceConfigs.unchanged')}
            </p>
          )}
        </div>
      );
    }
  }
}
