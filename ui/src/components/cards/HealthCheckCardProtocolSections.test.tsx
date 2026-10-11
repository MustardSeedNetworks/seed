/**
 * The daemon omits a zero connect time (`omitempty`), so a successful SQL or
 * LDAP check can arrive without `connectTimeMs`. The row must still render,
 * without a connect fact, rather than throw on the missing number.
 */
import { render, screen } from '@testing-library/react';
import type React from 'react';
import { useTranslation } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import type { HealthCheckRunResponse } from '../../types/generated/health-check-run-response';
import { HealthCheckCardProtocolSections } from './HealthCheckCardProtocolSections';

function Sections({ data }: { data: HealthCheckRunResponse }): React.ReactElement {
  const { t } = useTranslation('cards');
  return <HealthCheckCardProtocolSections data={data} t={t} />;
}

// Tooltip mirrors its trigger's text into a portal, so a bare text query
// matches the row twice.
const notTheTooltip = { ignore: '[role="tooltip"], script, style' } as const;

const base = { hasTests: true, pingResults: [], tcpResults: [], udpResults: [], httpResults: [] };

describe('HealthCheckCardProtocolSections', () => {
  it('renders SQL and LDAP checks that carry no connect time', () => {
    render(
      <Sections
        data={{
          ...base,
          enterpriseResults: {
            sqlResults: [
              {
                name: 'orders-db',
                success: true,
                driver: 'postgres',
                host: 'db1',
                port: 5432,
                totalTimeMs: 4,
              },
            ],
            ldapResults: [
              {
                name: 'directory',
                success: true,
                useTls: true,
                host: 'dc1',
                port: 636,
                totalTimeMs: 3,
              },
            ],
          },
        }}
      />,
    );

    expect(screen.getByText('orders-db', notTheTooltip)).toBeInTheDocument();
    expect(screen.getByText('directory', notTheTooltip)).toBeInTheDocument();
    expect(screen.queryByText(/^Connect /, notTheTooltip)).not.toBeInTheDocument();
  });

  it('shows the connect time when the daemon reports one', () => {
    render(
      <Sections
        data={{
          ...base,
          enterpriseResults: {
            sqlResults: [
              {
                name: 'orders-db',
                success: true,
                driver: 'postgres',
                host: 'db1',
                port: 5432,
                totalTimeMs: 4,
                connectTimeMs: 2.5,
              },
            ],
          },
        }}
      />,
    );

    expect(screen.getByText('Connect 2.5ms', notTheTooltip)).toBeInTheDocument();
  });
});
