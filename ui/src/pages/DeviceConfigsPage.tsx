/**
 * DeviceConfigsPage — saved device configurations and what changed (UI-SEED-25).
 *
 * The list is the devices backed up over SSH or NETCONF (P-D1); the detail is
 * the selected device's backup attempts, newest first, and the selected
 * attempt's change from the configuration before it (P-D2), with added and
 * removed lines highlighted. An operator adds, edits and removes devices,
 * runs a backup of one or all, and confirms or forgets each device's SSH host
 * key (UI-SEED-45). Below them, an admin runs the TFTP server that devices
 * push and pull files over (P-D4).
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { DeviceConfigsPreview } from '../components/previews/DeviceConfigsPreview';
import { GatedPreview } from '../components/ui/GatedPreview';
import { useRole } from '../contexts/RoleContext';
import {
  type BackupRunState,
  deleteDeviceConfigTarget,
  saveDeviceConfigTarget,
  useBackupRun,
  useDeviceConfigBackups,
  useDeviceConfigDiff,
  useDeviceConfigTargets,
} from '../hooks/useDeviceConfigs';
import { useTableLabels } from '../hooks/useTableLabels';
import { Alert } from '../shared/Alert';
import { Button } from '../shared/Button';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { SearchInput } from '../shared/SearchInput';
import { StatusPill } from '../shared/StatusPill';
import type {
  DeviceConfigBackup,
  DeviceConfigTarget,
  DeviceConfigTargetInput,
} from '../types/deviceConfigs';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
} from '../ui/ListDetail';
import { DiffView } from './deviceConfigs/DiffView';
import { HostKey } from './deviceConfigs/HostKey';
import { TargetForm } from './deviceConfigs/TargetForm';
import { TftpSession } from './deviceConfigs/TftpSession';

export function DeviceConfigsPage(): JSX.Element {
  return (
    <GatedPreview feature="compliance_advanced" preview={<DeviceConfigsPreview />}>
      <DeviceConfigsBody />
    </GatedPreview>
  );
}

function DeviceConfigsBody(): JSX.Element {
  const { t } = useTranslation(['pages']);
  // A saved configuration carries the device's secrets, so every read is
  // operator+; a viewer is told so instead of being shown a 403.
  const { canWrite } = useRole();
  if (!canWrite) {
    return <DetailEmpty>{t('deviceConfigs.operatorOnly')}</DetailEmpty>;
  }
  return <DeviceConfigsManager />;
}

/** The form's subject: a new device, or the one being edited. */
type Editing = { target?: DeviceConfigTarget } | null;

