/**
 * webauthn — the browser half of the passkey ceremonies.
 *
 * This did not exist. `MfaCard` called `/auth/webauthn/register/begin`, then
 * looked for a `window.seedWebAuthnRegister` helper that was never defined
 * anywhere, skipped it because the guard was always false, refreshed and
 * reported no error. So "Add a passkey" appeared to work and enrolled nothing:
 *
 *   POST /api/v1/auth/webauthn/register/begin  ->  200 {"publicKey":{...}}
 *   GET  /api/v1/auth/mfa/status               ->  webauthnCredentialCount: 0
 *
 * Nothing called `navigator.credentials` at all, so login was equally absent.
 *
 * The wire format is the WebAuthn JSON serialization the Go side (go-webauthn)
 * speaks: every binary field is base64url, and the browser needs ArrayBuffers.
 * Converting both ways is the whole substance of this module.
 */

import { api } from '../api';
import type { LoginResponse } from '../types/generated/login-response';
import type { WebAuthnCreationOptionsResponse } from '../types/generated/webauthn-creation-options-response';
import type { WebAuthnRequestOptionsResponse } from '../types/generated/webauthn-request-options-response';

/** base64UrlToBuffer decodes the padless base64url the server sends. */
export function base64UrlToBuffer(value: string): ArrayBuffer {
  const padded = value.replace(/-/g, '+').replace(/_/g, '/');
  const binary = atob(padded.padEnd(padded.length + ((4 - (padded.length % 4)) % 4), '='));
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  return bytes.buffer;
}

/** bufferToBase64Url encodes a credential field the way the server parses it. */
export function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/** isPasskeySupported reports whether this browser can run a ceremony at all. */
export function isPasskeySupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.PublicKeyCredential !== 'undefined' &&
    typeof navigator !== 'undefined' &&
    navigator.credentials !== undefined
  );
}

/**
 * registerPasskey runs the creation ceremony and enrols the result.
 *
 * The username is not sent to finish: register/finish reads it from the session
 * (`usernameFromContext`), because enrolment happens while already logged in.
 */
export async function registerPasskey(): Promise<void> {
  if (!isPasskeySupported()) {
    throw new Error('This browser cannot create a passkey.');
  }

  const options = await api.post<WebAuthnCreationOptionsResponse>(
    '/api/v1/auth/webauthn/register/begin',
    {},
  );
  const credential = (await navigator.credentials.create({
    publicKey: {
      ...options.publicKey,
      challenge: base64UrlToBuffer(options.publicKey.challenge),
      user: {
        ...options.publicKey.user,
        id: base64UrlToBuffer(options.publicKey.user.id),
      },
    },
  })) as PublicKeyCredential | null;

  if (!credential) {
    throw new Error('Passkey creation was cancelled.');
  }

  const response = credential.response as AuthenticatorAttestationResponse;
  await api.post('/api/v1/auth/webauthn/register/finish', {
    id: credential.id,
    rawId: bufferToBase64Url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
      attestationObject: bufferToBase64Url(response.attestationObject),
    },
  });
}

/**
 * loginWithPasskey runs the assertion ceremony and returns the session it wins.
 *
 * Sign-in is username-free: the browser offers whichever passkey it holds for
 * this server, and the passkey names the account. The ceremony itself travels
 * in an HttpOnly cookie that begin sets and finish consumes.
 */
export async function loginWithPasskey(): Promise<LoginResponse> {
  if (!isPasskeySupported()) {
    throw new Error('This browser cannot use a passkey.');
  }

  const options = await api.post<WebAuthnRequestOptionsResponse>(
    '/api/v1/auth/webauthn/login/begin',
    {},
  );
  const assertion = (await navigator.credentials.get({
    publicKey: {
      ...options.publicKey,
      challenge: base64UrlToBuffer(options.publicKey.challenge),
    },
  })) as PublicKeyCredential | null;

  if (!assertion) {
    throw new Error('Passkey sign-in was cancelled.');
  }

  const response = assertion.response as AuthenticatorAssertionResponse;

  return api.post<LoginResponse>('/api/v1/auth/webauthn/login/finish', {
    id: assertion.id,
    rawId: bufferToBase64Url(assertion.rawId),
    type: assertion.type,
    response: {
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
      authenticatorData: bufferToBase64Url(response.authenticatorData),
      signature: bufferToBase64Url(response.signature),
      userHandle: response.userHandle ? bufferToBase64Url(response.userHandle) : null,
    },
  });
}
