/**
 * Wire shape of /api/v1/tftp/session (P-D4). The route puts internal/tftp's
 * Status on the wire directly and publishes no schema, so this mirrors
 * tftp.go by hand. A stopped session carries no details, so the two states
 * are separate types rather than one with every field optional.
 */

export interface TftpSessionStopped {
  running: false;
}

export interface TftpSessionRunning {
  running: true;
  interface: string;
  address: string;
  port: number;
  /** The directory served for reads: `<data dir>/tftp`. */
  directory: string;
  /** Where uploads land: `<data dir>/tftp-incoming`. */
  uploadDirectory: string;
  allowUpload: boolean;
  /** IPv4 prefixes allowed to connect; absent means any host on the segment. */
  allowFrom?: string[];
  startedAt: string;
  /** The hard stop, however busy the session is. */
  expiresAt: string;
  idleTimeoutSeconds: number;
  activeTransfers: number;
}

export type TftpSession = TftpSessionStopped | TftpSessionRunning;

export interface TftpStartRequest {
  interface: string;
  allowUpload: boolean;
}
