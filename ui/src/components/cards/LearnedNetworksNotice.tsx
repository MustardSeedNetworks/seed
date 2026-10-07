/**
 * LearnedNetworksNotice — asks the operator about each learned network
 * (seed#3108).
 *
 * A learned network stays off until someone adds it. Before this notice the
 * only trace of one was a disabled row in Settings, so extended ranges went
 * unscanned unless someone went looking. Each row names the CIDR and where it
 * was learned; Add switches it on, Dismiss means it is never offered again.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useDecideNetwork, usePendingNetworks } from '../../hooks/useLearnedNetworks';
import { cn, radius, spacing } from '../../styles/theme';
import { Button } from '../ui/Button';

export function LearnedNetworksNotice(): JSX.Element | null {
  const { t } = useTranslation('cards');
  const pending = usePendingNetworks();
  const decide = useDecideNetwork();

  if (pending.length === 0) {
    return null;
  }

  return (
    <section
      className={cn(
        'stack-sm border border-status-info/30 bg-status-info/10',
        spacing.pad.sm,
        radius.md,
      )}
      aria-labelledby="learned-networks-title"
      data-testid="learned-networks"
    >
      <div>
        <h3 id="learned-networks-title" className="body-small font-medium text-text-primary">
          {t('discovery.learned.title', { count: pending.length })}
        </h3>
        <p className="caption text-text-secondary">{t('discovery.learned.body')}</p>
      </div>
      <ul className="stack-sm">
        {pending.map((network) => (
          <li
            key={network.cidr}
            className="flex flex-wrap items-center gap-compact"
            data-testid="learned-network"
            data-cidr={network.cidr}
          >
            <div className="min-w-0 flex-1">
              <p className="body-small font-mono text-text-primary">{network.cidr}</p>
              <p className="caption text-text-muted">{network.name}</p>
            </div>
            <Button
              size="xs"
              variant="solid"
              disabled={decide.isPending}
              aria-label={t('discovery.learned.addAria', { cidr: network.cidr })}
              onClick={() => decide.mutate({ cidr: network.cidr, decision: 'added' })}
              data-testid="learned-network-add"
            >
              {t('discovery.learned.add')}
            </Button>
            <Button
              size="xs"
              variant="secondary"
              disabled={decide.isPending}
              aria-label={t('discovery.learned.dismissAria', { cidr: network.cidr })}
              onClick={() => decide.mutate({ cidr: network.cidr, decision: 'dismissed' })}
              data-testid="learned-network-dismiss"
            >
              {t('discovery.learned.dismiss')}
            </Button>
          </li>
        ))}
      </ul>
      {decide.isError ? (
        <p className="caption text-status-error" role="alert" data-testid="learned-networks-error">
          {t('discovery.learned.error')}
        </p>
      ) : null}
    </section>
  );
}