function DeviceConfigsManager(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { targets, loading, error, reload } = useDeviceConfigTargets();
  const [backupsVersion, setBackupsVersion] = useState(0);
  // A finished run can add attempts and record a device's pending host key.
  const { run, start } = useBackupRun(() => {
    reload();
    setBackupsVersion((v) => v + 1);
  });
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [editing, setEditing] = useState<Editing>(null);
  const [removing, setRemoving] = useState<DeviceConfigTarget | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);

  const needle = query.trim().toLowerCase();
  const shown = targets.filter(
    (target) =>
      !needle ||
      target.name.toLowerCase().includes(needle) ||
      target.host.toLowerCase().includes(needle),
  );
  const selected = shown.find((target) => target.id === selectedId) ?? shown[0] ?? null;
  const running = run.kind === 'running';

  const save = async (input: DeviceConfigTargetInput): Promise<void> => {
    const saved = await saveDeviceConfigTarget(input, editing?.target?.id);
    setEditing(null);
    setSelectedId(saved.id);
    reload();
  };

  const remove = async (target: DeviceConfigTarget): Promise<void> => {
    setRemoving(null);
    try {
      await deleteDeviceConfigTarget(target.id);
      setRemoveError(null);
      reload();
    } catch (err: unknown) {
      setRemoveError(err instanceof Error ? err.message : String(err));
    }
  };

  const listError = error ?? removeError;

  return (
    <>
      {listError ? <Alert tone="error">{listError}</Alert> : null}

      <div className="flex flex-wrap items-center gap-3">
        <p className="min-w-0 flex-1 text-sm text-text-secondary">
          {loading
            ? t('common:status.loading')
            : t('deviceConfigs.deviceCount', { count: targets.length })}
        </p>
        <Button
          data-testid="config-run-all"
          size="sm"
          disabled={running || targets.length === 0}
          onClick={(): void => void start()}
        >
          {t('deviceConfigs.run.all')}
        </Button>
        <Button
          data-testid="config-target-add"
          variant="primary"
          size="sm"
          onClick={(): void => setEditing({})}
        >
          {t('deviceConfigs.addDevice')}
        </Button>
      </div>
      <RunStatus run={run} />

      <ListDetail>
        <RecordPane
          filter={
            <SearchInput
              value={query}
              onValueChange={setQuery}
              placeholder={t('deviceConfigs.filterPlaceholder', { count: targets.length })}
              label={t('deviceConfigs.filterLabel')}
              clearLabel={t('common:buttons.clear')}
              className="max-w-none"
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
          <TargetDetail
            key={selected.id}
            target={selected}
            backupsVersion={backupsVersion}
            running={running}
            onRun={(): void => void start([selected.id])}
            onEdit={(): void => setEditing({ target: selected })}
            onRemove={(): void => setRemoving(selected)}
            onHostKeyChanged={reload}
          />
        ) : (
          <DetailEmpty>
            {targets.length === 0
              ? t('deviceConfigs.emptyAddPrompt')
              : t('deviceConfigs.emptySelectPrompt')}
          </DetailEmpty>
        )}
      </ListDetail>

      <TftpSession />

      {editing ? (
        <TargetForm
          target={editing.target}
          onSubmit={save}
          onCancel={(): void => setEditing(null)}
        />
      ) : null}
      <ConfirmDialog
        open={removing !== null}
        title={t('deviceConfigs.remove.title', { name: removing?.name ?? '' })}
        message={t('deviceConfigs.remove.message')}
        confirmLabel={t('deviceConfigs.remove.confirm')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={(): void => {
          if (removing) {
            void remove(removing);
          }
        }}
        onCancel={(): void => setRemoving(null)}
      />
    </>
  );
}

function RunStatus({ run }: { run: BackupRunState }): JSX.Element | null {
  const { t } = useTranslation(['pages']);
  switch (run.kind) {
    case 'idle':
      return null;
    case 'running':
      return (
        <div data-testid="config-run-status">
          <Alert tone="info">{t('deviceConfigs.run.running')}</Alert>
        </div>
      );
    case 'finished':
      return (
        <div data-testid="config-run-status">
          <Alert tone={run.result.failed > 0 ? 'warning' : 'success'}>
            {t('deviceConfigs.run.finished', {
              succeeded: run.result.succeeded,
              failed: run.result.failed,
            })}
          </Alert>
        </div>
      );
    case 'failed':
      return (
        <div data-testid="config-run-status">
          <Alert tone="error">{t('deviceConfigs.run.failed', { reason: run.message })}</Alert>
        </div>
      );
  }
}

