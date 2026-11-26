/**
 * UsersSettings Component
 *
 * Settings → Users panel for the multi_user feature (seed#1191). Admin
 * accounts can list / create / promote / demote / delete users; non-
 * admins see their own row only via /api/v1/users/me. Creating users
 * is Pro-gated server-side via the multi_user feature; the Create
 * button is disabled with an explanatory tooltip on Free / Starter.
 *
 * SSO-linked users (seed#1198) appear with an Auth column indicating
 * google / microsoft / github so operators can distinguish local from
 * SSO identities. Editing role + active state works the same for both;
 * password change is local-auth only and the input is hidden for SSO
 * rows.
 */

import type { TFunction } from 'i18next';
import type React from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../../api/client';
import { useLicense } from '../../../contexts/LicenseContext';
import { useTableLabels } from '../../../hooks/useTableLabels';
import { Button } from '../../../shared/Button';
import { CategoryChip } from '../../../shared/CategoryChip';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { ConfirmDialog } from '../../../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../../../shared/DataTable';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Select } from '../../../shared/Select';
import { StatusPill, type StatusTone } from '../../../shared/StatusPill';
import { Tooltip } from '../../../shared/Tooltip';
import { Plus, Trash2, Users } from '../../ui/Icons';

interface UserRow {
  id: number;
  username: string;
  role: 'admin' | 'operator' | 'viewer';
  isActive: boolean;
  authProvider: 'local' | 'google' | 'microsoft' | 'github';
  email?: string;
  displayName?: string;
  lastLogin?: string;
  lockedUntilFuture?: boolean;
  createdAt: string;
  updatedAt: string;
}

const ROLES = ['admin', 'operator', 'viewer'] as const;
type Role = (typeof ROLES)[number];

function formatLastLogin(value: string | undefined): string {
  if (!value) return '—';
  try {
    return new Date(value).toLocaleString();
  } catch {
    return value;
  }
}

// i18next's TFunction carries the namespace tuple as a generic so the
// keys are typed against the loaded resource bundles. Use the project's
// concrete TFunction type rather than the loose (k: string) => string
// signature — that older helper signature is no longer assignable from
// useTranslation's return value under stricter TS.
type StatusT = TFunction<readonly ['settings', 'errors']>;

function statusOf(u: UserRow, t: StatusT): string {
  if (!u.isActive) return t('settings:users.status.disabled');
  if (u.lockedUntilFuture) return t('settings:users.status.locked');
  return t('settings:users.status.active');
}

function statusTone(u: UserRow): StatusTone {
  if (!u.isActive) return 'unknown';
  return u.lockedUntilFuture ? 'warning' : 'success';
}

function toRole(value: string): Role {
  return ROLES.find((r) => r === value) ?? 'viewer';
}

function listUsers(): Promise<UserRow[]> {
  return api.get<UserRow[] | null>('/api/v1/users').then((list) => list ?? []);
}

