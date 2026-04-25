export type EventType = "signup" | "login" | "deposit" | "custom";
export type Recommendation = "allow" | "review" | "deny";

export interface BehaviourSignals {
  mouseEvents?: number;
  mouseStraightLineRatio?: number;
  typingCadenceStddevMs?: number;
  pasteEvents?: number;
}

export interface BrowserSignals {
  sdkVersion?: string;
  userAgent?: string;
  language?: string;
  languages?: string[];
  timezone?: string;
  screen?: {
    width?: number;
    height?: number;
    colorDepth?: number;
    devicePixelRatio?: number;
  };
  hardwareConcurrency?: number;
  deviceMemory?: number;
  webdriver?: boolean;
  canvasHash?: string;
  webglVendor?: string;
  webglRenderer?: string;
  audioHash?: string;
  fontsHash?: string;
  storageQuota?: number;
  behaviour?: BehaviourSignals;
}

export interface CollectRequest {
  tenant_id?: string;
  session_id?: string;
  signals: BrowserSignals;
}

export interface ScoreRequest {
  visit_token: string;
  account_id: string;
  event_type: EventType;
  context?: {
    promo_id?: string;
    deposit_amount_minor?: number;
    deposit_currency?: string;
    email_hash?: string;
    email_domain?: string;
    phone_hash?: string;
  };
  client?: {
    ip?: string;
    user_agent?: string;
    accept_language?: string;
    country?: string;
    asn?: number;
    ja3?: string;
    ja4?: string;
    is_datacenter?: boolean;
    is_residential_proxy?: boolean;
  };
}

export interface ReasonCode {
  code: string;
  weight: number;
  detail: string;
}

export interface ScoreResponse {
  score: number;
  recommendation: Recommendation;
  reason_codes: ReasonCode[];
  device_id: string;
  cluster_id: string;
  linked_account_count: number;
  request_id: string;
  latency_ms: number;
}

export interface TenantRuleConfig {
  allowBelow: number;
  denyAtOrAbove: number;
  countryDenylist?: string[];
  disposableEmailDomains?: string[];
}

export interface TenantConfig {
  id: string;
  name: string;
  rules: TenantRuleConfig;
}

export interface VisitTokenPayload {
  tenant_id: string;
  visit_id: string;
  device_id: string;
  fingerprint_hash: string;
  signals: BrowserSignals;
  issued_at: number;
  expires_at: number;
}
