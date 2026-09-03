/**
 * ApiTokensSettings Component
 *
 * Settings panel for personal-access API tokens (Phase D-2 of
 * LICENSE_STRATEGY). Free and Starter tiers can see the panel and
 * their existing tokens but cannot mint new ones — the mint button
 * is disabled with a tooltip explaining the Pro requirement.
 *
 * Token plaintext is shown ONLY at creation time (server-side: only
 * SHA-256 hex is stored). The freshly minted token stays visible in
 * the UI until the user dismisses it.
 */

import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { api } from '../../../api/client';
import { useLicense } from '../../../contexts/LicenseContext';
import { useRole } from '../../../contexts/RoleContext';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Input } from '../../ui/Input';
import { Key, Plus, Trash2 } from '../../ui/icons';

interface ApiToken {
  id: string;
  name: string;
  prefix: string;
  createdAt: string;
  lastUsedAt?: string;
  revokedAt?: string;
}

interface MintTokenResponse {
  id: string;
  name: string;
  token: string;
  prefix: string;
  createdAt: string;
}

const ZERO_TIME_PREFIX = '0001-01-01';

function isZeroTime(value: string | undefined): boolean {
  return !value || value.startsWith(ZERO_TIME_PREFIX);
}

function formatDate(value: string | undefined): string {
  if (isZeroTime(value)) return '—';
  try {
    return new Date(value as string).toLocaleDateString();
  } catch {
    return value as string;
  }
}

