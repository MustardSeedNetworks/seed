/**
 * DashboardPage (UI-SEED-22) — the signed-in user's own page of cards. They
 * choose which cards it shows and in what order, and the layout follows them
 * to any browser they sign in from. Until they save one, it shows a default.
 *
 * Customizing edits a draft in a list beside the grid, which previews it;
 * nothing is stored until Save. Buttons move a widget rather than dragging
 * it, so the editor works the same from a keyboard and on a phone.
 */
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, IconButton } from '../components/ui/Button';
import { ChevronDown, ChevronUp, LayoutDashboard, Plus, X } from '../components/ui/Icons';
import { iconSizes } from '../constants/sizes';
import { useDashboardLayout } from '../hooks/useDashboardLayout';
import { cn, input } from '../styles/theme';
import { CardGrid } from '../ui/CardGrid';
import { DashboardWidget, widgetLabel } from './dashboard/DashboardWidget';
import {
  availableWidgets,
  type DashboardWidgetId,
  DEFAULT_DASHBOARD,
  isDashboardWidget,
  moveWidget,
  resolveLayout,
} from './dashboard/layout';

const selectClass = cn(input.base, input.state.default, input.size.sm, 'body-small');

interface EditorProps {
  draft: DashboardWidgetId[];
  saving: boolean;
  saveError: string | null;
  onChange: (draft: DashboardWidgetId[]) => void;
  onSave: () => void;
  onCancel: () => void;
}

function DashboardEditor({
  draft,
  saving,
  saveError,
  onChange,
  onSave,
  onCancel,
}: EditorProps): JSX.Element {
  const { t } = useTranslation('pages');
  const available = availableWidgets(draft);
  const [adding, setAdding] = useState<DashboardWidgetId | null>(null);
  // The choice falls back to the first unused widget once the chosen one is added.
  const toAdd = adding !== null && available.includes(adding) ? adding : available[0];

  return (
    <section
      data-testid="dashboard-editor"
      className="flex flex-col gap-default rounded-lg border border-surface-border bg-surface-raised pad"
    >
      <h2 className="text-sm font-semibold text-text-primary">{t('dashboard.editorTitle')}</h2>
      {draft.length === 0 ? (
        <p className="text-sm text-text-muted">{t('dashboard.editorEmpty')}</p>
      ) : (
        <ol className="flex flex-col divide-y divide-surface-border">
          {draft.map((id, index) => {
            const label = widgetLabel(t, id);
            return (
              <li
                key={id}
                data-testid={`dashboard-row-${id}`}
                className="flex items-center gap-compact py-compact"
              >
                <span className="min-w-0 flex-1 text-sm text-text-primary">{label}</span>
                <IconButton
                  icon={<ChevronUp className={iconSizes.md} />}
                  aria-label={t('dashboard.moveUp', { widget: label })}
                  disabled={index === 0}
                  onClick={(): void => onChange(moveWidget(draft, index, -1))}
                  data-testid={`dashboard-up-${id}`}
                />
                <IconButton
                  icon={<ChevronDown className={iconSizes.md} />}
                  aria-label={t('dashboard.moveDown', { widget: label })}
                  disabled={index === draft.length - 1}
                  onClick={(): void => onChange(moveWidget(draft, index, 1))}
                  data-testid={`dashboard-down-${id}`}
                />
                <IconButton
                  icon={<X className={iconSizes.md} />}
                  aria-label={t('dashboard.remove', { widget: label })}
                  onClick={(): void => onChange(draft.filter((w) => w !== id))}
                  data-testid={`dashboard-remove-${id}`}
                />
              </li>
            );
          })}
        </ol>
      )}
      {toAdd === undefined ? null : (
        <div className="flex flex-wrap items-end gap-compact">
          <label className="flex flex-col gap-tight text-xs text-text-muted">
            {t('dashboard.addLabel')}
            <select
              value={toAdd}
              onChange={({ target: { value } }): void =>
                setAdding(isDashboardWidget(value) ? value : null)
              }
              data-testid="dashboard-add-choice"
              className={selectClass}
            >
              {available.map((id) => (
                <option key={id} value={id}>
                  {widgetLabel(t, id)}
                </option>
              ))}
            </select>
          </label>
          <Button
            size="sm"
            variant="outline"
            leftIcon={<Plus className={iconSizes.md} />}
            onClick={(): void => onChange([...draft, toAdd])}
            data-testid="dashboard-add"
          >
            {t('dashboard.add')}
          </Button>
        </div>
      )}
      {saveError === null ? null : (
        <p
          role="alert"
          data-testid="dashboard-save-error"
          className="text-sm text-status-error-strong"
        >
          {t('dashboard.saveFailed', { reason: saveError })}
        </p>
      )}
      <div className="flex flex-wrap gap-compact">
        <Button size="sm" loading={saving} onClick={onSave} data-testid="dashboard-save">
          {t('dashboard.save')}
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel} data-testid="dashboard-cancel">
          {t('dashboard.cancel')}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={(): void => onChange([...DEFAULT_DASHBOARD])}
          data-testid="dashboard-reset"
        >
          {t('dashboard.reset')}
        </Button>
      </div>
    </section>
  );
}

export function DashboardPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { layout, error, save } = useDashboardLayout();
  // Non-null while customizing: the unsaved layout the grid previews.
  const [draft, setDraft] = useState<DashboardWidgetId[] | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  if (layout === null) {
    return error === null ? (
      <p className="body-small" data-testid="dashboard-loading">
        {t('common:status.loading')}
      </p>
    ) : (
      <p role="alert" data-testid="dashboard-error" className="text-sm text-status-error-strong">
        {t('dashboard.loadFailed', { reason: error })}
      </p>
    );
  }

  const shown = draft ?? resolveLayout(layout.widgets, layout.saved);

  const saveDraft = async (widgets: DashboardWidgetId[]): Promise<void> => {
    setSaving(true);
    const refusal = await save(widgets);
    setSaving(false);
    setSaveError(refusal);
    if (refusal === null) {
      setDraft(null);
    }
  };

  return (
    <>
      {draft === null ? (
        <div className="flex justify-end">
          <Button
            size="sm"
            variant="outline"
            leftIcon={<LayoutDashboard className={iconSizes.md} />}
            onClick={(): void => setDraft(shown)}
            data-testid="dashboard-customize"
          >
            {t('dashboard.customize')}
          </Button>
        </div>
      ) : (
        <DashboardEditor
          draft={draft}
          saving={saving}
          saveError={saveError}
          onChange={setDraft}
          onSave={(): void => {
            saveDraft(draft).catch(() => undefined);
          }}
          onCancel={(): void => {
            setDraft(null);
            setSaveError(null);
          }}
        />
      )}
      {shown.length === 0 ? (
        <p
          data-testid="dashboard-empty"
          className="rounded-lg border border-surface-border bg-surface-raised pad text-sm text-text-secondary"
        >
          {t('dashboard.empty')}
        </p>
      ) : (
        <CardGrid>
          {shown.map((id) => (
            // display: contents keeps each card a grid item while the wrapper
            // names the widget, so its order is testable.
            <div key={id} data-testid={`dashboard-widget-${id}`} className="contents">
              <DashboardWidget id={id} />
            </div>
          ))}
        </CardGrid>
      )}
    </>
  );
}
