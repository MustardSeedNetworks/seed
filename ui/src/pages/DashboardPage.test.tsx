/**
 * DashboardPage (UI-SEED-22): the default shows until a layout is saved; a
 * draft adds, removes and reorders cards without storing anything until
 * Save; a refused save keeps the draft and says why.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/client';
import i18n from '../i18n';
import type { DashboardLayout } from '../types/generated/dashboard-layout';

const store: { layout: DashboardLayout; refuse: string | null } = {
  layout: { widgets: [], saved: false },
  refuse: null,
};

const get = vi.fn((): Promise<DashboardLayout> => Promise.resolve(store.layout));
const put = vi.fn((_path: string, body: { widgets: string[] }): Promise<DashboardLayout> => {
  if (store.refuse !== null) {
    return Promise.reject(
      new ApiError(
        400,
        { error: 'That dashboard layout cannot be saved', details: store.refuse },
        undefined,
        null,
      ),
    );
  }
  store.layout = { widgets: body.widgets, saved: true };
  return Promise.resolve(store.layout);
});

vi.mock('../api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api')>()),
  api: { get: () => get(), put: (path: string, body: { widgets: string[] }) => put(path, body) },
}));

// The cards themselves are their pages' concern; here a widget is its id.
vi.mock('./dashboard/DashboardWidget', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./dashboard/DashboardWidget')>()),
  DashboardWidget: ({ id }: { id: string }) => <span>{id}</span>,
}));

const { DashboardPage } = await import('./DashboardPage');

function shown(): string[] {
  return screen
    .getAllByTestId(/^dashboard-widget-/)
    .map((el) => el.getAttribute('data-testid')?.replace('dashboard-widget-', '') ?? '');
}

beforeEach(async () => {
  store.layout = { widgets: [], saved: false };
  store.refuse = null;
  get.mockClear();
  put.mockClear();
  await i18n.changeLanguage('en');
});

describe('DashboardPage', () => {
  it('shows the default until a layout is saved', async () => {
    render(<DashboardPage />);
    await waitFor(() => expect(shown()).toEqual(['link', 'gateway', 'dns', 'network']));
  });

  it('shows a saved layout in order, skipping retired widgets', async () => {
    store.layout = { widgets: ['dns', 'retired', 'publicIp'], saved: true };
    render(<DashboardPage />);
    await waitFor(() => expect(shown()).toEqual(['dns', 'publicIp']));
  });

  it('says so when a saved dashboard is empty', async () => {
    store.layout = { widgets: [], saved: true };
    render(<DashboardPage />);
    expect(await screen.findByTestId('dashboard-empty')).toBeInTheDocument();
  });

  it('adds, removes and reorders a draft, and stores it only on Save', async () => {
    const user = userEvent.setup();
    render(<DashboardPage />);
    await user.click(await screen.findByTestId('dashboard-customize'));

    await user.click(screen.getByTestId('dashboard-down-link'));
    await user.click(screen.getByTestId('dashboard-remove-dns'));
    await user.selectOptions(screen.getByTestId('dashboard-add-choice'), 'driverStats');
    await user.click(screen.getByTestId('dashboard-add'));
    await user.click(screen.getByTestId('dashboard-up-driverStats'));

    // The grid previews the draft; nothing has been sent yet.
    expect(shown()).toEqual(['gateway', 'link', 'driverStats', 'network']);
    expect(put).not.toHaveBeenCalled();

    await user.click(screen.getByTestId('dashboard-save'));
    await waitFor(() => expect(screen.queryByTestId('dashboard-editor')).not.toBeInTheDocument());
    expect(put).toHaveBeenCalledWith('/api/v1/users/me/dashboard', {
      widgets: ['gateway', 'link', 'driverStats', 'network'],
    });
    expect(shown()).toEqual(['gateway', 'link', 'driverStats', 'network']);
  });

  it('offers no move past either end', async () => {
    const user = userEvent.setup();
    render(<DashboardPage />);
    await user.click(await screen.findByTestId('dashboard-customize'));
    expect(screen.getByTestId('dashboard-up-link')).toBeDisabled();
    expect(screen.getByTestId('dashboard-down-network')).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Move Gateway up' })).toBeEnabled();
  });

  it('cancel drops the draft and keeps the saved layout', async () => {
    const user = userEvent.setup();
    render(<DashboardPage />);
    await user.click(await screen.findByTestId('dashboard-customize'));
    await user.click(screen.getByTestId('dashboard-remove-link'));
    await user.click(screen.getByTestId('dashboard-cancel'));
    expect(shown()).toEqual(['link', 'gateway', 'dns', 'network']);
    expect(put).not.toHaveBeenCalled();
  });

  it('keeps the draft and says why when the server refuses it', async () => {
    store.refuse = 'widget "link" appears twice';
    const user = userEvent.setup();
    render(<DashboardPage />);
    await user.click(await screen.findByTestId('dashboard-customize'));
    await user.click(screen.getByTestId('dashboard-remove-dns'));
    await user.click(screen.getByTestId('dashboard-save'));

    expect(await screen.findByTestId('dashboard-save-error')).toHaveTextContent(
      'Not saved: widget "link" appears twice',
    );
    expect(screen.getByTestId('dashboard-editor')).toBeInTheDocument();
    expect(shown()).toEqual(['link', 'gateway', 'network']);
  });

  it('restores the default into the draft', async () => {
    store.layout = { widgets: ['publicIp'], saved: true };
    const user = userEvent.setup();
    render(<DashboardPage />);
    await user.click(await screen.findByTestId('dashboard-customize'));
    await user.click(screen.getByTestId('dashboard-reset'));
    expect(shown()).toEqual(['link', 'gateway', 'dns', 'network']);
  });
});
