import { hmacHex, sha256Hex, signJwt, verifyJwt } from "./crypto";
import { renderDmcaDraft, type DmcaDraftInput } from "./dmca";

export interface Env {
  AEGIS_ENV: string;
  PLATFORM_HMAC_SECRET: string;
  DEFAULT_TENANT_SECRET: string;
  POSTMARK_API_TOKEN?: string;
}

interface TenantConfig {
  id: string;
  slug: string;
  origin: string;
  jwtSecret: string;
  jwtKid: string;
  cookieName: string;
  verifyBaseUrl: string;
  csp: string;
}

function json(data: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(data, null, 2), {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
}

function getCookie(request: Request, name: string): string | undefined {
  const cookie = request.headers.get("cookie") ?? "";
  return cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(`${name}=`))
    ?.slice(name.length + 1);
}

function tenantFromRequest(request: Request, env: Env): TenantConfig {
  const url = new URL(request.url);
  const tenant = url.searchParams.get("tenant") || request.headers.get("x-aegis-tenant") || "tenant_demo";
  return {
    id: tenant,
    slug: tenant.replace(/^tenant_/, ""),
    origin: request.headers.get("x-aegis-origin") || "https://origin.example.test",
    jwtSecret: env.DEFAULT_TENANT_SECRET || "local-dev-secret-with-32-byte-minimum",
    jwtKid: "k_2026_04",
    cookieName: "gk_session",
    verifyBaseUrl: `${url.origin}/verify`,
    csp: "default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'none'",
  };
}

function looksLikeBot(request: Request): boolean {
  const accept = request.headers.get("accept");
  const ua = request.headers.get("user-agent");
  const cfBotScore = request.headers.get("cf-bot-score");
  return !accept || !ua || cfBotScore === "1";
}

function withCsp(response: Response, csp: string): Response {
  const headers = new Headers(response.headers);
  const contentType = headers.get("content-type") || "";
  if (contentType.includes("text/html")) {
    headers.set("content-security-policy", csp);
  }
  return new Response(response.body, { status: response.status, statusText: response.statusText, headers });
}

async function redirectToVerify(request: Request, tenant: TenantConfig, env: Env): Promise<Response> {
  const returnUrl = new URL(request.url);
  const state = await hmacHex(env.PLATFORM_HMAC_SECRET || tenant.jwtSecret, `${tenant.id}:${returnUrl.toString()}:${Date.now()}`);
  const verifyUrl = new URL(tenant.verifyBaseUrl);
  verifyUrl.searchParams.set("tenant", tenant.id);
  verifyUrl.searchParams.set("return", returnUrl.toString());
  verifyUrl.searchParams.set("state", state);

  return new Response(null, {
    status: 302,
    headers: {
      location: verifyUrl.toString(),
      "set-cookie": `gk_state=${state}; Path=/; Max-Age=180; Secure; HttpOnly; SameSite=Strict`,
      "cache-control": "no-store",
    },
  });
}

async function handleProxy(request: Request, env: Env): Promise<Response> {
  const tenant = tenantFromRequest(request, env);
  if (looksLikeBot(request)) return new Response("bot-like request blocked", { status: 403 });

  const token = getCookie(request, tenant.cookieName);
  if (!token) return redirectToVerify(request, tenant, env);

  try {
    await verifyJwt(token, tenant.jwtSecret, tenant.id);
  } catch {
    return redirectToVerify(request, tenant, env);
  }

  const incoming = new URL(request.url);
  const origin = new URL(tenant.origin);
  origin.pathname = incoming.pathname;
  origin.search = incoming.search;

  const upstream = await fetch(new Request(origin, request));
  return withCsp(upstream, tenant.csp);
}

