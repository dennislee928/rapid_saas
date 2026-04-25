import type { CollectRequest, ScoreRequest, ScoreResponse, TenantConfig, VisitTokenPayload } from "../shared/types";
import { scoreEvent } from "./scoring";
import { signVisitToken, verifyVisitToken } from "./token";

export interface Env {
  TOKEN_SECRET?: string;
  TENANTS_JSON?: string;
  EVENTS_QUEUE?: Queue<QueueEvent>;
}

export interface QueueEvent {
  type: "collect" | "score";
  tenant_id: string;
  occurred_at: number;
  payload: unknown;
}

const SDK_VERSION = "0.1.0";
const encoder = new TextEncoder();

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    return handleRequest(request, env, ctx);
  }
};

export async function handleRequest(request: Request, env: Env, ctx?: ExecutionContext): Promise<Response> {
  const started = Date.now();
  const url = new URL(request.url);

  if (request.method === "GET" && (url.pathname === "/health" || url.pathname === "/healthz")) {
    return json({ ok: true, service: "tiltguard-worker", sdk_version: SDK_VERSION });
  }

  if (request.method === "POST" && url.pathname === "/collect") {
    return collect(request, env, ctx);
  }

  if (request.method === "POST" && url.pathname === "/v1/score") {
    return score(request, env, ctx, started);
  }

  return json({ error: "not_found", message: "Use POST /collect or POST /v1/score." }, 404);
}

async function collect(request: Request, env: Env, ctx?: ExecutionContext): Promise<Response> {
  const body = await request.json<CollectRequest>();
  const tenantId = body.tenant_id || "ten_demo";
  const now = Date.now();
  const fingerprintHash = await sha256Hex(JSON.stringify(stableSignalSubset(body.signals)));
  const deviceId = `dev_${fingerprintHash.slice(0, 26)}`;
  const visitId = `vis_${crypto.randomUUID().replaceAll("-", "")}`;
  const payload: VisitTokenPayload = {
    tenant_id: tenantId,
    visit_id: visitId,
    device_id: deviceId,
    fingerprint_hash: fingerprintHash,
    signals: body.signals,
    issued_at: now,
    expires_at: now + 15 * 60 * 1000
  };
  const visitToken = await signVisitToken(payload, tokenSecret(env));

  enqueue(ctx, env, {
    type: "collect",
    tenant_id: tenantId,
    occurred_at: now,
    payload: { visit_id: visitId, device_id: deviceId, fingerprint_hash: fingerprintHash }
  });

  return json({ visit_token: visitToken, visit_id: visitId, device_id: deviceId, expires_at: payload.expires_at }, 202);
}

async function score(request: Request, env: Env, ctx: ExecutionContext | undefined, started: number): Promise<Response> {
  const tenant = authenticate(request, env);
  if (!tenant) {
    return json({ error: "unauthorized", message: "Missing or invalid bearer API key." }, 401);
  }

  const body = await request.json<ScoreRequest>();
  const visit = await verifyVisitToken(body.visit_token, tokenSecret(env));
  if (visit.tenant_id !== tenant.id) {
    return json({ error: "tenant_mismatch", message: "Visit token tenant does not match API key tenant." }, 403);
  }

  const reuse = demoReuseLookup(body, visit);
  const ruleScore = scoreEvent({ request: body, visit, tenantRules: tenant.rules, reuse });
  const response: ScoreResponse = {
    ...ruleScore,
    device_id: visit.device_id,
    cluster_id: `clu_${visit.fingerprint_hash.slice(0, 20)}`,
    linked_account_count: reuse.clusterAccountCount,
    request_id: `req_${crypto.randomUUID().replaceAll("-", "")}`,
    latency_ms: Date.now() - started
  };

  enqueue(ctx, env, {
    type: "score",
    tenant_id: tenant.id,
    occurred_at: Date.now(),
    payload: {
      account_id: body.account_id,
      event_type: body.event_type,
      device_id: response.device_id,
      cluster_id: response.cluster_id,
      score: response.score,
      recommendation: response.recommendation,
      reason_codes: response.reason_codes
    }
  });

  return json(response);
}

function authenticate(request: Request, env: Env): TenantConfig | null {
  const header = request.headers.get("authorization");
  const token = header?.match(/^Bearer\s+(.+)$/i)?.[1];
  if (!token) {
    return null;
  }
  return tenants(env)[token] ?? null;
}

function tenants(env: Env): Record<string, TenantConfig> {
  if (!env.TENANTS_JSON) {
    return {
      dev_tiltguard_demo_key: {
        id: "ten_demo",
        name: "Demo Operator",
        rules: { allowBelow: 30, denyAtOrAbove: 70, disposableEmailDomains: ["mailinator.com", "tempmail.test"] }
      }
    };
  }
  return JSON.parse(env.TENANTS_JSON) as Record<string, TenantConfig>;
}

function tokenSecret(env: Env): string {
  return env.TOKEN_SECRET ?? "dev-only-token-secret";
}

function demoReuseLookup(request: ScoreRequest, visit: VisitTokenPayload) {
  const accountHint = request.account_id.toLowerCase();
  const riskyDevice = visit.fingerprint_hash.endsWith("0") || accountHint.includes("ring");
  return {
    fingerprintAccounts30d: riskyDevice ? 4 : 0,
    clusterAccountCount: riskyDevice ? 5 : 1,
    signupCount24h: request.event_type === "signup" && riskyDevice ? 7 : 1,
    ipSignupCount1h: request.client?.is_datacenter ? 4 : 1
  };
}

function stableSignalSubset(signals: CollectRequest["signals"]) {
  return {
    canvasHash: signals.canvasHash,
    webglVendor: signals.webglVendor,
    webglRenderer: signals.webglRenderer,
    audioHash: signals.audioHash,
    fontsHash: signals.fontsHash,
    screen: signals.screen,
    timezone: signals.timezone,
    language: signals.language
  };
}

function enqueue(ctx: ExecutionContext | undefined, env: Env, event: QueueEvent): void {
  if (!env.EVENTS_QUEUE) {
    return;
  }
  const send = () => env.EVENTS_QUEUE ? env.EVENTS_QUEUE.send(event) : Promise.resolve(undefined);
  if (ctx) {
    ctx.waitUntil(send());
    return;
  }
  void send();
}

async function sha256Hex(value: string): Promise<string> {
  const bytes = new Uint8Array(await crypto.subtle.digest("SHA-256", encoder.encode(value)));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body, null, 2), {
    status,
    headers: { "content-type": "application/json; charset=utf-8" }
  });
}
