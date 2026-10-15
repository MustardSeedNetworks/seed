import type React from 'react';
import type { UseFormRegisterReturn } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { cn, input, spacing, status as statusColor } from '../../styles/theme';

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
      <label
        htmlFor="setup-token"
        className={cn(
          'block body-small font-medium text-text-primary',
          spacing.margin.bottom.inline,
        )}
      >
        {t('setupToken.label')}
      </label>
      <input
        id="setup-token"
        data-testid="setup-token"
        type="password"
        autoComplete="off"
        required={true}
        {...registration}
        className={cn(input.base, input.state.default, input.size.md)}
        placeholder={t('setupToken.placeholder')}
      />
      {tokenFile ? (
        <p className={cn('caption text-text-muted', spacing.margin.top.inline)}>
          {t('setupToken.help', { path: tokenFile })}
        </p>
      ) : null}
      {error ? <p className={cn('caption mt-tight', statusColor.text.error)}>{error}</p> : null}
    </div>
  );
}
