/**
 * GroupForm — create/edit dialog for a device group.
 *
 * Members are picked from the caller's polling targets: a group is a named set
 * of them, and PUT replaces the whole list, so the form always sends every
 * member it shows ticked.
 */

import { type FormEvent, type JSX, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, ModalFooter } from '../../components/ui/Modal';
import type { DeviceGroupRequest } from '../../types/generated/device-group-request';
import type { DeviceGroupResponse } from '../../types/generated/device-group-response';
import type { PollingTargetResponse } from '../../types/generated/polling-target-response';

export interface GroupFormProps {
  mode: 'create' | 'edit';
  initial: DeviceGroupRequest;
  targets: PollingTargetResponse[];
  onSubmit: (input: DeviceGroupRequest) => Promise<void>;
  onCancel: () => void;
}

export function GroupForm({
  mode,
  initial,
  targets,
  onSubmit,
  onCancel,
}: GroupFormProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [form, setForm] = useState<DeviceGroupRequest>(initial);
  const [query, setQuery] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const id = useId();

  const members = new Set(form.members);
  const needle = query.trim().toLowerCase();
  const shown = targets.filter(
    (target) =>
      !needle ||
      target.name.toLowerCase().includes(needle) ||
      target.ipAddress.toLowerCase().includes(needle),
  );

  function toggle(targetId: string, on: boolean): void {
    setForm((prev) => ({
      ...prev,
      members: on
        ? [...prev.members, targetId]
        : prev.members.filter((member) => member !== targetId),
    }));
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    if (!form.name.trim()) {
      setFormError(t('deviceGroups.nameRequired'));
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit({ ...form, name: form.name.trim() });
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('deviceGroups.saveFailed'));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      isOpen
      onClose={onCancel}
      title={
        mode === 'create' ? t('deviceGroups.formCreateTitle') : t('deviceGroups.formEditTitle')
      }
      // A stray click outside would throw away a half-picked member list.
      closeOnBackdropClick={false}
    >
      <form
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div
            role="alert"
            data-testid="group-form-error"
            className="mb-content rounded-md border border-status-error/40 bg-status-error/10 pad-xs text-sm text-status-error-strong"
          >
            {formError}
          </div>
        ) : null}

        <div className="stack">
          <div>
            <label htmlFor={`${id}-name`} className={labelClass}>
              {t('deviceGroups.fieldName')}
            </label>
            <input
              id={`${id}-name`}
              type="text"
              value={form.name}
              onChange={(e): void => setForm((prev) => ({ ...prev, name: e.target.value }))}
              required
              data-testid="group-name"
              className={`mt-tight ${inputClass}`}
            />
          </div>
          <div>
            <label htmlFor={`${id}-description`} className={labelClass}>
              {t('deviceGroups.fieldDescription')}
            </label>
            <input
              id={`${id}-description`}
              type="text"
              value={form.description ?? ''}
              onChange={(e): void => setForm((prev) => ({ ...prev, description: e.target.value }))}
              data-testid="group-description"
              className={`mt-tight ${inputClass}`}
            />
          </div>
          <fieldset className="stack-xs">
            <legend className={labelClass}>
              {t('deviceGroups.fieldMembers', { count: form.members.length })}
            </legend>
            {targets.length === 0 ? (
              <p data-testid="group-no-targets" className="body-small">
                {t('deviceGroups.noTargets')}
              </p>
            ) : (
              <>
                <input
                  type="search"
                  value={query}
                  onChange={(e): void => setQuery(e.target.value)}
                  placeholder={t('deviceGroups.memberFilterPlaceholder', { count: targets.length })}
                  aria-label={t('deviceGroups.memberFilterLabel')}
                  className={inputClass}
                />
                <ul className="max-h-64 divide-y divide-surface-border overflow-y-auto rounded-md border border-surface-border">
                  {shown.map((target) => (
                    <li key={target.id}>
                      <label className="flex min-h-[44px] cursor-pointer items-center gap-compact px-cell py-2 text-sm text-text-primary hover:bg-surface-hover">
                        <input
                          type="checkbox"
                          checked={members.has(target.id)}
                          onChange={(e): void => toggle(target.id, e.target.checked)}
                          data-testid={`group-member-${target.id}`}
                        />
                        <span className="min-w-0 flex-1 truncate">{target.name}</span>
                        <span className="figure text-xs text-text-muted">{target.ipAddress}</span>
                      </label>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </fieldset>
        </div>

        <ModalFooter>
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md px-3 py-2 text-sm text-text-muted hover:text-text-primary"
          >
            {t('common:buttons.cancel')}
          </button>
          <button
            type="submit"
            disabled={submitting}
            data-testid="group-save"
            className="rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:control-disabled"
          >
            {submitting
              ? t('common:common.saving')
              : mode === 'create'
                ? t('deviceGroups.addGroup')
                : t('deviceGroups.saveChanges')}
          </button>
        </ModalFooter>
      </form>
    </Modal>
  );
}

const labelClass: string = 'block text-xs font-medium uppercase tracking-wide text-text-muted';

const inputClass: string =
  'w-full rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none';

export function emptyGroup(): DeviceGroupRequest {
  return { name: '', description: '', members: [] };
}

/** groupToInput strips the audit columns the server manages. */
export function groupToInput(group: DeviceGroupResponse): DeviceGroupRequest {
  return { name: group.name, description: group.description, members: group.members };
}
