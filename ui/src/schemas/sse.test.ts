import { describe, expect, it } from 'vitest';
import { parseSseCardUpdate, parseSseMessage } from './sse';

describe('parseSseMessage', () => {
  it('accepts a minimal envelope with type only', () => {
    const msg = parseSseMessage({ type: 'ping', payload: null });
    expect(msg).not.toBeNull();
    expect(msg?.type).toBe('ping');
  });

  it('accepts an envelope with arbitrary payload', () => {
    const msg = parseSseMessage({
      type: 'card_update',
      payload: { cardId: 'wlan0', data: { rssi: -65 } },
    });
    expect(msg).not.toBeNull();
    expect(msg?.type).toBe('card_update');
  });

  it('rejects missing type', () => {
    expect(parseSseMessage({ payload: 'oops' })).toBeNull();
  });

  it('rejects non-string type', () => {
    expect(parseSseMessage({ type: 42, payload: null })).toBeNull();
  });

  it('rejects non-objects', () => {
    expect(parseSseMessage(null)).toBeNull();
    expect(parseSseMessage('not an envelope')).toBeNull();
    expect(parseSseMessage(42)).toBeNull();
  });
});

describe('parseSseCardUpdate', () => {
  it('accepts a minimal card update', () => {
    const update = parseSseCardUpdate({ cardId: 'wlan0', data: null });
    expect(update).not.toBeNull();
    expect(update?.cardId).toBe('wlan0');
  });

  it('accepts an update with optional interface', () => {
    const update = parseSseCardUpdate({
      cardId: 'wlan0',
      data: { rssi: -65 },
      interface: 'wlan0',
    });
    expect(update?.interface).toBe('wlan0');
  });

  it('rejects missing cardId', () => {
    expect(parseSseCardUpdate({ data: 'whatever' })).toBeNull();
  });

  it('rejects non-string cardId', () => {
    expect(parseSseCardUpdate({ cardId: 42, data: null })).toBeNull();
  });

  it('rejects non-objects', () => {
    expect(parseSseCardUpdate(null)).toBeNull();
    expect(parseSseCardUpdate('wlan0')).toBeNull();
  });

  it('rejects non-string interface when present', () => {
    expect(parseSseCardUpdate({ cardId: 'wlan0', data: null, interface: 42 })).toBeNull();
  });
});
