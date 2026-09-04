/**
 * webauthn ceremony tests.
 *
 * The module exists because the ceremony did not. MfaCard called
 * /auth/webauthn/register/begin, then looked for a window.seedWebAuthnRegister
 * helper defined nowhere, skipped it, and reported success — verified against a
 * running daemon:
 *
 *   POST /api/v1/auth/webauthn/register/begin -> 200 {"publicKey":{...}}
 *   GET  /api/v1/auth/mfa/status              -> webauthnCredentialCount: 0
 *
 * These cover the part that is easy to get quietly wrong: the base64url
 * round-trip, and that the finish call sends what the server parses.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  base64UrlToBuffer,
  bufferToBase64Url,
  isPasskeySupported,
  loginWithPasskey,
  registerPasskey,
} from './webauthn';

const mockPost = vi.fn<(path: string, body?: unknown) => Promise<unknown>>();
vi.mock('../api', () => ({
  api: { post: (path: string, body?: unknown): Promise<unknown> => mockPost(path, body) },
}));

function buf(...values: number[]): ArrayBuffer {
  return new Uint8Array(values).buffer;
}

beforeEach(() => {
  vi.stubGlobal('PublicKeyCredential', class {});
  vi.stubGlobal('navigator', { credentials: { create: vi.fn(), get: vi.fn() } });
});

afterEach(() => {
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});

describe('base64url', () => {
  it('round-trips bytes the server would send', () => {
    const original = buf(0, 1, 250, 251, 252, 253, 254, 255);
    expect(bufferToBase64Url(original)).not.toContain('+');
    expect(bufferToBase64Url(original)).not.toContain('/');
    expect(bufferToBase64Url(original)).not.toContain('=');
    expect(new Uint8Array(base64UrlToBuffer(bufferToBase64Url(original)))).toEqual(
      new Uint8Array(original),
    );
  });

  it('decodes a padless value, which is what the wire format uses', () => {
    // "AAAAAAAAAAE" is the user id the daemon returned for admin: 8 bytes, no
    // padding. A decoder that requires padding throws on exactly this.
    expect(new Uint8Array(base64UrlToBuffer('AAAAAAAAAAE'))).toEqual(
      new Uint8Array([0, 0, 0, 0, 0, 0, 0, 1]),
    );
  });
});

describe('isPasskeySupported', () => {
  it('is false when the browser has no WebAuthn', () => {
    vi.stubGlobal('PublicKeyCredential', undefined);
    expect(isPasskeySupported()).toBe(false);
  });

  it('is true when it does', () => {
    expect(isPasskeySupported()).toBe(true);
  });
});

describe('registerPasskey', () => {
  it('runs the ceremony and finishes it with base64url fields', async () => {
    mockPost.mockImplementation((endpoint: string) => {
      if (endpoint.endsWith('/register/begin')) {
        return Promise.resolve({
          publicKey: {
            challenge: 'AAAAAAAAAAE',
            user: { id: 'AAAAAAAAAAE', name: 'admin', displayName: 'admin' },
          },
        });
      }

      return Promise.resolve({});
    });
    const create = vi.fn(() =>
      Promise.resolve({
        id: 'cred-1',
        type: 'public-key',
        rawId: buf(1, 2, 3),
        response: { clientDataJSON: buf(4, 5), attestationObject: buf(6, 7) },
      }),
    );
    vi.stubGlobal('navigator', { credentials: { create, get: vi.fn() } });

    await registerPasskey();

    // The browser must receive ArrayBuffers, not the strings the server sent.
    const passed = create.mock.calls.at(0)?.at(0) as unknown as {
      publicKey: PublicKeyCredentialCreationOptions;
    };
    expect(passed.publicKey.challenge).toBeInstanceOf(ArrayBuffer);
    expect(passed.publicKey.user.id).toBeInstanceOf(ArrayBuffer);

    const [path, body] = mockPost.mock.calls.at(-1) as [string, Record<string, unknown>];
    expect(path).toBe('/api/v1/auth/webauthn/register/finish');
    expect(body.rawId).toBe(bufferToBase64Url(buf(1, 2, 3)));
    expect((body.response as Record<string, string>).attestationObject).toBe(
      bufferToBase64Url(buf(6, 7)),
    );
  });

  it('reports a cancelled ceremony instead of claiming success', async () => {
    mockPost.mockResolvedValue({
      publicKey: { challenge: 'AAAA', user: { id: 'AAAA', name: 'a', displayName: 'a' } },
    });
    vi.stubGlobal('navigator', {
      credentials: { create: vi.fn(() => Promise.resolve(null)), get: vi.fn() },
    });

    await expect(registerPasskey()).rejects.toThrow(/cancelled/i);
  });

  it('refuses on a browser with no WebAuthn rather than calling begin', async () => {
    vi.stubGlobal('PublicKeyCredential', undefined);

    await expect(registerPasskey()).rejects.toThrow(/cannot create a passkey/i);
    expect(mockPost).not.toHaveBeenCalled();
  });
});

describe('loginWithPasskey', () => {
  it('sends the username as a query parameter, which is where finish reads it', async () => {
    mockPost.mockImplementation((endpoint: string) => {
      if (endpoint.endsWith('/login/begin')) {
        return Promise.resolve({ publicKey: { challenge: 'AAAAAAAAAAE' } });
      }

      return Promise.resolve({ token: 'access-token' });
    });
    const get = vi.fn(() =>
      Promise.resolve({
        id: 'cred-1',
        type: 'public-key',
        rawId: buf(9),
        response: {
          clientDataJSON: buf(1),
          authenticatorData: buf(2),
          signature: buf(3),
          userHandle: null,
        },
      }),
    );
    vi.stubGlobal('navigator', { credentials: { create: vi.fn(), get } });

    const result = await loginWithPasskey('ad min');

    const [path] = mockPost.mock.calls.at(-1) as [string, unknown];
    expect(path).toBe('/api/v1/auth/webauthn/login/finish?username=ad%20min');
    expect(result.token).toBe('access-token');
  });
});
