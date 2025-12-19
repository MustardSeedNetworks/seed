/**
 * ProfileManagement Component
 *
 * Full-page profile management interface for MSP profiles (#754).
 * Allows creating, editing, duplicating, and deleting profiles.
 *
 * Features:
 * - Profile list with search/filter
 * - Create new profile
 * - Edit/Delete/Duplicate actions
 * - Import/Export functionality
 * - Set default profile
 */

import { useState, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useProfileContext } from "../../contexts/ProfileContext";
import { radius, spacing } from "../../styles/theme";
import type { Profile, ProfileRequest } from "../../types/profile";
import { ProfileEditor } from "./ProfileEditor";

interface ProfileManagementProps {
  onClose: () => void;
}

/**
 * Full-screen management UI for MSP profiles (create, edit, delete, duplicate, import/export).
 */
export function ProfileManagement({ onClose }: ProfileManagementProps) {
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
  const [searchQuery, setSearchQuery] = useState("");
  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  // Filter profiles by search
  const filteredProfiles = useMemo(() => {
    if (!searchQuery.trim()) return profiles;
    const query = searchQuery.toLowerCase();
    return profiles.filter(
      (p) => p.name.toLowerCase().includes(query) || p.description?.toLowerCase().includes(query)
    );
  }, [profiles, searchQuery]);

  // Handlers
  const handleCreate = useCallback(() => {
    setEditingProfile(null);
    setIsCreating(true);
    setIsEditorOpen(true);
  }, []);

  const handleEdit = useCallback((profile: Profile) => {
    setEditingProfile(profile);
    setIsCreating(false);
    setIsEditorOpen(true);
  }, []);

  const handleSave = useCallback(
    async (data: ProfileRequest) => {
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
    },
    [isCreating, editingProfile, createProfile, updateProfile]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      const success = await deleteProfile(id);
      if (success) {
        setDeleteConfirm(null);
      }
    },
    [deleteProfile]
  );

  const handleDuplicate = useCallback(
    async (profile: Profile) => {
      await duplicateProfile(profile.id);
    },
    [duplicateProfile]
  );

  const handleSetActive = useCallback(
    async (id: string) => {
      await switchProfile(id);
    },
    [switchProfile]
  );

  const handleExport = useCallback(async () => {
    await downloadProfiles();
  }, [downloadProfiles]);

  const handleBack = useCallback(() => {
    onClose();
  }, [onClose]);

  return (
    <div className="min-h-screen bg-surface-base">
      {/* Header */}
      <header className={`bg-surface-raised border-b border-surface-border ${spacing.pad.md}`}>
        <div className="max-w-6xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={handleBack}
              className="p-2 hover:bg-surface-hover rounded-lg transition-colors"
              aria-label={t("common.back", "Back")}
            >
              <svg
                className="w-5 h-5 text-text-secondary"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            </button>
            <div>
              <h1 className="heading-lg text-text-primary">
                {t("profile.management", "Profile Management")}
              </h1>
              <p className="body-small text-text-muted">
                {t("profile.managementDesc", "Create and manage client-specific configurations")}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleExport}
              className={`${spacing.pad.sm} ${radius.md} border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary body-small font-medium flex items-center gap-2`}
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              {t("profile.export", "Export")}
            </button>
            <button
              type="button"
              onClick={handleCreate}
              className={`${spacing.pad.sm} ${radius.md} bg-brand-primary hover:bg-brand-primary-hover text-white body-small font-medium flex items-center gap-2`}
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              {t("profile.create", "Create Profile")}
            </button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className={`max-w-6xl mx-auto ${spacing.pad.lg}`}>
        {/* Search bar */}
        <div className="mb-6">
          <div className="relative">
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-text-muted"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              type="text"
              placeholder={t("profile.searchPlaceholder", "Search profiles...")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className={`w-full pl-10 pr-4 py-2.5 ${radius.md} border border-surface-border bg-surface-base text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary`}
            />
          </div>
        </div>

        {/* Error message */}
        {error && (
          <div
            className={`mb-6 ${spacing.pad.md} ${radius.md} bg-status-error/10 border border-status-error/20 text-status-error`}
          >
            {error}
          </div>
        )}

        {/* Loading state */}
        {isLoading && profiles.length === 0 && (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-brand-primary" />
            <p className="mt-4 text-text-muted">{t("common.loading", "Loading...")}</p>
          </div>
        )}

        {/* Profile list */}
        {!isLoading && filteredProfiles.length === 0 ? (
          <div className="text-center py-12">
            <svg
              className="mx-auto w-16 h-16 text-text-muted mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
              />
            </svg>
            <h3 className="heading-md text-text-primary mb-2">
              {searchQuery
                ? t("profile.noResults", "No profiles found")
                : t("profile.noProfiles", "No profiles yet")}
            </h3>
            <p className="text-text-muted mb-6">
              {searchQuery
                ? t("profile.noResultsDesc", "Try adjusting your search criteria")
                : t("profile.noProfilesDesc", "Create your first profile to get started")}
            </p>
            {!searchQuery && (
              <button
                type="button"
                onClick={handleCreate}
                className={`${spacing.pad.sm} px-4 ${radius.md} bg-brand-primary hover:bg-brand-primary-hover text-white body-small font-medium`}
              >
                {t("profile.createFirst", "Create Your First Profile")}
              </button>
            )}
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredProfiles.map((profile) => (
              <ProfileCard
                key={profile.id}
                profile={profile}
                isActive={profile.id === activeProfile?.id}
                onEdit={() => handleEdit(profile)}
                onDelete={() => setDeleteConfirm(profile.id)}
                onDuplicate={() => handleDuplicate(profile)}
                onSetActive={() => handleSetActive(profile.id)}
              />
            ))}
          </div>
        )}
      </main>

      {/* Profile editor modal */}
      {isEditorOpen && (
        <ProfileEditor
          profile={editingProfile}
          onSave={handleSave}
          onCancel={() => setIsEditorOpen(false)}
          isLoading={isLoading}
        />
      )}

      {/* Delete confirmation modal */}
      {deleteConfirm && (
        <DeleteConfirmModal
          profileName={profiles.find((p) => p.id === deleteConfirm)?.name || ""}
          onConfirm={() => handleDelete(deleteConfirm)}
          onCancel={() => setDeleteConfirm(null)}
          isLoading={isLoading}
        />
      )}
    </div>
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
}: ProfileCardProps) {
  const { t } = useTranslation();
  const [showMenu, setShowMenu] = useState(false);

  return (
    <div
      className={`${radius.lg} border ${isActive ? "border-brand-primary ring-2 ring-brand-primary/20" : "border-surface-border"} bg-surface-raised overflow-hidden`}
    >
      {/* Card header */}
      <div className={`${spacing.pad.md} border-b border-surface-border`}>
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="heading-sm text-text-primary truncate">{profile.name}</h3>
              {profile.is_default && (
                <span className="caption px-1.5 py-0.5 rounded bg-brand-primary/10 text-brand-primary font-medium">
                  {t("profile.default", "Default")}
                </span>
              )}
              {isActive && (
                <span className="caption px-1.5 py-0.5 rounded bg-status-success/10 text-status-success font-medium">
                  {t("profile.active", "Active")}
                </span>
              )}
            </div>
            {profile.description && (
              <p className="body-small text-text-muted line-clamp-2">{profile.description}</p>
            )}
          </div>

          {/* Actions menu */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setShowMenu(!showMenu)}
              className="p-1.5 hover:bg-surface-hover rounded transition-colors"
              aria-label={t("common.actions", "Actions")}
            >
              <svg className="w-5 h-5 text-text-muted" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z" />
              </svg>
            </button>

            {showMenu && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setShowMenu(false)} />
                <div
                  className={`absolute right-0 top-full mt-1 w-40 ${radius.md} border border-surface-border bg-surface-raised shadow-lg z-20 overflow-hidden`}
                >
                  <button
                    type="button"
                    onClick={() => {
                      setShowMenu(false);
                      onEdit();
                    }}
                    className={`w-full text-left ${spacing.pad.sm} hover:bg-surface-hover body-small text-text-primary`}
                  >
                    {t("common.edit", "Edit")}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowMenu(false);
                      onDuplicate();
                    }}
                    className={`w-full text-left ${spacing.pad.sm} hover:bg-surface-hover body-small text-text-primary`}
                  >
                    {t("common.duplicate", "Duplicate")}
                  </button>
                  {!isActive && (
                    <button
                      type="button"
                      onClick={() => {
                        setShowMenu(false);
                        onSetActive();
                      }}
                      className={`w-full text-left ${spacing.pad.sm} hover:bg-surface-hover body-small text-brand-primary`}
                    >
                      {t("profile.setActive", "Set as Active")}
                    </button>
                  )}
                  {!profile.is_default && !isActive && (
                    <button
                      type="button"
                      onClick={() => {
                        setShowMenu(false);
                        onDelete();
                      }}
                      className={`w-full text-left ${spacing.pad.sm} hover:bg-surface-hover body-small text-status-error`}
                    >
                      {t("common.delete", "Delete")}
                    </button>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Card footer */}
      <div className={`${spacing.pad.sm} bg-surface-base flex items-center justify-between`}>
        <span className="caption text-text-muted">
          {t("profile.updated", "Updated")} {new Date(profile.updated_at).toLocaleDateString()}
        </span>
        {!isActive && (
          <button
            type="button"
            onClick={onSetActive}
            className="caption font-medium text-brand-primary hover:text-brand-primary-hover"
          >
            {t("profile.activate", "Activate")}
          </button>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Delete Confirmation Modal
// ============================================================================

interface DeleteConfirmModalProps {
  profileName: string;
  onConfirm: () => void;
  onCancel: () => void;
  isLoading: boolean;
}

function DeleteConfirmModal({
  profileName,
  onConfirm,
  onCancel,
  isLoading,
}: DeleteConfirmModalProps) {
  const { t } = useTranslation();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-black/50" onClick={onCancel} />
      <div
        className={`relative w-full max-w-md ${radius.lg} bg-surface-raised shadow-xl ${spacing.pad.lg}`}
      >
        <h3 className="heading-md text-text-primary mb-2">
          {t("profile.deleteConfirm", "Delete Profile?")}
        </h3>
        <p className="body-small text-text-secondary mb-6">
          {t(
            "profile.deleteConfirmDesc",
            'Are you sure you want to delete "{{name}}"? This action cannot be undone.',
            { name: profileName }
          )}
        </p>
        <div className="flex justify-end gap-3">
          <button
            type="button"
            onClick={onCancel}
            disabled={isLoading}
            className={`${spacing.pad.sm} px-4 ${radius.md} border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary body-small font-medium disabled:opacity-50`}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isLoading}
            className={`${spacing.pad.sm} px-4 ${radius.md} bg-status-error hover:bg-status-error/90 text-white body-small font-medium disabled:opacity-50`}
          >
            {isLoading ? t("common.deleting", "Deleting...") : t("common.delete", "Delete")}
          </button>
        </div>
      </div>
    </div>
  );
}

export default ProfileManagement;
