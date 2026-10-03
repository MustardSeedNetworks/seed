import { createHmac } from 'node:crypto';

const BASE32 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';

function decodeBase32(secret: string): Buffer {
  let bits = '';
  for (const char of secret.replace(/=+$/, '').toUpperCase()) {
    const value = BASE32.indexOf(char);
    if (value === -1) {
      throw new Error(`totp: ${char} is not a base32 character`);
    }
    bits += value.toString(2).padStart(5, '0');
  }
  const bytes: number[] = [];
  for (let i = 0; i + 8 <= bits.length; i += 8) {
    bytes.push(Number.parseInt(bits.slice(i, i + 8), 2));
  }
  return Buffer.from(bytes);
}

/**
 * RFC 6238 code for `secret`, `step` periods away from now — the parameters
 * internal/auth/totp.go verifies with (SHA-1, six digits, 30 s period).
 */
export function totpCode(secret: string, step = 0): string {
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30_000) + step));
  const digest = createHmac('sha1', decodeBase32(secret)).update(counter).digest();
  const offset = (digest.at(-1) ?? 0) & 0x0f;
  return String((digest.readUInt32BE(offset) & 0x7fffffff) % 1_000_000).padStart(6, '0');
}

/** A six-digit code the server's ±1 period skew window cannot accept. */
export function wrongTotpCode(secret: string): string {
  const accepted = new Set([-1, 0, 1, 2].map((step) => totpCode(secret, step)));
  for (let candidate = 0; ; candidate++) {
    const code = String(candidate).padStart(6, '0');
    if (!accepted.has(code)) {
      return code;
    }
  }
}
