/**
 * WifiSurveyCard Component
 *
 * Purpose: Manages Wifi site surveys - allows creating floor plan-based Wifi signal mapping campaigns.
 * Shows active and completed surveys with ability to create, pause, complete, and delete surveys.
 *
 * Key Features:
 * - Survey creation: dialog to create new survey with custom floor plan image
 * - Survey types: supports different survey modes (e.g., signal mapping, interference detection)
 * - Active surveys: shows in-progress and paused surveys with controls
 * - Completed surveys: displays finished surveys with ability to view results
 * - Survey controls: start, pause, complete, delete operations
 * - Floor plan visualization: SurveyView component displays floor plan with sample points
 * - Sample tracking: displays number of signal samples collected
 * - Status: warning (active survey), success (ready/completed) - fixes #737
 *
 * Usage:
 * ```typescript
 * <WifiSurveyCard isWifi={wifiConnected} />
 * ```
 *
 * Dependencies: useSurvey hook, SurveyView component, Card UI components, Icons
 * State: Manages surveys list, selected survey, create dialog state, fetches from API
 */

import type React from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { type Survey, type SurveyType, useSurvey } from '../../hooks/useSurvey';
import { LogComponents, logger } from '../../lib/logger';
import {
  button,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  modal,
  radius,
  spacing,
} from '../../styles/theme';
import { SurveyView } from '../survey/SurveyView';
import { Card, type Status } from '../ui/card';
import { Activity } from '../ui/icons';

interface WifiSurveyCardProps {
  isWifi: boolean;
  /** Current Wifi interface name - fix #572: no hardcoded interface names */
  currentInterface?: string;
}

/**
 * Manages Wifi site surveys for signal mapping with floor plan integration.
 */
