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
import { Checkbox } from '../../../shared/Checkbox';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Tooltip } from '../../../shared/Tooltip';
import { button as buttonTokens, cn, icon as iconTokens, layout } from '../../../styles/theme';
import { Bell } from '../../ui/Icons';
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
    trap: t('alertEscalation.channelTrap'),
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
            <FormField
              id={`escalation-rule-${ladder.id}`}
              label={t('alertEscalation.rule')}
              hint={t('alertEscalation.ruleHelp')}
            >
              {(control) => (
                <Input
                  {...control}
                  data-testid={`escalation-rule-${li}`}
                  list={RULE_OPTIONS_ID}
                  autoComplete="off"
                  value={ladder.rule}
                  onChange={(e): void => {
                    const rule = e.target.value;
                    editLadder(ladder.id, (l) => ({ ...l, rule }));
                  }}
                />
              )}
            </FormField>

            {ladder.stages.map((stage, si) => (
              <div
                key={stage.id}
                data-testid={`escalation-stage-${li}-${si}`}
                className="flex flex-wrap items-end gap-default"
              >
                <div className="w-28">
                  <FormField
                    id={`escalation-after-${stage.id}`}
                    label={t('alertEscalation.stageAfter', { stage: si + 1 })}
                  >
                    {(control) => (
                      <Input
                        {...control}
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
                      />
                    )}
                  </FormField>
                </div>
                <FormField group label={t('alertEscalation.stageChannels', { stage: si + 1 })}>
                  <div className="flex flex-wrap items-center gap-default min-h-9">
                    {ESCALATION_CHANNELS.map((channel) => (
                      <Checkbox
                        key={channel}
                        label={channelLabel[channel]}
                        data-testid={`escalation-channel-${li}-${si}-${channel}`}
                        checked={stage.channels.includes(channel)}
                        onChange={(on: boolean): void => {
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
                    ))}
                  </div>
                </FormField>
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

            <FormField
              id={`escalation-repeat-${ladder.id}`}
              label={t('alertEscalation.repeat')}
              hint={t('alertEscalation.repeatHelp')}
            >
              {(control) => (
                <Input
                  {...control}
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
                />
              )}
            </FormField>
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
