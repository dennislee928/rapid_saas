export interface Env {
  SECURITY_ENDPOINTS?: KVNamespace;
  SECURITY_WEBHOOK_QUEUE?: Queue<WebhookQueueMessage>;
  RATE_LIMITER?: DurableObjectNamespace;
  BODY_CAP_BYTES?: string;
  ENDPOINTS_JSON?: string;
  LIBSQL_ENDPOINTS_URL?: string;
  LIBSQL_AUTH_TOKEN?: string;
  DEFAULT_RATE_LIMIT_COUNT?: string;
  DEFAULT_RATE_LIMIT_WINDOW_SECONDS?: string;
  [secretName: string]: unknown;
}

type Provider = "github" | "stripe" | "slack" | "generic";

interface EndpointConfig {
  id: string;
  tenantId?: string;
  provider: Provider;
  enabled?: boolean;
  secret?: string;
  secretEnv?: string;
  bodyCapBytes?: number;
  rateLimit?: {
    count: number;
    windowSeconds: number;
  };
  queueTopic?: string;
  metadata?: Record<string, unknown>;
}

export interface WebhookQueueMessage {
  event_id: string;
  tenant_id: string;
  type: "security.webhook.received";
  schema_version: 1;
  occurred_at: string;
  idempotency_key: string;
  payload: WebhookQueuePayload;
  trace_id?: string;
  endpointId: string;
  tenantId: string;
  provider: Provider;
  receivedAt: string;
  headers: Record<string, string>;
  bodyBase64: string;
  bodySha256: string;
  sourceIp: string;
  topic?: string;
  metadata?: Record<string, unknown>;
}

interface WebhookQueuePayload {
  endpointId: string;
  tenantId: string;
  provider: Provider;
  receivedAt: string;
  headers: Record<string, string>;
  bodyBase64: string;
  bodySha256: string;
  sourceIp: string;
  topic?: string;
  metadata?: Record<string, unknown>;
}

interface RateLimitRequest {
  key: string;
  count: number;
  windowSeconds: number;
}

const DEFAULT_BODY_CAP_BYTES = 1024 * 1024;
const DEFAULT_RATE_LIMIT_COUNT = 60;
const DEFAULT_RATE_LIMIT_WINDOW_SECONDS = 60;
const TEXT_ENCODER = new TextEncoder();
const HEX = "0123456789abcdef";

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    return handleRequest(request, env, ctx);
  }
};

