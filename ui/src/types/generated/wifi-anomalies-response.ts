/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiAnomaliesResponse {
  anomalies: Anomaly[];
  status: Status;
}
export interface Anomaly {
  defKey: string;
  category: string;
  severity: string;
  subject: SubjectRef;
  title: string;
  description: string;
  impact?: string;
  recommendation: string;
  standards?: string[];
  evidence?: {
    [k: string]: string;
  };
  followUps?: FollowUp[];
  firstSeen: string;
  lastSeen: string;
  count: number;
}
export interface SubjectRef {
  kind: string;
  id: string;
}
export interface FollowUp {
  kind: string;
  label: string;
  action: string;
  capability?: string;
}
export interface Status {
  captureActive: boolean;
  source?: string;
  ssids: number;
  aps: number;
  bsses: number;
  stations: number;
  anomalies: number;
  lastEvaluated?: string;
}
