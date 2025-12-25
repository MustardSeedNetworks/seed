/**
 * ProfileManagement Component
 *
 * Modal-style profile management interface for MSP profiles (#754).
 * Centered modal similar to HelpModal.
 *
 * Features:
 * - Profile list with search/filter
 * - Create new profile
 * - Edit/Delete/Duplicate actions with visible buttons
 * - Import/Export functionality
 * - Set default profile
 */

import { useState, useCallback, useMemo, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useProfileContext } from "../../contexts/ProfileContext";
import {
  cn,
  icon as iconTokens,
  layout,
  modal,
  radius,
  spacing,
} from "../../styles/theme";
import type { Profile, ProfileRequest } from "../../types/profile";
import { ProfileEditor } from "./ProfileEditor";

interface ProfileManagementProps {
  onClose: () => void;
}

/**
 * Modal-style management UI for MSP profiles (create, edit, delete, duplicate, import/export).
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

  const modalRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Filter profiles by search
  const filteredProfiles = useMemo(() => {
    if (!searchQuery.trim()) return profiles;
    const query = searchQuery.toLowerCase();
    return profiles.filter(
      (p) =>
        p.name.toLowerCase().includes(query) ||
        p.description?.toLowerCase().includes(query)
    );
  }, [profiles, searchQuery]);

  // Handle ESC key to close modal
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    // Focus the close button when modal opens
    setTimeout(() => closeButtonRef.current?.focus(), 100);

    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

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

  return (
    <>
      {/* Modal overlay */}
      <div className={modal.overlay}>
        {/* Backdrop */}
        <div className={modal.backdrop} onClick={onClose} aria-hidden="true" />

        {/* Modal content */}
        <div
          ref={modalRef}
          role="dialog"
          aria-modal="true"
          aria-labelledby="profile-modal-title"
          className={cn(
            "relative",
            modal.content,
            modal.size.lg,
            "flex flex-col"
          )}
          style={{ maxHeight: "85vh" }}
        >
          {/* Header */}
          <div
            className={cn(
              layout.flex.between,
              spacing.pad.lg,
              "border-b border-surface-border shrink-0"
            )}
          >
            <div>
              <h2 id="profile-modal-title" className="heading-2">
                {t("profile.management", "Profile Management")}
              </h2>
              <p className="body-small text-text-muted mt-1">
                {t(
                  "profile.managementDesc",
                  "Create and manage client-specific configurations"
                )}
              </p>
            </div>
            <button
              ref={closeButtonRef}
              onClick={onClose}
              className={cn(
                "p-2",
                radius.md,
                "hover:bg-surface-hover active:bg-surface-hover text-text-muted touch-manipulation focus:outline-none focus:ring-2 focus:ring-brand-primary"
              )}
              aria-label={t("common.close", "Close")}
            >
              <svg
                className={iconTokens.size.lg}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>

          {/* Actions bar */}
          <div
            className={cn(
              spacing.pad.default,
              "border-b border-surface-border bg-surface-base flex items-center gap-2 shrink-0"
            )}
          >
            <button
              type="button"
              onClick={handleCreate}
              className={cn(
                spacing.pad.sm,
                "px-4",
                radius.md,
                "bg-brand-primary hover:bg-brand-primary-hover text-text-inverse body-small font-medium flex items-center gap-2"
              )}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              {t("profile.create", "Create Profile")}
            </button>
            <button
              type="button"
              onClick={handleExport}
              className={cn(
                spacing.pad.sm,
                "px-4",
                radius.md,
                "border border-surface-border bg-surface-raised hover:bg-surface-hover text-text-primary body-small font-medium flex items-center gap-2"
              )}
              title={t("profile.export", "Export All")}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              {t("profile.export", "Export")}
            </button>

            {/* Search bar */}
            <div className="relative flex-1 ml-4">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted"
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
                placeholder={t(
                  "profile.searchPlaceholder",
                  "Search profiles..."
                )}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className={cn(
                  "w-full pl-9 pr-4 py-2",
                  radius.md,
                  "border border-surface-border bg-surface-base text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary body-small"
                )}
              />
            </div>
          </div>

          {/* Error message */}
          {error && (
            <div
              className={cn(
                "mx-4 mt-4",
                spacing.pad.sm,
                radius.md,
                "bg-status-error/10 border border-status-error/20 text-status-error body-small shrink-0"
              )}
            >
              {error}
            </div>
          )}

          {/* Scrollable content */}
          <div className={cn(spacing.pad.lg, "overflow-y-auto flex-1")}>
            {/* Loading state */}
            {isLoading && profiles.length === 0 && (
              <div className="text-center py-12">
                <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-brand-primary" />
                <p className="mt-3 body-small text-text-muted">
                  {t("common.loading", "Loading...")}
                </p>
              </div>
            )}

            {/* Profile grid */}
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
                <h3 className="body font-medium text-text-primary mb-2">
                  {searchQuery
                    ? t("profile.noResults", "No profiles found")
                    : t("profile.noProfiles", "No profiles yet")}
                </h3>
                <p className="body-small text-text-muted mb-6">
                  {searchQuery
                    ? t(
                        "profile.noResultsDesc",
                        "Try adjusting your search criteria"
                      )
                    : t(
                        "profile.noProfilesDesc",
                        "Create your first profile to get started"
                      )}
                </p>
                {!searchQuery && (
                  <button
                    type="button"
                    onClick={handleCreate}
                    className={cn(
                      spacing.pad.sm,
                      "px-4",
                      radius.md,
                      "bg-brand-primary hover:bg-brand-primary-hover text-text-inverse body-small font-medium"
                    )}
                  >
                    {t("profile.createFirst", "Create Your First Profile")}
                  </button>
                )}
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
          </div>
        </div>
      </div>

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
}: ProfileCardProps) {
  const { t } = useTranslation();

  return (
    <div
      className={cn(
        radius.lg,
        "border",
        isActive
          ? "border-brand-primary ring-2 ring-brand-primary/20"
          : "border-surface-border",
        "bg-surface-raised overflow-hidden"
      )}
    >
      {/* Card content */}
      <div className={cn(spacing.pad.default)}>
        <div className="flex items-start justify-between mb-2">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="body-small font-medium text-text-primary truncate">
                {profile.name}
              </h3>
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
              <p className="caption text-text-muted mt-1 line-clamp-2">
                {profile.description}
              </p>
            )}
          </div>
        </div>

        {/* Updated date */}
        <p className="caption text-text-muted mb-3">
          {t("profile.updated", "Updated")}{" "}
          {new Date(profile.updated_at).toLocaleDateString()}
        </p>

        {/* Action buttons - always visible */}
        <div className="flex items-center gap-2 flex-wrap">
          {/* Edit button */}
          <button
            type="button"
            onClick={onEdit}
            className={cn(
              spacing.chip.sm,
              radius.md,
              "border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary caption font-medium flex items-center gap-1.5"
            )}
            title={t("common.edit", "Edit")}
          >
            <svg
              className="w-3.5 h-3.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
              />
            </svg>
            {t("common.edit", "Edit")}
          </button>

          {/* Clone/Duplicate button */}
          <button
            type="button"
            onClick={onDuplicate}
            className={cn(
              spacing.chip.sm,
              radius.md,
              "border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary caption font-medium flex items-center gap-1.5"
            )}
            title={t("common.clone", "Clone")}
          >
            <svg
              className="w-3.5 h-3.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
              />
            </svg>
            {t("common.clone", "Clone")}
          </button>

          {/* Delete button - only if not default and not active */}
          {!profile.is_default && !isActive && (
            <button
              type="button"
              onClick={onDelete}
              className={cn(
                spacing.chip.sm,
                radius.md,
                "border border-status-error/30 bg-status-error/5 hover:bg-status-error/10 text-status-error caption font-medium flex items-center gap-1.5"
              )}
              title={t("common.delete", "Delete")}
            >
              <svg
                className="w-3.5 h-3.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
              {t("common.delete", "Delete")}
            </button>
          )}

          {/* Activate button - only if not active */}
          {!isActive && (
            <button
              type="button"
              onClick={onSetActive}
              className={cn(
                spacing.chip.sm,
                radius.md,
                "bg-brand-primary hover:bg-brand-primary-hover text-text-inverse caption font-medium flex items-center gap-1.5 ml-auto"
              )}
              title={t("profile.activate", "Activate")}
            >
              <svg
                className="w-3.5 h-3.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
              {t("profile.activate", "Activate")}
            </button>
          )}
        </div>
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
    // z-60 required: nested modal must appear above parent modal (z-50)
    <div className={cn(modal.overlay, "z-60")}>
      <div className={modal.backdrop} onClick={onCancel} />
      <div
        className={cn(
          "relative",
          modal.content,
          modal.size.sm,
          modal.padding.md
        )}
      >
        <h3 className="heading-2 text-text-primary mb-2">
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
            className={cn(
              spacing.pad.sm,
              "px-4",
              radius.md,
              "border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary body-small font-medium disabled:opacity-50"
            )}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isLoading}
            className={cn(
              spacing.pad.sm,
              "px-4",
              radius.md,
              "bg-status-error hover:bg-status-error/90 text-text-inverse body-small font-medium disabled:opacity-50"
            )}
          >
            {isLoading
              ? t("common.deleting", "Deleting...")
              : t("common.delete", "Delete")}
          </button>
        </div>
      </div>
    </div>
  );
}

export default ProfileManagement;