export function UsersSettings(): React.ReactElement {
  const { t } = useTranslation(['settings', 'errors', 'common']);
  const { status: licenseStatus, refresh: refreshLicense } = useLicense();

  const [users, setUsers] = useState<UserRow[]>([]);
  const [me, setMe] = useState<UserRow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create-user form
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState<Role>('viewer');
  const [creating, setCreating] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<UserRow | null>(null);
  const labels = useTableLabels<UserRow>({ loading: t('settings:users.title') });

  const isAdmin = me?.role === 'admin';
  const roleOptions = [
    { value: 'admin', label: t('settings:users.roles.admin') },
    { value: 'operator', label: t('settings:users.roles.operator') },
    { value: 'viewer', label: t('settings:users.roles.viewer') },
  ];

  const columns: DataTableColumn<UserRow>[] = [
    {
      key: 'username',
      header: t('settings:users.columns.username'),
      sortValue: (u) => u.username,
      cell: (u) => (
        <span data-testid={`user-row-${u.username}`}>
          {u.username}
          {me?.username === u.username && (
            <span className="ml-1 text-xs text-text-muted">{t('settings:users.currentUser')}</span>
          )}
        </span>
      ),
    },
    {
      key: 'role',
      header: t('settings:users.columns.role'),
      sortValue: (u) => u.role,
      cell: (u) =>
        isAdmin && me?.username !== u.username ? (
          <Select
            aria-label={t('settings:users.columns.role')}
            value={u.role}
            options={roleOptions}
            onValueChange={(value) => void handleRoleChange(u, toRole(value))}
            data-testid={`role-select-${u.username}`}
          />
        ) : (
          t(`settings:users.roles.${u.role}`)
        ),
    },
    {
      key: 'authProvider',
      header: t('settings:users.columns.authProvider'),
      sortValue: (u) => u.authProvider,
      cell: (u) => <CategoryChip label={t(`settings:users.providers.${u.authProvider}`)} />,
    },
    {
      key: 'lastLogin',
      header: t('settings:users.columns.lastLogin'),
      sortValue: (u) => u.lastLogin ?? '',
      cell: (u) => formatLastLogin(u.lastLogin),
    },
    {
      key: 'status',
      header: t('settings:users.columns.status'),
      sortValue: (u) => statusOf(u, t),
      cell: (u) => <StatusPill tone={statusTone(u)} label={statusOf(u, t)} />,
    },
  ];
  const canCreate =
    isAdmin && licenseStatus !== null && Boolean(licenseStatus?.features?.includes?.('multi_user'));

  const refresh = async (): Promise<void> => {
    setError(null);
    try {
      await refreshLicense();
      const meRow = await api.get<UserRow>('/api/v1/users/me');
      setMe(meRow);
      if (meRow.role === 'admin') {
        setUsers(await listUsers());
      } else {
        setUsers([meRow]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors:users.loadFailed'));
    }
    setLoading(false);
  };

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleCreate = async (): Promise<void> => {
    if (!newUsername.trim() || !newPassword) return;
    setCreating(true);
    setError(null);
    try {
      await api.post('/api/v1/users', {
        username: newUsername.trim(),
        password: newPassword,
        role: newRole,
      });
      setNewUsername('');
      setNewPassword('');
      setNewRole('viewer');
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors:users.createFailed'));
    }
    setCreating(false);
  };

  const handleRoleChange = async (target: UserRow, role: Role): Promise<void> => {
    if (role === target.role) return;
    setError(null);
    try {
      await api.patch(`/api/v1/users/${target.username}`, { role });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors:users.updateRoleFailed'));
    }
  };

  const handleDelete = async (target: UserRow): Promise<void> => {
    setDeleteTarget(null);
    setError(null);
    try {
      await api.delete(`/api/v1/users/${target.username}`);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors:users.deleteFailed'));
    }
  };

  const tierLabel = licenseStatus?.tier ?? 'Free';

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-compact">
          <Users className="w-4 h-4" />
          <span>{t('settings:users.title')}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm" data-testid="users-settings-section">
        <p className="text-sm text-text-secondary">{t('settings:users.description')}</p>

        {isAdmin && !canCreate && (
          <div className="rounded-lg border border-status-warning/30 bg-status-warning/5 pad-sm text-sm text-status-warning-strong">
            {t('errors:users.featureRequired')} <strong>{tierLabel}</strong>
          </div>
        )}

        {error ? (
          <div
            className="rounded-lg border border-status-error/30 bg-status-error/5 pad-sm text-sm text-status-error-strong"
            data-testid="users-settings-error"
          >
            {error}
          </div>
        ) : null}

        {isAdmin && (
          <div className="flex flex-wrap items-end gap-compact">
            <FormField
              label={t('settings:users.columns.username')}
              id="new-username"
              className="flex-1 min-w-[12rem]"
            >
              {(control) => (
                <Input
                  {...control}
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  placeholder="alice"
                  maxLength={64}
                  disabled={!canCreate || creating}
                  data-testid="new-username-input"
                />
              )}
            </FormField>
            <FormField
              label={t('common:labels.password')}
              id="new-password"
              className="flex-1 min-w-[12rem]"
            >
              {(control) => (
                <Input
                  {...control}
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="≥ 12 chars, mixed case + digit + symbol"
                  disabled={!canCreate || creating}
                  data-testid="new-password-input"
                />
              )}
            </FormField>
            <FormField label={t('settings:users.columns.role')} className="min-w-[8rem]">
              {(control) => (
                <Select
                  {...control}
                  value={newRole}
                  options={roleOptions}
                  onValueChange={(value) => setNewRole(toRole(value))}
                  disabled={!canCreate || creating}
                  data-testid="new-role-select"
                />
              )}
            </FormField>
            <Tooltip text={canCreate ? undefined : t('errors:users.featureRequired')}>
              <Button
                variant="primary"
                icon={<Plus className="w-4 h-4" />}
                disabled={!canCreate || creating || !newUsername.trim() || !newPassword}
                loading={creating}
                onClick={() => void handleCreate()}
                data-testid="create-user-button"
              >
                {t('settings:users.addUser')}
              </Button>
            </Tooltip>
          </div>
        )}

        {loading ? (
          <div className="text-sm text-text-muted">{t('common:status.loading')}</div>
        ) : users.length === 0 ? (
          <div className="text-sm text-text-muted">{t('settings:users.none')}</div>
        ) : (
          <DataTable
            rows={users}
            columns={columns}
            getRowKey={(u) => String(u.id)}
            caption={t('settings:users.title')}
            empty={null}
            labels={labels}
            pageSize={10}
            defaultSort={{ key: 'username', direction: 'asc' }}
            data-testid="users-table"
            rowActions={(u) =>
              isAdmin && me?.username !== u.username ? (
                <Button
                  variant="destructive"
                  size="sm"
                  icon={<Trash2 className="w-3 h-3" />}
                  onClick={() => setDeleteTarget(u)}
                  data-testid={`delete-user-${u.username}`}
                >
                  {t('settings:users.deleteUser')}
                </Button>
              ) : null
            }
          />
        )}
      </div>
      <ConfirmDialog
        open={deleteTarget !== null}
        title={t('settings:users.confirmDelete', { username: deleteTarget?.username ?? '' })}
        message={t('settings:users.confirmDeletePrompt', {
          username: deleteTarget?.username ?? '',
        })}
        confirmLabel={t('settings:users.deleteUser')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={() => {
          if (deleteTarget) void handleDelete(deleteTarget);
        }}
        onCancel={() => setDeleteTarget(null)}
      />
    </CollapsibleSection>
  );
}