export function WiFiSurveyCard({
  isWifi,
  currentInterface = '',
}: WifiSurveyCardProps): React.ReactElement {
  const { t } = useTranslation('cards');
  const {
    surveys,
    loading,
    error,
    createSurvey,
    deleteSurvey,
    startSurvey,
    pauseSurvey,
    completeSurvey,
  } = useSurvey();

  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [selectedSurvey, setSelectedSurvey] = useState<Survey | null>(null);

  const activeSurveys = surveys.filter((s) => s.status === 'in_progress' || s.status === 'paused');

  // Fixes #737: Use "success" for no surveys (ready state) instead of confusing "?" badge
  const getCardStatus = (): Status => {
    if (activeSurveys.length > 0) {
      return 'warning'; // Active work needs attention
    }
    return 'success'; // Ready or completed - system is healthy
  };

  const handleCreateSurvey = async (
    name: string,
    surveyType: SurveyType,
    iface: string,
  ): Promise<void> => {
    try {
      const newSurvey = await createSurvey({
        name,
        surveyType,
        interface: iface,
      });
      setShowCreateDialog(false);
      // Automatically open the survey view for setup
      setSelectedSurvey(newSurvey);
    } catch (err) {
      logger.error(LogComponents.Survey, 'Failed to create survey', err);
    }
  };

  const handleDelete = async (id: string): Promise<void> => {
    if (confirm(t('survey.confirmDelete'))) {
      await deleteSurvey(id);
    }
  };

  const getSurveyTypeLabel = (type: SurveyType): string => {
    switch (type) {
      case 'passive':
        return t('survey.typePassiveLabel');
      case 'active':
        return t('survey.typeActiveLabel');
      case 'throughput':
        return t('survey.typeThroughputLabel');
      default:
        return type;
    }
  };

  const getStatusLabel = (status: string): string => {
    switch (status) {
      case 'in_progress':
        return t('survey.inProgress');
      case 'paused':
        return t('survey.paused');
      case 'completed':
        return t('survey.completed');
      case 'created':
        return t('survey.created');
      default:
        return status.replace('_', ' ').replace(/\b\w/g, (l) => l.toUpperCase());
    }
  };

  // Helper function to render survey content based on loading state and surveys
  const renderSurveyContent = (): React.ReactElement => {
    if (loading && surveys.length === 0) {
      return (
        <div class={cn('text-center', spacing.pad.lg, 'text-text-muted body-small')}>
          {t('survey.loading')}
        </div>
      );
    }
    if (surveys.length === 0) {
      return (
        <div class={cn('text-center', spacing.pad.lg, 'text-text-muted')}>
          <p class={cn('body-small', spacing.margin.bottom.inline)}>{t('survey.noSurveys')}</p>
          <button
            type="button"
            onClick={(): void => setShowCreateDialog(true)}
            class="body-small text-brand-primary hover:underline"
          >
            {t('survey.createFirst')}
          </button>
        </div>
      );
    }
    return (
      <div class="stack-sm">
        {surveys.slice(0, 3).map((survey) => (
          <div
            key={survey.id}
            class={cn(
              'border border-surface-border',
              radius.md,
              'pad-sm hover:bg-surface-hover transition-colors',
            )}
          >
            <div class={layout.flex.between}>
              <button
                type="button"
                class="flex-1 min-w-0 text-left cursor-pointer bg-transparent border-none p-0"
                onClick={(): void => setSelectedSurvey(survey)}
              >
                <div class={layout.inline.default}>
                  <h4 class="font-medium body-small truncate">{survey.name}</h4>
                  <span class="caption text-text-muted">{getStatusLabel(survey.status)}</span>
                </div>
                <div
                  class={cn(
                    layout.inline.comfortable,
                    spacing.margin.top.inline,
                    'caption text-text-muted',
                  )}
                >
                  <span>{getSurveyTypeLabel(survey.surveyType)}</span>
                  <span>
                    {survey.samples?.length ?? 0} {t('survey.samples').toLowerCase()}
                  </span>
                </div>
              </button>
              <div class={cn(layout.inline.tight, spacing.margin.left.inline)}>
                {survey.status === 'created' ? (
                  <button
                    type="button"
                    onClick={(): void => {
                      startSurvey(survey.id).catch(() => undefined);
                    }}
                    class={cn(
                      button.size.xs,
                      'caption border border-surface-border',
                      radius.md,
                      'hover:bg-surface-hover',
                    )}
                    title={t('survey.start')}
                  >
                    ▶
                  </button>
                ) : null}
                {survey.status === 'in_progress' ? (
                  <button
                    type="button"
                    onClick={(): void => {
                      pauseSurvey(survey.id).catch(() => undefined);
                    }}
                    class={cn(
                      button.size.xs,
                      'caption border border-surface-border',
                      radius.md,
                      'hover:bg-surface-hover',
                    )}
                    title={t('survey.pause')}
                  >
                    ⏸
                  </button>
                ) : null}
                {survey.status === 'paused' ? (
                  <>
                    <button
                      type="button"
                      onClick={(): void => {
                        startSurvey(survey.id).catch(() => undefined);
                      }}
                      class={cn(
                        button.size.xs,
                        'caption border border-surface-border',
                        radius.md,
                        'hover:bg-surface-hover',
                      )}
                      title={t('survey.resume')}
                    >
                      ▶
                    </button>
                    <button
                      type="button"
                      onClick={(): void => {
                        completeSurvey(survey.id).catch(() => undefined);
                      }}
                      class={cn(
                        button.size.xs,
                        'caption border border-surface-border',
                        radius.md,
                        'hover:bg-surface-hover',
                      )}
                      title={t('survey.complete')}
                    >
                      ✓
                    </button>
                  </>
                ) : null}
                <button
                  type="button"
                  onClick={(): void => {
                    handleDelete(survey.id).catch(() => undefined);
                  }}
                  class={cn(
                    button.size.xs,
                    'caption border border-surface-border',
                    radius.md,
                    'hover:bg-status-error/10 text-status-error',
                  )}
                  title={t('survey.delete')}
                >
                  ×
                </button>
              </div>
            </div>
          </div>
        ))}
        {surveys.length > 3 ? (
          <div class={cn('text-center caption text-text-muted', spacing.padding.top.tight)}>
            {t('survey.more', { count: surveys.length - 3 })}
          </div>
        ) : null}
      </div>
    );
  };

  return (
    <>
      <Card
        title={t('survey.title')}
        status={getCardStatus()}
        icon={<Activity class={iconTokens.size.md} />}
        headerAction={
          <button
            type="button"
            onClick={(e: React.MouseEvent): void => {
              e.stopPropagation();
              setShowCreateDialog(true);
            }}
            class="caption font-medium text-brand-primary hover:underline"
          >
            {t('survey.new')}
          </button>
        }
      >
        {isWifi ? null : (
          <div
            class={cn(
              'bg-status-warning/10 border border-status-warning/20 text-status-warning',
              spacing.pad.sm,
              radius.md,
              'body-small',
              spacing.margin.bottom.heading,
            )}
          >
            {t('survey.wifiRequired')}
          </div>
        )}

        {error ? (
          <div
            class={cn(
              'bg-status-error/10 border border-status-error/20 text-status-error',
              spacing.pad.sm,
              radius.md,
              'body-small',
              spacing.margin.bottom.heading,
            )}
          >
            {error}
          </div>
        ) : null}

        {renderSurveyContent()}
      </Card>

      {showCreateDialog ? (
        <CreateSurveyDialog
          onClose={(): void => setShowCreateDialog(false)}
          onCreate={handleCreateSurvey}
          t={t}
          currentInterface={currentInterface}
        />
      ) : null}

      {selectedSurvey ? (
        <SurveyView
          survey={selectedSurvey}
          onClose={(): void => setSelectedSurvey(null)}
          onUpdate={(): void => {
            // Refresh surveys list when survey is updated
            // The useSurvey hook will automatically refresh
          }}
        />
      ) : null}
    </>
  );
}