async function handleVerify(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const tenant = tenantFromRequest(request, env);
  const state = url.searchParams.get("state") || "";
  const returnUrl = url.searchParams.get("return") || "/";
  const html = `<!doctype html>
<html><head><title>GateKeep Verify</title></head>
<body>
  <h1>GateKeep hosted verification stub</h1>
  <p>Operational placeholder only. This is not legal advice or Ofcom certification.</p>
  <p>Tenant: ${tenant.id}</p>
  <form method="post" action="/callback/${tenant.id}?state=${encodeURIComponent(state)}&return=${encodeURIComponent(returnUrl)}">
    <input type="hidden" name="vendor_verification_id" value="persona_stub_verification" />
    <button type="submit">Simulate successful adult age assurance</button>
  </form>
</body></html>`;
  return new Response(html, { headers: { "content-type": "text/html; charset=utf-8", "cache-control": "no-store" } });
}

async function handleCallback(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const tenantId = url.pathname.split("/").pop() || "tenant_demo";
  const tenant = tenantFromRequest(new Request(`${url.origin}/?tenant=${tenantId}`), env);
  const state = url.searchParams.get("state");
  const stateCookie = getCookie(request, "gk_state");
  if (!state || !stateCookie || state !== stateCookie) return json({ error: "state_mismatch" }, { status: 400 });

  const form = request.method === "POST" ? await request.formData() : new FormData();
  const vendorVerificationID = String(form.get("vendor_verification_id") || "persona_stub_verification");
  const now = Math.floor(Date.now() / 1000);
  const payload = {
    iss: "gatekeep.aegis.adult",
    aud: tenant.id,
    sub: `session_${crypto.randomUUID()}`,
    iat: now,
    exp: now + 60 * 60 * 24 * 30,
    kid: tenant.jwtKid,
    vrf: "persona",
    vid_hash: await sha256Hex(`${vendorVerificationID}:${tenant.jwtSecret}`),
    ag: true,
    geo: request.headers.get("cf-ipcountry") || "GB",
    v: 1,
  };
  const token = await signJwt(payload, tenant.jwtSecret);

  return new Response(null, {
    status: 302,
    headers: {
      location: url.searchParams.get("return") || "/",
      "set-cookie": `${tenant.cookieName}=${token}; Path=/; Max-Age=2592000; Secure; HttpOnly; SameSite=Lax`,
      "cache-control": "no-store",
    },
  });
}

async function handleUpload(request: Request): Promise<Response> {
  if (request.method !== "POST") return json({ error: "method_not_allowed" }, { status: 405 });
  const body = await request.arrayBuffer();
  const contentHash = await sha256Hex(body);
  return json({
    asset_id: `asset_${crypto.randomUUID()}`,
    content_sha256: contentHash,
    workflow: "queued_for_hasher",
    source_retention: "delete_original_within_24h",
    note: "MVP stub: content is not persisted by this Worker.",
  });
}

async function handleTakedownDraft(request: Request): Promise<Response> {
  if (request.method !== "POST") return json({ error: "method_not_allowed" }, { status: 405 });
  const input = (await request.json()) as Partial<DmcaDraftInput>;
  const draft = renderDmcaDraft({
    toAddress: input.toAddress || "abuse@example.test",
    fromAddress: input.fromAddress || "takedowns@demo.reclaim.aegis.adult",
    replyTo: input.replyTo || "rights-holder@example.test",
    claimantName: input.claimantName || "Rights Holder",
    workDescription: input.workDescription || "Customer-provided copyrighted work description.",
    referenceUrl: input.referenceUrl || "https://customer.example/work",
    candidateUrl: input.candidateUrl || "https://candidate.example/leak",
    assetLabel: input.assetLabel || "asset label",
    matchScore: input.matchScore ?? 0.92,
    detectionMethod: input.detectionMethod || "pHash review threshold",
    detectedAt: input.detectedAt || new Date().toISOString(),
    signature: input.signature || "Typed Name",
  });
  return json({ status: "draft_requires_human_review", auto_send: false, draft });
}

export async function handleRequest(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  if (url.pathname === "/healthz") return json({ ok: true, service: "aegis-adult-worker" });
  if (url.pathname === "/verify") return handleVerify(request, env);
  if (url.pathname.startsWith("/callback/")) return handleCallback(request, env);
  if (url.pathname === "/api/reclaim/upload") return handleUpload(request);
  if (url.pathname === "/api/reclaim/takedown-draft") return handleTakedownDraft(request);
  return handleProxy(request, env);
}

export default {
  fetch: handleRequest,
};

