/**
 * The Profiles tab: which profile is active and the door to managing them. It
 * used to be the rail footer's account menu; the rail now carries one product
 * control (the interface) and everything about profiles lives here.
 */
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useProfileContext } from '../../../contexts/profileContext';
import { Button } from '../../../shared/Button';
import { RadioGroup } from '../../../shared/RadioGroup';

interface ProfilesSettingsProps {
  /** Opens the profile manager (create, edit, delete). */
  onManage: () => void;
}

export function ProfilesSettings({ onManage }: ProfilesSettingsProps): JSX.Element {
  const { t } = useTranslation();
  const { profiles, activeProfile, switchProfile, isLoading } = useProfileContext();

  return (
    <div className="space-y-4" data-testid="profiles-settings">
      {profiles.length === 0 ? (
        <p className="text-sm text-text-muted">{t('profile.noProfiles')}</p>
      ) : (
        <RadioGroup
          name="active-profile"
          label={t('profile.switch')}
          value={activeProfile?.id ?? ''}
          onValueChange={(id): void => {
            void switchProfile(id);
          }}
          options={profiles.map((profile) => ({
            value: profile.id,
            label: profile.name,
            disabled: isLoading,
          }))}
        />
      )}
      <Button variant="secondary" onClick={onManage} data-testid="settings-profile-manage">
        {t('labels.manageProfiles')}
      </Button>
    </div>
  );
}
