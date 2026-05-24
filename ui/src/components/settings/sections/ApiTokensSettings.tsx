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
import { api } from '../../../api/client';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Input } from '../../ui/Input';
import { Key, Plus, Trash2 } from '../../ui/icons';

interface LicenseStatus {
  tier: string;
  tierValue: number;
  isTrialMode: boolean;
  trialDaysLeft?: number;
  canMintTokens: boolean;
  activated: boolean;
}

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
  const [licenseStatus, setLicenseStatus] = useState<LicenseStatus | null>(null);
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newTokenName, setNewTokenName] = useState('');
  const [minting, setMinting] = useState(false);
  const [mintedToken, setMintedToken] = useState<MintTokenResponse | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const [status, tokenList] = await Promise.all([
        api.get<LicenseStatus>('/api/v1/license'),
        api.get<ApiToken[] | null>('/api/v1/tokens'),
      ]);
      setLicenseStatus(status);
      setTokens(tokenList ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load API tokens');
    } finally {
      setLoading(false);
    }
  }, []);

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
      setError(err instanceof Error ? err.message : 'Failed to mint token');
    } finally {
      setMinting(false);
    }
  }, [newTokenName, refresh]);

  const handleRevoke = useCallback(
    async (token: ApiToken): Promise<void> => {
      const ok = window.confirm(
        `Revoke token "${token.name}"? Any script using it will stop working immediately.`,
      );
      if (!ok) return;
      setError(null);
      try {
        await api.delete(`/api/v1/tokens/${token.id}`);
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to revoke token');
      }
    },
    [refresh],
  );

  const canMint = licenseStatus?.canMintTokens ?? false;
  const tierLabel = licenseStatus?.tier ?? 'Free';

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-2">
          <Key className="w-4 h-4" />
          <span>API Tokens</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm">
        <p className="text-sm text-text-secondary">
          Personal-access tokens for programmatic API calls (scripts, monitoring, CI). Available on
          the <strong>Pro</strong> tier. The web UI works on every tier without a token.
        </p>

        {!canMint && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm text-amber-200">
            Current tier: <strong>{tierLabel}</strong>. Minting API tokens requires Pro. Start a
            14-day trial with <code>seed license trial</code>, or activate a Pro key with{' '}
            <code>seed license activate -k &lt;KEY&gt;</code>.
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-status-error/30 bg-status-error/5 p-3 text-sm text-status-error">
            {error}
          </div>
        )}

        {mintedToken && (
          <div className="rounded-lg border border-status-success/40 bg-status-success/5 p-3 stack-xs">
            <div className="text-sm font-medium text-status-success">
              Token created — copy it now. It will not be shown again.
            </div>
            <code className="block break-all rounded bg-surface-raised px-2 py-1 text-xs">
              {mintedToken.token}
            </code>
            <div className="flex gap-2">
              <Button
                variant="outline"
                tone="green"
                size="sm"
                onClick={() => void navigator.clipboard?.writeText(mintedToken.token)}
              >
                Copy
              </Button>
              <Button variant="ghost" tone="gray" size="sm" onClick={() => setMintedToken(null)}>
                I&apos;ve saved it
              </Button>
            </div>
          </div>
        )}

        <div className="flex items-end gap-2">
          <div className="flex-1">
            <label className="block text-xs text-text-muted mb-1" htmlFor="api-token-name">
              Token name
            </label>
            <Input
              id="api-token-name"
              value={newTokenName}
              onChange={(e) => setNewTokenName(e.target.value)}
              placeholder="e.g. monitoring-prod"
              maxLength={64}
              disabled={!canMint || minting}
            />
          </div>
          <Button
            variant="solid"
            tone="violet"
            size="md"
            leftIcon={<Plus className="w-4 h-4" />}
            disabled={!canMint || minting || newTokenName.trim().length === 0}
            loading={minting}
            title={canMint ? undefined : 'API token minting requires the Pro tier'}
            onClick={() => void handleMint()}
          >
            Create token
          </Button>
        </div>

        {loading ? (
          <div className="text-sm text-text-muted">Loading…</div>
        ) : tokens.length === 0 ? (
          <div className="text-sm text-text-muted">No API tokens yet.</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="text-xs text-text-muted text-left">
              <tr>
                <th className="py-2 pr-2">Name</th>
                <th className="py-2 pr-2">Prefix</th>
                <th className="py-2 pr-2">Created</th>
                <th className="py-2 pr-2">Last used</th>
                <th className="py-2 pr-2">Status</th>
                <th className="py-2" />
              </tr>
            </thead>
            <tbody>
              {tokens.map((t) => {
                const revoked = !isZeroTime(t.revokedAt);
                return (
                  <tr key={t.id} className="border-t border-surface-border">
                    <td className="py-2 pr-2">{t.name}</td>
                    <td className="py-2 pr-2 font-mono text-xs">{t.prefix}…</td>
                    <td className="py-2 pr-2">{formatDate(t.createdAt)}</td>
                    <td className="py-2 pr-2">{formatDate(t.lastUsedAt)}</td>
                    <td className="py-2 pr-2">
                      {revoked ? (
                        <span className="text-status-error">revoked</span>
                      ) : (
                        <span className="text-status-success">active</span>
                      )}
                    </td>
                    <td className="py-2 text-right">
                      {!revoked && (
                        <Button
                          variant="ghost"
                          tone="red"
                          size="xs"
                          leftIcon={<Trash2 className="w-3 h-3" />}
                          onClick={() => void handleRevoke(t)}
                        >
                          Revoke
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
