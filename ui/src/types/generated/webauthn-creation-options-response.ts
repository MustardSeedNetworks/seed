/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WebAuthnCreationOptionsResponse {
  publicKey: WebAuthnCreationOptions;
}
export interface WebAuthnCreationOptions {
  rp: WebAuthnRelyingParty;
  user: WebAuthnUserEntity;
  challenge: string;
  pubKeyCredParams: WebAuthnCredentialParameter[];
  timeout: number;
  authenticatorSelection: WebAuthnAuthenticatorSelection;
  attestation: 'none';
}
export interface WebAuthnRelyingParty {
  name: string;
  id: string;
}
export interface WebAuthnUserEntity {
  name: string;
  displayName: string;
  id: string;
}
export interface WebAuthnCredentialParameter {
  type: 'public-key';
  alg: number;
}
export interface WebAuthnAuthenticatorSelection {
  requireResidentKey: boolean;
  residentKey: 'required';
  userVerification: 'required';
}
