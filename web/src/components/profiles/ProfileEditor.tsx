/**
 * ProfileEditor Component - Modal for creating/editing profiles
 */

import { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { radius, spacing } from "../../styles/theme";
import type { Profile, ProfileRequest } from "../../types/profile";

interface ProfileEditorProps {
  profile: Profile | null;
  onSave: (data: ProfileRequest) => Promise<void>;
  onCancel: () => void;
  isLoading: boolean;
}

/**
 *
 */
export function ProfileEditor({ profile, onSave, onCancel, isLoading }: ProfileEditorProps) {
  const { t } = useTranslation();
  const isEditing = profile !== null;

  const [name, setName] = useState(profile?.name || "");
  const [description, setDescription] = useState(profile?.description || "");
  const [isDefault, setIsDefault] = useState(profile?.is_default || false);
  const [notes, setNotes] = useState((profile?.config as { notes?: string })?.notes || "");

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      await onSave({
        name,
        description,
        is_default: isDefault,
        config: { notes },
      });
    },
    [name, description, isDefault, notes, onSave]
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-black/50" onClick={onCancel} />
      <div
        className={`relative w-full max-w-lg ${radius.lg} bg-surface-raised shadow-xl overflow-hidden`}
      >
        {/* Header */}
        <div className={`${spacing.pad.md} border-b border-surface-border`}>
          <h2 className="heading-md text-text-primary">
            {isEditing ? t("profile.edit", "Edit Profile") : t("profile.create", "Create Profile")}
          </h2>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit}>
          <div className={`${spacing.pad.md} space-y-4 max-h-96 overflow-y-auto`}>
            {/* Name */}
            <div>
              <label className="block body-small font-medium text-text-primary mb-1">
                {t("profile.name", "Name")} *
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className={`w-full ${spacing.pad.sm} ${radius.md} border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary`}
                placeholder={t("profile.namePlaceholder", "e.g., Client A")}
              />
            </div>

            {/* Description */}
            <div>
              <label className="block body-small font-medium text-text-primary mb-1">
                {t("profile.description", "Description")}
              </label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className={`w-full ${spacing.pad.sm} ${radius.md} border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary`}
                placeholder={t("profile.descriptionPlaceholder", "Brief description")}
              />
            </div>

            {/* Notes */}
            <div>
              <label className="block body-small font-medium text-text-primary mb-1">
                {t("profile.notes", "Notes")}
              </label>
              <textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                rows={3}
                className={`w-full ${spacing.pad.sm} ${radius.md} border border-surface-border bg-surface-base text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary resize-none`}
                placeholder={t("profile.notesPlaceholder", "Contact info, VPN requirements, etc.")}
              />
            </div>

            {/* Default checkbox */}
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={isDefault}
                onChange={(e) => setIsDefault(e.target.checked)}
                className="w-4 h-4 rounded border-surface-border text-brand-primary focus:ring-brand-primary"
              />
              <span className="body-small text-text-primary">
                {t("profile.setAsDefault", "Set as default profile")}
              </span>
            </label>
          </div>

          {/* Footer */}
          <div
            className={`${spacing.pad.md} border-t border-surface-border flex justify-end gap-3`}
          >
            <button
              type="button"
              onClick={onCancel}
              disabled={isLoading}
              className={`${spacing.pad.sm} px-4 ${radius.md} border border-surface-border bg-surface-base hover:bg-surface-hover text-text-primary body-small font-medium disabled:opacity-50`}
            >
              {t("common.cancel", "Cancel")}
            </button>
            <button
              type="submit"
              disabled={isLoading || !name.trim()}
              className={`${spacing.pad.sm} px-4 ${radius.md} bg-brand-primary hover:bg-brand-primary-hover text-white body-small font-medium disabled:opacity-50`}
            >
              {isLoading
                ? t("common.saving", "Saving...")
                : isEditing
                  ? t("common.save", "Save")
                  : t("common.create", "Create")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default ProfileEditor;