export function ApiTokensSettings(): React.ReactElement {
  const { t } = useTranslation(['settings', 'common']);
  // License state is sourced from the shared LicenseProvider so every
  // tier-aware UI surface stays in sync; this panel used to fetch it
  // inline (pre-PR-A4).
  const { status: licenseStatus, refresh: refreshLicense } = useLicense();
  // #1226: viewers are read-only — block the mint flow regardless of tier.
  const { canWrite } = useRole();
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newTokenName, setNewTokenName] = useState('');
  const [minting, setMinting] = useState(false);
  const [mintedToken, setMintedToken] = useState<MintTokenResponse | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const [, tokenList] = await Promise.all([
        refreshLicense(),
        api.get<ApiToken[] | null>('/api/v1/tokens'),
      ]);
      setTokens(tokenList ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiTokens.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [refreshLicense, t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleMint = useCallback(async (): Promise<void> => {
    const trimmed = newTokenName.trim();
    if (!trimmed) return;
    setMinting(true);
    setError(null);
    try {
      const created = await api.post<MintTokenResponse>('/api/v1/tokens', { name: trimmed });
      setMintedToken(created);
      setNewTokenName('');
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiTokens.mintFailed'));
    } finally {
      setMinting(false);
    }
  }, [newTokenName, refresh, t]);

  const handleRevoke = useCallback(
    async (token: ApiToken): Promise<void> => {
      const ok = window.confirm(t('apiTokens.confirmRevoke', { name: token.name }));
      if (!ok) return;
      setError(null);
      try {
        await api.delete(`/api/v1/tokens/${token.id}`);
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : t('apiTokens.revokeFailed'));
      }
    },
    [refresh, t],
  );

  const canMint = licenseStatus?.canMintTokens ?? false;
  const tierLabel = licenseStatus?.tier ?? 'Free';

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-compact">
          <Key className="w-4 h-4" />
          <span>{t('apiTokens.title')}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm">
        <p className="text-sm text-text-secondary">
          <Trans
            i18nKey="apiTokens.description"
            ns="settings"
            components={{ strong: <strong /> }}
          />
        </p>

        {!canMint && (
          <div className="rounded-lg border border-status-warning/30 bg-status-warning/5 pad-sm text-sm text-status-warning">
            <Trans
              i18nKey="apiTokens.tierWarning"
              ns="settings"
              values={{ tier: tierLabel }}
              components={{ tier: <strong />, code: <code />, code2: <code /> }}
            />
          </div>
        )}

        {error ? (
          <div className="rounded-lg border border-status-error/30 bg-status-error/5 pad-sm text-sm text-status-error">
            {error}
          </div>
        ) : null}

        {mintedToken ? (
          <div className="rounded-lg border border-status-success/40 bg-status-success/5 pad-sm stack-xs">
            <div className="text-sm font-medium text-status-success">{t('apiTokens.created')}</div>
            <code className="block break-all rounded bg-surface-raised px-cell py-compact text-xs">
              {mintedToken.token}
            </code>
            <div className="flex gap-compact">
              <Button
                variant="outline"
                tone="green"
                size="sm"
                onClick={() => void navigator.clipboard?.writeText(mintedToken.token)}
              >
                {t('apiTokens.copy')}
              </Button>
              <Button variant="ghost" tone="gray" size="sm" onClick={() => setMintedToken(null)}>
                {t('apiTokens.saved')}
              </Button>
            </div>
          </div>
        ) : null}

        <div className="flex items-end gap-compact">
          <div className="flex-1">
            <label className="block text-xs text-text-muted mb-tight" htmlFor="api-token-name">
              {t('apiTokens.tokenName')}
            </label>
            <Input
              id="api-token-name"
              value={newTokenName}
              onChange={(e) => setNewTokenName(e.target.value)}
              placeholder={t('apiTokens.namePlaceholder')}
              maxLength={64}
              disabled={!canMint || !canWrite || minting}
            />
          </div>
          <Button
            variant="solid"
            tone="violet"
            size="md"
            leftIcon={<Plus className="w-4 h-4" />}
            disabled={!canMint || !canWrite || minting || newTokenName.trim().length === 0}
            loading={minting}
            title={
              !canWrite
                ? t('apiTokens.mintNeedsOperator')
                : !canMint
                  ? t('apiTokens.mintNeedsPro')
                  : undefined
            }
            onClick={() => void handleMint()}
          >
            {t('apiTokens.createToken')}
          </Button>
        </div>

        {loading ? (
          <div className="text-sm text-text-muted">{t('common:status.loading')}</div>
        ) : tokens.length === 0 ? (
          <div className="text-sm text-text-muted">{t('apiTokens.none')}</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="text-xs text-text-muted text-left">
              <tr>
                <th className="py-row pr-2">{t('apiTokens.colName')}</th>
                <th className="py-row pr-2">{t('apiTokens.colPrefix')}</th>
                <th className="py-row pr-2">{t('apiTokens.colCreated')}</th>
                <th className="py-row pr-2">{t('apiTokens.lastUsed')}</th>
                <th className="py-row pr-2">{t('apiTokens.colStatus')}</th>
                <th className="py-row" />
              </tr>
            </thead>
            <tbody>
              {tokens.map((token) => {
                const revoked = !isZeroTime(token.revokedAt);
                return (
                  <tr key={token.id} className="border-t border-surface-border">
                    <td className="py-row pr-2">{token.name}</td>
                    <td className="py-row pr-2 font-mono text-xs">{token.prefix}…</td>
                    <td className="py-row pr-2">{formatDate(token.createdAt)}</td>
                    <td className="py-row pr-2">{formatDate(token.lastUsedAt)}</td>
                    <td className="py-row pr-2">
                      {revoked ? (
                        <span className="text-status-error">{t('apiTokens.statusRevoked')}</span>
                      ) : (
                        <span className="text-status-success">{t('apiTokens.statusActive')}</span>
                      )}
                    </td>
                    <td className="py-row text-right">
                      {!revoked && (
                        <Button
                          variant="ghost"
                          tone="red"
                          size="xs"
                          leftIcon={<Trash2 className="w-3 h-3" />}
                          disabled={!canWrite}
                          title={canWrite ? undefined : t('apiTokens.revokeNeedsOperator')}
                          onClick={() => void handleRevoke(token)}
                        >
                          {t('apiTokens.revoke')}
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