export async function handleRequest(request: Request, env: Env, _ctx?: ExecutionContext): Promise<Response> {
  const url = new URL(request.url);

  if (request.method === "GET" && (url.pathname === "/health" || url.pathname === "/healthz")) {
    return json({
      ok: true,
      service: "security-webhook-ingress",
      bindings: {
        endpointsKv: Boolean(env.SECURITY_ENDPOINTS),
        queue: Boolean(env.SECURITY_WEBHOOK_QUEUE),
        rateLimiter: Boolean(env.RATE_LIMITER),
        libsql: Boolean(env.LIBSQL_ENDPOINTS_URL)
      }
    });
  }

  const match = url.pathname.match(/^\/(?:w|webhooks|ingest)\/([A-Za-z0-9_.:-]+)$/);
  if (!match) {
    return json({ error: "not_found", message: "Use POST /w/:endpointId, /webhooks/:endpointId, or /ingest/:endpointId." }, 404);
  }

  if (request.method !== "POST") {
    return json({ error: "method_not_allowed", message: "Webhook endpoints only accept POST." }, 405);
  }

  const endpointId = match[1];
  const endpoint = await resolveEndpoint(endpointId, env);
  if (!endpoint) {
    return json({ error: "endpoint_not_found", message: `No webhook endpoint config found for '${endpointId}'.` }, 404);
  }

  if (endpoint.enabled === false) {
    return json({ error: "endpoint_disabled", message: `Webhook endpoint '${endpointId}' is disabled.` }, 403);
  }

  const bodyCapBytes = endpoint.bodyCapBytes ?? intVar(env.BODY_CAP_BYTES, DEFAULT_BODY_CAP_BYTES);
  const body = await readBodyWithCap(request, bodyCapBytes);
  if (!body.ok) {
    return json({ error: "body_too_large", message: `Request body exceeds ${bodyCapBytes} bytes.` }, 413);
  }

  const secret = resolveSecret(endpoint, env);
  if (!secret) {
    return json({
      error: "secret_not_configured",
      message: "Endpoint must provide secret or secretEnv pointing to an environment secret."
    }, 500);
  }

  const verification = await verifyWebhook(endpoint.provider, request.headers, body.bytes, secret);
  if (!verification.ok) {
    return json({ error: "signature_verification_failed", message: verification.message }, 401);
  }

  const rate = endpoint.rateLimit ?? {
    count: intVar(env.DEFAULT_RATE_LIMIT_COUNT, DEFAULT_RATE_LIMIT_COUNT),
    windowSeconds: intVar(env.DEFAULT_RATE_LIMIT_WINDOW_SECONDS, DEFAULT_RATE_LIMIT_WINDOW_SECONDS)
  };
  const sourceIp = request.headers.get("cf-connecting-ip") ?? "unknown";
  const limited = await checkRateLimit(env, {
    key: `${endpoint.id}:${sourceIp}`,
    count: rate.count,
    windowSeconds: rate.windowSeconds
  });
  if (!limited.ok) {
    return json({ error: "rate_limited", message: "Rate limit exceeded." }, 429, {
      "Retry-After": String(limited.retryAfterSeconds)
    });
  }

  if (!env.SECURITY_WEBHOOK_QUEUE) {
    return json({
      error: "queue_not_configured",
      message: "SECURITY_WEBHOOK_QUEUE binding is required to enqueue verified webhook events."
    }, 503);
  }

  const tenantId = endpoint.tenantId ?? stringMetadata(endpoint.metadata, "tenant_id") ?? "tenant_demo";
  const receivedAt = new Date().toISOString();
  const headers = headersToObject(request.headers);
  const bodyBase64 = arrayBufferToBase64(body.bytes);
  const bodySha256 = await sha256Hex(body.bytes);
  const idempotencyKey = request.headers.get("idempotency-key") ?? request.headers.get("x-request-id") ?? bodySha256;
  const payload: WebhookQueuePayload = {
    endpointId: endpoint.id,
    tenantId,
    provider: endpoint.provider,
    receivedAt,
    headers,
    bodyBase64,
    bodySha256,
    sourceIp,
    topic: endpoint.queueTopic,
    metadata: endpoint.metadata
  };

  await env.SECURITY_WEBHOOK_QUEUE.send({
    event_id: crypto.randomUUID(),
    tenant_id: tenantId,
    type: "security.webhook.received",
    schema_version: 1,
    occurred_at: receivedAt,
    idempotency_key: idempotencyKey,
    payload,
    trace_id: request.headers.get("traceparent") ?? request.headers.get("x-request-id") ?? undefined,
    ...payload
  });

  return json({ ok: true, endpointId: endpoint.id, provider: endpoint.provider }, 202);
}

async function resolveEndpoint(endpointId: string, env: Env): Promise<EndpointConfig | null> {
  const kvEndpoint = await resolveEndpointFromKv(endpointId, env);
  if (kvEndpoint) {
    return kvEndpoint;
  }

  const libsqlEndpoint = await resolveEndpointFromLibsql(endpointId, env);
  if (libsqlEndpoint) {
    return libsqlEndpoint;
  }

  const staticEndpoint = resolveEndpointFromStaticJson(endpointId, env);
  if (staticEndpoint) {
    return staticEndpoint;
  }

  return null;
}

async function resolveEndpointFromKv(endpointId: string, env: Env): Promise<EndpointConfig | null> {
  if (!env.SECURITY_ENDPOINTS) {
    return null;
  }

  const raw = await env.SECURITY_ENDPOINTS.get(`endpoint:${endpointId}`);
  if (!raw) {
    return null;
  }

  return normalizeEndpointConfig(endpointId, JSON.parse(raw));
}

async function resolveEndpointFromLibsql(endpointId: string, env: Env): Promise<EndpointConfig | null> {
  if (!env.LIBSQL_ENDPOINTS_URL) {
    return null;
  }

  const response = await fetch(`${env.LIBSQL_ENDPOINTS_URL.replace(/\/$/, "")}/webhook-endpoints/${encodeURIComponent(endpointId)}`, {
    headers: env.LIBSQL_AUTH_TOKEN ? { Authorization: `Bearer ${env.LIBSQL_AUTH_TOKEN}` } : undefined
  });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`libSQL endpoint lookup failed with HTTP ${response.status}`);
  }

  return normalizeEndpointConfig(endpointId, await response.json());
}

function resolveEndpointFromStaticJson(endpointId: string, env: Env): EndpointConfig | null {
  if (!env.ENDPOINTS_JSON) {
    return null;
  }

  const parsed = JSON.parse(env.ENDPOINTS_JSON) as Record<string, unknown>;
  const raw = parsed[endpointId];
  return raw ? normalizeEndpointConfig(endpointId, raw) : null;
}

