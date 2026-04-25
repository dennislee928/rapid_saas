import { authenticateApiKey, assertScope, clampAllowedBbox, mintSubscribeToken, verifySubscribeToken, type ApiPrincipal } from "./auth";
import { GroupHub } from "./group-hub";
import { normalizePosition, parseBboxParam, type Bbox, type PositionInput } from "./protocol";

export { GroupHub };

export interface Env {
  GROUP_HUB?: DurableObjectNamespace;
  DEMO_API_KEYS?: string;
  SUBSCRIBE_JWT_SECRET?: string;
  COALESCE_MS?: string;
  DEFAULT_RETENTION_TTL_SECONDS?: string;
  SUBSCRIBE_TOKEN_TTL_SECONDS?: string;
  LIBSQL_URL?: string;
  LIBSQL_AUTH_TOKEN?: string;
}

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
      service: "realtime-geospatial-api",
      bindings: {
        groupHub: Boolean(env.GROUP_HUB),
        libsql: Boolean(env.LIBSQL_URL),
        demoKeys: Boolean(env.DEMO_API_KEYS)
      }
    });
  }

  if (request.method === "GET" && url.pathname === "/dashboard") {
    return dashboardHtml();
  }

  if (request.method === "GET" && url.pathname === "/dashboard/api/summary") {
    return json({
      live_ccu: 0,
      publish_rate: 0,
      top_groups: [],
      note: "Stub endpoint; wire to Analytics Engine and Turso usage_buckets for production."
    });
  }

  if (request.method === "POST" && url.pathname === "/v1/positions") {
    return publishOne(request, env);
  }

  if (request.method === "POST" && url.pathname === "/v1/positions/batch") {
    return publishBatch(request, env);
  }

  if (request.method === "POST" && url.pathname === "/v1/subscribe-token") {
    return issueSubscribeToken(request, env);
  }

  const wsMatch = url.pathname.match(/^\/v1\/groups\/([^/]+)\/ws$/);
  if (request.method === "GET" && wsMatch) {
    return subscribe(request, env, decodeURIComponent(wsMatch[1]));
  }

  return json({ error: "not_found" }, 404);
}

async function publishOne(request: Request, env: Env): Promise<Response> {
  const principal = await requireApiPrincipal(request, env);
  if (principal instanceof Response) {
    return principal;
  }
  let position: PositionInput;
  try {
    position = normalizePosition(await request.json());
  } catch (error) {
    return json({ error: "bad_request", message: messageOf(error) }, 400);
  }
  if (!assertScope(principal, "publish", position.group_id)) {
    return json({ error: "forbidden", message: "API key cannot publish to this group." }, 403);
  }
  await publishToGroups(env, principal, [position]);
  return new Response(null, { status: 204 });
}

async function publishBatch(request: Request, env: Env): Promise<Response> {
  const principal = await requireApiPrincipal(request, env);
  if (principal instanceof Response) {
    return principal;
  }
  try {
    const body = await request.json<{ items?: unknown[] }>();
    if (!Array.isArray(body.items) || body.items.length === 0 || body.items.length > 1000) {
      return json({ error: "bad_request", message: "items must contain 1 to 1000 positions." }, 400);
    }
    const positions = body.items.map((item) => normalizePosition(item));
    const forbidden = positions.find((item) => !assertScope(principal, "publish", item.group_id));
    if (forbidden) {
      return json({ error: "forbidden", message: `API key cannot publish to group ${forbidden.group_id}.` }, 403);
    }
    await publishToGroups(env, principal, positions);
    return json({ ok: true, accepted: positions.length }, 202);
  } catch (error) {
    return json({ error: "bad_request", message: messageOf(error) }, 400);
  }
}

async function issueSubscribeToken(request: Request, env: Env): Promise<Response> {
  const principal = await requireApiPrincipal(request, env);
  if (principal instanceof Response) {
    return principal;
  }
  if (!assertScope(principal, "admin")) {
    return json({ error: "forbidden", message: "subscribe-token requires an admin-scoped API key." }, 403);
  }

  const body = await request.json<{ group_id?: string; ttl_s?: number; allowed_filters?: { bbox?: Bbox } }>();
  if (!body.group_id || !/^[A-Za-z0-9_.:-]{1,128}$/.test(body.group_id)) {
    return json({ error: "bad_request", message: "group_id is required." }, 400);
  }
  const secret = env.SUBSCRIBE_JWT_SECRET ?? "dev-only-change-me";
  const ttl = body.ttl_s ?? intVar(env.SUBSCRIBE_TOKEN_TTL_SECONDS, 300);
  return json(await mintSubscribeToken({
    tenant_id: principal.tenant_id,
    group_id: body.group_id,
    ttl_s: ttl,
    allowed_filters: body.allowed_filters
  }, secret));
}

