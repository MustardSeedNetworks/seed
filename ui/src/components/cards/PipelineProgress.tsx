import { Check, Circle, FileText, Loader2, Scan, Search, Shield, X } from 'lucide-react';
/**
 * Pipeline Progress Component
 *
 * Displays multi-phase discovery pipeline progress with:
 * - Phase steps with icons and status indicators
 * - Progress bar for current phase
 * - Current target being processed
 * - Elapsed time and cancel button
 */
import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PipelineStatus } from '../../hooks/usePipelineStatus';
import { button, cn, icon as iconTokens, radius, spacing } from '../../styles/theme';

interface PipelineProgressProps {
  status: PipelineStatus;
  onCancel?: () => void;
}

// Phase metadata for display
const PHASE_CONFIG: Record<
  string,
  { icon: React.ComponentType<{ className?: string }>; labelKey: string }
> = {
  enumeration: { icon: Search, labelKey: 'pipeline.phases.enumeration' },
  resolution: { icon: FileText, labelKey: 'pipeline.phases.resolution' },
  scanning: { icon: Scan, labelKey: 'pipeline.phases.scanning' },
  assessment: { icon: Shield, labelKey: 'pipeline.phases.assessment' },
};

// Map phase name to display name (fallback for missing i18n)
const PHASE_DISPLAY_NAMES: Record<string, string> = {
  enumeration: 'Enumeration',
  resolution: 'Resolution',
  scanning: 'Scanning',
  assessment: 'Assessment',
};

function formatDuration(ms: number): string {
  // Fixes #957: Handle negative values from clock skew
  const safeMs = Math.max(0, ms);
  if (safeMs < 1000) {
    return `${safeMs}ms`;
  }
  const seconds = Math.floor(safeMs / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
}

// Fixes #939: Simple hash for better key uniqueness instead of truncation
function simpleHash(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash &= hash; // Convert to 32bit integer
  }
  return hash.toString(36);
}

/** Helper to render phase icon based on state */
function renderPhaseIcon(
  isComplete: boolean,
  isCurrent: boolean,
  isRunning: boolean,
  Icon: React.ComponentType<{ class?: string }>,
): React.ReactElement {
  if (isComplete) {
    return <Check class={iconTokens.size.sm} />;
  }
  if (isCurrent && isRunning) {
    return <Loader2 class={cn(iconTokens.size.sm, 'animate-spin')} />;
  }
  return <Icon class={iconTokens.size.sm} />;
}

