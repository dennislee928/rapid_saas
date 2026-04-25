import type { DlqItem, Endpoint, RuleDocument, TimelineEvent } from "./types";

export const endpoints: Endpoint[] = [
  {
    id: "ep_01HV9_falcon",
    name: "Falcon alerts / prod",
    preset: "CrowdStrike",
    status: "healthy",
    ingressUrl: "https://in.router.example/w/ep_01HV9_falcon",
    events24h: 18420,
    p95AckMs: 42,
    lastSeen: "37s ago"
  },
  {
    id: "ep_01HV9_github",
    name: "GitHub Advanced Security",
    preset: "GitHub HMAC",
    status: "muted",
    ingressUrl: "https://in.router.example/w/ep_01HV9_github",
    events24h: 271,
    p95AckMs: 31,
    lastSeen: "8m ago"
  },
  {
    id: "ep_01HV9_wazuh",
    name: "Wazuh manager / eu-west",
    preset: "Generic shared secret",
    status: "failing",
    ingressUrl: "https://in.router.example/w/ep_01HV9_wazuh",
    events24h: 5208,
    p95AckMs: 88,
    lastSeen: "2m ago"
  }
];

export const timelineEvents: TimelineEvent[] = [
  {
    id: "evt_01JDXR4V7N",
    endpoint: "Falcon alerts / prod",
    source: "crowdstrike",
    status: "delivered",
    severity: "critical",
    receivedAt: "12:44:18",
    latencyMs: 612,
    message: "Host containment alert routed to #secops-critical"
  },
  {
    id: "evt_01JDXR4J2M",
    endpoint: "Wazuh manager / eu-west",
    source: "wazuh",
    status: "failed",
    severity: "high",
    receivedAt: "12:43:51",
    latencyMs: 4010,
    message: "Slack destination returned 429; retry scheduled"
  },
  {
    id: "evt_01JDXR3ZRA",
    endpoint: "GitHub Advanced Security",
    source: "github",
    status: "dropped",
    severity: "medium",
    receivedAt: "12:41:09",
    message: "Rule suppressed Dependabot advisory below high severity"
  },
  {
    id: "evt_01JDXR2KQ5",
    endpoint: "Falcon alerts / prod",
    source: "crowdstrike",
    status: "dlq",
    severity: "critical",
    receivedAt: "12:36:27",
    latencyMs: 30000,
    message: "PagerDuty Events v2 destination exhausted retries"
  }
];

export const activeRuleDocument: RuleDocument = {
  endpoint_id: "ep_01HV9_falcon",
  version: 7,
  rules: [
    {
      id: "rule_critical_hosts",
      name: "Critical endpoint isolation",
      enabled: true,
      filter_jsonlogic: {
        and: [{ "==": [{ var: "severity" }, "critical"] }, { in: [{ var: "device.region" }, ["eu", "us"]] }]
      },
      transform_kind: "template",
      destination_id: "dest_slack_secops",
      on_match: "forward"
    },
    {
      id: "rule_low_noise",
      name: "Drop low-confidence detections",
      enabled: true,
      filter_jsonlogic: {
        "<": [{ var: "confidence" }, 40]
      },
      transform_kind: "passthrough",
      destination_id: "dest_null",
      on_match: "drop"
    }
  ]
};

export const dlqItems: DlqItem[] = [
  {
    id: "dlq_01JDXNY7P9",
    endpoint: "Falcon alerts / prod",
    destination: "PagerDuty primary",
    failedAt: "18m ago",
    attempts: 5,
    reason: "HTTP 503 from upstream"
  },
  {
    id: "dlq_01JDXMZ10C",
    endpoint: "Wazuh manager / eu-west",
    destination: "Slack #secops",
    failedAt: "44m ago",
    attempts: 5,
    reason: "rate_limited"
  }
];

export const quota = {
  used: 238940,
  limit: 500000,
  resetLabel: "resets May 1"
};