function normalizeEndpointConfig(endpointId: string, raw: unknown): EndpointConfig {
  const config = raw as Partial<EndpointConfig>;
  if (!config.provider || !["github", "stripe", "slack", "generic"].includes(config.provider)) {
    throw new Error(`Endpoint '${endpointId}' must configure provider as github, stripe, slack, or generic.`);
  }

  return {
    ...config,
    id: config.id ?? endpointId,
    provider: config.provider
  };
}

function resolveSecret(endpoint: EndpointConfig, env: Env): string | null {
  if (endpoint.secret) {
    return endpoint.secret;
  }

  if (!endpoint.secretEnv) {
    return null;
  }

  const value = env[endpoint.secretEnv];
  return typeof value === "string" && value.length > 0 ? value : null;
}

async function verifyWebhook(
  provider: Provider,
  headers: Headers,
  body: Uint8Array,
  secret: string
): Promise<{ ok: true } | { ok: false; message: string }> {
  switch (provider) {
    case "github":
      return verifyGithub(headers, body, secret);
    case "stripe":
      return verifyStripeStyle(headers, body, secret);
    case "slack":
      return verifySlack(headers, body, secret);
    case "generic":
      return verifyGeneric(headers, body, secret);
  }
}

async function verifyGithub(headers: Headers, body: Uint8Array, secret: string) {
  const actual = headers.get("x-hub-signature-256");
  if (!actual?.startsWith("sha256=")) {
    return { ok: false as const, message: "Missing X-Hub-Signature-256 sha256 signature." };
  }

  const expected = `sha256=${await hmacSha256Hex(secret, body)}`;
  return timingSafeEqual(actual, expected)
    ? { ok: true as const }
    : { ok: false as const, message: "GitHub signature mismatch." };
}

async function verifyStripeStyle(headers: Headers, body: Uint8Array, secret: string) {
  const signature = headers.get("stripe-signature");
  if (!signature) {
    return { ok: false as const, message: "Missing Stripe-Signature header." };
  }

  const parts = parseCommaHeader(signature);
  const timestamp = parts.get("t");
  const signatures = parts.getAll("v1");
  if (!timestamp || signatures.length === 0) {
    return { ok: false as const, message: "Stripe-Signature must include t and v1 fields." };
  }

  if (!timestampWithinTolerance(timestamp, 300)) {
    return { ok: false as const, message: "Stripe-style timestamp is outside tolerance." };
  }

  const payload = concatBytes(TEXT_ENCODER.encode(`${timestamp}.`), body);
  const expected = await hmacSha256Hex(secret, payload);
  return signatures.some((candidate) => timingSafeEqual(candidate, expected))
    ? { ok: true as const }
    : { ok: false as const, message: "Stripe-style signature mismatch." };
}

async function verifySlack(headers: Headers, body: Uint8Array, secret: string) {
  const timestamp = headers.get("x-slack-request-timestamp");
  const actual = headers.get("x-slack-signature");
  if (!timestamp || !actual?.startsWith("v0=")) {
    return { ok: false as const, message: "Missing Slack timestamp or v0 signature header." };
  }

  if (!timestampWithinTolerance(timestamp, 300)) {
    return { ok: false as const, message: "Slack timestamp is outside tolerance." };
  }

  const payload = concatBytes(TEXT_ENCODER.encode(`v0:${timestamp}:`), body);
  const expected = `v0=${await hmacSha256Hex(secret, payload)}`;
  return timingSafeEqual(actual, expected)
    ? { ok: true as const }
    : { ok: false as const, message: "Slack signature mismatch." };
}

async function verifyGeneric(headers: Headers, body: Uint8Array, secret: string) {
  const actual = headers.get("x-webhook-signature") ?? headers.get("x-signature");
  if (!actual) {
    return { ok: false as const, message: "Missing X-Webhook-Signature or X-Signature header." };
  }

  const expectedHex = await hmacSha256Hex(secret, body);
  const expected = actual.startsWith("sha256=") ? `sha256=${expectedHex}` : expectedHex;
  return timingSafeEqual(actual, expected)
    ? { ok: true as const }
    : { ok: false as const, message: "Generic signature mismatch." };
}

