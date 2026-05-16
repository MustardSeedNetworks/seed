/**
 * Specialty protocol sections rendered inside HealthCheckCard.
 *
 * Covers SQL, file shares, LDAP, RTSP, DICOM, HL7 MLLP, FHIR, LTI/LMS,
 * OPC-UA, and Modbus TCP. Each section is gated on its result array
 * being non-empty; the parent HealthCheckCard already handles the
 * ping/TCP/UDP/HTTP sections inline.
 */

import type { TFunction } from 'i18next';
import { cn, layout, radius, spacing } from '../../styles/theme';
import { CollapsibleSection } from '../ui/CollapsibleSection';
import { StatusBadge } from '../ui/StatusBadge';
import type { HealthCheckData } from './healthCheckCardTypes';

interface HealthCheckCardProtocolSectionsProps {
  data: HealthCheckData;
  t: TFunction;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Renders ten optional protocol sections each gated on result-array presence
export function HealthCheckCardProtocolSections({
  data,
  t,
}: HealthCheckCardProtocolSectionsProps): JSX.Element {
  return (
    <>
      {/* SQL/Database Results */}
      {data.enterpriseResults?.sqlResults && data.enterpriseResults.sqlResults.length > 0 ? (
        <CollapsibleSection
          title={t('health.database')}
          count={data.enterpriseResults.sqlResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.enterpriseResults.sqlResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: SQL result rendering with connect/query metrics */}
          {data.enterpriseResults.sqlResults.map((r) => (
            <div
              key={`sql-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted">
                    {r.driver} • {r.host}:{r.port}
                  </span>
                </div>
                {r.success && r.serverVersion ? (
                  <span class="caption text-text-muted ml-6">{r.serverVersion}</span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success ? (
                  <div class="caption text-text-muted">
                    Connect: {r.connectTimeMs.toFixed(1)}ms
                    {r.queryTimeMs !== undefined ? ` • Query: ${r.queryTimeMs.toFixed(1)}ms` : null}
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* File Share Results (SMB/NFS) */}
      {data.enterpriseResults?.fileShareResults &&
      data.enterpriseResults.fileShareResults.length > 0 ? (
        <CollapsibleSection
          title={t('health.fileShares')}
          count={data.enterpriseResults.fileShareResults.length}
          variant="compact"
          defaultOpen={true}
          status={
            data.enterpriseResults.fileShareResults.some((r) => !r.success) ? 'error' : 'success'
          }
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: File share result rendering with read/write speeds */}
          {data.enterpriseResults.fileShareResults.map((r) => (
            <div
              key={`fileshare-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted">
                    {r.protocol.toUpperCase()} • /{r.host}/{r.share}
                  </span>
                </div>
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.connectTimeMs.toFixed(1)}ms</div>
                {r.success && (r.readSpeedMbps !== undefined || r.writeSpeedMbps !== undefined) ? (
                  <div class="caption text-text-muted">
                    {r.readSpeedMbps !== undefined ? `R: ${r.readSpeedMbps.toFixed(1)} MB/s` : null}
                    {r.readSpeedMbps !== undefined && r.writeSpeedMbps !== undefined ? ' • ' : null}
                    {r.writeSpeedMbps !== undefined
                      ? `W: ${r.writeSpeedMbps.toFixed(1)} MB/s`
                      : null}
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* LDAP Results */}
      {data.enterpriseResults?.ldapResults && data.enterpriseResults.ldapResults.length > 0 ? (
        <CollapsibleSection
          title="LDAP"
          count={data.enterpriseResults.ldapResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.enterpriseResults.ldapResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: LDAP result rendering with connect/bind metrics */}
          {data.enterpriseResults.ldapResults.map((r) => (
            <div
              key={`ldap-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted">
                    {r.useTls ? 'LDAPS' : 'LDAP'} • {r.host}:{r.port}
                  </span>
                </div>
                {r.success && r.serverInfo ? (
                  <span class="caption text-text-muted ml-6">{r.serverInfo}</span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success ? (
                  <div class="caption text-text-muted">
                    Connect: {r.connectTimeMs.toFixed(1)}ms
                    {r.bindTimeMs !== undefined ? ` • Bind: ${r.bindTimeMs.toFixed(1)}ms` : null}
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* RTSP Video Results */}
      {data.videoResults?.rtspResults && data.videoResults.rtspResults.length > 0 ? (
        <CollapsibleSection
          title={t('health.rtsp')}
          count={data.videoResults.rtspResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.videoResults.rtspResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: RTSP result rendering with codec/resolution info */}
          {data.videoResults.rtspResults.map((r) => (
            <div
              key={`rtsp-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted truncate max-w-48" title={r.url}>
                    {r.url}
                  </span>
                </div>
                {r.success && (r.codec || r.resolution) ? (
                  <span class="caption text-text-muted ml-6">
                    {r.codec ? r.codec : null}
                    {r.codec && r.resolution ? ' • ' : null}
                    {r.resolution ? r.resolution : null}
                  </span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.connectTimeMs.toFixed(1)}ms</div>
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* DICOM Results */}
      {data.medicalResults?.dicomResults && data.medicalResults.dicomResults.length > 0 ? (
        <CollapsibleSection
          title="DICOM"
          count={data.medicalResults.dicomResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.medicalResults.dicomResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: DICOM result rendering with AE title and C-ECHO metrics */}
          {data.medicalResults.dicomResults.map((r) => (
            <div
              key={`dicom-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted">
                    {r.host}:{r.port} • AE: {r.aeTitle}
                  </span>
                </div>
                {r.success && r.serverAeTitle ? (
                  <span class="caption text-text-muted ml-6">Server AE: {r.serverAeTitle}</span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success && r.echoTimeMs !== undefined ? (
                  <div class="caption text-text-muted">C-ECHO: {r.echoTimeMs.toFixed(1)}ms</div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* HL7 MLLP Results */}
      {data.medicalResults?.hl7Results && data.medicalResults.hl7Results.length > 0 ? (
        <CollapsibleSection
          title="HL7 MLLP"
          count={data.medicalResults.hl7Results.length}
          variant="compact"
          defaultOpen={true}
          status={data.medicalResults.hl7Results.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: HL7 result rendering with ACK code and response metrics */}
          {data.medicalResults.hl7Results.map((r) => (
            <div
              key={`hl7-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted">
                    {r.host}:{r.port}
                  </span>
                </div>
                {r.success && (r.ackCode || r.serverVersion) ? (
                  <span class="caption text-text-muted ml-6">
                    {r.ackCode ? `ACK: ${r.ackCode}` : null}
                    {r.ackCode && r.serverVersion ? ' • ' : null}
                    {r.serverVersion ? r.serverVersion : null}
                  </span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success && r.responseTimeMs !== undefined ? (
                  <div class="caption text-text-muted">
                    Response: {r.responseTimeMs.toFixed(1)}ms
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* FHIR Results */}
      {data.medicalResults?.fhirResults && data.medicalResults.fhirResults.length > 0 ? (
        <CollapsibleSection
          title="FHIR R4"
          count={data.medicalResults.fhirResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.medicalResults.fhirResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: FHIR result rendering with version and resource count */}
          {data.medicalResults.fhirResults.map((r) => (
            <div
              key={`fhir-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted truncate max-w-48" title={r.baseUrl}>
                    {r.baseUrl}
                  </span>
                </div>
                {r.success && (r.fhirVersion || r.serverName) ? (
                  <span class="caption text-text-muted ml-6">
                    {r.fhirVersion ? `v${r.fhirVersion}` : null}
                    {r.fhirVersion && r.serverName ? ' • ' : null}
                    {r.serverName ? r.serverName : null}
                  </span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success && r.resourceCount !== undefined ? (
                  <div class="caption text-text-muted">{r.resourceCount} resources</div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* LTI/LMS Results */}
      {data.educationResults?.ltiResults && data.educationResults.ltiResults.length > 0 ? (
        <CollapsibleSection
          title="LTI/LMS"
          count={data.educationResults.ltiResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.educationResults.ltiResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {data.educationResults.ltiResults.map((r) => (
            <div
              key={`lti-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted truncate max-w-48" title={r.launchUrl}>
                    {r.launchUrl}
                  </span>
                </div>
                {r.success && r.ltiVersion ? (
                  <span class="caption text-text-muted ml-6">LTI {r.ltiVersion}</span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* OPC-UA Results */}
      {data.industrialResults?.opcuaResults && data.industrialResults.opcuaResults.length > 0 ? (
        <CollapsibleSection
          title="OPC-UA"
          count={data.industrialResults.opcuaResults.length}
          variant="compact"
          defaultOpen={true}
          status={data.industrialResults.opcuaResults.some((r) => !r.success) ? 'error' : 'success'}
        >
          {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: OPC-UA result rendering with security and product info */}
          {data.industrialResults.opcuaResults.map((r) => (
            <div
              key={`opcua-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted truncate max-w-48" title={r.endpointUrl}>
                    {r.endpointUrl}
                  </span>
                </div>
                {r.success && (r.securityMode || r.productName) ? (
                  <span class="caption text-text-muted ml-6">
                    {r.securityMode ? r.securityMode : null}
                    {r.securityMode && r.productName ? ' • ' : null}
                    {r.productName ? r.productName : null}
                  </span>
                ) : null}
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success && r.serverState ? (
                  <div class="caption text-text-muted">{r.serverState}</div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}

      {/* Modbus TCP Results */}
      {data.industrialResults?.modbusResults && data.industrialResults.modbusResults.length > 0 ? (
        <CollapsibleSection
          title="Modbus TCP"
          count={data.industrialResults.modbusResults.length}
          variant="compact"
          defaultOpen={true}
          status={
            data.industrialResults.modbusResults.some((r) => !r.success) ? 'error' : 'success'
          }
        >
          {data.industrialResults.modbusResults.map((r) => (
            <div
              key={`modbus-${r.name}`}
              class={cn(
                layout.flex.between,
                spacing.pad.xs,
                radius.default,
                r.success ? 'bg-surface-raised' : 'bg-status-error/10',
              )}
            >
              <div class={layout.stack.compact}>
                <div class="flex items-center gap-2">
                  <StatusBadge status={r.success ? 'success' : 'error'} />
                  <span class="body-small font-medium">{r.name}</span>
                  <span class="caption text-text-muted">
                    {r.host}:{r.port} • Unit {r.unitId}
                  </span>
                </div>
                {r.error ? <span class="caption text-status-error ml-6">{r.error}</span> : null}
              </div>
              <div class="text-right">
                <div class="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                {r.success && r.registerValue !== undefined ? (
                  <div class="caption text-text-muted">
                    Reg: 0x{r.registerValue.toString(16).toUpperCase().padStart(4, '0')}
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </CollapsibleSection>
      ) : null}
    </>
  );
}
