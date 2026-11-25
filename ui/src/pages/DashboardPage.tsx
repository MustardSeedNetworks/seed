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
import { ChevronDown, ChevronUp, LayoutDashboard, Plus, X } from '../components/ui/Icons';
import { useDashboardLayout } from '../hooks/useDashboardLayout';
import { Alert } from '../shared/Alert';
import { Button } from '../shared/Button';
import { Card } from '../shared/Card';
import { EmptyState } from '../shared/EmptyState';
import { FormField } from '../shared/FormField';
import { Select } from '../shared/Select';
import { Skeleton } from '../shared/Skeleton';
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
    <Card
      data-testid="dashboard-editor"
      title={t('dashboard.editorTitle')}
      className="flex flex-col gap-4"
    >
      {draft.length === 0 ? (
        <p className="text-sm text-text-muted">{t('dashboard.editorEmpty')}</p>
      ) : (
        <ol className="flex flex-col divide-y divide-hairline">
          {draft.map((id, index) => {
            const label = widgetLabel(t, id);
            return (
              <li
                key={id}
                data-testid={`dashboard-row-${id}`}
                className="flex items-center gap-2 py-2"
              >
                <span className="min-w-0 flex-1 text-sm text-text-primary">{label}</span>
                <Button
                  variant="ghost"
                  iconOnly
                  icon={<ChevronUp className="h-4 w-4" />}
                  aria-label={t('dashboard.moveUp', { widget: label })}
                  disabled={index === 0}
                  onClick={(): void => onChange(moveWidget(draft, index, -1))}
                  data-testid={`dashboard-up-${id}`}
                />
                <Button
                  variant="ghost"
                  iconOnly
                  icon={<ChevronDown className="h-4 w-4" />}
                  aria-label={t('dashboard.moveDown', { widget: label })}
                  disabled={index === draft.length - 1}
                  onClick={(): void => onChange(moveWidget(draft, index, 1))}
                  data-testid={`dashboard-down-${id}`}
                />
                <Button
                  variant="ghost"
                  iconOnly
                  icon={<X className="h-4 w-4" />}
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
        <div className="flex flex-wrap items-end gap-2">
          <FormField label={t('dashboard.addLabel')} className="w-56">
            {(control) => (
              <Select
                {...control}
                value={toAdd}
                onValueChange={(value): void => setAdding(isDashboardWidget(value) ? value : null)}
                options={available.map((id) => ({ value: id, label: widgetLabel(t, id) }))}
                data-testid="dashboard-add-choice"
              />
            )}
          </FormField>
          <Button
            variant="secondary"
            icon={<Plus className="h-4 w-4" />}
            onClick={(): void => onChange([...draft, toAdd])}
            data-testid="dashboard-add"
          >
            {t('dashboard.add')}
          </Button>
        </div>
      )}
      {saveError === null ? null : (
        <div data-testid="dashboard-save-error">
          <Alert tone="error">{t('dashboard.saveFailed', { reason: saveError })}</Alert>
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        <Button variant="primary" loading={saving} onClick={onSave} data-testid="dashboard-save">
          {t('dashboard.save')}
        </Button>
        <Button variant="secondary" onClick={onCancel} data-testid="dashboard-cancel">
          {t('dashboard.cancel')}
        </Button>
        <Button
          variant="ghost"
          onClick={(): void => onChange([...DEFAULT_DASHBOARD])}
          data-testid="dashboard-reset"
        >
          {t('dashboard.reset')}
        </Button>
      </div>
    </Card>
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
      <div data-testid="dashboard-loading" role="status" className="flex flex-col gap-2">
        <span className="sr-only">{t('common:status.loading')}</span>
        <Skeleton className="h-32 w-full" />
      </div>
    ) : (
      <div data-testid="dashboard-error">
        <Alert tone="error">{t('dashboard.loadFailed', { reason: error })}</Alert>
      </div>
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
            variant="secondary"
            icon={<LayoutDashboard className="h-4 w-4" />}
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
        <div data-testid="dashboard-empty">
          <EmptyState
            icon={<LayoutDashboard className="h-6 w-6" aria-hidden="true" />}
            title={t('dashboard.emptyTitle')}
            hint={t('dashboard.emptyHint')}
          />
        </div>
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
