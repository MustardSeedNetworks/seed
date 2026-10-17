/**
 * useLifecycleFeed
 *
 * Loads and saves the opt-in EoX feed (P-D9, UI-SEED-47). Owner decision
 * 2026-10-14: the feed is off until an admin answers on, and "no" is the
 * default answer, so an unanswered setting edits as off. The client secret is
 * write-only: the server reports whether one is stored, and an empty secret on
 * save keeps it.
 */

import { useEffect, useState } from 'react';
import { api } from '../api';
import type { LifecycleFeedResponse } from '../types/generated/lifecycle-feed-response';
import type { LifecycleFeedUpdate } from '../types/generated/lifecycle-feed-update';
import type { SaveStatus } from '../types/settings';

const FEED = '/api/v1/lifecycle/feed';

/** The feed as the section edits it. */
export interface FeedDraft {
  choice: LifecycleFeedUpdate['choice'];
  endpoint: string;
  tokenUrl: string;
  clientId: string;
  /** Newly entered secret; empty means keep the stored one. */
  clientSecret: string;
  clearClientSecret: boolean;
}

interface UseLifecycleFeedResult {
  /** What the server holds, or null until it has loaded. */
  stored: LifecycleFeedResponse | null;
  draft: FeedDraft;
  setDraft: React.Dispatch<React.SetStateAction<FeedDraft>>;
  status: SaveStatus;
  /** Why the load or the last save failed, for display. */
  error: string;
  save: () => Promise<void>;
}

const EMPTY_DRAFT: FeedDraft = {
  choice: 'off',
  endpoint: '',
  tokenUrl: '',
  clientId: '',
  clientSecret: '',
  clearClientSecret: false,
};

function draftOf(stored: LifecycleFeedResponse): FeedDraft {
  return {
    choice: stored.choice === 'on' ? 'on' : 'off',
    endpoint: stored.endpoint,
    tokenUrl: stored.tokenUrl,
    clientId: stored.clientId,
    clientSecret: '',
    clearClientSecret: false,
  };
}

export function useLifecycleFeed(): UseLifecycleFeedResult {
  const [stored, setStored] = useState<LifecycleFeedResponse | null>(null);
  const [draft, setDraft] = useState<FeedDraft>(EMPTY_DRAFT);
  const [status, setStatus] = useState<SaveStatus>('idle');
  const [error, setError] = useState('');

  const accept = (response: LifecycleFeedResponse): void => {
    setStored(response);
    setDraft(draftOf(response));
  };

  useEffect((): void => {
    api
      .get<LifecycleFeedResponse>(FEED)
      .then(accept)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, [accept]);

  const save = async (): Promise<void> => {
    setStatus('saving');
    setError('');
    const update: LifecycleFeedUpdate = {
      choice: draft.choice,
      endpoint: draft.endpoint.trim(),
      tokenUrl: draft.tokenUrl.trim(),
      clientId: draft.clientId.trim(),
    };
    if (draft.clientSecret !== '') {
      update.clientSecret = draft.clientSecret;
    } else if (draft.clearClientSecret) {
      update.clearClientSecret = true;
    }
    await api
      .put<LifecycleFeedResponse>(FEED, update)
      .then((response) => {
        accept(response);
        setStatus('saved');
        setTimeout(() => setStatus('idle'), 2000);
      })
      .catch((err: unknown) => {
        setStatus('error');
        setError(err instanceof Error ? err.message : String(err));
      });
  };

  return { stored, draft, setDraft, status, error, save };
}
