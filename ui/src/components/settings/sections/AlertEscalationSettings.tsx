/**
 * AlertEscalationSettings Component
 *
 * Purpose: edit the per-rule escalation ladders (P-B2, #3186). A ladder
 * re-sends an alert nobody has acknowledged, on the channels each stage names,
 * a set time after the alert was raised. Before this the ladders existed only
 * in the settings API.
 *
 * It saves on an explicit action, like the webhook section: the list is
 * replaced whole, and the server refuses a ladder that could not run with its
 * reason, which is shown as given.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import {
  ESCALATION_CHANNELS,
  type EscalationChannel,
  type EscalationLadderDraft,
  MAX_ESCALATION_STAGES,
  newLadderDraft,
  stageDraft,
  useAlertEscalationSettings,
} from '../../../hooks/useAlertEscalationSettings';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
} from '../../../styles/theme';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Bell } from '../../ui/Icons';
import { Tooltip } from '../../ui/Tooltip';
import { AutoSaveIndicator } from './AutoSaveIndicator';

const RULE_OPTIONS_ID = 'alert-escalation-rules';

/** Minutes in the form, seconds on the wire; an empty field is 0. */
function toSeconds(minutes: string): number {
  return Math.round(Number(minutes) * 60);
}

function toMinutes(seconds: number): string {
  return seconds === 0 ? '' : String(seconds / 60);
}

