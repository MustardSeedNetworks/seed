/**
 * TargetForm.i18n.test.tsx — the polling target dialog speaks the operator's
 * language.
 *
 * S1-14d: two of the form's strings came from the locale files and the rest —
 * the title, every field label, the close and cancel buttons, the submit
 * button in all three states and both errors — were hardcoded English that
 * PollingTargetsPage.i18n.test.tsx never reached, because the dialog only
 * mounts after a click. `SNMP`, `IP` and the version values stay as they are.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import { emptyInput, TargetForm, type TargetFormProps } from './TargetForm';

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  vi.restoreAllMocks();
  await i18n.changeLanguage('en');
});

function renderForm(
  mode: TargetFormProps['mode'],
  onSubmit: TargetFormProps['onSubmit'] = vi.fn(),
): void {
  render(<TargetForm mode={mode} initial={emptyInput()} onSubmit={onSubmit} onCancel={vi.fn()} />);
}

describe('TargetForm — Spanish, with no English left behind', () => {
  it('titles and labels the create dialog', () => {
    renderForm('create');

    expect(screen.getByRole('heading', { name: 'Agregar destino de sondeo' })).toBeVisible();
    for (const label of [
      'Nombre',
      'Dirección IP',
      'Versión de SNMP',
      'Intervalo de sondeo (segundos)',
    ]) {
      expect(screen.getByText(label)).toBeVisible();
    }
    for (const english of [
      'Add polling target',
      'Name',
      'IP address',
      'SNMP version',
      'Poll interval (seconds)',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }
    expect(screen.getByRole('button', { name: 'Cerrar' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Cancelar' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Agregar destino' })).toBeVisible();
    for (const english of ['Close', 'Cancel', 'Add target']) {
      expect(screen.queryByRole('button', { name: english })).toBeNull();
    }
  });

  it('titles the edit dialog and its save button', () => {
    renderForm('edit');

    expect(screen.getByRole('heading', { name: 'Editar destino de sondeo' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Guardar cambios' })).toBeVisible();
    expect(screen.queryByText('Edit polling target')).toBeNull();
    expect(screen.queryByText('Save changes')).toBeNull();
  });

  it('says in Spanish that the name and address are required', async () => {
    renderForm('create');
    // The inputs carry `required`; submit the form itself so the handler's
    // own check runs rather than the browser's constraint validation.
    const form = screen.getByTestId('target-save').closest('form');
    if (!form) {
      throw new Error('TargetForm renders no form');
    }
    form.noValidate = true;

    await userEvent.click(screen.getByTestId('target-save'));

    expect(screen.getByTestId('target-form-error')).toHaveTextContent(
      'El nombre y la dirección IP son obligatorios.',
    );
  });

  it('shows the saving state and a failed save in Spanish', async () => {
    let reject: (reason: unknown) => void = () => {};
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((_, r) => {
          reject = r;
        }),
    );
    renderForm('create', onSubmit);
    await userEvent.type(screen.getByTestId('target-name'), 'core-01');
    await userEvent.type(screen.getByTestId('target-ip'), '192.0.2.2');

    await userEvent.click(screen.getByTestId('target-save'));
    expect(screen.getByTestId('target-save')).toHaveTextContent('Guardando…');
    expect(screen.queryByText('Saving…')).toBeNull();

    reject('refused');
    expect(await screen.findByTestId('target-form-error')).toHaveTextContent('No se pudo guardar');
  });
});
