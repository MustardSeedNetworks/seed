/**
 * useAlertEscalationSettings
 *
 * Loads and saves the per-rule escalation ladders (P-B2, #3186) through the
 * main settings endpoint. Like the webhook section it saves on an explicit
 * action: the server replaces the whole list and refuses a ladder that could
 * not run, so a debounced write of a half-built stage would only be refused.
 *
 * The rule picker offers the rules alerts in the inbox carry. A ladder may
 * still name a rule that has not fired yet; the server is the judge of what
 * runs.
 */

import { useEffect, useState } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { AlertsListResponse } from '../types/alerts';
import type { SaveStatus } from '../types/settings';

export const ESCALATION_CHANNELS = ['webhook', 'email', 'syslog'] as const;

export type EscalationChannel = (typeof ESCALATION_CHANNELS)[number];

/** Mirrors the server's escalation.MaxStages. */
export const MAX_ESCALATION_STAGES = 5;

/** One stage: fires afterSeconds after the alert was raised. */
export interface EscalationStage {
  afterSeconds: number;
  channels: EscalationChannel[];
}

/** One rule's ladder, the wire shape of alerts.escalations[]. */
export interface EscalationLadder {
  rule: string;
  stages: EscalationStage[];
  /** Re-sends the last stage at this period; 0 sends it once. */
  repeatSeconds: number;
}

/**
 * A ladder and its stages being edited. The ids are list keys only, so
 * removing a ladder or a stage does not hand its inputs to the next one; they
 * never reach the server.
 */
export interface EscalationStageDraft extends EscalationStage {
  id: number;
}

export interface EscalationLadderDraft {
  id: number;
  rule: string;
  stages: EscalationStageDraft[];
  repeatSeconds: number;
}

/** The alerts slice of GET /api/v1/settings. */
interface EscalationSettingsResponse {
  alerts?: { escalations?: EscalationLadder[] };
}

let lastDraftId = 0;

function draftId(): number {
  lastDraftId += 1;
  return lastDraftId;
}

export function stageDraft(stage: EscalationStage): EscalationStageDraft {
  return { ...stage, id: draftId() };
}

export function toDraft(ladder: EscalationLadder): EscalationLadderDraft {
  return { ...ladder, id: draftId(), stages: ladder.stages.map(stageDraft) };
}

/** Five minutes after the alert, on the webhook: the common first step. */
export function newLadderDraft(): EscalationLadderDraft {
  return toDraft({
    rule: '',
    stages: [{ afterSeconds: 300, channels: ['webhook'] }],
    repeatSeconds: 0,
  });
}

interface UseAlertEscalationSettingsResult {
  ladders: EscalationLadderDraft[];
  setLadders: React.Dispatch<React.SetStateAction<EscalationLadderDraft[]>>;
  /** Distinct rules carried by alerts in the inbox, sorted. */
  ruleOptions: string[];
  status: SaveStatus;
  /** The server's reason when a save was refused, for display. */
  error: string;
  save: () => Promise<void>;
}

export function useAlertEscalationSettings(): UseAlertEscalationSettingsResult {
  const [ladders, setLadders] = useState<EscalationLadderDraft[]>([]);
  const [ruleOptions, setRuleOptions] = useState<string[]>([]);
  const [status, setStatus] = useState<SaveStatus>('idle');
  const [error, setError] = useState('');

  const load = async (): Promise<void> => {
    await Promise.all([
      api
        .get<EscalationSettingsResponse>('/api/v1/settings')
        .then((data) => setLadders((data.alerts?.escalations ?? []).map(toDraft)))
        .catch((err: unknown) => {
          logger.error(LogComponents.CONFIG, 'Failed to fetch alert escalation settings', err);
        }),
      api
        .get<AlertsListResponse>('/api/v1/alerts?limit=1000')
        .then((data) => {
          const rules = new Set<string>();
          for (const alert of data.alerts) {
            if (alert.rule) {
              rules.add(alert.rule);
            }
          }
          setRuleOptions([...rules].sort());
        })
        .catch((err: unknown) => {
          logger.error(LogComponents.CONFIG, 'Failed to fetch alert rules', err);
        }),
    ]);
  };

  const save = async (): Promise<void> => {
    setStatus('saving');
    setError('');
    const escalations: EscalationLadder[] = ladders.map(({ rule, stages, repeatSeconds }) => ({
      rule: rule.trim(),
      stages: stages.map(({ afterSeconds, channels }) => ({ afterSeconds, channels })),
      repeatSeconds,
    }));
    await api
      .put('/api/v1/settings', { alerts: { escalations } })
      .then(() => {
        setStatus('saved');
        setTimeout(() => setStatus('idle'), 2000);
      })
      .catch((err: unknown) => {
        setStatus('error');
        setError(err instanceof Error ? err.message : '');
      });
  };

  useEffect((): void => {
    load().catch(() => undefined);
  }, [load]);

  return { ladders, setLadders, ruleOptions, status, error, save };
}
