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
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../../api/client';
import { useLicense } from '../../../contexts/LicenseContext';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Input } from '../../ui/Input';
import { Plus, Trash2, Users } from '../../ui/icons';

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

export function UsersSettings(): React.ReactElement {
  const { t } = useTranslation(['settings', 'errors']);
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

  const isAdmin = me?.role === 'admin';
  const canCreate =
    isAdmin && licenseStatus !== null && Boolean(licenseStatus?.features?.includes?.('multi_user'));

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      await refreshLicense();
      const meRow = await api.get<UserRow>('/api/v1/users/me');
      setMe(meRow);
      if (meRow.role === 'admin') {
        const list = await api.get<UserRow[]>('/api/v1/users');
        setUsers(list ?? []);
      } else {
        setUsers([meRow]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  }, [refreshLicense]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleCreate = useCallback(async (): Promise<void> => {
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
      setError(err instanceof Error ? err.message : 'Failed to create user');
    } finally {
      setCreating(false);
    }
  }, [newUsername, newPassword, newRole, refresh]);

  const handleRoleChange = useCallback(
    async (target: UserRow, role: Role): Promise<void> => {
      if (role === target.role) return;
      setError(null);
      try {
        await api.patch(`/api/v1/users/${target.username}`, { role });
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update role');
      }
    },
    [refresh],
  );

  const handleDelete = useCallback(
    async (target: UserRow): Promise<void> => {
      const ok = window.confirm(
        t('settings:users.confirmDelete', { username: target.username }) +
          '\n\n' +
          t('settings:users.confirmDeletePrompt', { username: target.username }),
      );
      if (!ok) return;
      setError(null);
      try {
        await api.delete(`/api/v1/users/${target.username}`);
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete user');
      }
    },
    [refresh, t],
  );

  const tierLabel = licenseStatus?.tier ?? 'Free';

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-2">
          <Users className="w-4 h-4" />
          <span>{t('settings:users.title')}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm" data-testid="users-settings-section">
        <p className="text-sm text-text-secondary">{t('settings:users.description')}</p>

        {isAdmin && !canCreate && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm text-amber-200">
            {t('errors:users.featureRequired')} <strong>{tierLabel}</strong>
          </div>
        )}

        {error && (
          <div
            className="rounded-lg border border-status-error/30 bg-status-error/5 p-3 text-sm text-status-error"
            data-testid="users-settings-error"
          >
            {error}
          </div>
        )}

        {isAdmin && (
          <div className="flex flex-wrap items-end gap-2">
            <div className="flex-1 min-w-[12rem]">
              <label className="block text-xs text-text-muted mb-1" htmlFor="new-username">
                {t('settings:users.columns.username')}
              </label>
              <Input
                id="new-username"
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                placeholder="alice"
                maxLength={64}
                disabled={!canCreate || creating}
                data-testid="new-username-input"
              />
            </div>
            <div className="flex-1 min-w-[12rem]">
              <label className="block text-xs text-text-muted mb-1" htmlFor="new-password">
                Password
              </label>
              <Input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="≥ 12 chars, mixed case + digit + symbol"
                disabled={!canCreate || creating}
                data-testid="new-password-input"
              />
            </div>
            <div className="min-w-[8rem]">
              <label className="block text-xs text-text-muted mb-1" htmlFor="new-role">
                {t('settings:users.columns.role')}
              </label>
              <select
                id="new-role"
                value={newRole}
                onChange={(e) => setNewRole(e.target.value as Role)}
                disabled={!canCreate || creating}
                className="w-full rounded-lg border border-surface-border bg-surface-raised px-2 py-1.5 text-sm"
                data-testid="new-role-select"
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {t(`settings:users.roles.${r}`)}
                  </option>
                ))}
              </select>
            </div>
            <Button
              variant="solid"
              tone="violet"
              size="md"
              leftIcon={<Plus className="w-4 h-4" />}
              disabled={!canCreate || creating || !newUsername.trim() || !newPassword}
              loading={creating}
              title={canCreate ? undefined : t('errors:users.featureRequired')}
              onClick={() => void handleCreate()}
              data-testid="create-user-button"
            >
              {t('settings:users.addUser')}
            </Button>
          </div>
        )}

        {loading ? (
          <div className="text-sm text-text-muted">{t('common:status.loading', 'Loading…')}</div>
        ) : users.length === 0 ? (
          <div className="text-sm text-text-muted">No users.</div>
        ) : (
          <table className="w-full text-sm" data-testid="users-table">
            <thead className="text-xs text-text-muted text-left">
              <tr>
                <th className="py-2 pr-2">{t('settings:users.columns.username')}</th>
                <th className="py-2 pr-2">{t('settings:users.columns.role')}</th>
                <th className="py-2 pr-2">{t('settings:users.columns.authProvider')}</th>
                <th className="py-2 pr-2">{t('settings:users.columns.lastLogin')}</th>
                <th className="py-2 pr-2">{t('settings:users.columns.status')}</th>
                <th className="py-2" />
              </tr>
            </thead>
            <tbody>
              {users.map((u) => {
                const isSelf = me?.username === u.username;
                return (
                  <tr
                    key={u.id}
                    className="border-t border-surface-border"
                    data-testid={`user-row-${u.username}`}
                  >
                    <td className="py-2 pr-2">
                      {u.username}
                      {isSelf && (
                        <span className="ml-1 text-xs text-text-muted">
                          {t('settings:users.currentUser')}
                        </span>
                      )}
                    </td>
                    <td className="py-2 pr-2">
                      {isAdmin && !isSelf ? (
                        <select
                          value={u.role}
                          onChange={(e) => void handleRoleChange(u, e.target.value as Role)}
                          className="rounded border border-surface-border bg-surface-raised px-1 py-0.5 text-xs"
                          data-testid={`role-select-${u.username}`}
                        >
                          {ROLES.map((r) => (
                            <option key={r} value={r}>
                              {t(`settings:users.roles.${r}`)}
                            </option>
                          ))}
                        </select>
                      ) : (
                        t(`settings:users.roles.${u.role}`)
                      )}
                    </td>
                    <td className="py-2 pr-2">
                      <span className="rounded bg-surface-raised px-1.5 py-0.5 text-xs">
                        {t(`settings:users.providers.${u.authProvider}`)}
                      </span>
                    </td>
                    <td className="py-2 pr-2">{formatLastLogin(u.lastLogin)}</td>
                    <td className="py-2 pr-2">{statusOf(u, t)}</td>
                    <td className="py-2 text-right">
                      {isAdmin && !isSelf && (
                        <Button
                          variant="ghost"
                          tone="red"
                          size="xs"
                          leftIcon={<Trash2 className="w-3 h-3" />}
                          onClick={() => void handleDelete(u)}
                          data-testid={`delete-user-${u.username}`}
                        >
                          {t('settings:users.deleteUser')}
                        </Button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </CollapsibleSection>
  );
}