async function subscribe(request: Request, env: Env, groupId: string): Promise<Response> {
  if (request.headers.get("upgrade")?.toLowerCase() !== "websocket") {
    return json({ error: "upgrade_required" }, 426);
  }
  if (!env.GROUP_HUB) {
    return json({ error: "durable_object_not_configured" }, 503);
  }
  const url = new URL(request.url);
  const token = url.searchParams.get("token");
  if (!token) {
    return json({ error: "missing_token" }, 401);
  }
  let claims;
  try {
    claims = await verifySubscribeToken(token, env.SUBSCRIBE_JWT_SECRET ?? "dev-only-change-me");
  } catch (error) {
    return json({ error: "unauthorized", message: messageOf(error) }, 401);
  }
  if (claims.group_id !== groupId) {
    return json({ error: "forbidden", message: "Token group does not match requested group." }, 403);
  }

  let bbox: Bbox | undefined;
  try {
    bbox = clampAllowedBbox(parseBboxParam(url.searchParams.get("bbox")), claims.allowed_filters?.bbox);
  } catch (error) {
    return json({ error: "bad_request", message: messageOf(error) }, 400);
  }

  const target = new URL("https://group-hub/internal/subscribe");
  target.searchParams.set("tenant_id", claims.tenant_id);
  target.searchParams.set("group_id", groupId);
  target.searchParams.set("snapshot", url.searchParams.get("snapshot") ?? "true");
  if (bbox) {
    target.searchParams.set("bbox", JSON.stringify(bbox));
  }

  return groupStub(env, claims.tenant_id, groupId).fetch(new Request(target, {
    headers: request.headers
  }));
}

async function publishToGroups(env: Env, principal: ApiPrincipal, positions: PositionInput[]): Promise<void> {
  if (!env.GROUP_HUB) {
    return;
  }
  const byGroup = new Map<string, PositionInput[]>();
  for (const position of positions) {
    byGroup.set(position.group_id, [...(byGroup.get(position.group_id) ?? []), position]);
  }
  await Promise.all([...byGroup].map(([groupId, items]) => {
    return groupStub(env, principal.tenant_id, groupId).fetch("https://group-hub/internal/publish", {
      method: "POST",
      body: JSON.stringify({ items })
    });
  }));
}

function groupStub(env: Env, tenantId: string, groupId: string): DurableObjectStub {
  if (!env.GROUP_HUB) {
    throw new Error("GROUP_HUB binding is not configured.");
  }
  const id = env.GROUP_HUB.idFromName(`${tenantId}::${groupId}`);
  return env.GROUP_HUB.get(id);
}

async function requireApiPrincipal(request: Request, env: Env): Promise<ApiPrincipal | Response> {
  const principal = await authenticateApiKey(request, env);
  if (!principal) {
    return json({ error: "unauthorized", message: "Missing or invalid API key." }, 401);
  }
  return principal;
}

function dashboardHtml(): Response {
  return new Response(`<!doctype html>
<html lang="en">
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Realtime Geo API Dashboard</title>
<style>
  body { margin: 0; font-family: ui-sans-serif, system-ui; background: #07130f; color: #edf7ef; }
  main { max-width: 960px; margin: 0 auto; padding: 48px 24px; }
  .hero { border: 1px solid #254434; border-radius: 28px; padding: 32px; background: radial-gradient(circle at top right, #1f6b48, transparent 38%), #0d1e17; }
  .grid { display: grid; gap: 16px; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); margin-top: 24px; }
  .card { border: 1px solid #254434; border-radius: 18px; padding: 18px; background: #0b1913; }
  strong { display: block; font-size: 32px; }
</style>
<main>
  <section class="hero">
    <p>Rapid SaaS MVP</p>
    <h1>Realtime Geospatial API</h1>
    <p>Dashboard stub for tenant creation, API key management, live CCU, write rate, and group health.</p>
  </section>
  <section class="grid">
    <div class="card"><span>Live CCU</span><strong>0</strong></div>
    <div class="card"><span>Publish rate</span><strong>0/s</strong></div>
    <div class="card"><span>Groups</span><strong>0</strong></div>
  </section>
</main>
</html>`, {
    headers: { "content-type": "text/html; charset=utf-8" }
  });
}

function intVar(value: string | undefined, fallback: number): number {
  const parsed = value ? Number.parseInt(value, 10) : Number.NaN;
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json; charset=utf-8" }
  });
}