async function checkRateLimit(env: Env, input: RateLimitRequest): Promise<{ ok: true } | { ok: false; retryAfterSeconds: number }> {
  if (!env.RATE_LIMITER) {
    return { ok: true };
  }

  const id = env.RATE_LIMITER.idFromName(input.key);
  const response = await env.RATE_LIMITER.get(id).fetch("https://rate-limit/check", {
    method: "POST",
    body: JSON.stringify(input)
  });
  if (!response.ok) {
    throw new Error(`Rate limiter failed with HTTP ${response.status}`);
  }

  return response.json();
}

export class RateLimiter {
  constructor(private readonly state: DurableObjectState) {}

  async fetch(request: Request): Promise<Response> {
    if (request.method !== "POST") {
      return json({ error: "method_not_allowed" }, 405);
    }

    const input = await request.json<RateLimitRequest>();
    const now = Date.now();
    const windowMs = Math.max(1, input.windowSeconds) * 1000;
    const record = await this.state.storage.get<{ resetAt: number; used: number }>(input.key);
    const current = !record || record.resetAt <= now ? { resetAt: now + windowMs, used: 0 } : record;
    current.used += 1;
    await this.state.storage.put(input.key, current);

    if (current.used > input.count) {
      return json({ ok: false, retryAfterSeconds: Math.max(1, Math.ceil((current.resetAt - now) / 1000)) });
    }

    return json({ ok: true });
  }
}

async function readBodyWithCap(request: Request, capBytes: number): Promise<{ ok: true; bytes: Uint8Array } | { ok: false }> {
  if (!request.body) {
    return { ok: true, bytes: new Uint8Array() };
  }

  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    total += value.byteLength;
    if (total > capBytes) {
      await reader.cancel();
      return { ok: false };
    }
    chunks.push(value);
  }

  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return { ok: true, bytes };
}

function intVar(value: string | undefined, fallback: number): number {
  const parsed = value ? Number.parseInt(value, 10) : Number.NaN;
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function headersToObject(headers: Headers): Record<string, string> {
  const output: Record<string, string> = {};
  headers.forEach((value, key) => {
    output[key] = value;
  });
  return output;
}

function stringMetadata(metadata: Record<string, unknown> | undefined, key: string): string | null {
  const value = metadata?.[key];
  return typeof value === "string" && value.length > 0 ? value : null;
}

async function hmacSha256Hex(secret: string, payload: Uint8Array): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    TEXT_ENCODER.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  return bytesToHex(new Uint8Array(await crypto.subtle.sign("HMAC", key, payload)));
}

async function sha256Hex(payload: Uint8Array): Promise<string> {
  return bytesToHex(new Uint8Array(await crypto.subtle.digest("SHA-256", payload)));
}

function bytesToHex(bytes: Uint8Array): string {
  let output = "";
  for (const byte of bytes) {
    output += HEX[byte >> 4] + HEX[byte & 0x0f];
  }
  return output;
}

function timingSafeEqual(a: string, b: string): boolean {
  const left = TEXT_ENCODER.encode(a);
  const right = TEXT_ENCODER.encode(b);
  let diff = left.length ^ right.length;
  const length = Math.max(left.length, right.length);
  for (let i = 0; i < length; i += 1) {
    diff |= (left[i] ?? 0) ^ (right[i] ?? 0);
  }
  return diff === 0;
}

function timestampWithinTolerance(timestamp: string, toleranceSeconds: number): boolean {
  const seconds = Number.parseInt(timestamp, 10);
  if (!Number.isFinite(seconds)) {
    return false;
  }

  return Math.abs(Math.floor(Date.now() / 1000) - seconds) <= toleranceSeconds;
}

function concatBytes(left: Uint8Array, right: Uint8Array): Uint8Array {
  const output = new Uint8Array(left.byteLength + right.byteLength);
  output.set(left, 0);
  output.set(right, left.byteLength);
  return output;
}

function arrayBufferToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

function parseCommaHeader(header: string): Map<string, string> & { getAll(name: string): string[] } {
  const values = new Map<string, string[]>();
  for (const part of header.split(",")) {
    const [name, ...rest] = part.split("=");
    if (!name || rest.length === 0) {
      continue;
    }
    const key = name.trim();
    const value = rest.join("=").trim();
    values.set(key, [...(values.get(key) ?? []), value]);
  }

  return {
    get(name: string) {
      return values.get(name)?.[0] ?? null;
    },
    getAll(name: string) {
      return values.get(name) ?? [];
    }
  } as Map<string, string> & { getAll(name: string): string[] };
}

function json(body: unknown, status = 200, headers?: HeadersInit): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...headers
    }
  });
}
