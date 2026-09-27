/**
 * chrome.i18n.test.tsx — the shared primitives and app chrome speak the
 * operator's language.
 *
 * #2839: these components reach every route, and their copy lives in prop
 * values and string literals — the shape the shared JSX-text gate cannot see.
 * Under `es` a screen reader announced "Close modal", "Notifications" and
 * "Status: success" on every page. Each case asserts the Spanish name and that
 * the English one is gone: a key that silently falls back renders English,
 * which a single-locale test cannot see.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Server } from 'lucide-react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Router } from 'wouter';
import { memoryLocation } from 'wouter/memory-location';

import i18n from '../../i18n';
import { useTestRunStore } from '../../stores/testRunStore';
import { Breadcrumbs } from '../../ui/Breadcrumbs';
import { SidebarLayout } from '../../ui/Sidebar';
import { Alert } from './Alert';
import { CommandPalette } from './CommandPalette';
import { type Column, DataTable } from './DataTable';
import { DeviceSelector } from './DeviceSelector';
import { Fab } from './Fab';
import { Modal } from './Modal';
import { StatusBadge } from './StatusBadge';
import { ToastProvider } from './Toast';
import { useToast } from './useToast';

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  await i18n.changeLanguage('en');
});

function withQuery(node: ReactNode): ReactNode {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

describe('shared chrome — Spanish, with no English left behind', () => {
  it('names the Alert dismiss button', () => {
    render(
      <Alert status="info" onDismiss={() => {}}>
        x
      </Alert>,
    );

    expect(screen.getByRole('button', { name: 'Descartar alerta' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Dismiss alert')).not.toBeInTheDocument();
  });

  it('labels the command palette, its search, its actions and the theme toggle', () => {
    // cmdk constructs a ResizeObserver; the suite-wide stub is an arrow function.
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe(): void {}
        unobserve(): void {}
        disconnect(): void {}
      },
    );
    // …and scrolls the active item into view, which jsdom does not implement.
    Element.prototype.scrollIntoView = (): void => {};
    render(
      <CommandPalette
        groups={[]}
        open={true}
        onOpenChange={() => {}}
        onToggleTheme={() => {}}
        isDark={false}
      />,
    );

    expect(screen.getByRole('button', { name: 'Cerrar paleta de comandos' })).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Buscar páginas y acciones…')).toBeInTheDocument();
    expect(screen.getByText('Acciones')).toBeInTheDocument();
    expect(screen.getByText('Cambiar a modo oscuro')).toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: 'Paleta de comandos' })).toBeInTheDocument();
    for (const english of ['Actions', 'Switch to dark mode']) {
      expect(screen.queryByText(english)).not.toBeInTheDocument();
    }
    expect(screen.queryByPlaceholderText('Search pages and actions…')).not.toBeInTheDocument();
    // The prototype stub must not leak into other test files.
    Reflect.deleteProperty(Element.prototype, 'scrollIntoView');
  });

  it('labels the DataTable empty, loading and search states', () => {
    const columns: Column<{ name: string }>[] = [
      { key: 'name', header: 'N', accessor: (r) => r.name },
    ];
    const { rerender } = render(
      <DataTable data={[]} columns={columns} keyExtractor={(r) => r.name} />,
    );

    expect(screen.getByText('No se encontraron datos')).toBeInTheDocument();
    expect(screen.queryByText('No data found')).not.toBeInTheDocument();
    const search = screen.getByPlaceholderText('Buscar…');
    fireEvent.change(search, { target: { value: 'x' } });
    expect(screen.getByRole('button', { name: 'Limpiar búsqueda' })).toBeInTheDocument();

    rerender(<DataTable data={[]} columns={columns} keyExtractor={(r) => r.name} loading={true} />);
    expect(screen.getByRole('status', { name: 'Cargando datos...' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Loading data')).not.toBeInTheDocument();
  });

  it('shows the DeviceSelector placeholder', () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((async () => ({
      ok: true,
      status: 200,
      json: async () => ({ devices: [] }),
    })) as unknown as typeof fetch);
    render(withQuery(<DeviceSelector value="" onChange={() => {}} />));

    expect(screen.getByText('Seleccionar dispositivo')).toBeInTheDocument();
    expect(screen.queryByText('Select device')).not.toBeInTheDocument();
  });

  it('names the Fab after a partial run', () => {
    useTestRunStore.getState().reset();
    const { rerender } = render(<Fab />);
    act(() => {
      const runId = useTestRunStore.getState().start();
      useTestRunStore.getState().settlePartial(runId);
    });
    rerender(<Fab />);

    expect(
      screen.getByRole('button', {
        name: 'Algunas comprobaciones no terminaron: toque para volver a ejecutar todas las pruebas',
      }),
    ).toBeInTheDocument();
    useTestRunStore.getState().reset();
  });

  it('names both Modal close controls', () => {
    render(
      <Modal isOpen={true} onClose={() => {}} title="t" showCloseButton={true}>
        body
      </Modal>,
    );

    expect(screen.getAllByRole('button', { name: 'Cerrar ventana' })).toHaveLength(2);
    expect(screen.queryByLabelText('Close modal')).not.toBeInTheDocument();
  });

  it('labels the toast region, each toast and its dismiss button', async () => {
    function Trigger(): React.JSX.Element {
      const { addToast } = useToast();
      return (
        <button type="button" onClick={() => addToast('hola', 'success', 0)}>
          go
        </button>
      );
    }
    render(
      <ToastProvider>
        <Trigger />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByText('go'));

    await waitFor(() => {
      expect(screen.getByRole('region', { name: 'Notificaciones' })).toBeInTheDocument();
    });
    expect(screen.getByRole('alert', { name: 'Notificación: éxito' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Descartar notificación' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Notification: success')).not.toBeInTheDocument();
  });

  it.each([
    ['success', 'Estado: correcto'],
    ['warning', 'Estado: advertencia'],
    ['error', 'Estado: error'],
    ['unknown', 'Estado: desconocido'],
    ['loading', 'Estado: cargando'],
  ] as const)('names the %s StatusBadge', (status, spanish) => {
    render(<StatusBadge status={status} />);

    expect(screen.getByRole('img', { name: spanish })).toBeInTheDocument();
    expect(screen.queryByLabelText(`Status: ${status}`)).not.toBeInTheDocument();
  });

  it('labels the breadcrumb trail and its home link', () => {
    const { hook } = memoryLocation({ path: '/network' });
    render(
      <Router hook={hook}>
        <Breadcrumbs />
      </Router>,
    );

    expect(screen.getByRole('navigation', { name: 'Ruta de navegación' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Inicio' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Breadcrumb')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Home')).not.toBeInTheDocument();
  });

  it('names the phone drawer scrim', () => {
    render(
      <SidebarLayout groups={[{ label: 'g', items: [{ path: '/', label: 'x', icon: Server }] }]}>
        <div />
      </SidebarLayout>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Abrir menú' }));

    expect(screen.getAllByRole('button', { name: 'Cerrar menú' }).length).toBeGreaterThan(0);
    expect(screen.queryByLabelText('Close menu')).not.toBeInTheDocument();
  });
});
