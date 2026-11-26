import { ConfirmDialog } from '../../shared/ConfirmDialog';
import { Modal } from '../../shared/Modal';
import { SearchInput } from '../../shared/SearchInput';
import { Tooltip } from '../../shared/Tooltip';
/** Create, edit, switch and transfer saved profiles. */

import type React from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useProfileContext } from '../../contexts/profileContext';
import { cn, radius, spacing } from '../../styles/theme';
import type { Profile, ProfileRequest } from '../../types/profile';
import { WriteGate } from '../ui/WriteGate';
import { ProfileEditor } from './ProfileEditor';

interface ProfileManagementProps {
  onClose: () => void;
}

/**
 * Modal-style management UI for MSP profiles (create, edit, delete, duplicate, import/export).
 */
export function ProfileManagement({ onClose }: ProfileManagementProps): React.ReactElement {
  const { t } = useTranslation();
  const {
    profiles,
    activeProfile,
    isLoading,
    error,
    createProfile,
    updateProfile,
    deleteProfile,
    switchProfile,
    duplicateProfile,
    downloadProfiles,
  } = useProfileContext();

  // Local state
  const [searchQuery, setSearchQuery] = useState('');
  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  // Filter profiles by search
  const query = searchQuery.toLowerCase();
  const filteredProfiles = searchQuery.trim()
    ? profiles.filter(
        (p) => p.name.toLowerCase().includes(query) || p.description?.toLowerCase().includes(query),
      )
    : profiles;

  // Handlers
  const handleCreate = (): void => {
    setEditingProfile(null);
    setIsCreating(true);
    setIsEditorOpen(true);
  };

  const handleEdit = (profile: Profile): void => {
    setEditingProfile(profile);
    setIsCreating(false);
    setIsEditorOpen(true);
  };

  const handleSave = async (data: ProfileRequest): Promise<void> => {
    if (isCreating) {
      const created = await createProfile(data);
      if (created) {
        setIsEditorOpen(false);
      }
    } else if (editingProfile) {
      const updated = await updateProfile(editingProfile.id, data);
      if (updated) {
        setIsEditorOpen(false);
      }
    }
  };

  const handleDelete = async (id: string): Promise<void> => {
    const success = await deleteProfile(id);
    if (success) {
      setDeleteConfirm(null);
    }
  };

  const handleDuplicate = async (profile: Profile): Promise<void> => {
    await duplicateProfile(profile.id);
  };

  const handleSetActive = async (id: string): Promise<void> => {
    await switchProfile(id);
  };

  const handleExport = async (): Promise<void> => {
    await downloadProfiles();
  };

  return (
    <>
      <Modal
        open
        onClose={onClose}
        title={t('profile.management')}
        closeLabel={t('common.close')}
        data-testid="profile-modal"
      >
        <p className="body-small text-text-muted mb-content">{t('profile.managementDesc')}</p>

        {/* Actions bar */}
        <div
          className={cn(
            spacing.pad.default,
            'border-b border-surface-border bg-surface-base flex items-center gap-compact shrink-0',
          )}
        >
          <WriteGate>
            <button
              type="button"
              onClick={handleCreate}
              data-testid="profile-create"
              className={cn(
                spacing.pad.sm,
                'px-4',
                radius.md,
                'bg-brand-primary hover:bg-brand-accent text-on-brand body-small font-medium flex items-center gap-compact',
              )}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              {t('profile.create')}
            </button>
          </WriteGate>
          <Tooltip text={t('profile.export')}>
            <button
              type="button"
              onClick={handleExport}
              className={cn(
                spacing.pad.sm,
                'px-4',
                radius.md,
                'border border-surface-border bg-surface-raised hover:bg-surface-hover text-text-primary body-small font-medium flex items-center gap-compact',
              )}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              {t('profile.export')}
            </button>
          </Tooltip>

          {/* Search bar */}
          <div className="flex-1 ml-content">
            <SearchInput
              value={searchQuery}
              onValueChange={setSearchQuery}
              label={t('profile.searchPlaceholder')}
              clearLabel={t('profile.clearSearch')}
              placeholder={t('profile.searchPlaceholder')}
            />
          </div>
        </div>

        {/* Error message */}
        {error ? (
          <div
            className={cn(
              'mt-content',
              spacing.pad.sm,
              radius.md,
              'bg-status-error/10 border border-status-error/20 text-status-error-strong body-small shrink-0',
            )}
          >
            {error}
          </div>
        ) : null}

        {/* Scrollable content */}
        <div className="mt-content">
          {/* Loading state */}
          {isLoading && profiles.length === 0 ? (
            <div className="text-center py-centered">
              <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-brand-primary" />
              <p className="mt-heading body-small text-text-muted">{t('common.loading')}</p>
            </div>
          ) : null}

          {/* Profile grid */}
          {!isLoading && filteredProfiles.length === 0 ? (
            <div className="text-center py-centered">
              <svg
                className="mx-auto w-16 h-16 text-text-muted mb-content"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                />
              </svg>
              <h3 className="body font-medium text-text-primary mb-2">
                {searchQuery ? t('profile.noResults') : t('profile.noProfiles')}
              </h3>
              <p className="body-small text-text-muted mb-section">
                {searchQuery ? t('profile.noResultsDesc') : t('profile.noProfilesDesc')}
              </p>
              {searchQuery ? null : (
                <WriteGate>
                  <button
                    type="button"
                    onClick={handleCreate}
                    className={cn(
                      spacing.pad.sm,
                      'px-4',
                      radius.md,
                      'bg-brand-primary hover:bg-brand-accent text-on-brand body-small font-medium',
                    )}
                  >
                    {t('profile.createFirst')}
                  </button>
                </WriteGate>
              )}
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-comfortable">
              {filteredProfiles.map((profile) => (
                <ProfileCard
                  key={profile.id}
                  profile={profile}
                  isActive={profile.id === activeProfile?.id}
                  onEdit={(): void => handleEdit(profile)}
                  onDelete={(): void => setDeleteConfirm(profile.id)}
                  onDuplicate={(): void => {
                    handleDuplicate(profile).catch(() => undefined);
                  }}
                  onSetActive={(): void => {
                    handleSetActive(profile.id).catch(() => undefined);
                  }}
                />
              ))}
            </div>
          )}
        </div>
      </Modal>
      {/* Profile editor modal */}
      {isEditorOpen ? (
        <ProfileEditor
          profile={editingProfile}
          onSave={handleSave}
          onCancel={(): void => setIsEditorOpen(false)}
          isLoading={isLoading}
        />
      ) : null}
      {/* Delete confirmation modal */}
      {deleteConfirm ? (
        <ConfirmDialog
          open
          title={t('profile.deleteConfirm')}
          message={t('profile.deleteConfirmDesc', {
            name: profiles.find((p) => p.id === deleteConfirm)?.name || '',
          })}
          confirmLabel={t('common.delete')}
          cancelLabel={t('common.cancel')}
          closeLabel={t('common.close')}
          onConfirm={(): void => {
            handleDelete(deleteConfirm).catch(() => undefined);
          }}
          onCancel={(): void => setDeleteConfirm(null)}
          confirming={isLoading}
        />
      ) : null}
    </>
  );
}

