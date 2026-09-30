/**
 * Specialty protocol sections rendered inside HealthCheckCard.
 *
 * Covers SQL, file shares, LDAP, RTSP, DICOM, HL7 MLLP, FHIR, LTI/LMS,
 * OPC-UA, and Modbus TCP. Each section is gated on its result array
 * being non-empty; the parent HealthCheckCard already handles the
 * ping/TCP/UDP/HTTP sections inline.
 */

import type { TFunction } from 'i18next';
import type { JSX } from 'react';
import { cn, layout, radius, spacing, status as statusColor } from '../../styles/theme';
import { CollapsibleSection } from '../ui/CollapsibleSection';
import { StatusBadge } from '../ui/StatusBadge';
import { Tooltip } from '../ui/Tooltip';
import type { HealthCheckData } from './healthCheckCardTypes';
import { failuresFirst } from './healthCheckResultOrder';

/** One protocol result, reduced to what its row shows. */
interface ProtocolRow {
  name: string;
  target: string;
  success: boolean;
  totalMs: number;
  /** Secondary facts, shown only for a successful check. */
  details: (string | null)[];
  error?: string;
}

interface ProtocolSection {
  id: string;
  title: string;
  rows: ProtocolRow[];
}

const ms = (v: number): string => `${v.toFixed(1)}ms`;

function sections(data: HealthCheckData, t: TFunction<'cards'>): ProtocolSection[] {
  const { enterpriseResults: ent, videoResults: video, medicalResults: med } = data;
  const { educationResults: edu, industrialResults: ind } = data;
  const optional = <V,>(v: V | undefined, f: (v: V) => string): string | null =>
    v === undefined || v === '' ? null : f(v);

  return [
    {
      id: 'sql',
      title: t('health.database'),
      rows: (ent?.sqlResults ?? []).map((r) => ({
        name: r.name,
        target: `${r.driver} • ${r.host}:${r.port}`,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [
          t('health.detail.connect', { ms: ms(r.connectTimeMs) }),
          optional(r.queryTimeMs, (v) => t('health.detail.query', { ms: ms(v) })),
          r.serverVersion ?? null,
        ],
        error: r.error,
      })),
    },
    {
      id: 'fileshare',
      title: t('health.fileShares'),
      rows: (ent?.fileShareResults ?? []).map((r) => ({
        name: r.name,
        target: `${r.protocol.toUpperCase()} • /${r.host}/${r.share}`,
        success: r.success,
        totalMs: r.connectTimeMs,
        details: [
          optional(r.readSpeedMbps, (v) => t('health.detail.read', { rate: v.toFixed(1) })),
          optional(r.writeSpeedMbps, (v) => t('health.detail.write', { rate: v.toFixed(1) })),
        ],
        error: r.error,
      })),
    },
    {
      id: 'ldap',
      title: 'LDAP',
      rows: (ent?.ldapResults ?? []).map((r) => ({
        name: r.name,
        target: `${r.useTls ? 'LDAPS' : 'LDAP'} • ${r.host}:${r.port}`,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [
          t('health.detail.connect', { ms: ms(r.connectTimeMs) }),
          optional(r.bindTimeMs, (v) => t('health.detail.bind', { ms: ms(v) })),
          r.serverInfo ?? null,
        ],
        error: r.error,
      })),
    },
    {
      id: 'rtsp',
      title: t('health.rtsp'),
      rows: (video?.rtspResults ?? []).map((r) => ({
        name: r.name,
        target: r.url,
        success: r.success,
        totalMs: r.connectTimeMs,
        details: [r.codec ?? null, r.resolution ?? null],
        error: r.error,
      })),
    },
    {
      id: 'dicom',
      title: 'DICOM',
      rows: (med?.dicomResults ?? []).map((r) => ({
        name: r.name,
        target: `${r.host}:${r.port} • ${t('health.detail.aeTitle', { title: r.aeTitle })}`,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [
          optional(r.echoTimeMs, (v) => t('health.detail.echo', { ms: ms(v) })),
          optional(r.serverAeTitle, (v) => t('health.detail.serverAeTitle', { title: v })),
        ],
        error: r.error,
      })),
    },
    {
      id: 'hl7',
      title: 'HL7 MLLP',
      rows: (med?.hl7Results ?? []).map((r) => ({
        name: r.name,
        target: `${r.host}:${r.port}`,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [
          optional(r.responseTimeMs, (v) => t('health.detail.response', { ms: ms(v) })),
          optional(r.ackCode, (v) => t('health.detail.ack', { code: v })),
          r.serverVersion ?? null,
        ],
        error: r.error,
      })),
    },
    {
      id: 'fhir',
      title: 'FHIR R4',
      rows: (med?.fhirResults ?? []).map((r) => ({
        name: r.name,
        target: r.baseUrl,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [
          optional(r.fhirVersion, (v) => `v${v}`),
          r.serverName ?? null,
          optional(r.resourceCount, (v) => t('health.detail.resources', { count: v })),
        ],
        error: r.error,
      })),
    },
    {
      id: 'lti',
      title: 'LTI/LMS',
      rows: (edu?.ltiResults ?? []).map((r) => ({
        name: r.name,
        target: r.launchUrl,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [optional(r.ltiVersion, (v) => `LTI ${v}`)],
        error: r.error,
      })),
    },
    {
      id: 'opcua',
      title: 'OPC-UA',
      rows: (ind?.opcuaResults ?? []).map((r) => ({
        name: r.name,
        target: r.endpointUrl,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [r.serverState ?? null, r.securityMode ?? null, r.productName ?? null],
        error: r.error,
      })),
    },
    {
      id: 'modbus',
      title: 'Modbus TCP',
      rows: (ind?.modbusResults ?? []).map((r) => ({
        name: r.name,
        target: `${r.host}:${r.port} • ${t('health.detail.unit', { id: r.unitId })}`,
        success: r.success,
        totalMs: r.totalTimeMs,
        details: [
          optional(r.registerValue, (v) =>
            t('health.detail.register', {
              value: `0x${v.toString(16).toUpperCase().padStart(4, '0')}`,
            }),
          ),
        ],
        error: r.error,
      })),
    },
  ];
}

