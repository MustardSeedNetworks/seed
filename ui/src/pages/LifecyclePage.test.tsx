/**
 * Hardware lifecycle (UI-SEED-47): parts are ranked by how far past their
 * end-of-life they are, only a passed sale or support milestone claims a
 * state, an EoX import round-trips and refusals are worded, and a viewer reads
 * without the import control.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import type { EoXRecord, LifecycleDevice } from '../types/generated/lifecycle-devices-response';

const c3750: EoXRecord = {
  productId: 'WS-C3750X-48P-S',
  description: 'Catalyst 3750X 48 Port PoE IP Base',
  endOfSale: '2016-10-30',
  endOfSwMaintenance: '2017-10-30',
  endOfSecuritySupport: '2019-10-30',
  lastDateOfSupport: '2021-10-31',
  bulletin: 'https://www.cisco.com/c/en/us/products/eol-notice.html',
  migrationProductId: 'WS-C3850-48P-S',
};

const c9300: EoXRecord = {
  productId: 'C9300-48P',
  endOfSale: '2026-09-01',
  lastDateOfSupport: '2031-09-30',
  bulletin: 'javascript:alert(1)',
};

/** The server's assessment: before an import, every part is not listed. */
function devices(withData: boolean): LifecycleDevice[] {
  return [
    {
      ip: '10.0.0.2',
      name: 'core-sw',
      components: [
        withData
          ? { productId: 'C9300-48P', name: 'Switch 1', status: 'end_of_sale', record: c9300 }
          : { productId: 'C9300-48P', name: 'Switch 1', status: 'not_listed' },
        { productId: 'GLC-TE', name: 'Gi1/0/1 SFP', status: 'not_listed' },
      ],
    },
    {
      ip: '10.0.0.9',
      components: [
        withData
          ? { productId: 'WS-C3750X-48P-S', status: 'end_of_support', record: c3750 }
          : { productId: 'WS-C3750X-48P-S', status: 'not_listed' },
      ],
    },
  ];
}

const { ApiError } = await import('../api/client');

let imported = false;
let refusal: InstanceType<typeof ApiError> | null = null;
const puts: unknown[] = [];
let role: CurrentUser['role'] = 'operator';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    api: {
      get: async (path: string): Promise<unknown> => {
        if (path.includes('/users/me')) {
          return { username: 'u', role, isActive: true };
        }
        return { records: imported ? 2 : 0, devices: devices(imported) };
      },
      put: async (_path: string, body: unknown): Promise<unknown> => {
        puts.push(body);
        if (refusal) {
          throw refusal;
        }
        imported = true;
        return { records: [c3750, c9300] };
      },
    },
  };
});

const { LifecyclePage } = await import('./LifecyclePage');

function renderPage(): void {
  render(
    <RoleProvider isAuthenticated={true}>
      <LifecyclePage />
    </RoleProvider>,
  );
}

const eoxFile = (text: string): File => new File([text], 'eox.json', { type: 'application/json' });

async function importFile(user: ReturnType<typeof userEvent.setup>, file: File): Promise<void> {
  await waitFor(() =>
    expect(screen.getByTestId('lifecycle-import')).not.toHaveAttribute('aria-disabled'),
  );
  await user.upload(screen.getByTestId('lifecycle-import-file'), file);
}