/** Keyed by target, so a new device starts on its own newest backup. */
function TargetDetail({
  target,
  backupsVersion,
  running,
  onRun,
  onEdit,
  onRemove,
  onHostKeyChanged,
}: {
  target: DeviceConfigTarget;
  backupsVersion: number;
  running: boolean;
  onRun: () => void;
  onEdit: () => void;
  onRemove: () => void;
  onHostKeyChanged: () => void;
}): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { backups, error } = useDeviceConfigBackups(target.id, backupsVersion);
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
      <div className="flex flex-wrap gap-2">
        <Button data-testid="config-run-one" size="sm" disabled={running} onClick={onRun}>
          {t('deviceConfigs.run.one')}
        </Button>
        <Button data-testid="config-target-edit" size="sm" onClick={onEdit}>
          {t('deviceConfigs.edit')}
        </Button>
        <Button
          data-testid="config-target-remove"
          variant="destructive"
          size="sm"
          onClick={onRemove}
        >
          {t('deviceConfigs.remove.action')}
        </Button>
      </div>
      <HostKey target={target} onChanged={onHostKeyChanged} />
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
    return <Alert tone="error">{t('deviceConfigs.backupsFailed', { reason: error })}</Alert>;
  }
  return <BackupList backups={backups} pickedId={pickedId} onPick={onPick} />;
}

function BackupList({
  backups,
  pickedId,
  onPick,
}: {
  backups: DeviceConfigBackup[] | null;
  pickedId: string | null;
  onPick: (id: string) => void;
}): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<DeviceConfigBackup>({ loading: t('deviceConfigs.loadingBackups') });
  const columns: DataTableColumn<DeviceConfigBackup>[] = [
    {
      key: 'takenAt',
      header: t('deviceConfigs.columnTaken'),
      cell: (backup) => (
        <span data-testid={`config-backup-${backup.id}`} className="figure text-text-primary">
          {new Date(backup.takenAt).toLocaleString()}
        </span>
      ),
      sortValue: (backup) => backup.takenAt,
    },
    {
      key: 'status',
      header: t('deviceConfigs.columnResult'),
      cell: (backup) => (
        <StatusPill
          tone={backup.status === 'ok' ? 'success' : 'error'}
          label={t(`deviceConfigs.status.${backup.status}`)}
        />
      ),
      sortValue: (backup) => backup.status,
    },
  ];
  return (
    <div className="space-y-1">
      <p className="kicker">{t('deviceConfigs.backupsHeading')}</p>
      <DataTable
        rows={backups ?? []}
        columns={columns}
        getRowKey={(backup) => backup.id}
        caption={t('deviceConfigs.backupsHeading')}
        labels={labels}
        loading={backups === null}
        defaultSort={{ key: 'takenAt', direction: 'desc' }}
        pageSize={10}
        onRowClick={(backup): void => onPick(backup.id)}
        {...(pickedId === null ? {} : { selectedRowKey: pickedId })}
        empty={
          <div data-testid="config-backups-empty">
            <EmptyState title={t('deviceConfigs.noBackups')} />
          </div>
        }
      />
    </div>
  );
}

/** A failed attempt holds no configuration, so it shows why instead of a diff. */
function BackupChange({ backup }: { backup: DeviceConfigBackup }): JSX.Element {
  const { t } = useTranslation(['pages']);
  if (backup.status !== 'ok') {
    return (
      <div data-testid="config-backup-failed">
        <Alert tone="error">
          {t('deviceConfigs.attemptFailed', {
            status: t(`deviceConfigs.status.${backup.status}`),
            reason: backup.error || '—',
          })}
        </Alert>
      </div>
    );
  }
  return <SavedChange backupId={backup.id} />;
}

function SavedChange({ backupId }: { backupId: string }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const state = useDeviceConfigDiff(backupId);

  switch (state.kind) {
    case 'loading':
      return <p className="text-sm text-text-secondary">{t('common:status.loading')}</p>;
    case 'first':
      return (
        <div data-testid="config-diff-first">
          <Alert tone="info">{t('deviceConfigs.firstBackup')}</Alert>
        </div>
      );
    case 'error':
      return <Alert tone="error">{t('deviceConfigs.diffFailed', { reason: state.message })}</Alert>;
    case 'ready': {
      const { diff } = state;
      return (
        <div className="space-y-3">
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
            <p data-testid="config-diff-unchanged" className="text-sm text-text-secondary">
              {t('deviceConfigs.unchanged')}
            </p>
          )}
        </div>
      );
    }
  }
}