export const PipelineProgress: React.NamedExoticComponent<PipelineProgressProps> = memo(
  function pipelineProgress({ status, onCancel }: PipelineProgressProps): React.ReactElement {
    const { t } = useTranslation('cards');

    const isRunning =
      status.state === 'enumerating' ||
      status.state === 'resolving' ||
      status.state === 'scanning' ||
      status.state === 'assessing';

    // Fixes #913: Handle case where currentPhase is not in enabledPhases (returns -1)
    // Default to 0 to show first phase as current instead of all phases as complete
    const rawPhaseIndex = status.enabledPhases.indexOf(status.currentPhase);
    const currentPhaseIndex = rawPhaseIndex >= 0 ? rawPhaseIndex : 0;

    return (
      <div class={cn('space-y-3', spacing.pad.sm)}>
        {/* Current phase header */}
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            {isRunning ? (
              <Loader2 class={cn(iconTokens.size.sm, 'text-brand-primary animate-spin')} />
            ) : null}
            <span class="body-small font-medium text-text-primary">
              {t('pipeline.phaseProgress', {
                current: status.phaseNumber,
                total: status.totalPhases,
                defaultValue: `Phase ${status.phaseNumber} of ${status.totalPhases}`,
              })}
              : {PHASE_DISPLAY_NAMES[status.currentPhase] || status.currentPhase}
            </span>
          </div>
          {isRunning && onCancel ? (
            <button
              type="button"
              onClick={onCancel}
              class={cn(
                button.base,
                button.size.sm,
                button.variant.secondary,
                'flex items-center gap-1',
              )}
              aria-label={t('pipeline.cancel', { defaultValue: 'Cancel' })}
            >
              <X class={iconTokens.size.xs} />
              <span class="hidden sm:inline">
                {t('pipeline.cancel', { defaultValue: 'Cancel' })}
              </span>
            </button>
          ) : null}
        </div>

        {/* Progress bar */}
        <div class="space-y-1">
          <div class={cn('h-2 bg-surface-sunken overflow-hidden', radius.default)}>
            <div
              class={cn('h-full bg-brand-primary transition-all duration-300', radius.default)}
              style={{ width: `${Math.min(status.percentComplete, 100)}%` }}
            />
          </div>
          <div class="flex justify-between caption text-text-muted">
            <span>
              {status.processedCount} / {status.totalCount}{' '}
              {t('pipeline.devices', { defaultValue: 'devices' })}
            </span>
            <span>{Math.round(status.percentComplete)}%</span>
          </div>
        </div>

        {/* Current target and timing */}
        <div class="flex flex-wrap items-center gap-x-4 gap-y-1 caption text-text-muted">
          {status.currentTarget ? (
            <span>
              {t('pipeline.scanning', { defaultValue: 'Scanning' })}:{' '}
              <span class="font-mono text-text-secondary">{status.currentTarget}</span>
            </span>
          ) : null}
          <span>
            {t('pipeline.elapsed', { defaultValue: 'Elapsed' })}: {formatDuration(status.elapsedMs)}
          </span>
          {status.estimatedRemainMs > 0 && (
            <span>
              {t('pipeline.remaining', { defaultValue: 'Remaining' })}:{' '}
              {formatDuration(status.estimatedRemainMs)}
            </span>
          )}
        </div>

        {/* Phase stepper */}
        <div class="flex items-center justify-between gap-1 pt-2 border-t border-surface-border">
          {status.enabledPhases.map((phase, index) => {
            const config = PHASE_CONFIG[phase];
            const ICON = config?.icon || Circle;
            const isComplete = index < currentPhaseIndex;
            const isCurrent = index === currentPhaseIndex;
            const isPending = index > currentPhaseIndex;

            // Get duration if phase is complete
            const duration = status.phaseDurations[phase];

            return (
              <div
                key={phase}
                class={cn('flex flex-col items-center gap-1 flex-1', isPending && 'opacity-50')}
              >
                {/* Icon with status indicator */}
                <div
                  class={cn(
                    'flex items-center justify-center w-8 h-8',
                    radius.full,
                    isComplete && 'bg-status-success text-text-inverse',
                    isCurrent && 'bg-brand-primary text-text-inverse',
                    isPending && 'bg-surface-sunken text-text-muted',
                  )}
                >
                  {renderPhaseIcon(isComplete, isCurrent, isRunning, ICON)}
                </div>

                {/* Phase name */}
                <span
                  class={cn(
                    'caption text-center',
                    isCurrent ? 'text-text-primary font-medium' : 'text-text-muted',
                  )}
                >
                  {PHASE_DISPLAY_NAMES[phase] || phase}
                </span>

                {/* Duration if complete */}
                {duration !== undefined && (
                  <span class="caption text-text-muted">{formatDuration(duration)}</span>
                )}
              </div>
            );
          })}
        </div>

        {/* Errors if any */}
        {status.errors.length > 0 && (
          <div class={cn('p-2 bg-status-error/10 border border-status-error/30', radius.default)}>
            <span class="caption text-status-error font-medium">
              {t('pipeline.errors', { defaultValue: 'Errors' })}:
            </span>
            <ul class="mt-1 space-y-0.5">
              {/* Fixes #926, #939: Use error content hash for stable keys */}
              {status.errors.map((error, i) => (
                <li key={`${i}-${simpleHash(error)}`} class="caption text-status-error">
                  {error}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    );
  },
);
