import { activeRuleDocument, dlqItems, endpoints, quota, timelineEvents } from "./mock-data";
import type { DlqItem, Endpoint, RuleDocument, TimelineEvent } from "./types";

const TENANT_ID = process.env.ROUTER_TENANT_ID ?? "tenant_demo";
const API_BASE = process.env.ROUTER_API_BASE_URL;

type RouterEndpoint = {
  id: string;
  name: string;
  source_preset?: string;
  enabled: boolean;
};

type DeliveryLog = {
  id: string;
  endpoint_id: string;
  status: TimelineEvent["status"];
  http_status?: number;
  latency_ms?: number;
  error?: string;
  received_at: number;
};

type DlqEntry = {
  id: string;
  endpoint_id: string;
  destination_id?: string;
  last_error?: string;
  attempts: number;
  parked_at: number;
};

type UsageSummary = {
  ingressed: number;
  forwarded: number;
  failed: number;
};

type RouterRule = {
  id: string;
  endpoint_id: string;
  name: string;
  enabled: boolean;
  filter_jsonlogic?: Record<string, unknown>;
  transform_kind: RuleDocument["rules"][number]["transform_kind"];
  destination_id?: string;
  on_match: RuleDocument["rules"][number]["on_match"];
};

export async function getDashboardData() {
  const [liveEndpoints, liveEvents, liveQuota] = await Promise.all([
    fetchJSON<RouterEndpoint[]>("/v1/endpoints"),
    fetchJSON<DeliveryLog[]>("/v1/delivery-logs?limit=20"),
    fetchJSON<UsageSummary>("/v1/usage")
  ]);

  return {
    endpoints: liveEndpoints ? mapEndpoints(liveEndpoints) : endpoints,
    timelineEvents: liveEvents ? mapEvents(liveEvents, liveEndpoints ?? []) : timelineEvents,
    quota: liveQuota ? mapQuota(liveQuota) : quota
  };
}

export async function getDlqData() {
  const [liveDlq, liveQuota] = await Promise.all([
    fetchJSON<DlqEntry[]>("/v1/dlq?limit=50"),
    fetchJSON<UsageSummary>("/v1/usage")
  ]);

  return {
    dlqItems: liveDlq ? mapDlq(liveDlq) : dlqItems,
    quota: liveQuota ? mapQuota(liveQuota) : quota
  };
}

export async function getRulesData() {
  const [liveRules, liveQuota] = await Promise.all([
    fetchJSON<RouterRule[]>("/v1/rules"),
    fetchJSON<UsageSummary>("/v1/usage")
  ]);

  return {
    activeRuleDocument: liveRules ? mapRules(liveRules) : activeRuleDocument,
    quota: liveQuota ? mapQuota(liveQuota) : quota
  };
}

async function fetchJSON<T>(path: string): Promise<T | null> {
  if (!API_BASE) {
    return null;
  }
  try {
    const response = await fetch(`${API_BASE.replace(/\/$/, "")}${path}`, {
      headers: { "x-tenant-id": TENANT_ID },
      next: { revalidate: 5 }
    });
    if (!response.ok) {
      return null;
    }
    return response.json() as Promise<T>;
  } catch {
    return null;
  }
}

function mapEndpoints(items: RouterEndpoint[]): Endpoint[] {
  return items.map((item) => ({
    id: item.id,
    name: item.name,
    preset: item.source_preset ?? "generic",
    status: item.enabled ? "healthy" : "muted",
    ingressUrl: `/w/${item.id}`,
    events24h: 0,
    p95AckMs: 0,
    lastSeen: "live API"
  }));
}

function mapEvents(items: DeliveryLog[], endpointList: RouterEndpoint[]): TimelineEvent[] {
  const names = new Map(endpointList.map((endpoint) => [endpoint.id, endpoint.name]));
  return items.map((item) => ({
    id: item.id,
    endpoint: names.get(item.endpoint_id) ?? item.endpoint_id,
    source: "router-api",
    status: item.status,
    severity: item.status === "failed" || item.status === "dlq" ? "high" : "low",
    receivedAt: formatTime(item.received_at),
    latencyMs: item.latency_ms,
    message: item.error || `Delivery ${item.status}${item.http_status ? ` with HTTP ${item.http_status}` : ""}`
  }));
}

function mapDlq(items: DlqEntry[]): DlqItem[] {
  return items.map((item) => ({
    id: item.id,
    endpoint: item.endpoint_id,
    destination: item.destination_id ?? "unknown destination",
    failedAt: formatTime(item.parked_at),
    attempts: item.attempts,
    reason: item.last_error ?? "parked"
  }));
}

function mapQuota(summary: UsageSummary) {
  return {
    used: summary.ingressed,
    limit: 500000,
    resetLabel: `${summary.forwarded} forwarded / ${summary.failed} failed`
  };
}

function mapRules(rules: RouterRule[]): RuleDocument {
  return {
    endpoint_id: rules[0]?.endpoint_id ?? "ep_live",
    version: 1,
    rules: rules.map((rule) => ({
      id: rule.id,
      name: rule.name,
      enabled: rule.enabled,
      filter_jsonlogic: rule.filter_jsonlogic ?? {},
      transform_kind: rule.transform_kind,
      destination_id: rule.destination_id ?? "",
      on_match: rule.on_match
    }))
  };
}

function formatTime(ms: number) {
  if (!ms) {
    return "unknown";
  }
  return new Intl.DateTimeFormat("en", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  }).format(new Date(ms));
}
