/**
 * SetupSsoOptions — the sign-in alternatives under the setup form.
 *
 * `/api/v1/sso/providers` lists only providers that are enabled AND have a
 * ClientID (`internal/api/v1/handlers_oauth.go::initOAuthManager`, #720), so
 * presence in the list is enough to offer one. Nothing renders while none is.
 */

import type React from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { LogComponents, logger } from '../../lib/logger';
import { button, cn, radius } from '../../styles/theme';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

export function SetupSsoOptions(): React.JSX.Element | null {
  const { t: tCommon } = useTranslation('common');
  const [ssoProviders, setSsoProviders] = useState<string[]>([]);

  // Fixes #769, #720.
  useEffect(() => {
    fetch(`${API_BASE}/api/v1/sso/providers`)
      .then((res) => (res.ok ? res.json() : { providers: [] }))
      .then((data: { providers?: string[] }) => {
        const providers = data.providers ?? [];
        setSsoProviders(providers);
        logger.info(LogComponents.SETUP, 'SSO provider list loaded', {
          count: providers.length,
        });
      })
      .catch((err: unknown) => {
        setSsoProviders([]);
        logger.warn(LogComponents.SETUP, 'Failed to load SSO providers', {
          error: err instanceof Error ? err.message : String(err),
        });
      });
  }, []);

  const isProviderEnabled = (name: string): boolean =>
    ssoProviders.some((p) => p.toLowerCase() === name.toLowerCase());

  if (ssoProviders.length === 0) {
    return null;
  }

  return (
    <>
      {/* Separator */}
      <div className="relative my-6">
        <div className="absolute inset-0 flex items-center" aria-hidden="true">
          <div className="w-full border-t border-surface-border" />
        </div>
        <div className="relative flex justify-center">
          <span className="px-cell bg-surface-raised text-sm text-text-muted">{tCommon('or')}</span>
        </div>
      </div>

      <div className="flex flex-col stack-sm">
        {isProviderEnabled('google') && (
          <button
            type="button"
            onClick={() => {
              window.location.href = `${API_BASE}/api/v1/sso/login?provider=google`;
            }}
            className={cn(
              'w-full',
              button.size.md,
              'bg-status-info text-text-inverse',
              radius.md,
              'font-medium hover:bg-status-info/85 focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2 focus:ring-offset-surface-base disabled:control-disabled',
            )}
          >
            {tCommon('buttons.signInWithGoogle')}
          </button>
        )}
        {isProviderEnabled('microsoft') && (
          <button
            type="button"
            onClick={() => {
              window.location.href = `${API_BASE}/api/v1/sso/login?provider=microsoft`;
            }}
            className={cn(
              'w-full',
              button.size.md,
              'bg-surface-sunken text-text-primary',
              radius.md,
              'font-medium hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-surface-border focus:ring-offset-2 focus:ring-offset-surface-base border border-surface-border disabled:control-disabled',
            )}
          >
            {tCommon('buttons.signInWithMicrosoft')}
          </button>
        )}
        {isProviderEnabled('github') && (
          <button
            type="button"
            onClick={() => {
              window.location.href = `${API_BASE}/api/v1/sso/login?provider=github`;
            }}
            className={cn(
              'w-full',
              button.size.md,
              'bg-surface-sunken text-text-primary',
              radius.md,
              'font-medium hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-surface-border focus:ring-offset-2 focus:ring-offset-surface-base border border-surface-border disabled:control-disabled',
            )}
          >
            {tCommon('buttons.signInWithGitHub')}
          </button>
        )}
      </div>
    </>
  );
}
