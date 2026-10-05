/**
 * PathDiscoveryCard tests: the trace request carries what the form shows.
 *
 * The form's target, protocol and port are read with useWatch (UI-SEED-41
 * slice 7) and submitted through handleSubmit, so these pin both: the Trace
 * button stays disabled until a target is typed, and a submitted or quick
 * trace posts the current protocol and port.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import type { PathResponse } from '../../types';
import { PathDiscoveryCard } from './PathDiscoveryCard';

const api = vi.hoisted(() => ({
  post: vi.fn<(url: string, body: unknown) => Promise<PathResponse>>(),
}));
vi.mock('../../api', () => ({ api }));

describe('PathDiscoveryCard', () => {
  beforeEach(() => {
    api.post.mockReset();
    api.post.mockResolvedValue({} as PathResponse);
  });

  it('posts the typed target with the chosen protocol and port', async () => {
    const user = userEvent.setup();
    render(<PathDiscoveryCard />);
    const trace = screen.getByRole('button', { name: i18n.t('cards:pathDiscovery.trace') });
    expect(trace).toBeDisabled();

    await user.type(
      screen.getByPlaceholderText(i18n.t('cards:pathDiscovery.enterTarget')),
      'example.com',
    );
    await user.selectOptions(
      screen.getByRole('combobox', { name: i18n.t('cards:pathDiscovery.protocol') }),
      'tcp',
    );
    const port = screen.getByPlaceholderText(i18n.t('cards:pathDiscovery.port'));
    await user.clear(port);
    await user.type(port, '443');
    expect(trace).toBeEnabled();
    await user.click(trace);

    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
    expect(api.post).toHaveBeenCalledWith('/api/v1/path/path', {
      source: 'self',
      destination: 'example.com',
      method: 'both',
      protocol: 'tcp',
      port: 443,
    });
  });

  it('a quick target fills the field and traces it over ICMP', async () => {
    const user = userEvent.setup();
    render(<PathDiscoveryCard dnsServer="9.9.9.9" />);
    await user.click(screen.getByRole('button', { name: i18n.t('cards:pathDiscovery.dns') }));

    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
    expect(api.post).toHaveBeenCalledWith(
      '/api/v1/path/path',
      expect.objectContaining({ destination: '9.9.9.9', protocol: 'icmp', port: undefined }),
    );
    expect(screen.getByPlaceholderText(i18n.t('cards:pathDiscovery.enterTarget'))).toHaveValue(
      '9.9.9.9',
    );
  });
});