describe('LifecyclePage', () => {
  beforeEach(() => {
    imported = false;
    refusal = null;
    puts.length = 0;
    role = 'operator';
  });

  it('lists every part as not listed, claiming no state, until EoX data is imported', async () => {
    renderPage();

    const part = await screen.findByTestId('lifecycle-part-10.0.0.2#0');
    expect(part).toHaveTextContent('C9300-48P');
    expect(part).toHaveTextContent('core-sw · Switch 1');
    expect(part).toHaveTextContent('Not listed');
    expect(part).toHaveAttribute('data-state', 'unknown');
    // A device without a name is named by its address.
    expect(screen.getByTestId('lifecycle-part-10.0.0.9#0')).toHaveTextContent('10.0.0.9');
    expect(screen.getByTestId('lifecycle-counts')).toHaveTextContent(
      '3 parts · No EoX data imported',
    );
    expect(screen.getByTestId('lifecycle-not-listed')).toHaveTextContent('No EoX data yet.');
  });

  it('imports an EoX file, then ranks the part past end of support first', async () => {
    const user = userEvent.setup();
    renderPage();
    const body = { EOXRecord: [{ EOLProductID: 'WS-C3750X-48P-S' }] };
    await importFile(user, eoxFile(JSON.stringify(body)));

    expect(await screen.findByTestId('lifecycle-import-result')).toHaveTextContent(
      'Imported EoX data for 2 products.',
    );
    expect(puts).toEqual([body]);
    expect(screen.getByTestId('lifecycle-counts')).toHaveTextContent(
      '3 parts · EoX data for 2 products',
    );

    const rows = screen.getAllByTestId(/^lifecycle-part-/);
    expect(rows.map((row) => row.dataset.testid)).toEqual([
      'lifecycle-part-10.0.0.9#0',
      'lifecycle-part-10.0.0.2#0',
      'lifecycle-part-10.0.0.2#1',
    ]);
    expect(rows[0]).toHaveAttribute('data-state', 'crit');
    expect(rows[0]).toHaveTextContent('End of support');
    expect(rows[1]).toHaveAttribute('data-state', 'warn');

    // The first part opens: its dates, its replacement and its bulletin.
    expect(screen.getByTestId('lifecycle-device')).toHaveTextContent('10.0.0.9');
    expect(screen.getByTestId('lifecycle-last-support')).toHaveTextContent('2021-10-31');
    expect(screen.getByText('WS-C3850-48P-S')).toBeVisible();
    expect(screen.getByTestId('lifecycle-bulletin')).toHaveAttribute('href', c3750.bulletin);
  });

  it('names an unannounced date and links no bulletin that is not https', async () => {
    imported = true;
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByTestId('lifecycle-part-10.0.0.2#0'));

    expect(screen.getByTestId('lifecycle-device')).toHaveTextContent('core-sw (10.0.0.2)');
    expect(screen.getByTestId('lifecycle-end-of-sale')).toHaveTextContent('2026-09-01');
    expect(screen.getAllByText('Not announced')).toHaveLength(2);
    expect(screen.queryByTestId('lifecycle-bulletin')).toBeNull();

    await user.click(screen.getByTestId('lifecycle-part-10.0.0.2#1'));
    expect(screen.getByTestId('lifecycle-not-listed')).toHaveTextContent(
      'The imported EoX data has no record for this product.',
    );
  });

  it('refuses a file that is not JSON without asking the server', async () => {
    const user = userEvent.setup();
    renderPage();
    await importFile(user, eoxFile('PID,EndOfSale'));

    expect(await screen.findByTestId('lifecycle-import-result')).toHaveTextContent(
      'That file is not a Cisco EoX API response.',
    );
    expect(puts).toEqual([]);
  });

  it.each([
    [400, 'That file is not a Cisco EoX API response.'],
    [413, 'That file is larger than 8 MB.'],
  ])('words a %i refusal', async (status, wording) => {
    refusal = new ApiError(status, { error: 'server reason' }, undefined, null);
    const user = userEvent.setup();
    renderPage();
    await importFile(user, eoxFile('{"EOXRecord": []}'));

    expect(await screen.findByTestId('lifecycle-import-result')).toHaveTextContent(wording);
    expect(screen.getByTestId('lifecycle-counts')).toHaveTextContent('No EoX data imported');
  });

  it('leaves a viewer able to read but not import', async () => {
    role = 'viewer';
    renderPage();

    await screen.findByTestId('lifecycle-part-10.0.0.2#0');
    // The tooltip keeps a disabled control focusable, so it is aria-disabled.
    await waitFor(() =>
      expect(screen.getByTestId('lifecycle-import')).toHaveAttribute('aria-disabled', 'true'),
    );
  });
});
