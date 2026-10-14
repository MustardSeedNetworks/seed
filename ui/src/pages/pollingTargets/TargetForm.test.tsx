/**
 * The form refuses what the poller refuses. `CredentialResolver.Resolve`
 * (internal/polling/snmp/credentials.go) treats a target with no
 * CredentialsID as a configuration error rather than a request for anonymous
 * polling, so an enabled target saved without one can only produce
 * ErrCredentialsUnresolved on every tick. The refusal belongs where the
 * operator can still act on it.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { UseDeviceCredentialsResult } from '../../hooks/useDeviceCredentials';
import type { Credentials } from '../../types/generated/device-credential-list-response';
import { emptyInput, TargetForm, type TargetFormProps, targetToInput } from './TargetForm';

const vault = vi.hoisted(() => ({
  current: { credentials: [], loading: false, error: null } as UseDeviceCredentialsResult,
}));

vi.mock('../../hooks/useDeviceCredentials', () => ({
  useDeviceCredentials: (): UseDeviceCredentialsResult => vault.current,
}));

const community: Credentials = {
  id: 'cred-000000000001',
  clientId: 'c',
  name: 'core community',
  kind: 'v2c',
  createdAt: '',
  updatedAt: '',
};

const v3User: Credentials = {
  id: 'cred-000000000002',
  clientId: 'c',
  name: 'noc v3',
  kind: 'v3',
  securityLevel: 'authPriv',
  snmpV3User: 'noc',
  snmpV3AuthProto: 'SHA',
  snmpV3PrivProto: 'AES',
  createdAt: '',
  updatedAt: '',
};

beforeEach(() => {
  vault.current = { credentials: [community, v3User], loading: false, error: null };
});

function fill(): Promise<void> {
  return (async (): Promise<void> => {
    await userEvent.type(screen.getByTestId('target-name'), 'core-01');
    await userEvent.type(screen.getByTestId('target-ip'), '10.44.10.2');
  })();
}

describe('TargetForm — credential refusal', () => {
  it('refuses to save an enabled target with no credential, and says the rule', async () => {
    const onSubmit = vi.fn<TargetFormProps['onSubmit']>();
    render(
      <TargetForm
        mode="create"
        initial={{ ...emptyInput(), enabled: true }}
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />,
    );
    await fill();

    await userEvent.click(screen.getByTestId('target-save'));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByTestId('target-form-error').textContent).toContain('needs a credential');
  });

  it('saves a disabled target with no credential', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(
      <TargetForm mode="create" initial={emptyInput()} onSubmit={onSubmit} onCancel={vi.fn()} />,
    );
    await fill();

    await userEvent.click(screen.getByTestId('target-save'));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it('saves an enabled target that carries a credential', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(
      <TargetForm
        mode="edit"
        initial={{
          ...emptyInput(),
          name: 'core-01',
          ipAddress: '10.44.10.2',
          enabled: true,
          credentialsId: 'cred-1',
        }}
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />,
    );

    await userEvent.click(screen.getByTestId('target-save'));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });
});

describe('TargetForm — collector chain', () => {
  /**
   * The repository fills an absent chain with its own default
   * (repository_polling_targets.go:124). `[]` is not that default, and a form
   * that mirrors a server default by hand is the defect #2393 removed
   * elsewhere in the UI.
   */
  it('sends no chain on create so the server default applies', () => {
    expect(emptyInput().collectorChain).toBeUndefined();
  });

  it('creates disabled, since a fresh form has no credential chosen', () => {
    expect(emptyInput().enabled).toBe(false);
  });

  it('shows an existing chain read-only rather than an editable mirror', () => {
    render(
      <TargetForm
        mode="edit"
        initial={targetToInput({
          id: 't',
          clientId: 'c',
          name: 'core-01',
          ipAddress: '10.44.10.2',
          snmpVersion: 'v2c',
          credentialsId: 'cred-1',
          pollIntervalSeconds: 300,
          enabled: true,
          collectorChain: ['sys_info', 'if_table'],
          lastStatus: 'ok',
          lastError: '',
          createdAt: '',
          updatedAt: '',
        })}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    const chain = screen.getByTestId('target-chain');
    expect(chain.textContent).toContain('sys_info');
    expect(chain.textContent).toContain('if_table');
    expect(chain.querySelector('input')).toBeNull();
  });
});

describe('TargetForm — credential picker', () => {
  function optionNames(): string[] {
    const select = screen.getByTestId('target-credential');
    return Array.from(select.querySelectorAll('option')).map((o) => o.textContent ?? '');
  }

  it('offers only the credentials whose kind matches the SNMP version', async () => {
    render(
      <TargetForm mode="create" initial={emptyInput()} onSubmit={vi.fn()} onCancel={vi.fn()} />,
    );
    expect(optionNames()).toEqual(['No credential', 'core community']);

    await userEvent.selectOptions(screen.getByTestId('target-version'), 'v3');

    expect(optionNames()).toEqual(['No credential', 'noc v3']);
  });

  it('saves an enabled target with the credential picked from the vault', async () => {
    const onSubmit = vi.fn<TargetFormProps['onSubmit']>().mockResolvedValue(undefined);
    render(
      <TargetForm
        mode="create"
        initial={{ ...emptyInput(), enabled: true }}
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />,
    );
    await fill();

    await userEvent.selectOptions(screen.getByTestId('target-credential'), community.id);
    await userEvent.click(screen.getByTestId('target-save'));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ enabled: true, snmpVersion: 'v2c', credentialsId: community.id }),
    );
  });

  it('drops the chosen credential when the version changes under it', async () => {
    const onSubmit = vi.fn<TargetFormProps['onSubmit']>().mockResolvedValue(undefined);
    render(
      <TargetForm
        mode="edit"
        initial={{
          ...emptyInput(),
          name: 'core-01',
          ipAddress: '10.44.10.2',
          credentialsId: community.id,
        }}
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByTestId('target-credential')).toHaveValue(community.id);

    await userEvent.selectOptions(screen.getByTestId('target-version'), 'v3');
    await userEvent.click(screen.getByTestId('target-save'));

    expect(screen.getByTestId('target-credential')).toHaveValue('');
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ credentialsId: undefined }));
  });

  it('summarises a v3 credential without any secret', async () => {
    render(
      <TargetForm
        mode="create"
        initial={{ ...emptyInput(), snmpVersion: 'v3' }}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    await userEvent.selectOptions(screen.getByTestId('target-credential'), v3User.id);

    expect(screen.getByTestId('target-credential-summary')).toHaveTextContent(
      'User noc · authPriv · SHA · AES',
    );
  });

  it('says when the vault holds nothing for the chosen version', () => {
    vault.current = { credentials: [v3User], loading: false, error: null };
    render(
      <TargetForm mode="create" initial={emptyInput()} onSubmit={vi.fn()} onCancel={vi.fn()} />,
    );

    expect(optionNames()).toEqual(['No credential']);
    expect(screen.getByTestId('target-credential-summary')).toHaveTextContent(
      'The vault holds no SNMP v2c credential.',
    );
  });

  it('reports a vault that failed to load instead of an empty list', () => {
    vault.current = { credentials: [], loading: false, error: 'Forbidden' };
    render(
      <TargetForm mode="create" initial={emptyInput()} onSubmit={vi.fn()} onCancel={vi.fn()} />,
    );

    expect(screen.getByTestId('target-credential-summary')).toHaveTextContent(
      'Could not load the credential vault: Forbidden',
    );
  });
});
