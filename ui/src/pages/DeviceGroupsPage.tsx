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
import { Tooltip } from '../components/ui/Tooltip';
import { useRole } from '../contexts/RoleContext';
import { useDeviceGroupSummary, useDeviceGroups } from '../hooks/useDeviceGroups';
import { usePollingTargets } from '../hooks/usePollingTargets';
import type { DeviceGroupResponse } from '../types/generated/device-group-response';
import type {
  DeviceGroupAlertCounts,
  DeviceGroupMemberResponse,
} from '../types/generated/device-group-summary-response';
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
      {error ? (
        <div
          role="alert"
          className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error-strong"
        >
          {error}
        </div>
      ) : null}

      <div className="flex-between">
        <p className="body-small">
          {loading
            ? t('common:status.loading')
            : t('deviceGroups.groupCount', { count: groups.length })}
        </p>
        <Tooltip text={readOnlyReason}>
          <button
            type="button"
            onClick={(): void => setShowCreate(true)}
            disabled={!canWrite}
            data-testid="group-add"
            className="inline-flex items-center gap-compact rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:cursor-not-allowed disabled:control-disabled"
          >
            <Plus className="h-4 w-4" />
            {t('deviceGroups.addGroup')}
          </button>
        </Tooltip>
      </div>

      <ListDetail>
        <RecordPane
          filter={
            <input
              type="search"
              value={query}
              onChange={(e): void => setQuery(e.target.value)}
              placeholder={t('deviceGroups.filterPlaceholder', { count: groups.length })}
              aria-label={t('deviceGroups.filterLabel')}
              className="w-full rounded-lg border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none"
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
                <Tooltip text={readOnlyReason}>
                  <button
                    type="button"
                    onClick={(): void => setEditing(selected)}
                    disabled={!canWrite}
                    data-testid="group-edit"
                    className="rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover disabled:cursor-not-allowed disabled:control-disabled"
                  >
                    {t('common:buttons.edit')}
                  </button>
                </Tooltip>
                <Tooltip text={readOnlyReason}>
                  <button
                    type="button"
                    onClick={(): void => {
                      if (
                        window.confirm(t('deviceGroups.confirmDelete', { name: selected.name }))
                      ) {
                        void deleteGroup(selected);
                      }
                    }}
                    disabled={!canWrite}
                    data-testid="group-delete"
                    className="rounded-md px-3 py-2 text-sm text-status-error-strong hover:bg-status-error/10 disabled:cursor-not-allowed disabled:control-disabled"
                  >
                    {t('common:buttons.delete')}
                  </button>
                </Tooltip>
              </>
            }
          >
            {deleteError ? (
              <p
                role="alert"
                data-testid="group-delete-error"
                className="body-small text-status-error-strong"
              >
                {t('deviceGroups.deleteFailed', { reason: deleteError })}
              </p>
            ) : null}
            {summaryError ? (
              <p role="alert" className="body-small text-status-error-strong">
                {t('deviceGroups.summaryFailed', { reason: summaryError })}
              </p>
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
    <span className="text-status-error-strong">
      {t('deviceGroups.openAlertTotal', { count: counts.total })}
      {parts.length > 0 ? ` · ${parts.join(', ')}` : ''}
    </span>
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

const MEMBER_DOT: Record<RecordState, string> = {
  ok: 'bg-status-success',
  warn: 'bg-status-warning',
  crit: 'bg-status-error',
  unknown: 'bg-text-disabled',
};

function MemberList({ members }: { members: DeviceGroupMemberResponse[] | null }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  return (
    <div className="stack-xs">
      <p className="caption">{t('deviceGroups.membersHeading')}</p>
      {members === null ? (
        <p className="body-small">{t('common:status.loading')}</p>
      ) : members.length === 0 ? (
        <p data-testid="group-members-empty" className="body-small">
          {t('deviceGroups.noMembers')}
        </p>
      ) : (
        <ul className="divide-y divide-surface-border overflow-hidden rounded-lg border border-surface-border">
          {members.map((member) => (
            <li
              key={member.target.id}
              data-testid={`group-summary-member-${member.target.id}`}
              data-state={memberState(member)}
              className="flex flex-wrap items-center gap-x-default gap-y-1 px-cell py-2"
            >
              <span
                aria-hidden="true"
                className={`h-2 w-2 shrink-0 rounded-full ${MEMBER_DOT[memberState(member)]}`}
              />
              <span className="min-w-0 flex-1 truncate text-sm text-text-primary">
                {member.target.name}
              </span>
              <span className="figure text-xs text-text-muted">{member.target.ipAddress}</span>
              <span className="text-xs text-text-secondary">{memberPoll(member, t)}</span>
              <span className="figure text-xs text-text-secondary">
                {t('deviceGroups.memberAlerts', { count: member.openAlerts })}
              </span>
            </li>
          ))}
        </ul>
      )}
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
