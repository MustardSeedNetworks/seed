/**
 * DeviceGroupsPage — List + detail over /api/v1/device-groups (UI-SEED-25).
 *
 * A device group is a named set of polling targets. Alert rules and scheduled
 * reports can be scoped to one, so this page is where the set is kept: the
 * list carries each group's name and size, the detail its members' last poll
 * and their open alerts from the group's summary read.
 */

import type { TFunction } from 'i18next';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus } from '../components/ui/Icons';
import { useRole } from '../contexts/RoleContext';
import { useDeviceGroupSummary, useDeviceGroups } from '../hooks/useDeviceGroups';
import { usePollingTargets } from '../hooks/usePollingTargets';
import { useTableLabels } from '../hooks/useTableLabels';
import { Alert } from '../shared/Alert';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { SearchInput } from '../shared/SearchInput';
import { StatusPill, type StatusTone } from '../shared/StatusPill';
import type { DeviceGroupResponse } from '../types/generated/device-group-response';
import type {
  DeviceGroupAlertCounts,
  DeviceGroupMemberResponse,
} from '../types/generated/device-group-summary-response';
import { GatedButton } from '../ui/GatedButton';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
  type RecordState,
} from '../ui/ListDetail';
import { emptyGroup, GroupForm, groupToInput } from './deviceGroups/GroupForm';

export function DeviceGroupsPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { canWrite } = useRole();
  const { groups, loading, error, create, update, remove } = useDeviceGroups();
  const { targets } = usePollingTargets();
  const [editing, setEditing] = useState<DeviceGroupResponse | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [deleting, setDeleting] = useState<DeviceGroupResponse | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const needle = query.trim().toLowerCase();
  const shown = groups.filter((group) => !needle || group.name.toLowerCase().includes(needle));
  const selected = shown.find((group) => group.id === selectedId) ?? shown[0] ?? null;
  const { summary, error: summaryError } = useDeviceGroupSummary(
    selected?.id ?? null,
    selected?.updatedAt ?? '',
  );

  const readOnlyReason = canWrite ? undefined : t('deviceGroups.readOnly');

  // A group an alert rule or scheduled report is scoped to cannot be deleted
  // (409); the server's reason says which, so it is shown as given.
  const deleteGroup = async (group: DeviceGroupResponse): Promise<void> => {
    setDeleteError(null);
    await remove(group.id)
      .then(() => setSelectedId(null))
      .catch((err: unknown) => setDeleteError(err instanceof Error ? err.message : String(err)));
  };

  return (
    <>
      {error ? <Alert tone="error">{error}</Alert> : null}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-text-secondary">
          {loading
            ? t('common:status.loading')
            : t('deviceGroups.groupCount', { count: groups.length })}
        </p>
        <GatedButton
          variant="primary"
          icon={<Plus className="h-4 w-4" />}
          reason={readOnlyReason}
          onClick={(): void => setShowCreate(true)}
          data-testid="group-add"
        >
          {t('deviceGroups.addGroup')}
        </GatedButton>
      </div>

      <ListDetail>
        <RecordPane
          filter={
            <SearchInput
              value={query}
              onValueChange={setQuery}
              placeholder={t('deviceGroups.filterPlaceholder', { count: groups.length })}
              label={t('deviceGroups.filterLabel')}
              clearLabel={t('common:buttons.clear')}
              className="max-w-none"
            />
          }
          empty={groups.length === 0 ? t('deviceGroups.noGroups') : t('deviceGroups.noMatch')}
        >
          {shown.map((group) => (
            <RecordRow
              key={group.id}
              data-testid={`group-row-${group.id}`}
              name={group.name}
              nameKind="prose"
              meta={group.description || undefined}
              value={t('deviceGroups.memberCount', { count: group.members.length })}
              selected={selected?.id === group.id}
              onSelect={(): void => {
                setSelectedId(group.id);
                setDeleteError(null);
              }}
            />
          ))}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow={t('deviceGroups.selectedGroup')}
            title={selected.name}
            meta={selected.description || undefined}
            actions={
              <>
                <GatedButton
                  reason={readOnlyReason}
                  onClick={(): void => setEditing(selected)}
                  data-testid="group-edit"
                >
                  {t('common:buttons.edit')}
                </GatedButton>
                <GatedButton
                  variant="destructive"
                  reason={readOnlyReason}
                  onClick={(): void => setDeleting(selected)}
                  data-testid="group-delete"
                >
                  {t('common:buttons.delete')}
                </GatedButton>
              </>
            }
          >
            {deleteError ? (
              <div data-testid="group-delete-error">
                <Alert tone="error">
                  {t('deviceGroups.deleteFailed', { reason: deleteError })}
                </Alert>
              </div>
            ) : null}
            {summaryError ? (
              <Alert tone="error">
                {t('deviceGroups.summaryFailed', { reason: summaryError })}
              </Alert>
            ) : null}
            <DetailFacts
              items={[
                {
                  label: t('deviceGroups.labelMembers'),
                  testId: 'group-member-total',
                  value: selected.members.length,
                },
                {
                  label: t('deviceGroups.labelOpenAlerts'),
                  testId: 'group-open-alerts',
                  value: summary ? <AlertCounts counts={summary.openAlerts} /> : '—',
                  prose: true,
                },
                {
                  label: t('deviceGroups.labelUpdated'),
                  value: new Date(selected.updatedAt).toLocaleString(),
                },
              ]}
            />
            <MemberList members={summary?.members ?? null} />
          </DetailPane>
        ) : (
          <DetailEmpty>
            {groups.length === 0
              ? t('deviceGroups.emptyAddPrompt')
              : t('deviceGroups.emptySelectPrompt')}
          </DetailEmpty>
        )}
      </ListDetail>

      {showCreate ? (
        <GroupForm
          mode="create"
          initial={emptyGroup()}
          targets={targets}
          onCancel={(): void => setShowCreate(false)}
          onSubmit={async (input): Promise<void> => {
            const created = await create(input);
            setSelectedId(created.id);
            setShowCreate(false);
          }}
        />
      ) : null}

      {editing ? (
        <GroupForm
          mode="edit"
          initial={groupToInput(editing)}
          targets={targets}
          onCancel={(): void => setEditing(null)}
          onSubmit={async (input): Promise<void> => {
            await update(editing.id, input);
            setEditing(null);
          }}
        />
      ) : null}

      <ConfirmDialog
        open={deleting !== null}
        title={t('deviceGroups.confirmDelete', { name: deleting?.name ?? '' })}
        message={t('deviceGroups.deleteMessage')}
        confirmLabel={t('common:buttons.delete')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={(): void => {
          if (deleting) {
            void deleteGroup(deleting);
          }
          setDeleting(null);
        }}
        onCancel={(): void => setDeleting(null)}
      />
    </>
  );
}

