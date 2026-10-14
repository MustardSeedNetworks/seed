/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WebAuthnRequestOptionsResponse {
  publicKey: WebAuthnRequestOptions;
}
export interface WebAuthnRequestOptions {
  challenge: string;
  timeout: number;
  rpId: string;
  allowCredentials: WebAuthnCredentialDescriptor[];
}
export interface WebAuthnCredentialDescriptor {
  type: 'public-key';
  id: string;
  transports?: ('usb' | 'nfc' | 'ble' | 'smart-card' | 'hybrid' | 'internal')[];
}
