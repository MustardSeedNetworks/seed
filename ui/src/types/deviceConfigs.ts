/**
 * Wire shapes of the configuration backup reads (P-D1, P-D2). The routes put
 * internal/deviceconfig's Target, Backup and Diff on the wire directly and
 * publish no schema, so these mirror deviceconfig.go by hand.
 */

export type DeviceConfigPlatform =
  | 'cisco-ios'
  | 'cisco-nxos'
  | 'arista-eos'
  | 'juniper-junos'
  | 'netconf';

export type DeviceConfigStatus =
  | 'ok'
  | 'auth_failed'
  | 'host_key_mismatch'
  | 'host_key_unconfirmed'
  | 'unreachable'
  | 'command_failed'
  | 'error';

export interface DeviceConfigTarget {
  id: string;
  name: string;
  host: string;
  port: number;
  platform: DeviceConfigPlatform;
  /** The SSH credential in the vault the backup signs in with. */
  credentialsId: string;
  /** The SSH host key fingerprint every backup must match. */
  hostKeySha256?: string;
  /** The key the device presented while none was pinned, awaiting confirmation. */
  pendingHostKeySha256?: string;
  updatedAt: string;
}

/** The target POST/PUT body. A port left out takes the platform's default. */
export interface DeviceConfigTargetInput {
  name: string;
  host: string;
  port?: number;
  platform: DeviceConfigPlatform;
  credentialsId: string;
}

export interface DeviceConfigTargetList {
  count: number;
  targets: DeviceConfigTarget[];
}

/** One backup attempt; a list leaves the configuration out. */
export interface DeviceConfigBackup {
  id: string;
  targetId: string;
  takenAt: string;
  status: DeviceConfigStatus;
  error?: string;
  sha256?: string;
}

export interface DeviceConfigBackupList {
  count: number;
  backups: DeviceConfigBackup[];
}

/** The finished backup run's job result. */
export interface DeviceConfigRunResult {
  succeeded: number;
  failed: number;
}

/** A backup's change from the target's previous configuration. */
export interface DeviceConfigDiff {
  targetId: string;
  backupId: string;
  takenAt: string;
  previousBackupId: string;
  previousTakenAt: string;
  added: number;
  removed: number;
  /** The change in unified diff format. */
  unified: string;
}