/** Only the severities with an open alert are named; none at all says so. */
function AlertCounts({ counts }: { counts: DeviceGroupAlertCounts }): JSX.Element {
  const { t } = useTranslation(['pages']);
  if (counts.total === 0) {
    return <span>{t('deviceGroups.noOpenAlerts')}</span>;
  }
  const parts = (['critical', 'error', 'warning', 'info'] as const)
    .filter((severity) => counts[severity] > 0)
    .map((severity) => t(`deviceGroups.severity.${severity}`, { count: counts[severity] }));
  return (
    <StatusPill
      tone="error"
      label={`${t('deviceGroups.openAlertTotal', { count: counts.total })}${parts.length > 0 ? ` · ${parts.join(', ')}` : ''}`}
    />
  );
}

/** A member is healthy only when its last poll succeeded; never polled is unknown. */
function memberState(member: DeviceGroupMemberResponse): RecordState {
  if (!member.target.enabled || !member.target.lastPolledAt) {
    return 'unknown';
  }
  if (member.target.lastStatus !== 'ok') {
    return 'crit';
  }
  return member.openAlerts > 0 ? 'warn' : 'ok';
}

const MEMBER_TONE: Record<RecordState, StatusTone> = {
  ok: 'success',
  warn: 'warning',
  crit: 'error',
  unknown: 'unknown',
};

const MEMBER_STATE_LABEL = {
  ok: 'deviceGroups.memberState.ok',
  warn: 'deviceGroups.memberState.warn',
  crit: 'deviceGroups.memberState.crit',
  unknown: 'deviceGroups.memberState.unknown',
} as const satisfies Record<RecordState, string>;

function MemberList({ members }: { members: DeviceGroupMemberResponse[] | null }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<DeviceGroupMemberResponse>({
    loading: t('deviceGroups.loadingMembers'),
  });
  const columns: DataTableColumn<DeviceGroupMemberResponse>[] = [
    {
      key: 'name',
      header: t('deviceGroups.columnName'),
      cell: (member) => (
        <span
          data-testid={`group-summary-member-${member.target.id}`}
          className="text-text-primary"
        >
          {member.target.name}
        </span>
      ),
      sortValue: (member) => member.target.name,
    },
    {
      key: 'address',
      header: t('deviceGroups.columnAddress'),
      cell: (member) => <span className="figure">{member.target.ipAddress}</span>,
      sortValue: (member) => member.target.ipAddress,
    },
    {
      key: 'state',
      header: t('deviceGroups.columnState'),
      cell: (member) => {
        const state = memberState(member);
        return <StatusPill tone={MEMBER_TONE[state]} label={t(MEMBER_STATE_LABEL[state])} />;
      },
      sortValue: (member) => memberState(member),
    },
    {
      key: 'lastPoll',
      header: t('deviceGroups.columnLastPoll'),
      cell: (member) => memberPoll(member, t),
      sortValue: (member) => member.target.lastPolledAt ?? '',
    },
    {
      key: 'alerts',
      header: t('deviceGroups.columnAlerts'),
      align: 'right',
      cell: (member) => (
        <span className="figure">
          {t('deviceGroups.memberAlerts', { count: member.openAlerts })}
        </span>
      ),
      sortValue: (member) => member.openAlerts,
    },
  ];
  return (
    <div className="space-y-1">
      <p className="kicker">{t('deviceGroups.membersHeading')}</p>
      <DataTable
        rows={members ?? []}
        columns={columns}
        getRowKey={(member) => member.target.id}
        caption={t('deviceGroups.membersHeading')}
        labels={labels}
        loading={members === null}
        pageSize={10}
        empty={
          <div data-testid="group-members-empty">
            <EmptyState title={t('deviceGroups.noMembers')} />
          </div>
        }
      />
    </div>
  );
}

function memberPoll(member: DeviceGroupMemberResponse, t: TFunction<['pages', 'common']>): string {
  const { target } = member;
  if (!target.enabled) {
    return t('pollingTargets.pollingPaused');
  }
  if (!target.lastPolledAt) {
    return t('pollingTargets.noPollYet');
  }
  if (target.lastStatus !== 'ok') {
    return target.lastError || t('pollingTargets.lastPollFailed');
  }
  return t('deviceGroups.polledAt', { time: new Date(target.lastPolledAt).toLocaleTimeString() });
}
