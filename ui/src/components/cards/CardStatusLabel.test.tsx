import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import type { CardStatus } from '../../shared/Card';
import { useCardStatusLabel } from './CardStatusLabel';

const statuses: CardStatus[] = ['success', 'warning', 'error', 'info', 'loading', 'unknown'];

describe('useCardStatusLabel', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each(statuses)('names the %s status in words', (status) => {
    const { result } = renderHook(() => useCardStatusLabel());

    expect(result.current(status)).toBe(`Status: ${status}`);
  });

  it('translates the label under es', async () => {
    await i18n.changeLanguage('es');
    const { result } = renderHook(() => useCardStatusLabel());

    expect(result.current('error')).toBe('Estado: error');
  });
});