export function AlertEscalationSettings(): JSX.Element {
  const { ladders, setLadders, ruleOptions, status, error, save } = useAlertEscalationSettings();
  const { t } = useTranslation('settings');
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');
  const channelLabel: Record<EscalationChannel, string> = {
    webhook: t('alertEscalation.channelWebhook'),
    email: t('alertEscalation.channelEmail'),
    syslog: t('alertEscalation.channelSyslog'),
  };

  const editLadder = (
    id: number,
    edit: (ladder: EscalationLadderDraft) => EscalationLadderDraft,
  ): void => {
    setLadders((current) => current.map((ladder) => (ladder.id === id ? edit(ladder) : ladder)));
  };

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Bell className={iconTokens.size.sm} />
          <span>{t('sections.alertEscalation')}</span>
          <AutoSaveIndicator status={status} />
        </div>
      }
      defaultOpen={false}
      data-testid="alert-escalation-section"
    >
      <div className="stack">
        <p className="body-small text-text-muted">{t('alertEscalation.description')}</p>
        <p className="caption text-text-muted">{t('alertEscalation.stageRules')}</p>

        <datalist id={RULE_OPTIONS_ID}>
          {ruleOptions.map((rule) => (
            <option key={rule} value={rule} />
          ))}
        </datalist>

        {ladders.length === 0 ? (
          <p data-testid="escalation-empty" className="body-small text-text-secondary">
            {t('alertEscalation.empty')}
          </p>
        ) : null}

        {ladders.map((ladder, li) => (
          <fieldset
            key={ladder.id}
            data-testid={`escalation-ladder-${li}`}
            disabled={!canWrite}
            className="stack-sm rounded-md border border-surface-border pad-sm"
          >
            <label className="stack-xs" htmlFor={`escalation-rule-${ladder.id}`}>
              <span className="body-small font-medium text-text-primary">
                {t('alertEscalation.rule')}
              </span>
              <input
                id={`escalation-rule-${ladder.id}`}
                data-testid={`escalation-rule-${li}`}
                type="text"
                list={RULE_OPTIONS_ID}
                autoComplete="off"
                value={ladder.rule}
                onChange={(e): void => {
                  const rule = e.target.value;
                  editLadder(ladder.id, (l) => ({ ...l, rule }));
                }}
                className={cn(inputTokens.base, 'w-full')}
              />
              <span className="caption text-text-muted">{t('alertEscalation.ruleHelp')}</span>
            </label>

            {ladder.stages.map((stage, si) => (
              <div
                key={stage.id}
                data-testid={`escalation-stage-${li}-${si}`}
                className="flex flex-wrap items-end gap-default"
              >
                <label className="stack-xs" htmlFor={`escalation-after-${stage.id}`}>
                  <span className="body-small font-medium text-text-primary">
                    {t('alertEscalation.stageAfter', { stage: si + 1 })}
                  </span>
                  <input
                    id={`escalation-after-${stage.id}`}
                    data-testid={`escalation-after-${li}-${si}`}
                    type="number"
                    min={1}
                    step={1}
                    inputMode="numeric"
                    value={toMinutes(stage.afterSeconds)}
                    onChange={(e): void => {
                      const afterSeconds = toSeconds(e.target.value);
                      editLadder(ladder.id, (l) => ({
                        ...l,
                        stages: l.stages.map((s, i) => (i === si ? { ...s, afterSeconds } : s)),
                      }));
                    }}
                    className={cn(inputTokens.base, 'w-24')}
                  />
                </label>
                <fieldset className="flex flex-wrap items-center gap-default min-h-9">
                  <legend className="sr-only">
                    {t('alertEscalation.stageChannels', { stage: si + 1 })}
                  </legend>
                  {ESCALATION_CHANNELS.map((channel) => (
                    <label
                      key={channel}
                      className="flex items-center gap-tight text-sm text-text-secondary"
                    >
                      <input
                        type="checkbox"
                        data-testid={`escalation-channel-${li}-${si}-${channel}`}
                        checked={stage.channels.includes(channel)}
                        onChange={(e): void => {
                          const on = e.target.checked;
                          editLadder(ladder.id, (l) => ({
                            ...l,
                            stages: l.stages.map((s, i) =>
                              i === si
                                ? {
                                    ...s,
                                    channels: on
                                      ? ESCALATION_CHANNELS.filter(
                                          (c) => c === channel || s.channels.includes(c),
                                        )
                                      : s.channels.filter((c) => c !== channel),
                                  }
                                : s,
                            ),
                          }));
                        }}
                      />
                      {channelLabel[channel]}
                    </label>
                  ))}
                </fieldset>
                <button
                  type="button"
                  data-testid={`escalation-remove-stage-${li}-${si}`}
                  disabled={ladder.stages.length === 1}
                  onClick={(): void => {
                    editLadder(ladder.id, (l) => ({
                      ...l,
                      stages: l.stages.filter((_, i) => i !== si),
                    }));
                  }}
                  className={cn(
                    buttonTokens.base,
                    buttonTokens.variant.ghost,
                    buttonTokens.size.sm,
                  )}
                >
                  {t('alertEscalation.removeStage', { stage: si + 1 })}
                </button>
              </div>
            ))}

            <div className={layout.inline.default}>
              <button
                type="button"
                data-testid={`escalation-add-stage-${li}`}
                disabled={ladder.stages.length >= MAX_ESCALATION_STAGES}
                onClick={(): void => {
                  editLadder(ladder.id, (l) => {
                    const last = l.stages[l.stages.length - 1];
                    // Five minutes after the last stage, on its channels: a
                    // stage must come after the one before it.
                    const next = stageDraft({
                      afterSeconds: (last?.afterSeconds ?? 0) + 300,
                      channels: last?.channels ?? ['webhook'],
                    });
                    return { ...l, stages: [...l.stages, next] };
                  });
                }}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.secondary,
                  buttonTokens.size.sm,
                )}
              >
                {t('alertEscalation.addStage')}
              </button>
              <button
                type="button"
                data-testid={`escalation-remove-${li}`}
                onClick={(): void => {
                  setLadders((current) => current.filter((l) => l.id !== ladder.id));
                }}
                className={cn(buttonTokens.base, buttonTokens.variant.ghost, buttonTokens.size.sm)}
              >
                {t('alertEscalation.removeLadder')}
              </button>
            </div>

            <label className="stack-xs" htmlFor={`escalation-repeat-${ladder.id}`}>
              <span className="body-small font-medium text-text-primary">
                {t('alertEscalation.repeat')}
              </span>
              <input
                id={`escalation-repeat-${ladder.id}`}
                data-testid={`escalation-repeat-${li}`}
                type="number"
                min={1}
                step={1}
                inputMode="numeric"
                value={toMinutes(ladder.repeatSeconds)}
                onChange={(e): void => {
                  const repeatSeconds = toSeconds(e.target.value);
                  editLadder(ladder.id, (l) => ({ ...l, repeatSeconds }));
                }}
                className={cn(inputTokens.base, 'w-24')}
              />
              <span className="caption text-text-muted">{t('alertEscalation.repeatHelp')}</span>
            </label>
          </fieldset>
        ))}

        {error === '' ? null : (
          <p data-testid="escalation-error" className="body-small text-status-error">
            {error}
          </p>
        )}

        <div className={layout.inline.default}>
          <Tooltip text={readOnlyReason}>
            <button
              type="button"
              data-testid="escalation-add"
              disabled={!canWrite}
              onClick={(): void => {
                setLadders((current) => [...current, newLadderDraft()]);
              }}
              className={cn(
                buttonTokens.base,
                buttonTokens.variant.secondary,
                buttonTokens.size.sm,
              )}
            >
              {t('alertEscalation.addLadder')}
            </button>
          </Tooltip>
          <Tooltip text={readOnlyReason}>
            <button
              type="button"
              data-testid="escalation-save"
              disabled={!canWrite || status === 'saving'}
              onClick={(): void => {
                save().catch(() => undefined);
              }}
              className={cn(buttonTokens.base, buttonTokens.variant.primary, buttonTokens.size.sm)}
            >
              {t('alertEscalation.save')}
            </button>
          </Tooltip>
        </div>
      </div>
    </CollapsibleSection>
  );
}
