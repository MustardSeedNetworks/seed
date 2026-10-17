/**
 * TargetForm — add or edit a configuration backup target (UI-SEED-45).
 *
 * A backup signs in over SSH with a credential from the vault. An operator
 * adding the first device has no SSH login stored yet, so the form can store
 * one in place rather than sending them to the API.
 */

import { type FormEvent, type JSX, type ReactNode, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api/client';
import { Modal, ModalFooter } from '../../components/ui/Modal';
import { useDeviceCredentials } from '../../hooks/useDeviceCredentials';
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
  const id = useId();
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
          <div
            role="alert"
            data-testid="config-target-form-error"
            className="mb-content rounded-md border border-status-error/40 bg-status-error/10 pad-xs text-sm text-status-error-strong"
          >
            {formError}
          </div>
        ) : null}

        <div className="stack">
          <Field id={`${id}-name`} label={t('deviceConfigs.form.name')}>
            <input
              id={`${id}-name`}
              value={name}
              onChange={(e): void => setName(e.target.value)}
              required
              data-testid="config-target-name"
              className={inputClass}
            />
          </Field>
          <Field id={`${id}-host`} label={t('deviceConfigs.form.host')}>
            <input
              id={`${id}-host`}
              value={host}
              onChange={(e): void => setHost(e.target.value)}
              required
              placeholder="10.0.0.1"
              data-testid="config-target-host"
              className={inputClass}
            />
          </Field>
          <Field id={`${id}-platform`} label={t('deviceConfigs.form.platform')}>
            <select
              id={`${id}-platform`}
              value={platform}
              onChange={(e): void => setPlatform(e.target.value as DeviceConfigPlatform)}
              data-testid="config-target-platform"
              className={inputClass}
            >
              {PLATFORMS.map((p) => (
                <option key={p} value={p}>
                  {t(`deviceConfigs.platform.${p}`)}
                </option>
              ))}
            </select>
          </Field>
          <Field id={`${id}-port`} label={t('deviceConfigs.form.port')}>
            <input
              id={`${id}-port`}
              type="number"
              min={1}
              max={65535}
              value={port}
              onChange={(e): void => setPort(e.target.value)}
              placeholder={platform === 'netconf' ? '830' : '22'}
              data-testid="config-target-port"
              className={inputClass}
            />
          </Field>
          <Field id={`${id}-credential`} label={t('deviceConfigs.form.credential')}>
            <select
              id={`${id}-credential`}
              value={credential}
              onChange={(e): void => setPicked(e.target.value)}
              disabled={vault.loading}
              data-testid="config-target-credential"
              className={inputClass}
            >
              <option value="" disabled>
                {vault.loading
                  ? t('common:status.loading')
                  : t('deviceConfigs.form.credentialPick')}
              </option>
              {sshLogins.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.sshUser ? `${c.name} (${c.sshUser})` : c.name}
                </option>
              ))}
              <option value={NEW_LOGIN}>{t('deviceConfigs.form.credentialNew')}</option>
            </select>
            {vault.error ? (
              <p className="mt-tight text-xs text-status-error-strong">
                {t('deviceConfigs.form.credentialsLoadFailed', { reason: vault.error })}
              </p>
            ) : null}
          </Field>
          {credential === NEW_LOGIN ? (
            <fieldset data-testid="config-target-new-login" className="stack-sm">
              <legend className="text-xs text-text-muted">
                {t('deviceConfigs.form.newLoginHint')}
              </legend>
              <Field id={`${id}-login-name`} label={t('deviceConfigs.form.loginName')}>
                <input
                  id={`${id}-login-name`}
                  value={login.name}
                  onChange={(e): void => setLogin({ ...login, name: e.target.value })}
                  required
                  data-testid="config-target-login-name"
                  className={inputClass}
                />
              </Field>
              <Field id={`${id}-login-user`} label={t('deviceConfigs.form.loginUser')}>
                <input
                  id={`${id}-login-user`}
                  value={login.user}
                  onChange={(e): void => setLogin({ ...login, user: e.target.value })}
                  required
                  autoComplete="off"
                  data-testid="config-target-login-user"
                  className={inputClass}
                />
              </Field>
              <Field id={`${id}-login-password`} label={t('deviceConfigs.form.loginPassword')}>
                <input
                  id={`${id}-login-password`}
                  type="password"
                  value={login.password}
                  onChange={(e): void => setLogin({ ...login, password: e.target.value })}
                  required
                  autoComplete="new-password"
                  data-testid="config-target-login-password"
                  className={inputClass}
                />
              </Field>
            </fieldset>
          ) : null}
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
            disabled={submitting || vault.loading}
            data-testid="config-target-save"
            className="rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:control-disabled"
          >
            {submitting
              ? t('common:common.saving')
              : target
                ? t('deviceConfigs.form.save')
                : t('deviceConfigs.form.add')}
          </button>
        </ModalFooter>
      </form>
    </Modal>
  );
}

function Field({
  id,
  label,
  children,
}: {
  id: string;
  label: string;
  children: ReactNode;
}): JSX.Element {
  return (
    <div>
      <label htmlFor={id} className={labelClass}>
        {label}
      </label>
      <div className="mt-tight">{children}</div>
    </div>
  );
}

const labelClass: string = 'block text-xs font-medium uppercase tracking-wide text-text-muted';

const inputClass: string =
  'w-full rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none';
