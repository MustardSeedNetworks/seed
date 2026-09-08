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
import { describe, expect, it, vi } from 'vitest';
import { emptyInput, TargetForm, type TargetFormProps, targetToInput } from './TargetForm';

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

  it('creates disabled, since the create form cannot yet attach a credential', () => {
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
