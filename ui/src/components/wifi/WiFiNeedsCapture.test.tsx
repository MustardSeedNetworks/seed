/**
 * Without a capture source the daemon lists the rules that cannot run in
 * `status.needsCapture` (seed#2351). Both Wi-Fi cards must name them, so an
 * empty anomaly stream is not read as a clean airspace.
 */

import { render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import type { WiFiAirspaceResponse } from '../../types/generated/wifi-airspace-response';
import type { Status, WiFiAnomaliesResponse } from '../../types/generated/wifi-anomalies-response';
import { WiFiAirspaceCard } from './WiFiAirspaceCard';
import { WiFiAnomaliesCard } from './WiFiAnomaliesCard';

let status: Status;

vi.mock('../../hooks/useWifiVisibility', () => ({
  useWifiAirspace: () => ({
    data: { clientsWithheld: false, status, ssids: [] } satisfies WiFiAirspaceResponse,
    isLoading: false,
    isError: false,
  }),
  useWifiAnomalies: () => ({
    data: { anomalies: [], status } satisfies WiFiAnomaliesResponse,
    isLoading: false,
    isError: false,
  }),
}));

const deauthFlood = { id: 'wifi-deauth-flood', title: 'Deauthentication flood' };

function scanOnly(needsCapture?: Status['needsCapture']): Status {
  return {
    captureActive: false,
    ssids: 0,
    aps: 0,
    bsses: 0,
    stations: 0,
    anomalies: 0,
    needsCapture,
  };
}

const cards = [
  ['airspace', WiFiAirspaceCard],
  ['anomalies', WiFiAnomaliesCard],
] as const;

describe.each(cards)('%s card', (_name, CardUnderTest) => {
  beforeEach(() => {
    status = scanOnly([deauthFlood]);
  });

  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('names each rule that needs a capture source', () => {
    render(<CardUnderTest />);
    const notice = screen.getByTestId('wifi-needs-capture');
    expect(notice).toHaveTextContent('These checks need monitor-mode capture and did not run:');
    expect(within(notice).getByTestId('wifi-needs-capture-rule')).toHaveTextContent(
      'Deauthentication flood',
    );
  });

  it('names it in Spanish', async () => {
    await i18n.changeLanguage('es');
    render(<CardUnderTest />);
    expect(screen.getByTestId('wifi-needs-capture')).toHaveTextContent(
      'Estas comprobaciones necesitan captura en modo monitor y no se ejecutaron:',
    );
  });

  it('shows nothing while capture covers every rule', () => {
    status = { ...scanOnly(), captureActive: true };
    render(<CardUnderTest />);
    expect(screen.queryByTestId('wifi-needs-capture')).not.toBeInTheDocument();
  });
});

describe('anomalies card status', () => {
  it('is not a success while capture-only rules did not run', () => {
    status = scanOnly([deauthFlood]);
    render(<WiFiAnomaliesCard />);
    expect(screen.getByRole('img', { name: 'Status: unknown' })).toBeInTheDocument();
  });

  it('is a success when every rule ran and found nothing', () => {
    status = { ...scanOnly(), captureActive: true };
    render(<WiFiAnomaliesCard />);
    expect(screen.getByRole('img', { name: 'Status: success' })).toBeInTheDocument();
  });
});