interface CreateSurveyDialogProps {
  onClose: () => void;
  onCreate: (name: string, type: SurveyType, iface: string) => void;
  t: ReturnType<typeof useTranslation<'cards'>>['t'];
  /** Current Wifi interface name - fix #572: no hardcoded interface names */
  currentInterface?: string;
}

function CreateSurveyDialog({
  onClose,
  onCreate,
  t,
  currentInterface = '',
}: CreateSurveyDialogProps): React.ReactElement {
  const [name, setName] = useState('');
  const [surveyType, setSurveyType] = useState<SurveyType>('passive');

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    if (name.trim()) {
      // Fix #572: Use interface from props instead of hardcoded "wlan0"
      onCreate(name.trim(), surveyType, currentInterface);
    }
  };

  return (
    <div class={modal.overlay}>
      <div
        class={cn(
          'bg-surface-raised',
          radius.md,
          spacing.pad.lg,
          'max-w-md w-full',
          spacing.pad.default,
        )}
      >
        <h2 class={cn('heading-2', spacing.margin.bottom.content)}>
          {t('survey.createNewSurvey')}
        </h2>
        <form onSubmit={handleSubmit}>
          <div class="stack">
            <div>
              <label for="survey-name" class={cn('label block', spacing.margin.bottom.tight)}>
                {t('survey.surveyName')}
              </label>
              <input
                id="survey-name"
                type="text"
                value={name}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setName(e.target.value)
                }
                class={cn(inputTokens.base, inputTokens.state.default, inputTokens.size.md)}
                placeholder={t('survey.namePlaceholder')}
                required={true}
              />
            </div>
            <div>
              <label class={cn('label block', spacing.margin.bottom.tight)} for="survey-type">
                {t('survey.surveyType')}
              </label>
              <select
                id="survey-type"
                value={surveyType}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setSurveyType(e.target.value as SurveyType)
                }
                class={cn(inputTokens.base, inputTokens.state.default, inputTokens.size.md)}
              >
                <option value="passive">{t('survey.typePassive')}</option>
                <option value="active">{t('survey.typeActive')}</option>
                <option value="throughput">{t('survey.typeThroughput')}</option>
              </select>
            </div>
          </div>
          <div class={cn(layout.inline.default, spacing.margin.top.section)}>
            <button
              type="button"
              onClick={onClose}
              class={cn(
                'flex-1',
                button.size.md,
                'border border-surface-border',
                radius.md,
                'hover:bg-surface-hover',
              )}
            >
              {t('survey.cancel')}
            </button>
            <button
              type="submit"
              class={cn(
                'flex-1',
                button.size.md,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'hover:bg-brand-primary/90',
              )}
            >
              {t('survey.create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
