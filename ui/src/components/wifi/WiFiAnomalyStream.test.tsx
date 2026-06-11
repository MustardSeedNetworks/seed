import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { Anomaly } from '../../types/generated/wifi-anomalies-response';
import { WiFiAnomalyStream } from './WiFiAnomalyStream';

function anomaly(overrides: Partial<Anomaly>): Anomaly {
  return {
    defKey: 'wifi-open-network',
    category: 'security',
    severity: 'warning',
    subject: { kind: 'bssid', id: '00:11:22:33:44:55' },
    title: 'Open network',
    description: 'no encryption',
    recommendation: 'enable WPA2/WPA3',
    firstSeen: '2026-01-01T00:00:00Z',
    lastSeen: '2026-01-01T00:00:00Z',
    count: 1,
    ...overrides,
  };
}

describe('WiFiAnomalyStream', () => {
  it('shows an empty state when there are no anomalies', () => {
    render(<WiFiAnomalyStream anomalies={[]} />);
    expect(screen.getByTestId('wifi-anomalies-empty')).toBeInTheDocument();
  });

  it('renders a row per anomaly with severity, title and subject', () => {
    render(<WiFiAnomalyStream anomalies={[anomaly({})]} />);
    expect(screen.getByTestId('wifi-anomaly-stream')).toBeInTheDocument();
    expect(screen.getAllByTestId('wifi-anomaly-row')).toHaveLength(1);
    expect(screen.getByText('Open network')).toBeInTheDocument();
    expect(screen.getByText('00:11:22:33:44:55')).toBeInTheDocument();
  });

  it('orders critical before error before warning before info', () => {
    render(
      <WiFiAnomalyStream
        anomalies={[
          anomaly({ defKey: 'a', severity: 'info', title: 'Info one' }),
          anomaly({ defKey: 'b', severity: 'critical', title: 'Critical one' }),
          anomaly({ defKey: 'c', severity: 'warning', title: 'Warning one' }),
          anomaly({ defKey: 'd', severity: 'error', title: 'Error one' }),
        ]}
      />,
    );
    const titles = screen.getAllByTestId('wifi-anomaly-row').map((r) => r.textContent ?? '');
    expect(titles[0]).toContain('Critical one');
    expect(titles[1]).toContain('Error one');
    expect(titles[2]).toContain('Warning one');
    expect(titles[3]).toContain('Info one');
  });
});
