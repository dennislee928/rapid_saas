export type EndpointStatus = "healthy" | "muted" | "failing";

export type Endpoint = {
  id: string;
  name: string;
  preset: string;
  status: EndpointStatus;
  ingressUrl: string;
  events24h: number;
  p95AckMs: number;
  lastSeen: string;
};

export type EventStatus = "queued" | "delivered" | "failed" | "dropped" | "dlq";

export type TimelineEvent = {
  id: string;
  endpoint: string;
  source: string;
  status: EventStatus;
  severity: "critical" | "high" | "medium" | "low";
  receivedAt: string;
  latencyMs?: number;
  message: string;
};

export type RuleDocument = {
  endpoint_id: string;
  version: number;
  rules: Array<{
    id: string;
    name: string;
    enabled: boolean;
    filter_jsonlogic: Record<string, unknown>;
    transform_kind: "passthrough" | "template" | "preset";
    destination_id: string;
    on_match: "forward" | "drop" | "continue";
  }>;
};

export type DlqItem = {
  id: string;
  endpoint: string;
  destination: string;
  failedAt: string;
  attempts: number;
  reason: string;
};
