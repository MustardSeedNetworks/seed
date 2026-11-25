import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { type LabelledStatus, useStatusLabel } from './useStatusLabel';

describe('useStatusLabel', () => {
  it.each<[LabelledStatus, string]>([
    ['success', 'Status: success'],
    ['warning', 'Status: warning'],
    ['error', 'Status: error'],
    ['unknown', 'Status: unknown'],
    ['loading', 'Status: loading'],
  ])('names %s in words', (status, label) => {
    const { result } = renderHook(() => useStatusLabel());
    expect(result.current(status)).toBe(label);
  });
});
