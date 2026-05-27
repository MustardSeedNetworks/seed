/**
 * SsoSettings Component
 *
 * Settings → Single Sign-On panel for seed#1198. Lets operators enable
 * Google / Microsoft / GitHub OAuth providers without editing
 * config.yaml. The backend already implements:
 *   - GET  /api/v1/sso/settings (read provider configs, auth required)
 *   - PUT  /api/v1/sso/update   (write a single provider, Pro-gated via
 *                                requireFeature("sso"))
 *
 * Each provider is a card with: enabled toggle, client ID + client
 * secret (masked; blank = keep current), redirect URL display, tenant
 * ID (Microsoft only), and Save. Free/Starter operators see the panel
 * read-only with the upgrade message.
 */

import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../../api/client';
import { useLicense } from '../../../contexts/LicenseContext';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Input } from '../../ui/Input';
import { Key } from '../../ui/icons';

type ProviderName = 'google' | 'microsoft' | 'github';

interface ProviderConfig {
  name: ProviderName;
  enabled: boolean;
  client_id: string;
  client_secret: string;
  redirect_url: string;
  tenant_id?: string;
  scopes?: string[];
}

interface SettingsResponse {
  providers: { name: string; enabled: boolean }[];
}

const PROVIDER_DEFAULTS: Record<ProviderName, ProviderConfig> = {
  google: { name: 'google', enabled: false, client_id: '', client_secret: '', redirect_url: '' },
  microsoft: {
    name: 'microsoft',
    enabled: false,
    client_id: '',
    client_secret: '',
    redirect_url: '',
    tenant_id: 'common',
  },
  github: { name: 'github', enabled: false, client_id: '', client_secret: '', redirect_url: '' },
};

function defaultRedirectUrl(): string {
  if (typeof window === 'undefined') return '';
  return `${window.location.origin}/api/v1/sso/callback`;
}

export function SsoSettings(): React.ReactElement {
  const { t } = useTranslation(['settings', 'errors']);
  const { status: licenseStatus } = useLicense();

  const [providers, setProviders] = useState<Record<ProviderName, ProviderConfig>>(() => ({
    google: { ...PROVIDER_DEFAULTS.google, redirect_url: defaultRedirectUrl() },
    microsoft: { ...PROVIDER_DEFAULTS.microsoft, redirect_url: defaultRedirectUrl() },
    github: { ...PROVIDER_DEFAULTS.github, redirect_url: defaultRedirectUrl() },
  }));
  const [loading, setLoading] = useState(true);
  const [savingProvider, setSavingProvider] = useState<ProviderName | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saveStatus, setSaveStatus] = useState<string | null>(null);

  const canEdit = Boolean(licenseStatus?.features?.includes?.('sso'));

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const resp = await api.get<SettingsResponse>('/api/v1/sso/settings');
      const enabledByName = new Map(resp.providers.map((p) => [p.name.toLowerCase(), p.enabled]));
      setProviders((prev) => ({
        google: { ...prev.google, enabled: enabledByName.get('google') ?? false },
        microsoft: { ...prev.microsoft, enabled: enabledByName.get('microsoft') ?? false },
        github: { ...prev.github, enabled: enabledByName.get('github') ?? false },
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load SSO settings');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const updateField = useCallback(
    <K extends keyof ProviderConfig>(
      name: ProviderName,
      key: K,
      value: ProviderConfig[K],
    ): void => {
      setProviders((prev) => ({ ...prev, [name]: { ...prev[name], [key]: value } }));
    },
    [],
  );

  const handleSave = useCallback(
    async (name: ProviderName): Promise<void> => {
      setSavingProvider(name);
      setError(null);
      setSaveStatus(null);
      try {
        const cfg = providers[name];
        await api.put('/api/v1/sso/update', {
          provider: cfg.name,
          enabled: cfg.enabled,
          client_id: cfg.client_id,
          client_secret: cfg.client_secret,
          redirect_url: cfg.redirect_url || defaultRedirectUrl(),
          tenant_id: cfg.tenant_id,
          scopes: cfg.scopes,
        });
        setSaveStatus(t('settings:sso.status.saved', { provider: name }));
        // Wipe the secret input now that it's saved server-side.
        updateField(name, 'client_secret', '');
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to save SSO provider');
      } finally {
        setSavingProvider(null);
      }
    },
    [providers, refresh, t, updateField],
  );

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-2">
          <Key className="w-4 h-4" />
          <span>{t('settings:sso.title')}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm" data-testid="sso-settings-section">
        <p className="text-sm text-text-secondary">{t('settings:sso.description')}</p>

        {!canEdit && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm text-amber-200">
            {t('errors:sso.featureRequired')}
          </div>
        )}

        {error && (
          <div
            className="rounded-lg border border-status-error/30 bg-status-error/5 p-3 text-sm text-status-error"
            data-testid="sso-error"
          >
            {error}
          </div>
        )}

        {saveStatus && (
          <div className="rounded-lg border border-status-success/30 bg-status-success/5 p-3 text-sm text-status-success">
            {saveStatus}
          </div>
        )}

        {loading ? (
          <div className="text-sm text-text-muted">{t('common:status.loading', 'Loading…')}</div>
        ) : (
          <div className="stack-sm">
            <ProviderCard
              name="google"
              cfg={providers.google}
              canEdit={canEdit}
              saving={savingProvider === 'google'}
              onChange={updateField}
              onSave={() => void handleSave('google')}
            />
            <ProviderCard
              name="microsoft"
              cfg={providers.microsoft}
              canEdit={canEdit}
              saving={savingProvider === 'microsoft'}
              onChange={updateField}
              onSave={() => void handleSave('microsoft')}
            />
            <ProviderCard
              name="github"
              cfg={providers.github}
              canEdit={canEdit}
              saving={savingProvider === 'github'}
              onChange={updateField}
              onSave={() => void handleSave('github')}
            />
          </div>
        )}
      </div>
    </CollapsibleSection>
  );
}