// ============================================================================
// Profile Card Component
// ============================================================================

interface ProfileCardProps {
  profile: Profile;
  isActive: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onDuplicate: () => void;
  onSetActive: () => void;
}

function ProfileCard({
  profile,
  isActive,
  onEdit,
  onDelete,
  onDuplicate,
  onSetActive,
}: ProfileCardProps): React.ReactElement {
  const { t } = useTranslation();

  return (
    <div
      className={cn(
        radius.lg,
        'border',
        isActive ? 'border-brand-primary ring-2 ring-brand-primary/20' : 'border-surface-border',
        'bg-surface-raised overflow-hidden',
      )}
    >
      {/* Card content */}
      <div className={cn(spacing.pad.default)}>
        <div className="flex items-start justify-between mb-2">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-compact flex-wrap">
              <h3 className="body-small font-medium text-text-primary truncate">{profile.name}</h3>
              {profile.isDefault ? (
                <span className="caption px-1.5 py-0.5 rounded bg-brand-primary/10 text-brand-primary-strong font-medium">
                  {t('profile.default')}
                </span>
              ) : null}
              {isActive ? (
                <span className="caption px-1.5 py-0.5 rounded bg-status-success/10 text-status-success-strong font-medium">
                  {t('profile.active')}
                </span>
              ) : null}
            </div>
            {profile.description ? (
              <p className="caption text-text-muted mt-tight line-clamp-2">{profile.description}</p>
            ) : null}
          </div>
        </div>

        {/* Updated date */}
        <p className="caption text-text-muted mb-heading">
          {t('profile.updated')} {new Date(profile.updatedAt).toLocaleDateString()}
        </p>

        {/* Action buttons - hidden for viewers (#1226). */}
        <WriteGate>
          <div className="flex items-center gap-compact flex-wrap">
            {/* Edit button */}
            <Tooltip text={t('common.edit')}>
              <button
                type="button"
                onClick={onEdit}
                className={cn(
                  spacing.chip.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary caption font-medium flex items-center gap-1.5',
                )}
              >
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
                {t('common.edit')}
              </button>
            </Tooltip>

            {/* Clone/Duplicate button */}
            <Tooltip text={t('common.clone')}>
              <button
                type="button"
                onClick={onDuplicate}
                className={cn(
                  spacing.chip.sm,
                  radius.md,
                  'border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary caption font-medium flex items-center gap-1.5',
                )}
              >
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                  />
                </svg>
                {t('common.clone')}
              </button>
            </Tooltip>

            {/* Delete button - only if not default and not active */}
            {profile.isDefault || isActive ? null : (
              <Tooltip text={t('common.delete')}>
                <button
                  type="button"
                  onClick={onDelete}
                  data-testid={`profile-delete-${profile.id}`}
                  className={cn(
                    spacing.chip.sm,
                    radius.md,
                    'border border-status-error/30 bg-status-error/5 hover:bg-status-error/10 text-status-error-strong caption font-medium flex items-center gap-1.5',
                  )}
                >
                  <svg
                    className="w-3.5 h-3.5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                  {t('common.delete')}
                </button>
              </Tooltip>
            )}

            {/* Activate button - only if not active */}
            {isActive ? null : (
              <Tooltip text={t('profile.activate')}>
                <button
                  type="button"
                  onClick={onSetActive}
                  className={cn(
                    spacing.chip.sm,
                    radius.md,
                    'bg-brand-primary hover:bg-brand-accent text-on-brand caption font-medium flex items-center gap-1.5 ml-auto',
                  )}
                >
                  <svg
                    className="w-3.5 h-3.5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                  {t('profile.activate')}
                </button>
              </Tooltip>
            )}
          </div>
        </WriteGate>
      </div>
    </div>
  );
}

export default ProfileManagement;