/**
 * The left column may shrink (min-w-0): its name and target truncate, with the
 * full value on the tooltip, and the details wrap. A flex child without it
 * cannot shrink below its content, which is what pushed the total past the
 * card's edge. The total is the one value that never gives way.
 */
function ProtocolResultRow({ row }: { row: ProtocolRow }): JSX.Element {
  const details = row.success ? row.details.filter((d): d is string => d !== null) : [];
  return (
    <div
      className={cn(
        'flex items-start justify-between',
        spacing.gap.compact,
        spacing.pad.xs,
        radius.default,
        row.success ? 'bg-surface-raised' : statusColor.bg.errorSoft,
      )}
    >
      <div className={cn('flex min-w-0 flex-1 items-start', spacing.gap.compact)}>
        <StatusBadge status={row.success ? 'success' : 'error'} />
        <div className={cn(layout.stack.tight, 'min-w-0 flex-1')}>
          <Tooltip text={row.name}>
            <span className="body-small font-medium truncate">{row.name}</span>
          </Tooltip>
          <Tooltip text={row.target}>
            <span className="caption text-text-muted truncate">{row.target}</span>
          </Tooltip>
          {details.length > 0 ? (
            <span className="caption text-text-muted break-words">{details.join(' • ')}</span>
          ) : null}
          {row.error ? (
            <span className="caption text-status-error break-words">{row.error}</span>
          ) : null}
        </div>
      </div>
      <span className="body-small font-mono shrink-0">{ms(row.totalMs)}</span>
    </div>
  );
}

interface HealthCheckCardProtocolSectionsProps {
  data: HealthCheckData;
  t: TFunction<'cards'>;
}

export function HealthCheckCardProtocolSections({
  data,
  t,
}: HealthCheckCardProtocolSectionsProps): JSX.Element {
  return (
    <>
      {sections(data, t)
        .filter((s) => s.rows.length > 0)
        .map((s) => (
          <CollapsibleSection
            key={s.id}
            title={s.title}
            count={s.rows.length}
            variant="compact"
            defaultOpen={true}
            status={s.rows.some((r) => !r.success) ? 'error' : 'success'}
          >
            {failuresFirst(s.rows, (r) => (r.success ? 2 : 0)).map((r) => (
              <ProtocolResultRow key={`${s.id}-${r.name}`} row={r} />
            ))}
          </CollapsibleSection>
        ))}
    </>
  );
}