interface ProviderCardProps {
  name: ProviderName;
  cfg: ProviderConfig;
  canEdit: boolean;
  saving: boolean;
  onChange: <K extends keyof ProviderConfig>(
    name: ProviderName,
    key: K,
    value: ProviderConfig[K],
  ) => void;
  onSave: () => void;
}

function ProviderCard({
  name,
  cfg,
  canEdit,
  saving,
  onChange,
  onSave,
}: ProviderCardProps): React.ReactElement {
  const { t } = useTranslation(['settings']);
  const isMicrosoft = name === 'microsoft';

  return (
    <div
      className="rounded-lg border border-surface-border bg-surface-raised p-3 stack-xs"
      data-testid={`sso-provider-${name}`}
    >
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium">{t(`settings:sso.providers.${name}`)}</h4>
        <label className="inline-flex items-center gap-1.5 text-xs">
          <input
            type="checkbox"
            checked={cfg.enabled}
            disabled={!canEdit || saving}
            onChange={(e) => onChange(name, 'enabled', e.target.checked)}
            data-testid={`sso-enable-${name}`}
          />
          {t('settings:sso.fields.enabled')}
        </label>
      </div>

      <div>
        <label className="block text-xs text-text-muted mb-1" htmlFor={`sso-${name}-cid`}>
          {t('settings:sso.fields.clientId')}
        </label>
        <Input
          id={`sso-${name}-cid`}
          value={cfg.client_id}
          onChange={(e) => onChange(name, 'client_id', e.target.value)}
          disabled={!canEdit || saving}
          data-testid={`sso-client-id-${name}`}
        />
      </div>

      <div>
        <label className="block text-xs text-text-muted mb-1" htmlFor={`sso-${name}-cs`}>
          {t('settings:sso.fields.clientSecret')}
        </label>
        <Input
          id={`sso-${name}-cs`}
          type="password"
          value={cfg.client_secret}
          onChange={(e) => onChange(name, 'client_secret', e.target.value)}
          placeholder={t('settings:sso.fields.clientSecretMask')}
          disabled={!canEdit || saving}
          data-testid={`sso-client-secret-${name}`}
        />
      </div>

      <div>
        <label className="block text-xs text-text-muted mb-1" htmlFor={`sso-${name}-ru`}>
          {t('settings:sso.fields.redirectUrl')}
        </label>
        <Input
          id={`sso-${name}-ru`}
          value={cfg.redirect_url || defaultRedirectUrl()}
          onChange={(e) => onChange(name, 'redirect_url', e.target.value)}
          disabled={!canEdit || saving}
          data-testid={`sso-redirect-${name}`}
        />
        <p className="text-xs text-text-muted mt-1">{t('settings:sso.fields.redirectUrlHint')}</p>
      </div>

      {isMicrosoft && (
        <div>
          <label className="block text-xs text-text-muted mb-1" htmlFor={`sso-${name}-tid`}>
            {t('settings:sso.fields.tenantId')}
          </label>
          <Input
            id={`sso-${name}-tid`}
            value={cfg.tenant_id ?? ''}
            onChange={(e) => onChange(name, 'tenant_id', e.target.value)}
            placeholder="common | organizations | <tenant-guid>"
            disabled={!canEdit || saving}
            data-testid={`sso-tenant-${name}`}
          />
          <p className="text-xs text-text-muted mt-1">{t('settings:sso.fields.tenantIdHint')}</p>
        </div>
      )}

      <div className="flex justify-end">
        <Button
          variant="solid"
          tone="violet"
          size="sm"
          disabled={!canEdit || saving}
          loading={saving}
          onClick={onSave}
          data-testid={`sso-save-${name}`}
        >
          {t('settings:sso.actions.save')}
        </Button>
      </div>
    </div>
  );
}
