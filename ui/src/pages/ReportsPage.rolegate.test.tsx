/**
 * ReportsPage role gating (#1254).
 *
 * ReportsCard already documented its contract — "Undefined for a viewer:
 * generate and delete are operator-gated" — and nothing implemented it. The
 * container passed both handlers to every role, so a viewer saw Generate and
 * Delete against POST /reports/generate and DELETE /reports/{id}, both
 * `minRole: op`.
 */

import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import type { ReportInfo } from '../types/generated/reports-response';

const reports: ReportInfo[] = [
  {
    id: 'r1',
    name: 'Executive summary',
    format: 'pdf',
    status: 'complete',
    createdAt: '2026-09-06T10:00:00Z',
  } as ReportInfo,
];

vi.mock('../hooks/useReports', () => ({
  useReports: () => ({
    reports,
    loading: false,
    error: null,
    generating: false,
    generate: vi.fn(),
    remove: vi.fn(),
  }),
}));

// The page is wrapped in RequireFeature("export_csv_json"); the licence is not
// what is under test, so the feature is present for both roles here.
vi.mock('../contexts/LicenseContext', () => ({
  useLicense: () => ({ hasFeature: () => true, loading: false }),
}));

vi.mock('../components/cards/SlaDashboardCard', () => ({
  SLADashboardCard: (): React.ReactElement => <div />,
}));

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/client', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

const { ReportsPage } = await import('./ReportsPage');

function renderAs(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role, isActive: true })
      : Promise.resolve({}),
  );
  render(
    <RoleProvider isAuthenticated={true}>
      <ReportsPage />
    </RoleProvider>,
  );
}

beforeEach(() => {
  mockGet.mockReset();
});

describe('ReportsPage — viewer gating', () => {
  it('offers neither generate nor delete to a viewer', async () => {
    renderAs('viewer');

    await waitFor(() => {
      expect(screen.getByTestId('report-row')).toBeTruthy();
    });
    expect(screen.queryByTestId('reports-generate')).toBeNull();
    expect(screen.queryByTestId('report-delete-r1')).toBeNull();
  });

  it('offers both to an operator', async () => {
    renderAs('operator');

    await waitFor(() => {
      expect(screen.getByTestId('reports-generate')).toBeTruthy();
    });
    expect(screen.getByTestId('report-delete-r1')).toBeTruthy();
  });
});
