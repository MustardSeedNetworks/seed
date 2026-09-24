/**
 * cards.i18n.test.tsx — the discovery, path and log cards speak the operator's
 * language in the copy no page suite reaches.
 *
 * S1-14d: these strings live in prop values, a phase-label table, a relative
 * time helper and a template literal — the shapes the shared JSX-text gate
 * cannot see. Under `es` a screen reader still announced "Start network
 * discovery scan" and a device row still read "Just now". Each case asserts
 * the Spanish text and that the English is gone: a key that silently falls
 * back renders English, which a single-locale test cannot see.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import { DiscoveryModal } from './DiscoveryModal';
import { NetworkDiscoveryCard } from './NetworkDiscoveryCard';
import type { NetworkDiscoveryData } from './networkDiscoveryCardTypes';
import { PathDiscoveryCard } from './PathDiscoveryCard';
import { ScanProgress } from './ScanProgress';

vi.mock('../../hooks/useEngineScan', () => ({
  useEngineScan: () => ({
    running: false,
    status: { state: 'idle', jobId: '', percentComplete: 0, error: null },
    startScan: vi.fn().mockResolvedValue(undefined),
    cancelScan: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('../../hooks/useEnginePhase', () => ({
  useEnginePhase: () => ({ phase: '' }),
}));

vi.mock('../../hooks/useNetworkDiscoveryAutoScan', () => ({
  useNetworkDiscoveryAutoScan: () => ({
    handleDeepScan: vi.fn().mockResolvedValue(undefined),
  }),
}));

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  vi.restoreAllMocks();
  await i18n.changeLanguage('en');
});

function withQuery(node: ReactNode): ReactNode {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

function discovery(lastSeen: string): NetworkDiscoveryData {
  return {
    devices: [
      {
        ip: '192.0.2.10',
        mac: '02:00:5e:10:00:10',
        hostname: 'core-01',
        vendor: 'Example',
        lastSeen,
        discoveryMethod: ['arp'],
        isLocal: false,
      },
    ],
    status: {
      scanning: false,
      deviceCount: 1,
      lastScan: lastSeen,
      subnet: '192.0.2.0/24',
      localIP: '192.0.2.5',
      interface: 'en0',
    },
  };
}

describe('discovery cards — Spanish, with no English left behind', () => {
  it('names the empty card and its scan button', () => {
    render(<NetworkDiscoveryCard data={null} onScan={() => {}} />);

    expect(
      screen.getByRole('button', { name: 'Iniciar escaneo de descubrimiento de red' }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText('Start network discovery scan')).not.toBeInTheDocument();
    expect(
      screen.getByLabelText('Descubrimiento de red: no hay datos disponibles'),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText('Network discovery - no data available')).toBeNull();
  });

  it('names the loading card', () => {
    render(<NetworkDiscoveryCard data={null} loading={true} />);

    expect(screen.getByLabelText('Descubrimiento de red en curso')).toBeInTheDocument();
    expect(screen.queryByLabelText('Network discovery scanning in progress')).toBeNull();
  });

  it('names the populated card, its full-screen and scan buttons, with a counted device', () => {
    render(<NetworkDiscoveryCard data={discovery(new Date().toISOString())} onScan={() => {}} />);

    expect(
      screen.getByLabelText('Descubrimiento de red: 1 dispositivo encontrado'),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/devices found/)).toBeNull();
    expect(
      screen.getByRole('button', { name: 'Abrir vista de pantalla completa' }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText('Open full screen view')).toBeNull();
    expect(screen.getByRole('button', { name: 'Iniciar escaneo de red' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Start network scan')).toBeNull();
  });

  it('names every engine phase, not only the two-word one', () => {
    for (const [phase, spanish, english] of [
      ['discovery', 'Descubrimiento', 'Discovery'],
      ['correlation', 'Correlacionando', 'Correlating'],
      ['name_resolution', 'Resolución de nombres', 'Name resolution'],
      ['enrichment', 'Enriquecimiento', 'Enrichment'],
      ['assessment', 'Evaluación', 'Assessment'],
    ] as const) {
      const { unmount } = render(<ScanProgress percent={40} phase={phase} />);
      expect(screen.getByText(`Escaneando — ${spanish}`)).toBeInTheDocument();
      expect(screen.queryByText(new RegExp(english))).toBeNull();
      unmount();
    }
  });

  it('reads a fresh device row and the modal close button in Spanish', () => {
    render(
      withQuery(
        <DiscoveryModal
          isOpen={true}
          onClose={() => {}}
          data={discovery(new Date().toISOString())}
        />,
      ),
    );

    expect(screen.getAllByText('Justo ahora').length).toBeGreaterThan(0);
    expect(screen.queryByText('Just now')).toBeNull();
    expect(screen.getByRole('button', { name: 'Cerrar' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Close' })).toBeNull();
  });

  it('labels the path trace port field', () => {
    render(withQuery(<PathDiscoveryCard />));
    fireEvent.change(screen.getByLabelText('Protocolo'), { target: { value: 'tcp' } });

    expect(screen.getByPlaceholderText('Puerto')).toBeInTheDocument();
    expect(screen.queryByPlaceholderText('Port')).toBeNull();
  });
});
