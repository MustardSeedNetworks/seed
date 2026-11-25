/**
 * GroupForm — create/edit dialog for a device group.
 *
 * Members are picked from the caller's polling targets: a group is a named set
 * of them, and PUT replaces the whole list, so the form always sends every
 * member it shows ticked.
 */

import { type FormEvent, type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useTableLabels } from '../../hooks/useTableLabels';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Modal } from '../../shared/Modal';
import { SearchInput } from '../../shared/SearchInput';
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
  const labels = useTableLabels<PollingTargetResponse>({
    loading: t('common:status.loading'),
    selectRow: (target) => t('deviceGroups.selectMember', { name: target.name }),
  });

  const needle = query.trim().toLowerCase();
  const shown = targets.filter(
    (target) =>
      !needle ||
      target.name.toLowerCase().includes(needle) ||
      target.ipAddress.toLowerCase().includes(needle),
  );
  const columns: DataTableColumn<PollingTargetResponse>[] = [
    {
      key: 'name',
      header: t('deviceGroups.columnName'),
      cell: (target) => <span className="text-text-primary">{target.name}</span>,
      sortValue: (target) => target.name,
    },
    {
      key: 'address',
      header: t('deviceGroups.columnAddress'),
      cell: (target) => <span className="figure">{target.ipAddress}</span>,
      sortValue: (target) => target.ipAddress,
    },
  ];

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
      open
      onClose={onCancel}
      closeLabel={t('common:buttons.close')}
      size="md"
      footer={
        <>
          <Button onClick={onCancel}>{t('common:buttons.cancel')}</Button>
          <Button
            type="submit"
            form="device-group-form"
            variant="primary"
            loading={submitting}
            data-testid="group-save"
          >
            {submitting
              ? t('common:common.saving')
              : mode === 'create'
                ? t('deviceGroups.addGroup')
                : t('deviceGroups.saveChanges')}
          </Button>
        </>
      }
      title={
        mode === 'create' ? t('deviceGroups.formCreateTitle') : t('deviceGroups.formEditTitle')
      }
      // A stray click outside would throw away a half-picked member list.
      closeOnBackdrop={false}
    >
      <form
        id="device-group-form"
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div data-testid="group-form-error" className="mb-4">
            <Alert tone="error">{formError}</Alert>
          </div>
        ) : null}

        <div className="space-y-4">
          <FormField label={t('deviceGroups.fieldName')} required>
            {(control) => (
              <Input
                {...control}
                value={form.name}
                onChange={(e): void => setForm((prev) => ({ ...prev, name: e.target.value }))}
                required
                data-testid="group-name"
              />
            )}
          </FormField>
          <FormField label={t('deviceGroups.fieldDescription')}>
            {(control) => (
              <Input
                {...control}
                value={form.description ?? ''}
                onChange={(e): void =>
                  setForm((prev) => ({ ...prev, description: e.target.value }))
                }
                data-testid="group-description"
              />
            )}
          </FormField>
          <fieldset className="space-y-2">
            <legend className="mb-1.5 text-sm font-medium text-text-secondary">
              {t('deviceGroups.fieldMembers', { count: form.members.length })}
            </legend>
            {targets.length === 0 ? (
              <div data-testid="group-no-targets">
                <EmptyState title={t('deviceGroups.noTargets')} />
              </div>
            ) : (
              <>
                <SearchInput
                  value={query}
                  onValueChange={setQuery}
                  placeholder={t('deviceGroups.memberFilterPlaceholder', { count: targets.length })}
                  label={t('deviceGroups.memberFilterLabel')}
                  clearLabel={t('common:buttons.clear')}
                  className="max-w-none"
                />
                <DataTable
                  rows={shown}
                  columns={columns}
                  getRowKey={(target) => target.id}
                  caption={t('deviceGroups.fieldMembers', { count: form.members.length })}
                  labels={labels}
                  pageSize={8}
                  empty={<EmptyState title={t('deviceGroups.memberNoMatch')} />}
                  selection={{
                    selectedKeys: new Set(form.members),
                    onChange: (keys): void => setForm((prev) => ({ ...prev, members: [...keys] })),
                  }}
                />
              </>
            )}
          </fieldset>
        </div>
      </form>
    </Modal>
  );
}

export function emptyGroup(): DeviceGroupRequest {
  return { name: '', description: '', members: [] };
}

/** groupToInput strips the audit columns the server manages. */
export function groupToInput(group: DeviceGroupResponse): DeviceGroupRequest {
  return { name: group.name, description: group.description, members: group.members };
}
