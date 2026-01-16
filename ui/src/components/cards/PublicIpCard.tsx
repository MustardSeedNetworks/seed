/**
 * PublicIPCard Component
 *
 * Purpose: Displays the public IPv4 and IPv6 addresses as seen from the internet.
 * Shows when these addresses were last checked and any lookup errors.
 *
 * Key Features:
 * - Dual-stack support: shows both IPv4 and IPv6 public addresses
 * - Timestamp: displays when addresses were last verified (e.g., "5m ago", "just now")
 * - Error handling: shows error message if lookup fails
 * - Status indication: success (has IP), error (lookup failed), unknown (no data)
 * - Geolocation: displays ISP/ASN and location information when available
 * - IP History: collapsible section showing previous IP addresses
 *
 * Usage:
 * ```typescript
 * <PublicIPCard
 *   data={publicIPData}
 *   loading={isFetching}
 * />
 * ```
 *
 * Dependencies: BaseCard, Card UI components, CollapsibleSection, Globe icon, theme utilities
 * State: Receives data from parent component via props
 */

import type React from "react";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import { icon as iconTokens } from "../../styles/theme";
import { CardDivider, CardRow, CardValue, type Status } from "../ui/Card";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { Globe } from "../ui/Icons";
import { BaseCard } from "./BaseCard";

/** IP history entry for tracking address changes */
export interface IpHistoryEntry {
  ip: string;
  firstSeen: string;
  lastSeen: string;
  city?: string;
  country?: string;
}

export interface PublicIpData {
  ipv4?: string;
  ipv6?: string;
  lastChecked: string;
  error?: string;
  // Geo fields
  isp?: string;
  asn?: string;
  org?: string;
  city?: string;
  region?: string;
  country?: string;
  countryCode?: string;
  lat?: number;
  lon?: number;
  // History
  history?: IpHistoryEntry[];
}

interface PublicIpCardProps {
  data: PublicIpData | null;
  loading?: boolean;
}

function formatLastChecked(isoDate: string): string {
  try {
    const date = new Date(isoDate);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) {
      return "just now";
    }
    if (diffMins < 60) {
      return `${diffMins}m ago`;
    }

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) {
      return `${diffHours}h ago`;
    }

    return date.toLocaleDateString();
  } catch {
    return "unknown";
  }
}

/**
 * Format a date range for history display
 */
function formatDateRange(firstSeen: string, lastSeen: string): string {
  try {
    const first = new Date(firstSeen);
    const last = new Date(lastSeen);
    const firstStr = first.toLocaleDateString();
    const lastStr = last.toLocaleDateString();

    if (firstStr === lastStr) {
      return firstStr;
    }
    return `${firstStr} - ${lastStr}`;
  } catch {
    return "unknown";
  }
}

/**
 * Format ISP/ASN display string
 * Shows "AS15169 Google LLC" style format
 */
function formatIspAsn(asn?: string, org?: string, isp?: string): string | null {
  const asnPart = asn ? `AS${asn.replace(/^AS/i, "")}` : null;
  const namePart = org || isp;

  if (asnPart && namePart) {
    return `${asnPart} ${namePart}`;
  }
  if (asnPart) {
    return asnPart;
  }
  if (namePart) {
    return namePart;
  }
  return null;
}

/**
 * Format location string from geo fields
 * Shows "City, Region, Country" format, omitting missing parts
 */
function formatLocation(city?: string, region?: string, country?: string): string | null {
  const parts = [city, region, country].filter(Boolean);
  return parts.length > 0 ? parts.join(", ") : null;
}

function getStatus(data: PublicIpData): Status {
  if (data.error && !data.ipv4 && !data.ipv6) {
    return "error";
  }
  if (data.ipv4 || data.ipv6) {
    return "success";
  }
  return "unknown";
}

export const PublicIpCard: React.NamedExoticComponent<PublicIpCardProps> = memo(
  function publicIpCard({ data, loading }: PublicIpCardProps): React.ReactElement {
    const { t } = useTranslation("cards");

    return (
      <BaseCard
        title={t("publicIp.title")}
        icon={<Globe class={iconTokens.size.md} />}
        data={data}
        loading={loading}
        getStatus={getStatus}
        loadingContent={<CardValue value={t("publicIp.checking")} size="lg" />}
        emptyMessage={t("publicIp.unableToDetect")}
      >
        {(ipData: PublicIpData): React.ReactElement => {
          const ispAsnDisplay = formatIspAsn(ipData.asn, ipData.org, ipData.isp);
          const locationDisplay = formatLocation(ipData.city, ipData.region, ipData.country);
          const hasHistory = ipData.history && ipData.history.length > 0;

          return (
            <>
              {/* IPv4 Address */}
              {ipData.ipv4 ? (
                <>
                  <p class="caption font-medium">{t("publicIp.ipv4")}</p>
                  <CardValue value={ipData.ipv4} size="lg" />
                </>
              ) : (
                <>
                  <p class="caption font-medium">{t("publicIp.ipv4")}</p>
                  <p class="body-small text-text-muted">{t("publicIp.notAvailable")}</p>
                </>
              )}

              <CardDivider />

              {/* IPv6 Address */}
              {ipData.ipv6 ? (
                <>
                  <p class="caption font-medium">{t("publicIp.ipv6")}</p>
                  <p class="body-small font-mono break-all text-text-primary">{ipData.ipv6}</p>
                </>
              ) : (
                <>
                  <p class="caption font-medium">{t("publicIp.ipv6")}</p>
                  <p class="body-small text-text-muted">{t("publicIp.notAvailable")}</p>
                </>
              )}

              {/* ISP/ASN - only show if available */}
              {ispAsnDisplay ? (
                <>
                  <CardDivider />
                  <CardRow label={t("publicIp.ispAsn")} value={ispAsnDisplay} />
                </>
              ) : null}

              {/* Location - only show if available */}
              {locationDisplay ? (
                <>
                  <CardDivider />
                  <CardRow label={t("publicIp.location")} value={locationDisplay} />
                </>
              ) : null}

              {/* Last checked */}
              {ipData.lastChecked ? (
                <>
                  <CardDivider />
                  <CardRow
                    label={t("publicIp.lastChecked")}
                    value={formatLastChecked(ipData.lastChecked)}
                  />
                </>
              ) : null}

              {/* Error if any */}
              {ipData.error ? (
                <>
                  <CardDivider />
                  <p class="caption text-status-error">{ipData.error}</p>
                </>
              ) : null}

              {/* IP History - collapsible section */}
              {hasHistory ? (
                <>
                  <CardDivider />
                  <CollapsibleSection
                    title={t("publicIp.history")}
                    count={ipData.history?.length}
                    variant="compact"
                    defaultOpen={false}
                  >
                    <div class="space-y-2">
                      {ipData.history?.map((entry, index) => {
                        const entryLocation = formatLocation(entry.city, undefined, entry.country);
                        return (
                          <div key={`${entry.ip}-${index}`} class="flex flex-col gap-0.5">
                            <div class="flex justify-between items-center">
                              <span class="body-small font-mono text-text-primary">{entry.ip}</span>
                              <span class="caption text-text-muted">
                                {formatDateRange(entry.firstSeen, entry.lastSeen)}
                              </span>
                            </div>
                            {entryLocation ? (
                              <span class="caption text-text-muted">{entryLocation}</span>
                            ) : null}
                          </div>
                        );
                      })}
                    </div>
                  </CollapsibleSection>
                </>
              ) : null}
            </>
          );
        }}
      </BaseCard>
    );
  },
);
