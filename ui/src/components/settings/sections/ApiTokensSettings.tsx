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

import type { MintTokenResponse } from '../../../types/generated/mint-token-response';

// The hand-typed mirror omitted `scope`, so a minted token's scope could not
// be shown or checked in this view (seed#2393).
export type { MintTokenResponse };

import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { api } from '../../../api/client';
import { useLicense } from '../../../contexts/LicenseContext';
import { useRole } from '../../../contexts/RoleContext';
import { useTableLabels } from '../../../hooks/useTableLabels';
import { Button as SharedButton } from '../../../shared/Button';
import { ConfirmDialog } from '../../../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../../../shared/DataTable';
import { StatusPill } from '../../../shared/StatusPill';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Key, Plus, Trash2 } from '../../ui/Icons';
import { Input } from '../../ui/Input';

interface ApiToken {
  id: string;
  name: string;
  prefix: string;
  createdAt: string;
  lastUsedAt?: string;
  revokedAt?: string;
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

function listTokens(): Promise<ApiToken[]> {
  return api.get<ApiToken[] | null>('/api/v1/tokens').then((list) => list ?? []);
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
  const [revokeTarget, setRevokeTarget] = useState<ApiToken | null>(null);
  const labels = useTableLabels<ApiToken>({ loading: t('apiTokens.title') });

  const columns: DataTableColumn<ApiToken>[] = [
    {
      key: 'name',
      header: t('apiTokens.colName'),
      sortValue: (token) => token.name,
      cell: (token) => token.name,
    },
    {
      key: 'prefix',
      header: t('apiTokens.colPrefix'),
      cell: (token) => <span className="font-mono text-xs">{token.prefix}…</span>,
    },
    {
      key: 'created',
      header: t('apiTokens.colCreated'),
      sortValue: (token) => token.createdAt,
      cell: (token) => formatDate(token.createdAt),
    },
    {
      key: 'lastUsed',
      header: t('apiTokens.lastUsed'),
      sortValue: (token) => (isZeroTime(token.lastUsedAt) ? '' : (token.lastUsedAt ?? '')),
      cell: (token) => formatDate(token.lastUsedAt),
    },
    {
      key: 'status',
      header: t('apiTokens.colStatus'),
      cell: (token) =>
        isZeroTime(token.revokedAt) ? (
          <StatusPill tone="success" label={t('apiTokens.statusActive')} />
        ) : (
          <StatusPill tone="error" label={t('apiTokens.statusRevoked')} />
        ),
    },
  ];

  const refresh = async (): Promise<void> => {
    setError(null);
    try {
      const [, tokenList] = await Promise.all([refreshLicense(), listTokens()]);
      setTokens(tokenList);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiTokens.loadFailed'));
    }
    setLoading(false);
  };

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleMint = async (): Promise<void> => {
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
    }
    setMinting(false);
  };

  const handleRevoke = async (token: ApiToken): Promise<void> => {
    setRevokeTarget(null);
    setError(null);
    try {
      await api.delete(`/api/v1/tokens/${token.id}`);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiTokens.revokeFailed'));
    }
  };

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
          <div className="rounded-lg border border-status-warning/30 bg-status-warning/5 pad-sm text-sm text-status-warning-strong">
            <Trans
              i18nKey="apiTokens.tierWarning"
              ns="settings"
              values={{ tier: tierLabel }}
              components={{ tier: <strong />, code: <code />, code2: <code /> }}
            />
          </div>
        )}

        {error ? (
          <div className="rounded-lg border border-status-error/30 bg-status-error/5 pad-sm text-sm text-status-error-strong">
            {error}
          </div>
        ) : null}

        {mintedToken ? (
          <div className="rounded-lg border border-status-success/40 bg-status-success/5 pad-sm stack-xs">
            <div className="text-sm font-medium text-status-success-strong">
              {t('apiTokens.created')}
            </div>
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
          <DataTable
            rows={tokens}
            columns={columns}
            getRowKey={(token) => token.id}
            caption={t('apiTokens.title')}
            empty={null}
            labels={labels}
            pageSize={10}
            defaultSort={{ key: 'created', direction: 'desc' }}
            rowActions={(token) =>
              isZeroTime(token.revokedAt) ? (
                <SharedButton
                  variant="destructive"
                  size="sm"
                  icon={<Trash2 className="w-3 h-3" />}
                  disabled={!canWrite}
                  title={canWrite ? undefined : t('apiTokens.revokeNeedsOperator')}
                  onClick={() => setRevokeTarget(token)}
                >
                  {t('apiTokens.revoke')}
                </SharedButton>
              ) : null
            }
          />
        )}
      </div>
      <ConfirmDialog
        open={revokeTarget !== null}
        title={t('apiTokens.confirmRevokeTitle', { name: revokeTarget?.name ?? '' })}
        message={t('apiTokens.confirmRevoke')}
        confirmLabel={t('apiTokens.revoke')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={() => {
          if (revokeTarget) void handleRevoke(revokeTarget);
        }}
        onCancel={() => setRevokeTarget(null)}
      />
    </CollapsibleSection>
  );
}
