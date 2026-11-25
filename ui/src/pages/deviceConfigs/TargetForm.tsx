/**
 * TargetForm — add or edit a configuration backup target (UI-SEED-45).
 *
 * A backup signs in over SSH with a credential from the vault. An operator
 * adding the first device has no SSH login stored yet, so the form can store
 * one in place rather than sending them to the API.
 */

import { type FormEvent, type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api/client';
import { Modal, ModalFooter } from '../../components/ui/Modal';
import { useDeviceCredentials } from '../../hooks/useDeviceCredentials';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Select } from '../../shared/Select';
import type {
  DeviceConfigPlatform,
  DeviceConfigTarget,
  DeviceConfigTargetInput,
} from '../../types/deviceConfigs';
import type { Credentials } from '../../types/generated/device-credential-list-response';
import type { DeviceCredentialRequest } from '../../types/generated/device-credential-request';

const PLATFORMS: readonly DeviceConfigPlatform[] = [
  'cisco-ios',
  'cisco-nxos',
  'arista-eos',
  'juniper-junos',
  'netconf',
];

/** The select's value for "store a new SSH login". */
const NEW_LOGIN = 'new';

export interface TargetFormProps {
  /** The target being edited; absent when adding one. */
  target?: DeviceConfigTarget;
  onSubmit: (input: DeviceConfigTargetInput) => Promise<void>;
  onCancel: () => void;
}

export function TargetForm({ target, onSubmit, onCancel }: TargetFormProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const vault = useDeviceCredentials();
  const [name, setName] = useState(target?.name ?? '');
  const [host, setHost] = useState(target?.host ?? '');
  const [port, setPort] = useState(target ? String(target.port) : '');
  const [platform, setPlatform] = useState<DeviceConfigPlatform>(target?.platform ?? 'cisco-ios');
  const [picked, setPicked] = useState(target?.credentialsId ?? '');
  const [login, setLogin] = useState({ name: '', user: '', password: '' });
  // A login stored by this form, which the vault list read on open lacks.
  const [stored, setStored] = useState<Credentials | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // The service refuses any other kind (checkSSHCredential).
  const sshLogins = [
    ...vault.credentials.filter((c) => c.kind === 'ssh'),
    ...(stored ? [stored] : []),
  ];
  // With no SSH login stored, storing one is the only way forward.
  const credential = picked || (!vault.loading && sshLogins.length === 0 ? NEW_LOGIN : '');

  async function credentialId(): Promise<string> {
    if (credential !== NEW_LOGIN) {
      return credential;
    }
    const body: DeviceCredentialRequest = {
      name: login.name.trim(),
      sshUser: login.user.trim(),
      sshPassword: login.password,
    };
    const saved = await api.post<Credentials>('/api/v1/device-credentials', body);
    // A retry after the device is refused must not store the login twice.
    setStored(saved);
    setPicked(saved.id);
    return saved.id;
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    if (!credential) {
      setFormError(t('deviceConfigs.form.credentialRequired'));
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit({
        name: name.trim(),
        host: host.trim(),
        port: port ? Number(port) : undefined,
        platform,
        credentialsId: await credentialId(),
      });
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('deviceConfigs.form.saveFailed'));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      isOpen
      onClose={onCancel}
      title={target ? t('deviceConfigs.form.editTitle') : t('deviceConfigs.form.addTitle')}
      // A stray click outside would throw away a half-typed device.
      closeOnBackdropClick={false}
    >
      <form
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
      >
        {formError ? (
          <div data-testid="config-target-form-error" className="mb-4">
            <Alert tone="error">{formError}</Alert>
          </div>
        ) : null}

        <div className="space-y-4">
          <FormField label={t('deviceConfigs.form.name')} required>
            {(control) => (
              <Input
                {...control}
                value={name}
                onChange={(e): void => setName(e.target.value)}
                required
                data-testid="config-target-name"
              />
            )}
          </FormField>
          <FormField label={t('deviceConfigs.form.host')} required>
            {(control) => (
              <Input
                {...control}
                value={host}
                onChange={(e): void => setHost(e.target.value)}
                required
                placeholder="10.0.0.1"
                data-testid="config-target-host"
              />
            )}
          </FormField>
          <FormField label={t('deviceConfigs.form.platform')}>
            {(control) => (
              <Select
                {...control}
                value={platform}
                onValueChange={(value): void => setPlatform(value as DeviceConfigPlatform)}
                options={PLATFORMS.map((p) => ({
                  value: p,
                  label: t(`deviceConfigs.platform.${p}`),
                }))}
                data-testid="config-target-platform"
              />
            )}
          </FormField>
          <FormField label={t('deviceConfigs.form.port')}>
            {(control) => (
              <Input
                {...control}
                type="number"
                min={1}
                max={65535}
                value={port}
                onChange={(e): void => setPort(e.target.value)}
                placeholder={platform === 'netconf' ? '830' : '22'}
                data-testid="config-target-port"
              />
            )}
          </FormField>
          <FormField
            label={t('deviceConfigs.form.credential')}
            {...(vault.error
              ? { error: t('deviceConfigs.form.credentialsLoadFailed', { reason: vault.error }) }
              : {})}
          >
            {(control) => (
              <Select
                {...control}
                value={credential}
                onValueChange={setPicked}
                disabled={vault.loading}
                placeholder={
                  vault.loading
                    ? t('common:status.loading')
                    : t('deviceConfigs.form.credentialPick')
                }
                options={[
                  ...sshLogins.map((c) => ({
                    value: c.id,
                    label: c.sshUser ? `${c.name} (${c.sshUser})` : c.name,
                  })),
                  { value: NEW_LOGIN, label: t('deviceConfigs.form.credentialNew') },
                ]}
                data-testid="config-target-credential"
              />
            )}
          </FormField>
          {credential === NEW_LOGIN ? (
            <fieldset data-testid="config-target-new-login" className="space-y-3">
              <legend className="mb-1.5 text-xs text-text-muted">
                {t('deviceConfigs.form.newLoginHint')}
              </legend>
              <FormField label={t('deviceConfigs.form.loginName')} required>
                {(control) => (
                  <Input
                    {...control}
                    value={login.name}
                    onChange={(e): void => setLogin({ ...login, name: e.target.value })}
                    required
                    data-testid="config-target-login-name"
                  />
                )}
              </FormField>
              <FormField label={t('deviceConfigs.form.loginUser')} required>
                {(control) => (
                  <Input
                    {...control}
                    value={login.user}
                    onChange={(e): void => setLogin({ ...login, user: e.target.value })}
                    required
                    autoComplete="off"
                    data-testid="config-target-login-user"
                  />
                )}
              </FormField>
              <FormField label={t('deviceConfigs.form.loginPassword')} required>
                {(control) => (
                  <Input
                    {...control}
                    type="password"
                    value={login.password}
                    onChange={(e): void => setLogin({ ...login, password: e.target.value })}
                    required
                    autoComplete="new-password"
                    data-testid="config-target-login-password"
                  />
                )}
              </FormField>
            </fieldset>
          ) : null}
        </div>

        <ModalFooter>
          <Button onClick={onCancel}>{t('common:buttons.cancel')}</Button>
          <Button
            type="submit"
            variant="primary"
            loading={submitting}
            disabled={vault.loading}
            data-testid="config-target-save"
          >
            {submitting
              ? t('common:common.saving')
              : target
                ? t('deviceConfigs.form.save')
                : t('deviceConfigs.form.add')}
          </Button>
        </ModalFooter>
      </form>
    </Modal>
  );
}
