import type React from 'react';
import type { UseFormRegisterReturn } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { spacing } from '../../styles/theme';

interface SetupTokenFieldProps {
  registration: UseFormRegisterReturn<'setupToken'>;
  error?: string;
  /** Host file the daemon wrote the token to; reading it proves host access. */
  tokenFile?: string;
}

/**
 * The setup token the operator copies from a file on the seed host. The
 * daemon never sends it over HTTP, so a network peer cannot finish setup.
 */
export function SetupTokenField({
  registration,
  error,
  tokenFile,
}: SetupTokenFieldProps): React.JSX.Element {
  const { t } = useTranslation('setup');
  return (
    <div className={spacing.margin.bottom.content}>
      <FormField
        id="setup-token"
        label={t('setupToken.label')}
        required
        {...(tokenFile ? { hint: t('setupToken.help', { path: tokenFile }) } : {})}
        {...(error ? { error } : {})}
      >
        {(control) => (
          <Input
            {...control}
            data-testid="setup-token"
            type="password"
            autoComplete="off"
            required={true}
            {...registration}
            placeholder={t('setupToken.placeholder')}
          />
        )}
      </FormField>
    </div>
  );
}
