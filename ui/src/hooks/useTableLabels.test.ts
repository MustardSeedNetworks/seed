import { renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import i18n from '../i18n';
import { useTableLabels } from './useTableLabels';

beforeEach(async () => {
  await i18n.changeLanguage('es');
});
afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('useTableLabels', () => {
  it('speaks Spanish with no English left behind', () => {
    const { result } = renderHook(() => useTableLabels<string>({ loading: 'Cargando' }));
    expect(result.current.pagination).toBe('Páginas de la tabla');
    expect(result.current.previousPage).toBe('Página anterior');
    expect(result.current.pageStatus(2, 5, 113)).toBe('Página 2 de 5 (113 filas)');
    expect(result.current.actions).toBe('Acciones');
  });
});
